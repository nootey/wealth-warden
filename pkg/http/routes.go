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
	"wealth-warden/pkg/http/endpoints"
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
	SavingsService           *services.SavingsService
}

func NewRouteInitializer(router *gin.Engine, cfg *config.Config, db *gorm.DB) *RouteInitializer {
	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	recActionRepo := repositories.NewReoccurringActionsRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(userRepo, loggingService)
	userService := services.NewUserService(cfg, userRepo)
	recActionService := services.NewReoccurringActionService(recActionRepo, authService, loggingService)
	inflowService := services.NewInflowService(cfg, authService, loggingService, recActionService, inflowRepo)
	outflowService := services.NewOutflowService(cfg, authService, loggingService, recActionService, outflowRepo)
	savingsService := services.NewSavingsService(cfg, authService, loggingService, recActionService, savingsRepo)

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
		SavingsService:           savingsService,
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
	savingsHandler := handlers.NewSavingsHandler(r.SavingsService)

	// Protected routes
	authGroup := r.Router.Group(apiPrefixV1, middleware.WebClientAuthentication())
	{
		endpoints.AuthRoutes(authGroup, authHandler)
		endpoints.UserRoutes(authGroup, userHandler)
		endpoints.InflowRoutes(authGroup, inflowHandler)
		endpoints.OutflowRoutes(authGroup, outflowHandler)
		endpoints.LoggingRoutes(authGroup, loggingHandler)
		endpoints.RecActionRoutes(authGroup, recActionHandler)
		endpoints.SavingsRoutes(authGroup, savingsHandler)
	}

	// Public routes
	publicGroup := r.Router.Group(apiPrefixV1)
	{
		endpoints.ExposedAuthRoutes(publicGroup, authHandler)
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
