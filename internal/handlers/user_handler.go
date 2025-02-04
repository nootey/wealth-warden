package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
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

// CreateUser adds a new user to the database.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User

	// Bind JSON request to the user struct
	if err := c.ShouldBindJSON(&user); err != nil {
		utils.ErrorMessage("Json bind error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	// Call service to create user
	err := h.Service.CreateUser(&user)
	if err != nil {
		utils.ErrorMessage("Create error", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	utils.SuccessMessage(user.Email, "User created successfully", http.StatusOK)(c.Writer, c.Request)
}
