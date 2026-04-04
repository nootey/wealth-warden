package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type SavingsHandler struct {
	service services.SavingsServiceInterface
	v       validators.Validator
}

func NewSavingsHandler(
	service services.SavingsServiceInterface,
	v validators.Validator,
) *SavingsHandler {
	return &SavingsHandler{
		service: service,
		v:       v,
	}
}

func (h *SavingsHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), h.GetGoals)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), h.GetGoalByID)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), h.InsertGoal)
	apiGroup.PUT("/:id", authz.RequireAllMW("manage_data"), h.UpdateGoal)
	apiGroup.DELETE("/:id", authz.RequireAllMW("manage_data"), h.DeleteGoal)

	apiGroup.GET("/:id/contributions", authz.RequireAllMW("view_data"), h.GetContributions)
	apiGroup.PUT("/:id/contributions", authz.RequireAllMW("manage_data"), h.InsertContribution)
	apiGroup.DELETE("/:id/contributions/:contrib_id", authz.RequireAllMW("manage_data"), h.DeleteContribution)
}

func (h *SavingsHandler) GetGoals(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	records, err := h.service.FetchGoals(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *SavingsHandler) GetGoalByID(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	record, err := h.service.FetchGoalByID(ctx, userID, id)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *SavingsHandler) InsertGoal(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.SavingGoalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err := h.service.InsertGoal(ctx, userID, &req)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Goal created", "Success", http.StatusOK)
}

func (h *SavingsHandler) UpdateGoal(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	var req models.SavingGoalUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.service.UpdateGoal(ctx, userID, id, &req)
	if err != nil {
		utils.ErrorMessage(c, "Update error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Goal updated", "Success", http.StatusOK)
}

func (h *SavingsHandler) DeleteGoal(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	id, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteGoal(ctx, userID, id); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Goal deleted", "Success", http.StatusOK)
}

func (h *SavingsHandler) GetContributions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	goalID, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	records, err := h.service.FetchContributions(ctx, userID, goalID)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *SavingsHandler) InsertContribution(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	goalID, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	var req models.SavingContributionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.v.ValidateStruct(req); err != nil {
		utils.ValidationFailed(c, err.Error(), err)
		return
	}

	_, err = h.service.InsertContribution(ctx, userID, goalID, &req)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Contribution added", "Success", http.StatusOK)
}

func (h *SavingsHandler) DeleteContribution(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	goalID, err := parseID(c, "id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	contribID, err := parseID(c, "contrib_id")
	if err != nil {
		utils.ErrorMessage(c, "param error", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := h.service.DeleteContribution(ctx, userID, goalID, contribID); err != nil {
		utils.ErrorMessage(c, "Delete error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Contribution deleted", "Success", http.StatusOK)
}

func parseID(c *gin.Context, param string) (int64, error) {
	s := c.Param(param)
	if s == "" {
		return 0, errors.New("invalid id provided")
	}
	return strconv.ParseInt(s, 10, 64)
}
