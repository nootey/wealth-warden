package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"
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

func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.Service.GetAllUsers()
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *UserHandler) GetUserById(c *gin.Context) {

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

	user, err := h.Service.FetchUserByID(id)
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *UserHandler) CreateInvitation(c *gin.Context) {
	var req models.InvitationRequest

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

	invitation := &models.Invitation{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		RoleID:      req.Role.ID,
	}

	err := h.Service.CreateInvitation(invitation)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "User invited", "Invitation link has been sent successfully.", http.StatusOK)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		utils.ErrorMessage(c, "Json bind error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	err := h.Service.CreateUser(&user)
	if err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, user.Email, "User created successfully", http.StatusOK)
}
