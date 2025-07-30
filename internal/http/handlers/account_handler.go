package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type AccountHandler struct {
	Service *services.AccountService
}

func NewAccountHandler(service *services.AccountService) *AccountHandler {
	return &AccountHandler{
		Service: service,
	}
}

func (h *AccountHandler) GetAccountTypes(c *gin.Context) {
	records, err := h.Service.FetchAllAccountTypes(c)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}
