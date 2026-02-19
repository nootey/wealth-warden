package http

import (
	"net/http"
	"wealth-warden/internal/bootstrap"
	httpHandlers "wealth-warden/internal/http/handlers"
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

	// Register handlers
	authHandler := httpHandlers.NewAuthHandler(r.Container.Config, wm, r.Container.AuthService)
	userHandler := httpHandlers.NewUserHandler(r.Container.UserService, validator)
	loggingHandler := httpHandlers.NewLoggingHandler(r.Container.LoggingService)
	accountHandler := httpHandlers.NewAccountHandler(r.Container.AccountService, validator)
	transactionHandler := httpHandlers.NewTransactionHandler(r.Container.TransactionService, validator)
	settingsHandler := httpHandlers.NewSettingsHandler(r.Container.SettingsService, validator)
	roleHandler := httpHandlers.NewRolePermissionHandler(r.Container.RoleService, validator)
	importHandler := httpHandlers.NewImportHandler(r.Container.ImportService, validator)
	exportHandler := httpHandlers.NewExportHandler(r.Container.ExportService, validator)
	investmentHandler := httpHandlers.NewInvestmentHandler(r.Container.InvestmentService, validator)
	notesHandler := httpHandlers.NewNotesHandler(r.Container.NotesService, validator)
	analyticsHandler := httpHandlers.NewAnalyticsHandler(r.Container.AnalyticsService, validator)

	// Register routes

	// Auth only routes
	authenticated := _v1.Group("",
		wm.WebClientAuthentication(),
	)

	// Auth + Permission gated routes
	protected := authenticated.Group("",
		middleware.InjectPerms(r.Container.AuthzService),
	)

	// Public routes
	public := _v1.Group("")
	{
		authHandler.PublicRoutes(public.Group("/auth"))
		userHandler.PublicRoutes(public.Group("/users"))
	}

	authHandler.Routes(authenticated.Group("/auth"))
	accountHandler.Routes(protected.Group("/accounts"))
	analyticsHandler.Routes(protected.Group("/analytics"))
	exportHandler.Routes(protected.Group("/exports"))
	importHandler.Routes(protected.Group("/imports"))
	investmentHandler.Routes(protected.Group("/investments"))
	loggingHandler.Routes(protected.Group("/logs"))
	notesHandler.Routes(protected.Group("/notes"))
	roleHandler.Routes(protected.Group("/users/roles"))
	settingsHandler.Routes(protected.Group("/settings"))
	transactionHandler.Routes(protected.Group("/transactions"))
	userHandler.Routes(protected.Group("/users"))

}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Wealth Warden server is running!"})
}
