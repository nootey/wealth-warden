package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
)

type AuthHandler struct {
	Service *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		Service: authService,
	}
}

func (h *AuthHandler) LoginUser(c *gin.Context) {

	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.LoginForm
	if err := c.ShouldBindJSON(&form); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	accessToken, refreshToken, expiresAt, err := h.Service.LoginUser(form.Email, form.Password, userAgent, loginIP, form.RememberMe)
	if err != nil {
		utils.ErrorMessage(c, "Login failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	// Set cookies and return success message as in your original function
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access", accessToken, 60*15, "/", h.Service.Config.WebClientDomain, h.Service.Config.Release, true)
	c.SetCookie("refresh", refreshToken, expiresAt, "/", h.Service.Config.WebClientDomain, h.Service.Config.Release, true)

	utils.SuccessMessage(c, "200", "Logged in", http.StatusOK)
}

func (h *AuthHandler) GetAuthUser(c *gin.Context) {
	queryParams := c.Request.URL.Query()
	withSecrets := queryParams.Get("withSecrets")
	includeSecrets := false
	if withSecrets == "true" {
		includeSecrets = true
	}

	user, err := h.Service.GetCurrentUser(c, includeSecrets)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) LogoutUser(c *gin.Context) {
	c.SetCookie("access", "", -1, "/", h.Service.Config.WebClientDomain, h.Service.Config.Release, true)
	c.SetCookie("refresh", "", -1, "/", h.Service.Config.WebClientDomain, h.Service.Config.Release, true)
	utils.SuccessMessage(c, "", "Logged out", http.StatusOK)
}
