package v1

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func RecActionRoutes(apiGroup *gin.RouterGroup, handler *handlers.ReoccurringActionHandler) {
	apiGroup.GET("/by-category", handler.GetAllActionsForCategory)
	apiGroup.POST("/delete", handler.DeleteReoccurringAction)
	apiGroup.GET("/available-record-years", handler.GetAvailableRecordYears)
}
