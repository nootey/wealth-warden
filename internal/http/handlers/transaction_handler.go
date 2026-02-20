package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	Service services.TransactionServiceInterface
	v       validators.Validator
}

func NewTransactionHandler(
	service services.TransactionServiceInterface,
	v validators.Validator,
) *TransactionHandler {
	return &TransactionHandler{
		Service: service,
		v:       v,
	}
}

func (h *TransactionHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("", authz.RequireAllMW("view_data"), h.GetTransactionsPaginated)
	ap.GET(":id", authz.RequireAllMW("view_data"), h.GetTransactionByID)
	ap.PUT("", authz.RequireAllMW("manage_data"), h.InsertTransaction)
	ap.PUT("/:id", authz.RequireAllMW("manage_data"), h.UpdateTransaction)
	ap.DELETE("/:id", authz.RequireAllMW("manage_data"), h.DeleteTransaction)
	ap.GET("transfers", authz.RequireAllMW("view_data"), h.GetTransfersPaginated)
	ap.PUT("transfers", authz.RequireAllMW("manage_data"), h.InsertTransfer)
	ap.DELETE("transfers/:id", authz.RequireAllMW("manage_data"), h.DeleteTransfer)
	ap.POST("/restore", authz.RequireAllMW("manage_data"), h.RestoreTransaction)
	ap.GET("categories", authz.RequireAllMW("view_data"), h.GetCategories)
	ap.GET("categories/:id", authz.RequireAllMW("view_data"), h.GetCategoryByID)
	ap.PUT("categories", authz.RequireAllMW("manage_data"), h.InsertCategory)
	ap.PUT("categories/:id", authz.RequireAllMW("manage_data"), h.UpdateCategory)
	ap.DELETE("categories/:id", authz.RequireAllMW("manage_data"), h.DeleteCategory)
	ap.GET("categories/groups", authz.RequireAllMW("view_data"), h.GetCategoryGroups)
	ap.GET("categories/groups/all", authz.RequireAllMW("view_data"), h.GetCategoriesWithGroups)
	ap.GET("categories/groups/:id", authz.RequireAllMW("view_data"), h.GetCategoryGroupByID)
	ap.PUT("categories/groups", authz.RequireAllMW("manage_data"), h.InsertCategoryGroup)
	ap.PUT("categories/groups/:id", authz.RequireAllMW("manage_data"), h.UpdateCategoryGroup)
	ap.DELETE("categories/groups/:id", authz.RequireAllMW("manage_data"), h.DeleteCategoryGroup)
	ap.POST("categories/restore", authz.RequireAllMW("manage_data"), h.RestoreCategory)
	ap.POST("categories/restore/name", authz.RequireAllMW("manage_data"), h.RestoreCategoryName)
	ap.GET("templates", authz.RequireAllMW("view_data"), h.GetTransactionTemplatesPaginated)
	ap.GET("templates/:id", authz.RequireAllMW("view_data"), h.GetTransactionTemplateByID)
	ap.GET("templates/count", authz.RequireAllMW("view_data"), h.GetTransactionTemplateCount)
	ap.PUT("templates", authz.RequireAllMW("manage_data"), h.InsertTransactionTemplate)
	ap.PUT("templates/:id", authz.RequireAllMW("manage_data"), h.UpdateTransactionTemplate)
	ap.POST("templates/:id/active", authz.RequireAllMW("manage_data"), h.ToggleTransactionTemplateActiveState)
	ap.DELETE("templates/:id", authz.RequireAllMW("manage_data"), h.DeleteTransactionTemplate)
}

func (h *TransactionHandler) GetTransactionsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

	accountIDStr := qp.Get("account_id")
	var accountID *int64
	if accountIDStr != "" {
		id, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			utils.ErrorMessage(c, "Error occurred", "account id must be a valid integer", http.StatusBadRequest, err)
			return
		}
		accountID = &id
	}

	records, totals, paginator, err := h.Service.FetchTransactionsPaginated(ctx, userID, p, includeDeleted, accountID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          gin.H{"records": records, "totals": totals},
	}

	c.JSON(http.StatusOK, response)
}

func (h *TransactionHandler) GetTransfersPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")
	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

	var accountID *int64
	if accIDStr := qp.Get("account_id"); accIDStr != "" {
		if id, err := strconv.ParseInt(accIDStr, 10, 64); err == nil {
			accountID = &id
		}
	}

	records, paginator, err := h.Service.FetchTransfersPaginated(ctx, userID, p, includeDeleted, accountID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")
	queryParams := c.Request.URL.Query()
	includeDeletedStr := queryParams.Get("deleted")

	includeDeleted := strings.EqualFold(includeDeletedStr, "true")

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

	record, err := h.Service.FetchTransactionByID(ctx, userID, id, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *TransactionHandler) GetCategories(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	q := c.Request.URL.Query()
	includeDeleted := strings.EqualFold(q.Get("deleted"), "true")

	records, err := h.Service.FetchAllCategories(ctx, userID, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) GetCategoryByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")
	q := c.Request.URL.Query()
	includeDeleted := strings.EqualFold(q.Get("deleted"), "true")

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

	records, err := h.Service.FetchCategoryByID(ctx, userID, id, includeDeleted)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) InsertTransaction(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.TransactionReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertTransaction(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) InsertTransfer(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.TransferReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertTransfer(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) InsertCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.CategoryReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertCategory(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	var record *models.TransactionReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateTransaction(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	var record *models.CategoryReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateCategory(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.DeleteTransaction(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteTransfer(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.DeleteTransfer(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.DeleteCategory(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreTransaction(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreTransaction(ctx, userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreCategory(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreCategory(ctx, userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}

func (h *TransactionHandler) RestoreCategoryName(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req *models.TrRestoreReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.Service.RestoreCategoryName(ctx, userID, req.ID); err != nil {
		utils.ErrorMessage(c, "Restore error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record restored", "Success", http.StatusOK)
}

func (h *TransactionHandler) GetTransactionTemplatesPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	records, paginator, err := h.Service.FetchTransactionTemplatesPaginated(ctx, userID, p)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *TransactionHandler) GetTransactionTemplateByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	record, err := h.Service.FetchTransactionTemplateByID(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *TransactionHandler) InsertTransactionTemplate(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.TransactionTemplateReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertTransactionTemplate(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateTransactionTemplate(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	var record *models.TransactionTemplateReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateTransactionTemplate(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) ToggleTransactionTemplateActiveState(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.ToggleTransactionTemplateActiveState(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "State toggled", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteTransactionTemplate(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.DeleteTransactionTemplate(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *TransactionHandler) GetTransactionTemplateCount(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	record, err := h.Service.GetTransactionTemplateCount(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *TransactionHandler) GetCategoryGroups(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchAllCategoryGroups(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) GetCategoriesWithGroups(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.Service.FetchAllCategoriesWithGroups(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *TransactionHandler) GetCategoryGroupByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	record, err := h.Service.FetchCategoryGroupByID(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, record)

}

func (h *TransactionHandler) InsertCategoryGroup(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.CategoryGroupReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.Service.InsertCategoryGroup(ctx, userID, record)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)
}

func (h *TransactionHandler) UpdateCategoryGroup(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	var record *models.CategoryGroupReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateCategoryGroup(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *TransactionHandler) DeleteCategoryGroup(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

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

	if err := h.Service.DeleteCategoryGroup(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
