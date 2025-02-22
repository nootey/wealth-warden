package http

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func recActionRoutes(apiGroup *gin.RouterGroup, handler *handlers.ReoccurringActionHandler) {
	apiGroup.GET("/get-all-reoccurring-actions-for-category", handler.GetAllActionsForCategory)

}
