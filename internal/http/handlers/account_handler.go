package handlers

import (
	"net/http"
	"strings"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	service services.AccountServiceInterface
	v       validators.Validator
}

func NewAccountHandler(
	service services.AccountServiceInterface,
	v validators.Validator,
) *AccountHandler {
	return &AccountHandler{
		service: service,
		v:       v,
	}
}

func (h *AccountHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), h.GetAccountsPaginated)
	apiGroup.GET("/all", authz.RequireAllMW("view_data"), h.GetAllAccounts)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), h.GetAccountByID)
	apiGroup.GET("/name/:name", authz.RequireAllMW("view_data"), h.GetAccountByName)
	apiGroup.GET("/subtype/:sub", authz.RequireAllMW("view_data"), h.GetAccountsBySubtype)
	apiGroup.GET("/type/:type", authz.RequireAllMW("view_data"), h.GetAccountsByType)
	apiGroup.GET("/types", authz.RequireAllMW("view_data"), h.GetAccountTypes)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), h.InsertAccount)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_data"), h.UpdateAccount)
	apiGroup.POST(":id/active", authz.RequireAllMW("manage_data"), h.ToggleAccountActiveState)
	apiGroup.DELETE(":id", authz.RequireAllMW("manage_data"), h.CloseAccount)
	apiGroup.DELETE(":id/purge", authz.RequireAllMW("root_access"), h.PurgeAccount)
	apiGroup.GET("/user/:userID", authz.RequireAllMW("access_backoffice"), h.GetAccountsForUser)
	apiGroup.POST(":id/projection/save", authz.RequireAllMW("manage_data"), h.SaveAccountProjection)
	apiGroup.POST(":id/projection/revert", authz.RequireAllMW("manage_data"), h.RevertAccountProjection)
	apiGroup.GET("/balances/:id/latest", authz.RequireAllMW("view_data"), h.GetLatestBalance)
	apiGroup.POST("/balances/backfill", authz.RequireAllMW("manage_data"), h.BackfillBalancesForUser)
	apiGroup.GET("/defaults/all", authz.RequireAllMW("view_data"), h.GetAccountsWithDefaults)
	apiGroup.GET("/defaults/types", authz.RequireAllMW("view_data"), h.GetAccountTypesWithoutDefaults)
	apiGroup.PATCH("/defaults/set/:id", authz.RequireAllMW("manage_data"), h.SetDefaultAccount)
	apiGroup.PATCH("/defaults/unset/:id", authz.RequireAllMW("manage_data"), h.UnsetDefaultAccount)
	apiGroup.GET("/sync/asset/:id", authz.RequireAllMW("manage_data"), h.SyncAssetPnL)
	apiGroup.GET("/sync/account/:acc_id", authz.RequireAllMW("manage_data"), h.SyncAccountPnL)
	apiGroup.POST("/sync/balances", authz.RequireAllMW("view_data"), h.SyncBalancesForUser)
	apiGroup.POST("/merge", authz.RequireAllMW("manage_data"), h.MergeAccounts)
}

func (h *AccountHandler) GetAccountsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeInactive := strings.EqualFold(qp.Get("inactive"), "true")
	classification := qp.Get("classification")

	records, paginator, err := h.service.FetchAccountsPaginated(ctx, userID, p, includeInactive, classification)
	if err != nil {
		_ = c.Error(err)
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

func (h *AccountHandler) GetAllAccounts(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	q := c.Request.URL.Query()
	includeInactive := strings.EqualFold(q.Get("inactive"), "true")
	includeTypes := strings.EqualFold(q.Get("types"), "true")

	records, err := h.service.FetchAllAccounts(ctx, userID, includeInactive, includeTypes)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountByID(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	qp := c.Request.URL.Query()
	if strings.EqualFold(qp.Get("initial_balance"), "true") {
		record, err := h.service.FetchAccountWithOpening(ctx, userID, id)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, record)
		return
	}

	records, err := h.service.FetchAccountByID(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountByName(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	name := c.Param("name")

	records, err := h.service.FetchAccountByName(ctx, userID, name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountTypes(c *gin.Context) {

	ctx := c.Request.Context()

	records, err := h.service.FetchAllAccountTypes(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountsBySubtype(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	sub := c.Param("sub")

	records, err := h.service.FetchAccountsBySubtype(ctx, userID, sub)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) GetAccountsByType(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	t := c.Param("type")

	records, err := h.service.FetchAccountsByType(ctx, userID, t)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)

}

func (h *AccountHandler) InsertAccount(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.AccountReq
	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.service.InsertAccount(ctx, userID, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record created", "Success", http.StatusOK)

}

func (h *AccountHandler) UpdateAccount(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var req *models.AccountReq

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.service.UpdateAccount(ctx, userID, id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)

}

func (h *AccountHandler) ToggleAccountActiveState(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.ToggleAccountActiveState(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Account status updated", "Success", http.StatusOK)

}

func (h *AccountHandler) CloseAccount(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.CloseAccount(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *AccountHandler) GetAccountsForUser(c *gin.Context) {

	ctx := c.Request.Context()

	userID, err := utils.ParseID(c, "userID")
	if err != nil {
		_ = c.Error(err)
		return
	}

	records, err := h.service.FetchAccountsForUser(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *AccountHandler) PurgeAccount(c *gin.Context) {

	ctx := c.Request.Context()
	actorID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.PurgeAccount(ctx, actorID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Account purged", "Success", http.StatusOK)
}

func (h *AccountHandler) BackfillBalancesForUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	from := c.Query("from")
	to := c.Query("to")

	if err := h.service.BackfillBalancesForUser(ctx, userID, from, to); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Backfill completed", "Success", http.StatusOK)
}

func (h *AccountHandler) SaveAccountProjection(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	var req *models.AccountProjectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	err = h.service.SaveAccountProjection(ctx, id, userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *AccountHandler) RevertAccountProjection(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	err = h.service.RevertAccountProjection(ctx, id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *AccountHandler) GetLatestBalance(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	rec, err := h.service.FetchLatestBalance(ctx, id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, rec)
}

func (h *AccountHandler) GetAccountsWithDefaults(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.service.FetchAccountsWithDefaults(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *AccountHandler) GetAccountTypesWithoutDefaults(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.service.FetchAccountTypesWithoutDefaults(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *AccountHandler) SetDefaultAccount(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	accountID, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.SetDefaultAccount(ctx, userID, accountID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Default set", "Success", http.StatusOK)
}

func (h *AccountHandler) UnsetDefaultAccount(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	accountID, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.UnsetDefaultAccount(ctx, userID, accountID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Default unset", "Success", http.StatusOK)
}

func (h *AccountHandler) SyncAssetPnL(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := utils.ParseID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.SyncAssetPnL(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "PnL sync queued", "Asset PnL recalculation has been queued", http.StatusOK)
}

func (h *AccountHandler) SyncBalancesForUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	if err := h.service.SyncForUser(ctx, userID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Sync completed", "Balance sync completed", http.StatusOK)
}

func (h *AccountHandler) MergeAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req struct {
		SourceID      int64 `json:"source_id" validate:"required"`
		DestinationID int64 `json:"destination_id" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := h.service.QueueAccountMerge(ctx, userID, req.SourceID, req.DestinationID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Merge queued", "Success", http.StatusOK)
}

func (h *AccountHandler) SyncAccountPnL(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	accID, err := utils.ParseID(c, "acc_id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.SyncAccountPnL(ctx, userID, accID); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "PnL sync queued", "Account PnL recalculation has been queued", http.StatusOK)
}
