package http

import (
	"net/http"
	"wealth-warden/internal/bootstrap"
	httpHandlers "wealth-warden/internal/http/handlers"
	"wealth-warden/internal/http/v1"
	"wealth-warden/internal/middleware"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
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

	validator := validators.NewValidator()

	authHandler := httpHandlers.NewAuthHandler(r.Container.AuthService)
	userHandler := httpHandlers.NewUserHandler(r.Container.UserService, validator)
	loggingHandler := httpHandlers.NewLoggingHandler(r.Container.LoggingService)
	accountHandler := httpHandlers.NewAccountHandler(r.Container.AccountService, validator)
	transactionHandler := httpHandlers.NewTransactionHandler(r.Container.TransactionService, validator)
	settingsHandler := httpHandlers.NewSettingsHandler(r.Container.SettingsService, validator)
	chartingHandler := httpHandlers.NewChartingHandler(r.Container.ChartingService, validator)
	roleHandler := httpHandlers.NewRolePermissionHandler(r.Container.RoleService, validator)
	statsHandler := httpHandlers.NewStatisticsHandler(r.Container.StatsService, validator)
	importHandler := httpHandlers.NewImportHandler(r.Container.ImportService, validator)

	//authRL := middleware.NewRateLimiter(5.0/60.0, 5) // 5 per minute, burst 3

	// Auth only routes
	authenticated := _v1.Group("",
		r.Container.AuthService.WebClientMiddleware.WebClientAuthentication(),
	)
	authRoutes := authenticated.Group("/auth")
	v1.AuthRoutes(authRoutes, authHandler)

	// Auth + Permission gated routes
	protected := authenticated.Group("",
		middleware.InjectPerms(r.Container.AuthzService),
	)

	userRoutes := protected.Group("/users")
	v1.UserRoutes(userRoutes, userHandler, roleHandler)

	loggingRoutes := protected.Group("/logs")
	v1.LoggingRoutes(loggingRoutes, loggingHandler)

	accountRoutes := protected.Group("/accounts")
	v1.AccountRoutes(accountRoutes, accountHandler)

	transactionRoutes := protected.Group("/transactions")
	v1.TransactionRoutes(transactionRoutes, transactionHandler)

	settingsRoutes := protected.Group("/settings")
	v1.SettingsRoutes(settingsRoutes, settingsHandler)

	chartingRoutes := protected.Group("/charts")
	v1.ChartingRoutes(chartingRoutes, chartingHandler)

	statsRoutes := protected.Group("/statistics")
	v1.StatsRoutes(statsRoutes, statsHandler)

	importRoutes := protected.Group("/imports")
	v1.ImportRoutes(importRoutes, importHandler)

	// Public routes
	public := _v1.Group("")
	{
		publicAuthRoutes := public.Group("/auth")
		v1.PublicAuthRoutes(publicAuthRoutes, authHandler)

		publicUserRoutes := public.Group("/users")
		v1.PublicUserRoutes(publicUserRoutes, userHandler)
	}
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}
