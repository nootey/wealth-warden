package endpoints

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/handlers"
)

func RecActionRoutes(apiGroup *gin.RouterGroup, handler *handlers.ReoccurringActionHandler) {
	apiGroup.GET("/get-all-reoccurring-actions-for-category", handler.GetAllActionsForCategory)
	apiGroup.POST("/delete-reoccurring-action", handler.DeleteReoccurringAction)
	apiGroup.GET("/get-available-record-years", handler.GetAvailableRecordYears)
}
