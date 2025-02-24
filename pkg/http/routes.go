package http

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"wealth-warden/internal/handlers"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/middleware"
)

type RouteInitializer struct {
	Router                   *gin.Engine
	Config                   *config.Config
	DB                       *gorm.DB
	AuthService              *services.AuthService
	UserService              *services.UserService
	InflowService            *services.InflowService
	OutflowService           *services.OutflowService
	LoggingService           *services.LoggingService
	ReoccurringActionService *services.ReoccurringActionService
}

func NewRouteInitializer(router *gin.Engine, cfg *config.Config, db *gorm.DB) *RouteInitializer {
	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	recActionRepo := repositories.NewReoccurringActionsRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(userRepo, loggingService)
	userService := services.NewUserService(cfg, userRepo)
	recActionService := services.NewReoccurringActionService(recActionRepo, authService)
	inflowService := services.NewInflowService(cfg, authService, loggingService, recActionService, inflowRepo)
	outflowService := services.NewOutflowService(cfg, authService, loggingService, recActionService, outflowRepo)

	return &RouteInitializer{
		Router:                   router,
		Config:                   cfg,
		DB:                       db,
		AuthService:              authService,
		UserService:              userService,
		InflowService:            inflowService,
		OutflowService:           outflowService,
		LoggingService:           loggingService,
		ReoccurringActionService: recActionService,
	}
}

func (r *RouteInitializer) InitEndpoints() {
	apiPrefixV1 := "/api/v1"

	r.Router.GET("/", rootHandler)
	r.Router.GET(apiPrefixV1+"/health", func(c *gin.Context) {
		healthCheck(c)
	})

	authHandler := handlers.NewAuthHandler(r.Config, r.AuthService)
	userHandler := handlers.NewUserHandler(r.UserService)
	inflowHandler := handlers.NewInflowHandler(r.InflowService)
	outflowHandler := handlers.NewOutflowHandler(r.OutflowService)
	loggingHandler := handlers.NewLoggingHandler(r.LoggingService)
	recActionHandler := handlers.NewReoccurringActionHandler(r.ReoccurringActionService)

	// Protected routes
	authGroup := r.Router.Group(apiPrefixV1, middleware.WebClientAuthentication())
	{
		authRoutes(authGroup, authHandler)
		userRoutes(authGroup, userHandler)
		inflowRoutes(authGroup, inflowHandler)
		outflowRoutes(authGroup, outflowHandler)
		loggingRoutes(authGroup, loggingHandler)
		recActionRoutes(authGroup, recActionHandler)
	}

	// Public routes
	publicGroup := r.Router.Group(apiPrefixV1)
	{
		exposedAuthRoutes(publicGroup, authHandler)
	}
}

// Root handler for basic server check
func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}

// Health check handler function
func healthCheck(c *gin.Context) {
	httpHealthStatus := "healthy"
	dbStatus := "healthy"

	// Check database connection
	err := database.PingMysqlDatabase()
	if err != nil {
		dbStatus = "unhealthy"
		httpHealthStatus = "degraded"
	}

	statusCode := http.StatusOK
	if httpHealthStatus == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status": gin.H{
			"api": gin.H{"http": httpHealthStatus},
			"services": gin.H{
				"database": gin.H{"mysql": dbStatus},
			},
		},
	})
}
