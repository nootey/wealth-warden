package http

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"wealth-warden/server/internal/handlers"
	"wealth-warden/server/internal/repositories"
	"wealth-warden/server/internal/services"
	"wealth-warden/server/pkg/config"
	"wealth-warden/server/pkg/database"
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

	userService := services.NewUserService(userRepo)

	userHandler := handlers.NewUserHandler(cfg, userService)

	authenticatedGroup := router.Group(apiPrefixV1)
	{
		userRoutes(authenticatedGroup, userHandler)
	}
}

func userRoutes(apiGroup *gin.RouterGroup, handler *handlers.UserHandler) {

	apiGroup.GET("/get-users", func(c *gin.Context) {
		handler.GetUsers(c)
	})
}
