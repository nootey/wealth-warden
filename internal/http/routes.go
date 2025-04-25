package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/handlers"
	"wealth-warden/internal/http/v1"
)

type RouteInitializerHTTP struct {
	Router    *gin.Engine
	Container *bootstrap.Container
}

func NewRouteInitializerHTTP(r *gin.Engine, container *bootstrap.Container) *RouteInitializerHTTP {

	return &RouteInitializerHTTP{
		Router:    r,
		Container: container,
	}
}

func (r *RouteInitializerHTTP) InitEndpoints() {
	api := r.Router.Group("/api")

	// Version 1
	_v1 := api.Group("/v1")

	r.Router.GET("/", rootHandler)
	r.initV1Routes(_v1)
}

func (r *RouteInitializerHTTP) initV1Routes(_v1 *gin.RouterGroup) {

	authHandler := handlers.NewAuthHandler(r.Container.AuthService)
	userHandler := handlers.NewUserHandler(r.Container.UserService)
	inflowHandler := handlers.NewInflowHandler(r.Container.InflowService)
	outflowHandler := handlers.NewOutflowHandler(r.Container.OutflowService)
	loggingHandler := handlers.NewLoggingHandler(r.Container.LoggingService)
	recActionHandler := handlers.NewReoccurringActionHandler(r.Container.ReoccurringActionService)
	budgetHandler := handlers.NewBudgetHandler(r.Container.BudgetService)
	savingsHandler := handlers.NewSavingsHandler(r.Container.SavingsService)

	// Protected routes
	authGroup := _v1.Group("/", r.Container.AuthService.WebClientMiddleware.WebClientAuthentication())
	{
		authRoutes := authGroup.Group("/auth")
		v1.AuthRoutes(authRoutes, authHandler)

		userRoutes := authGroup.Group("/users")
		v1.UserRoutes(userRoutes, userHandler)

		inflowRoutes := authGroup.Group("/inflows")
		v1.InflowRoutes(inflowRoutes, inflowHandler)

		outflowRoutes := authGroup.Group("/outflows")
		v1.OutflowRoutes(outflowRoutes, outflowHandler)

		loggingRoutes := authGroup.Group("/logs")
		v1.LoggingRoutes(loggingRoutes, loggingHandler)

		reoccurringRoutes := authGroup.Group("/reoccurring")
		v1.RecActionRoutes(reoccurringRoutes, recActionHandler)

		budgetRoutes := authGroup.Group("/budget")
		v1.BudgetRoutes(budgetRoutes, budgetHandler)

		savingsRoutes := authGroup.Group("/savings")
		v1.SavingsRoutes(savingsRoutes, savingsHandler)
	}

	// Public routes
	publicGroup := _v1.Group("")
	{
		publicAuthRoutes := publicGroup.Group("/auth")
		v1.PublicAuthRoutes(publicAuthRoutes, authHandler)
	}
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}
