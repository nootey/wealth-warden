package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

func (h *ImportHandler) GetImportsByImportType(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	importType := c.Param("import_type")

	records, err := h.Service.FetchImportsByImportType(userID, importType)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ImportHandler) GetImportByID(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	idStr := c.Param("id")
	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	importType := c.Param("import_type")

	records, err := h.Service.FetchImportByID(nil, id, userID, importType)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ImportHandler) GetStoredCustomImport(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Bad Request", "id must be an integer", http.StatusBadRequest, err)
		return
	}

	step := strings.ToLower(strings.TrimSpace(c.Query("step")))
	if step == "" {
		step = "cash"
	}

	imp, err := h.Service.FetchImportByID(nil, id, userID, "custom")
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}
	if imp == nil || imp.Type != "custom" {
		utils.ErrorMessage(c, "Not found", "import not found", http.StatusNotFound, nil)
		return
	}

	filePath := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), imp.Name+".json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	var payload models.TxnImportPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		utils.ErrorMessage(c, "Invalid file", "invalid JSON in stored import", http.StatusBadRequest, err)
		return
	}

	categories, filteredCount, apiErr := h.Service.ValidateCustomImport(&payload, step)
	if apiErr != nil {
		utils.ErrorMessage(c, "Error occurred", apiErr.Error(), http.StatusInternalServerError, nil)
		return
	}

	// choose the correct set for this step
	var set []models.JSONTxn
	switch step {
	case "investment", "investments":
		set = payload.Transfers
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
	step := strings.ToLower(strings.TrimSpace(c.Query("step")))

	var payload models.TxnImportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid Request", "Invalid JSON format", http.StatusBadRequest, err)
		return
	}

	categories, filteredCount, apiErr := h.Service.ValidateCustomImport(&payload, step)
	if apiErr != nil {
		utils.ErrorMessage(c, "Error occurred", apiErr.Error(), http.StatusInternalServerError, nil)
		return
	}

	var set []models.JSONTxn
	switch step {
	case "investment", "investments":
		set = payload.Transfers
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

func (h *ImportHandler) ImportTransactions(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	checkAccIDStr := c.Query("check_acc_id")
	if checkAccIDStr == "" {
		utils.ErrorMessage(c, "param error", "missing account ids", http.StatusBadRequest, nil)
		return
	}

	checkAccID, err := strconv.ParseInt(checkAccIDStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "check acc id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.ErrorMessage(c, "Invalid upload", "file is required", http.StatusBadRequest, err)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		utils.ErrorMessage(c, "Invalid upload", "cannot open uploaded file", http.StatusBadRequest, err)
		return
	}
	defer f.Close()

	var payload models.TxnImportPayload

	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if dec.More() {
		utils.ErrorMessage(c, "Invalid JSON", "unexpected data after JSON object", http.StatusBadRequest, nil)
		return
	}

	cmStr := c.PostForm("category_mappings")
	if cmStr != "" {
		var cms []models.CategoryMapping
		if err := json.Unmarshal([]byte(cmStr), &cms); err != nil {
			utils.ErrorMessage(c, "Invalid category_mappings", err.Error(), http.StatusBadRequest, err)
			return
		}
		payload.CategoryMappings = cms
	}

	// Validate
	if err := h.v.ValidateStruct(payload); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.ImportTransactions(userID, checkAccID, payload); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Transaction import successful", "Success", http.StatusOK)
}

func (h *ImportHandler) ImportAccounts(c *gin.Context) {

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	useBalancesStr := c.Query("use_balances")
	if useBalancesStr == "" {
		utils.ErrorMessage(c, "param error", "missing use_balances bool", http.StatusBadRequest, nil)
		return
	}

	useBalances, err := strconv.ParseBool(useBalancesStr)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "use_balances must be a valid boolean", http.StatusBadRequest, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.ErrorMessage(c, "Invalid upload", "file is required", http.StatusBadRequest, err)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		utils.ErrorMessage(c, "Invalid upload", "cannot open uploaded file", http.StatusBadRequest, err)
		return
	}
	defer f.Close()

	var payload models.AccImportPayload

	dec := json.NewDecoder(f)
	if err := dec.Decode(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if dec.More() {
		utils.ErrorMessage(c, "Invalid JSON", "unexpected data after JSON object", http.StatusBadRequest, nil)
		return
	}

	// Validate
	if err := h.v.ValidateStruct(payload); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.ImportAccounts(userID, payload, useBalances); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Account import successful", "Success", http.StatusOK)
}

func (h *ImportHandler) TransferInvestmentsFromImport(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	var payload models.InvestmentTransferPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid Request", "Invalid JSON body", http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(payload); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := h.Service.TransferInvestmentsFromImport(userID, payload); err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Investments transferred successfully", "Success", http.StatusOK)
}

func (h *ImportHandler) DeleteImport(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	if err := h.Service.DeleteImport(userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
