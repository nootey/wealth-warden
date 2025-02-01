package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
)

type AuthHandler struct {
	Service *services.AuthService
	Config  *config.Config
}

func NewAuthHandler(cfg *config.Config, authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		Service: authService,
		Config:  cfg,
	}
}

func (h *AuthHandler) LoginUser(c *gin.Context) {
	fmt.Println(c.Request.Body)
	c.JSON(http.StatusOK, gin.H{"status": "no"})
}
