package handlers

import (
	"net/http"
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

type UserHandler struct {
	service services.UserServiceInterface
	v       validators.Validator
}

func NewUserHandler(
	service services.UserServiceInterface,
	v validators.Validator,
) *UserHandler {
	return &UserHandler{
		service: service,
		v:       v,
	}
}

func (h *UserHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("manage_users"), h.GetUsersPaginated)
	apiGroup.GET("/search", authz.RequireAllMW("access_backoffice"), h.SearchUsers)
	apiGroup.GET("/:id", authz.RequireAllMW("manage_users"), h.GetUserById)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_users"), h.UpdateUser)
	apiGroup.DELETE(":id", authz.RequireAllMW("delete_users"), h.DeleteUser)

	apiGroup.GET("invitations", authz.RequireAllMW("view_data"), h.GetInvitationsPaginated)
	apiGroup.PUT("invitations", authz.RequireAllMW("view_data"), h.InsertInvitation)
	apiGroup.POST("invitations/resend/:id", h.ResendInvitation)
	apiGroup.DELETE("invitations/:id", authz.RequireAllMW("delete_users"), h.DeleteInvitation)
}

func (h *UserHandler) PublicRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/invitations/:hash", h.GetInvitationByHash)
	apiGroup.GET("/token", h.GetUserByToken)
}

func (h *UserHandler) SearchUsers(c *gin.Context) {

	users, err := h.service.SearchUsersByEmail(c.Request.Context(), c.Query("q"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetUsersPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

	records, paginator, err := h.service.FetchUsersPaginated(ctx, p, includeDeleted)
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

func (h *UserHandler) GetInvitationsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	records, paginator, err := h.service.FetchInvitationsPaginated(ctx, p)
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

func (h *UserHandler) GetUserById(c *gin.Context) {

	ctx := c.Request.Context()
	idStr := c.Param("id")
	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	user, err := h.service.FetchUserByID(ctx, id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetUserByToken(c *gin.Context) {

	ctx := c.Request.Context()
	tokenType := c.Query("type")
	tokenValue := c.Query("value")

	user, err := h.service.FetchUserByToken(ctx, tokenType, tokenValue)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetInvitationByHash(c *gin.Context) {

	ctx := c.Request.Context()
	hash := c.Param("hash")
	if hash == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid hash provided"))
		return
	}

	record, err := h.service.FetchInvitationByHash(ctx, hash)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *UserHandler) InsertInvitation(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvitationReq

	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	if err := utils.SanitizeStruct(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Internal, apperr.GenericMessage, err))
		return
	}

	_, err := h.service.InsertInvitation(ctx, userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "User invited", "Invitation link has been sent successfully.", http.StatusOK)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")
	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	var record *models.UserReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.service.UpdateUser(ctx, userID, id, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	if err := h.service.DeleteUser(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *UserHandler) ResendInvitation(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	_, err = h.service.ResendInvitation(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Invitation has been re-sent", "Success", http.StatusOK)
}

func (h *UserHandler) DeleteInvitation(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	if err := h.service.DeleteInvitation(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
