package http

import (
	"net/http"
	"wealth-warden/internal/bootstrap"
	httpHandlers "wealth-warden/internal/http/handlers"
	"wealth-warden/internal/http/routes"
	"wealth-warden/internal/middleware"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type RouteInitializerHTTP struct {
	Router    *gin.Engine
	Container *bootstrap.ServiceContainer
}

func NewRouteInitializerHTTP(r *gin.Engine, container *bootstrap.ServiceContainer) *RouteInitializerHTTP {

	return &RouteInitializerHTTP{
		Router:    r,
		Container: container,
	}
}

func (r *RouteInitializerHTTP) InitEndpoints(wm *middleware.WebClientMiddleware) {
	api := r.Router.Group("/api")

	r.Router.GET("/", rootHandler)
	r.initV1Routes(api, wm)
}

func (r *RouteInitializerHTTP) initV1Routes(_v1 *gin.RouterGroup, wm *middleware.WebClientMiddleware) {

	validator := validators.NewValidator()

	authHandler := httpHandlers.NewAuthHandler(r.Container.Config, wm, r.Container.AuthService)
	userHandler := httpHandlers.NewUserHandler(r.Container.UserService, validator)
	loggingHandler := httpHandlers.NewLoggingHandler(r.Container.LoggingService)
	accountHandler := httpHandlers.NewAccountHandler(r.Container.AccountService, validator)
	transactionHandler := httpHandlers.NewTransactionHandler(r.Container.TransactionService, validator)
	settingsHandler := httpHandlers.NewSettingsHandler(r.Container.SettingsService, validator)
	chartingHandler := httpHandlers.NewChartingHandler(r.Container.ChartingService, validator)
	roleHandler := httpHandlers.NewRolePermissionHandler(r.Container.RoleService, validator)
	statsHandler := httpHandlers.NewStatisticsHandler(r.Container.StatsService, validator)
	importHandler := httpHandlers.NewImportHandler(r.Container.ImportService, validator)
	exportHandler := httpHandlers.NewExportHandler(r.Container.ExportService, validator)
	investmentHandler := httpHandlers.NewInvestmentHandler(r.Container.InvestmentService, validator)
	notesHandler := httpHandlers.NewNotesHandler(r.Container.NotesService, validator)

	//authRL := middleware.NewRateLimiter(5.0/60.0, 5) // 5 per minute, burst 3

	// Auth only routes
	authenticated := _v1.Group("",
		wm.WebClientAuthentication(),
	)
	authRoutes := authenticated.Group("/auth")
	routes.AuthRoutes(authRoutes, authHandler)

	// Auth + Permission gated routes
	protected := authenticated.Group("",
		middleware.InjectPerms(r.Container.AuthzService),
	)

	userRoutes := protected.Group("/users")
	routes.UserRoutes(userRoutes, userHandler, roleHandler)

	loggingRoutes := protected.Group("/logs")
	routes.LoggingRoutes(loggingRoutes, loggingHandler)

	accountRoutes := protected.Group("/accounts")
	routes.AccountRoutes(accountRoutes, accountHandler)

	transactionRoutes := protected.Group("/transactions")
	routes.TransactionRoutes(transactionRoutes, transactionHandler)

	settingsRoutes := protected.Group("/settings")
	routes.SettingsRoutes(settingsRoutes, settingsHandler)

	chartingRoutes := protected.Group("/charts")
	routes.ChartingRoutes(chartingRoutes, chartingHandler)

	statsRoutes := protected.Group("/statistics")
	routes.StatsRoutes(statsRoutes, statsHandler)

	importRoutes := protected.Group("/imports")
	routes.ImportRoutes(importRoutes, importHandler)

	exportRoutes := protected.Group("/exports")
	routes.ExportRoutes(exportRoutes, exportHandler)

	investmentRoutes := protected.Group("/investments")
	routes.InvestmentRoutes(investmentRoutes, investmentHandler)

	noteRoutes := protected.Group("/notes")
	routes.NoteRoutes(noteRoutes, notesHandler)

	// Public routes
	public := _v1.Group("")
	{
		publicAuthRoutes := public.Group("/auth")
		routes.PublicAuthRoutes(publicAuthRoutes, authHandler)

		publicUserRoutes := public.Group("/users")
		routes.PublicUserRoutes(publicUserRoutes, userHandler)
	}
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}
