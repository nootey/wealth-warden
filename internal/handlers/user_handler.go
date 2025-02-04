package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// CreateUser adds a new user to the database.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User

	// Bind JSON request to the user struct
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Call service to create user
	err := h.Service.CreateUser(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user": user})
}
