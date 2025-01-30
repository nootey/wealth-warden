package http

import (
	"context"
	"errors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"time"
	"wealth-warden/server/pkg/config"
)

type Server struct {
	Router *gin.Engine
	server *http.Server
	logger *zap.Logger
}

func NewServer(cfg *config.Config, logger *zap.Logger, dbClient *gorm.DB) *Server {
	// Create a Router and attach middleware
	router := NewRouter(cfg, dbClient)

	return &Server{
		Router: router,
		logger: logger.Named("http-server"),
		server: &http.Server{
			Addr: ":" + cfg.HttpServerPort,
		},
	}
}

func (s *Server) Start() {
	s.logger.Info("Starting the server")

	// Attach recovery & log middleware
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

func NewRouter(cfg *config.Config, dbClient *gorm.DB) *gin.Engine {

	var router *gin.Engine

	if cfg.Release == "production" {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()

	} else {
		router = gin.Default()
	}

	// Global middlewares
	router.Use(gin.Recovery())

	// Initialize API routes
	InitEndpoints(router, cfg, dbClient)

	return router
}
