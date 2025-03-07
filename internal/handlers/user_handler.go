package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type UserHandler struct {
	Service *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		Service: userService,
	}
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.Service.GetAllUsers()
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *UserHandler) GetUserById(c *gin.Context) {

	idStr := c.Param("id")

	if idStr == "" {
		err := errors.New("invalid id provided")
		utils.ErrorMessage("param error", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	parsedID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.ErrorMessage("param error", "id must be a positive integer", http.StatusBadRequest)(c, err)
		return
	}
	uintID := uint(parsedID)

	user, err := h.Service.FetchUserByID(uintID)
	if err != nil {
		utils.ErrorMessage("Fetch error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		utils.ErrorMessage("Json bind error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	err := h.Service.CreateUser(&user)
	if err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage(user.Email, "User created successfully", http.StatusOK)(c.Writer, c.Request)
}
