package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
)

type TransactionHandler struct {
	Service *services.TransactionService
}

func NewTransactionHandler(service *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		Service: service,
	}
}

func (h *TransactionHandler) GetTransactionsPaginated(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
