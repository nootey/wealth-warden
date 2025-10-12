package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	Service *services.ImportService
	v       *validators.GoValidator
}

func NewImportHandler(
	service *services.ImportService,
	v *validators.GoValidator,
) *ImportHandler {
	return &ImportHandler{
		Service: service,
		v:       v,
	}
}

func (h *ImportHandler) ValidateCustomImport(c *gin.Context) {
	var payload models.CustomImportPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid Request", "Invalid JSON format", http.StatusBadRequest, err)
		return
	}

	if payload.Year == 0 {
		utils.ErrorMessage(c, "Validation Error", "Missing or invalid 'year' field", http.StatusBadRequest, nil)
		return
	}

	if payload.GeneratedAt.IsZero() {
		utils.ErrorMessage(c, "Validation Error", "Missing or invalid 'generated_at' field", http.StatusBadRequest, nil)
		return
	}

	if len(payload.Txns) == 0 {
		utils.ErrorMessage(c, "Validation Error", "No transactions found", http.StatusBadRequest, nil)
		return
	}

	for i, t := range payload.Txns {
		if t.TransactionType == "" {
			utils.ErrorMessage(c, "Validation Error",
				fmt.Sprintf("Transaction[%d]: missing transaction_type", i),
				http.StatusBadRequest, nil)
			return
		}

		tt := strings.ToLower(t.TransactionType)
		if tt != "inflow" && tt != "expense" && tt != "investments" && tt != "savings" {
			utils.ErrorMessage(c, "Validation Error",
				fmt.Sprintf("Transaction[%d]: invalid transaction_type '%s'", i, tt),
				http.StatusBadRequest, nil)
			return
		}

		if t.Amount == "" {
			utils.ErrorMessage(c, "Validation Error",
				fmt.Sprintf("Transaction[%d]: missing amount", i),
				http.StatusBadRequest, nil)
			return
		}

		if t.TxnDate.IsZero() {
			utils.ErrorMessage(c, "Validation Error",
				fmt.Sprintf("Transaction[%d]: missing or invalid txn_date", i),
				http.StatusBadRequest, nil)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"year":   payload.Year,
		"count":  len(payload.Txns),
		"sample": payload.Txns[0],
	})
}

func (h *ImportHandler) ImportFromJSON(c *gin.Context) {
	
}
