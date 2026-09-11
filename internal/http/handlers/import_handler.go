package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	Service services.ImportServiceInterface
	v       validators.Validator
}

func NewImportHandler(
	service services.ImportServiceInterface,
	v validators.Validator,
) *ImportHandler {
	return &ImportHandler{
		Service: service,
		v:       v,
	}
}

func (h *ImportHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/:import_type", authz.RequireAllMW("view_data"), h.GetImportsByImportType)
	apiGroup.GET("/:import_type/:id", authz.RequireAllMW("view_data"), h.GetStoredCustomImport)
	apiGroup.POST("custom/validate", authz.RequireAllMW("manage_data"), h.ValidateCustomImport)
	apiGroup.POST("custom/accounts", authz.RequireAllMW("manage_data"), h.ImportAccounts)
	apiGroup.POST("custom/categories", authz.RequireAllMW("manage_data"), h.ImportCategories)
	apiGroup.POST("custom/transactions", authz.RequireAllMW("manage_data"), h.ImportTransactions)
	apiGroup.POST("custom/investments", authz.RequireAllMW("manage_data"), h.TransferInvestmentsFromImport)
	apiGroup.POST("custom/savings", authz.RequireAllMW("manage_data"), h.TransferSavingsFromImport)
	apiGroup.POST("custom/repayments", authz.RequireAllMW("manage_data"), h.TransferRepaymentsFromImport)
	apiGroup.POST("custom/trades", authz.RequireAllMW("manage_data"), h.TransferInvestmentTrades)
	apiGroup.DELETE("/:id", authz.RequireAllMW("manage_data"), h.DeleteImport)
}

func (h *ImportHandler) GetImportsByImportType(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	importType := c.Param("import_type")

	records, err := h.Service.FetchImportsByImportType(ctx, userID, importType)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ImportHandler) GetStoredCustomImport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	step := strings.ToLower(strings.TrimSpace(c.Query("step")))
	if step == "" {
		step = "cash"
	}

	imp, err := h.Service.FetchImportByID(ctx, id, userID, "custom")
	if err != nil {
		_ = c.Error(err)
		return
	}
	if imp == nil || imp.Type != "custom" {
		_ = c.Error(apperr.New(apperr.NotFound, "Import not found"))
		return
	}

	filePath := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), imp.Name+".json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var payload models.TxnImportPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Internal, apperr.GenericMessage, err))
		return
	}

	categories, filteredCount, apiErr := h.Service.ValidateCustomImport(ctx, &payload, step)
	if apiErr != nil {
		_ = c.Error(apiErr)
		return
	}

	// choose the correct set for this step
	var set []models.JSONTxn
	switch step {
	case "investment", "investments":
		set = payload.InvestmentTransfers
	case "saving", "savings":
		set = payload.SavingsTransfers
	default: // "cash"
		set = payload.Txns
	}

	c.JSON(http.StatusOK, gin.H{
		"count":          len(set),
		"filtered_count": filteredCount,
		"categories":     categories,
		"step":           step,
	})
}

func (h *ImportHandler) ValidateCustomImport(c *gin.Context) {

	ctx := c.Request.Context()
	step := strings.ToLower(strings.TrimSpace(c.Query("step")))

	var payload models.TxnImportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	categories, filteredCount, apiErr := h.Service.ValidateCustomImport(ctx, &payload, step)
	if apiErr != nil {
		_ = c.Error(apiErr)
		return
	}

	var set []models.JSONTxn
	switch step {
	case "investment", "investments":
		set = payload.InvestmentTransfers
	case "saving", "savings":
		set = payload.SavingsTransfers
	case "repayment", "repayments":
		set = payload.RepaymentTransfers
	case "investment_trades":
		set = payload.TradeTransfers
	default: // "cash"
		set = payload.Txns
	}

	var sample models.JSONTxn
	if len(set) > 0 {
		sample = set[0]
	}

	c.JSON(http.StatusOK, gin.H{
		"count":          len(set),
		"filtered_count": filteredCount,
		"sample":         sample,
		"categories":     categories,
		"step":           step,
	})
}

func (h *ImportHandler) ImportAccounts(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	useBalancesStr := c.Query("use_balances")
	if useBalancesStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "use_balances is required"))
		return
	}

	useBalances, err := strconv.ParseBool(useBalancesStr)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "use_balances must be a valid boolean", err))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "A file is required", err))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The uploaded file could not be opened", err))
		return
	}
	defer func(f multipart.File) {
		if err := f.Close(); err != nil {
			_ = c.Error(err)
		}
	}(f)

	var payload models.AccImportPayload

	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The file is not valid JSON", err))
		return
	}

	if dec.More() {
		_ = c.Error(apperr.New(apperr.Invalid, "The file has unexpected data after the JSON object"))
		return
	}

	// Validate
	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.ImportAccounts(ctx, userID, payload, useBalances); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Account import successful", "Success", http.StatusOK)
}

func (h *ImportHandler) ImportCategories(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "A file is required", err))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The uploaded file could not be opened", err))
		return
	}
	defer func(f multipart.File) {
		if err := f.Close(); err != nil {
			_ = c.Error(err)
		}
	}(f)

	var payload models.CategoryImportPayload

	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The file is not valid JSON", err))
		return
	}

	if dec.More() {
		_ = c.Error(apperr.New(apperr.Invalid, "The file has unexpected data after the JSON object"))
		return
	}

	// Validate
	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.ImportCategories(ctx, userID, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Category import successful", "Success", http.StatusOK)
}

func (h *ImportHandler) ImportTransactions(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	checkAccIDStr := c.Query("check_acc_id")
	if checkAccIDStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "check_acc_id is required"))
		return
	}

	checkAccID, err := strconv.ParseInt(checkAccIDStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "check_acc_id must be a valid integer", err))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "A file is required", err))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The uploaded file could not be opened", err))
		return
	}
	defer func(f multipart.File) {
		if err := f.Close(); err != nil {
			_ = c.Error(err)
		}
	}(f)

	var payload models.TxnImportPayload

	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "The file is not valid JSON", err))
		return
	}

	if dec.More() {
		_ = c.Error(apperr.New(apperr.Invalid, "The file has unexpected data after the JSON object"))
		return
	}

	cmStr := c.PostForm("category_mappings")
	if cmStr != "" {
		var cms []models.CategoryMapping
		if err := json.Unmarshal([]byte(cmStr), &cms); err != nil {
			_ = c.Error(apperr.Wrap(apperr.Invalid, "category_mappings is not valid JSON", err))
			return
		}
		payload.CategoryMappings = cms
	}

	// Validate
	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.ImportTransactions(ctx, userID, checkAccID, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Transaction import successful", "Success", http.StatusOK)
}

func (h *ImportHandler) TransferInvestmentsFromImport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	var payload models.InvestmentTransferPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.TransferInvestmentsFromImport(ctx, userID, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Investments transferred successfully", "Success", http.StatusOK)
}

func (h *ImportHandler) TransferSavingsFromImport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	var payload models.SavingTransferPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.TransferSavingsFromImport(ctx, userID, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Savings transferred successfully", "Success", http.StatusOK)
}

func (h *ImportHandler) TransferRepaymentsFromImport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	var payload models.RepaymentTransferPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.TransferRepaymentsFromImport(ctx, userID, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Repayments transferred successfully", "Success", http.StatusOK)
}

func (h *ImportHandler) DeleteImport(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.Service.DeleteImport(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *ImportHandler) TransferInvestmentTrades(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "A file is required", err))
		return
	}
	defer func(file multipart.File) {
		// This runs after the response is written, so it can only be logged.
		if err := file.Close(); err != nil {
			_ = c.Error(err)
		}
	}(file)

	txnBytes, err := io.ReadAll(file)
	if err != nil {
		_ = c.Error(err)
		return
	}

	mappingsJSON := c.Request.FormValue("trade_mappings")
	var payload models.InvestmentTradesPayload
	if err := json.Unmarshal([]byte(mappingsJSON), &payload.TradeMappings); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "trade_mappings is not valid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(payload); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.Service.TransferInvestmentsTrades(ctx, userID, txnBytes, payload); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Investments transferred successfully", "Success", http.StatusOK)
}
