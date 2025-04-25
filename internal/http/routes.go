package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/handlers"
	"wealth-warden/internal/http/endpoints"
	"wealth-warden/pkg/database"
)

type RouteInitializer struct {
	Router    *gin.Engine
	Container *bootstrap.Container
}

func NewRouteInitializerHTTP(r *gin.Engine, container *bootstrap.Container) *RouteInitializer {

	return &RouteInitializer{
		Router:    r,
		Container: container,
	}
}

func (r *RouteInitializer) InitEndpoints() {
	apiPrefixV1 := "/api/v1"

	r.Router.GET("/", rootHandler)
	r.Router.GET(apiPrefixV1+"/health", func(c *gin.Context) {
		healthCheck(c)
	})

	authHandler := handlers.NewAuthHandler(r.Container.AuthService)
	userHandler := handlers.NewUserHandler(r.Container.UserService)
	inflowHandler := handlers.NewInflowHandler(r.Container.InflowService)
	outflowHandler := handlers.NewOutflowHandler(r.Container.OutflowService)
	loggingHandler := handlers.NewLoggingHandler(r.Container.LoggingService)
	recActionHandler := handlers.NewReoccurringActionHandler(r.Container.ReoccurringActionService)
	budgetHandler := handlers.NewBudgetHandler(r.Container.BudgetService)
	savingsHandler := handlers.NewSavingsHandler(r.Container.SavingsService)

	// Protected routes
	authGroup := r.Router.Group(apiPrefixV1, r.Container.AuthService.WebClientMiddleware.WebClientAuthentication())
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
