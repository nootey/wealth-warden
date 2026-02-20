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

type UserHandler struct {
	Service *services.UserService
	v       *validators.GoValidator
}

func NewUserHandler(
	service *services.UserService,
	v *validators.GoValidator,
) *UserHandler {
	return &UserHandler{
		Service: service,
		v:       v,
	}
}

func (h *UserHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("manage_users"), h.GetUsersPaginated)
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

func (h *UserHandler) GetUsersPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)
	includeDeleted := strings.EqualFold(qp.Get("include_deleted"), "true")

	records, paginator, err := h.Service.FetchUsersPaginated(ctx, p, includeDeleted)
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

func (h *UserHandler) GetInvitationsPaginated(c *gin.Context) {

	ctx := c.Request.Context()
	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	records, paginator, err := h.Service.FetchInvitationsPaginated(ctx, p)
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

func (h *UserHandler) GetUserById(c *gin.Context) {

	ctx := c.Request.Context()
	idStr := c.Param("id")
	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "param error", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	user, err := h.Service.FetchUserByID(ctx, id)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetUserByToken(c *gin.Context) {

	ctx := c.Request.Context()
	tokenType := c.Query("type")
	tokenValue := c.Query("value")

	user, err := h.Service.FetchUserByToken(ctx, tokenType, tokenValue)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetInvitationByHash(c *gin.Context) {

	ctx := c.Request.Context()
	hash := c.Param("hash")
	if hash == "" {
		err := errors.New("invalid hash provided")
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	record, err := h.Service.FetchInvitationByHash(ctx, hash)
	if err != nil {
		utils.ErrorMessage(c, "fetch error", err.Error(), http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *UserHandler) InsertInvitation(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.InvitationReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	validator := validators.NewValidator()
	if err := validator.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	if err := utils.SanitizeStruct(&req); err != nil {
		utils.ErrorMessage(c, "Sanitization error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	_, err := h.Service.InsertInvitation(ctx, userID, req)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "User invited", "Invitation link has been sent successfully.", http.StatusOK)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {

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

	var record *models.UserReq

	if err := c.ShouldBindJSON(&record); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.Service.UpdateUser(ctx, userID, id, record)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record updated", "Success", http.StatusOK)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {

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

	if err := h.Service.DeleteUser(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}

func (h *UserHandler) ResendInvitation(c *gin.Context) {

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

	_, err = h.Service.ResendInvitation(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Resend error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Invitation has been re-sent", "Success", http.StatusOK)
}

func (h *UserHandler) DeleteInvitation(c *gin.Context) {

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

	if err := h.Service.DeleteInvitation(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Record deleted", "Success", http.StatusOK)
}
