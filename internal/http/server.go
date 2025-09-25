package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wealth-warden/internal/bootstrap"
	appConfig "wealth-warden/pkg/config"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	"github.com/tavsec/gin-healthcheck/config"
	"go.uber.org/zap"
)

type Server struct {
	Router *gin.Engine
	server *http.Server
	logger *zap.Logger
}

func NewServer(container *bootstrap.Container, logger *zap.Logger) *Server {

	router := NewRouter(container, logger)

	addr := net.JoinHostPort(container.Config.Host, container.Config.HttpServer.Port)

	return &Server{
		Router: router,
		logger: logger,
		server: &http.Server{
			Addr:    addr,
			Handler: router,
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

	if container.Config.Release {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// Logging & recovery
	r.Use(container.Middleware.ErrorLogger())
	r.Use(ginzap.RecoveryWithZap(logger, true))

	// Health check (DB)
	sqlDB, err := container.DB.DB()
	if err != nil {
		panic(err)
	}

	sqlCheck := checks.SqlCheck{Sql: sqlDB}
	healthcheck.New(r, config.DefaultConfig(), []checks.Check{sqlCheck})

	// CORS
	c := defineCORS(container.Config)
	r.Use(cors.New(c))

	routeInitializer := NewRouteInitializerHTTP(r, container)
	routeInitializer.InitEndpoints()

	return r
}

func defineCORS(cfg *appConfig.Config) cors.Config {
	origins := cfg.CORS.AllowedOrigins

	allowList := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowList[strings.TrimSpace(o)] = struct{}{}
	}

	// Optional wildcard support (e.g., *.example.com)
	var wildcardSuffixes []string
	for _, sfx := range cfg.CORS.WildcardSuffixes {
		wildcardSuffixes = append(wildcardSuffixes, strings.ToLower(sfx))
	}
	return cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With", "wealth-warden-client"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			// Compare exact allow-list (scheme + host + optional port)
			if _, ok := allowList[origin]; ok {
				return true
			}
			// Optional wildcard suffix check
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := strings.ToLower(u.Hostname())
			for _, sfx := range wildcardSuffixes {
				if strings.HasSuffix(host, sfx) {
					// Optionally enforce scheme:
					if len(cfg.CORS.AllowedSchemes) > 0 {
						if u.Scheme == "" {
							return false
						}
						ok := false
						for _, sch := range cfg.CORS.AllowedSchemes {
							if u.Scheme == sch {
								ok = true
								break
							}
						}
						return ok
					}
					return true
				}
			}
			return false
		},
	}
}
