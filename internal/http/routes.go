package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/bootstrap"
	httpHandlers "wealth-warden/internal/http/handlers"
	"wealth-warden/internal/http/v1"
	"wealth-warden/internal/middleware"
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

	authHandler := httpHandlers.NewAuthHandler(r.Container.AuthService)
	userHandler := httpHandlers.NewUserHandler(r.Container.UserService)
	loggingHandler := httpHandlers.NewLoggingHandler(r.Container.LoggingService)
	accountHandler := httpHandlers.NewAccountHandler(r.Container.AccountService)
	transactionHandler := httpHandlers.NewTransactionHandler(r.Container.TransactionService)

	authRL := middleware.NewRateLimiter(5.0/60.0, 5) // 5 per minute, burst 3

	// Protected routes
	authGroup := _v1.Group("/", r.Container.AuthService.WebClientMiddleware.WebClientAuthentication())
	{
		authRoutes := authGroup.Group("/auth", authRL.Middleware())
		v1.AuthRoutes(authRoutes, authHandler)

		userRoutes := authGroup.Group("/users")
		v1.UserRoutes(userRoutes, userHandler)

		loggingRoutes := authGroup.Group("/logs")
		v1.LoggingRoutes(loggingRoutes, loggingHandler)

		accountRoutes := authGroup.Group("/accounts")
		v1.AccountRoutes(accountRoutes, accountHandler)

		transactionRoutes := authGroup.Group("/transactions")
		v1.TransactionRoutes(transactionRoutes, transactionHandler)

	}

	// Public routes
	publicGroup := _v1.Group("")
	{
		publicAuthRoutes := publicGroup.Group("/auth", authRL.Middleware())
		v1.PublicAuthRoutes(publicAuthRoutes, authHandler)
	}
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}
