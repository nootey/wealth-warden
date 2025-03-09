package http

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"wealth-warden/internal/handlers"
	"wealth-warden/internal/http/endpoints"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
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
	BudgetService            *services.BudgetService
	SavingsService           *services.SavingsService
}

func NewRouteInitializer(router *gin.Engine, cfg *config.Config, db *gorm.DB) *RouteInitializer {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg)

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	recActionRepo := repositories.NewReoccurringActionsRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, userRepo, loggingService, webClientMiddleware)
	userService := services.NewUserService(cfg, userRepo)
	recActionService := services.NewReoccurringActionService(recActionRepo, authService, loggingService)
	inflowService := services.NewInflowService(cfg, authService, loggingService, recActionService, inflowRepo)
	outflowService := services.NewOutflowService(cfg, authService, loggingService, recActionService, outflowRepo)
	budgetService := services.NewBudgetService(cfg, authService, loggingService, budgetRepo)
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
		BudgetService:            budgetService,
		SavingsService:           savingsService,
	}
}

func (r *RouteInitializer) InitEndpoints() {
	apiPrefixV1 := "/api/v1"

	r.Router.GET("/", rootHandler)
	r.Router.GET(apiPrefixV1+"/health", func(c *gin.Context) {
		healthCheck(c)
	})

	authHandler := handlers.NewAuthHandler(r.AuthService)
	userHandler := handlers.NewUserHandler(r.UserService)
	inflowHandler := handlers.NewInflowHandler(r.InflowService)
	outflowHandler := handlers.NewOutflowHandler(r.OutflowService)
	loggingHandler := handlers.NewLoggingHandler(r.LoggingService)
	recActionHandler := handlers.NewReoccurringActionHandler(r.ReoccurringActionService)
	budgetHandler := handlers.NewBudgetHandler(r.BudgetService)
	savingsHandler := handlers.NewSavingsHandler(r.SavingsService)

	// Protected routes
	authGroup := r.Router.Group(apiPrefixV1, r.AuthService.WebClientMiddleware.WebClientAuthentication())
	{
		authRoutes := authGroup.Group("/auth")
		endpoints.AuthRoutes(authRoutes, authHandler)

		userRoutes := authGroup.Group("/users")
		endpoints.UserRoutes(userRoutes, userHandler)

		inflowRoutes := authGroup.Group("/inflows")
		endpoints.InflowRoutes(inflowRoutes, inflowHandler)

		outflowRoutes := authGroup.Group("/outflows")
		endpoints.OutflowRoutes(outflowRoutes, outflowHandler)

		loggingRoutes := authGroup.Group("/logs")
		endpoints.LoggingRoutes(loggingRoutes, loggingHandler)

		reoccurringRoutes := authGroup.Group("/reoccurring")
		endpoints.RecActionRoutes(reoccurringRoutes, recActionHandler)

		budgetRoutes := authGroup.Group("/budget")
		endpoints.BudgetRoutes(budgetRoutes, budgetHandler)

		savingsRoutes := authGroup.Group("/savings")
		endpoints.SavingsRoutes(savingsRoutes, savingsHandler)
	}

	// Public routes
	publicGroup := r.Router.Group(apiPrefixV1)
	{
		publicAuthRoutes := publicGroup.Group("/auth")
		endpoints.PublicAuthRoutes(publicAuthRoutes, authHandler)
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
