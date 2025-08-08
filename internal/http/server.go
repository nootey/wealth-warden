package http

import (
	"context"
	"errors"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	"github.com/tavsec/gin-healthcheck/config"
	"go.uber.org/zap"
	"net/http"
	"time"
	"wealth-warden/internal/bootstrap"
)

type Server struct {
	Router *gin.Engine
	server *http.Server
	logger *zap.Logger
}

func NewServer(container *bootstrap.Container, logger *zap.Logger) *Server {

	router := NewRouter(container, logger)

	addr := container.Config.Host + ":" + container.Config.HttpServer.Port

	return &Server{
		Router: router,
		logger: logger,
		server: &http.Server{
			Addr: addr,
		},
	}
}

func (s *Server) Start() {

	s.logger.Info("Starting the server")

	s.server.Handler = s.Router.Handler()

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal("Failed to listen and serve", zap.Error(err))
		}
	}()
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down the server")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func NewRouter(container *bootstrap.Container, logger *zap.Logger) *gin.Engine {

	var r *gin.Engine
	var domainProtocol string

	if container.Config.Release {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		domainProtocol = "https://"

	} else {
		r = gin.Default()
		domainProtocol = "http://"
	}

	sqlDB, err := container.DB.DB()
	if err != nil {
		panic(err)
	}

	sqlCheck := checks.SqlCheck{Sql: sqlDB}
	healthcheck.New(r, config.DefaultConfig(), []checks.Check{sqlCheck})

	// Setup CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		domainProtocol + container.Config.WebClient.Domain,
		domainProtocol + container.Config.WebClient.Domain + ":" + container.Config.WebClient.Port,
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "wealth-warden-client"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Register custom logger middleware BEFORE recovery
	r.Use(container.Middleware.ErrorLogger())

	// Gin's recovery (to handle panics)
	r.Use(ginzap.RecoveryWithZap(logger, true))

	routeInitializer := NewRouteInitializerHTTP(r, container)
	routeInitializer.InitEndpoints()

	return r
}
