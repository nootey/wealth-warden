package http

import (
	"context"
	"errors"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
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

	router := NewRouter(container)

	addr := container.Config.Host + ":" + container.Config.HttpServerPort

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

	s.Router.Use(ginzap.Ginzap(s.logger, time.RFC3339, true), ginzap.RecoveryWithZap(s.logger, true))

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

func NewRouter(container *bootstrap.Container) *gin.Engine {

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

	// Setup CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		domainProtocol + container.Config.WebClientDomain,
		domainProtocol + container.Config.WebClientDomain + ":" + container.Config.WebClientPort,
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "wealth-warden-client"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	r.Use(gin.Recovery())

	routeInitializer := NewRouteInitializerHTTP(r, container)
	routeInitializer.InitEndpoints()

	return r
}
