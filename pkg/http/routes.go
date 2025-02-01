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

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, "Wealth Warden server")
}

// Health check handler function
func healthCheck(c *gin.Context, cfg *config.Config) {
	httpHealthStatus := "healthy"
	dbStatus := "healthy"

	// Check database connection
	err := database.PingMysqlDatabase()
	if err != nil {
		dbStatus = "unhealthy"
		httpHealthStatus = "degraded"
	}

	response := gin.H{
		"status": gin.H{
			"api": gin.H{
				"http": httpHealthStatus,
			},
			"services": gin.H{
				"database": gin.H{
					"mysql": dbStatus,
				},
			},
		},
	}

	statusCode := http.StatusOK
	if httpHealthStatus == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

func InitEndpoints(router *gin.Engine, cfg *config.Config, dbClient *gorm.DB) {
	apiPrefixV1 := "/api/v1"

	router.GET("/", func(c *gin.Context) {
		rootHandler(c)
	})

	router.GET(apiPrefixV1+"/health", func(c *gin.Context) {
		healthCheck(c, cfg)
	})

	userRepo := repositories.NewUserRepository(dbClient)

	authService := services.NewAuthService(userRepo)
	userService := services.NewUserService(userRepo)

	authHandler := handlers.NewAuthHandler(cfg, authService)
	userHandler := handlers.NewUserHandler(cfg, userService)

	authenticatedGroup := router.Group(apiPrefixV1, middleware.FrontendAuthMiddleware())
	{
		authRoutes(authenticatedGroup, authHandler)
		userRoutes(authenticatedGroup, userHandler)
	}

	unauthenticatedGroup := router.Group(apiPrefixV1)
	{
		nonAuthRoutes(unauthenticatedGroup, authHandler)
	}
}

func nonAuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {

	apiGroup.POST("/login", func(c *gin.Context) {
		handler.LoginUser(c)
	})
	apiGroup.POST("/logout", func(c *gin.Context) {
		handler.LogoutUser(c)
	})
}

func authRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/get-auth-user", func(c *gin.Context) {
		handler.GetAuthUser(c)
	})
}

func userRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {

	apiGroup.GET("/get-users", func(c *gin.Context) {
		handler.GetUsers(c)
	})
}
