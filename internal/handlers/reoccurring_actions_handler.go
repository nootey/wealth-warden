package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type ReoccurringActionHandler struct {
	Service *services.ReoccurringActionService
}

func NewReoccurringActionHandler(service *services.ReoccurringActionService) *ReoccurringActionHandler {
	return &ReoccurringActionHandler{
		Service: service,
	}
}

func (h *ReoccurringActionHandler) GetAllActionsForCategory(c *gin.Context) {

	categoryName := c.Query("categoryName")
	if categoryName == "" {
		err := errors.New("invalid category provided")
		utils.ErrorMessage("Request error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	actions, err := h.Service.FetchAllActionsForCategory(c, categoryName)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, actions)
}
