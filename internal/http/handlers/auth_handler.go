package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/utils"
)

type AuthHandler struct {
	Service *services.AuthService
}

func NewAuthHandler(
	service *services.AuthService,
) *AuthHandler {
	return &AuthHandler{
		Service: service,
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

	// Set cookies and return success message
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access", accessToken, int(constants.AccessCookieTTL.Seconds()), "/", h.Service.Config.WebClient.Domain, true, true)
	c.SetCookie("refresh", refreshToken, expiresAt, "/", h.Service.Config.WebClient.Domain, true, true)

	utils.SuccessMessage(c, "200", "Logged in", http.StatusOK)
}

func (h *AuthHandler) GetAuthUser(c *gin.Context) {
	user, err := h.Service.GetCurrentUser(c)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) LogoutUser(c *gin.Context) {
	c.SetCookie("access", "", -1, "/", h.Service.Config.WebClient.Domain, true, true)
	c.SetCookie("refresh", "", -1, "/", h.Service.Config.WebClient.Domain, true, true)
	utils.SuccessMessage(c, "", "Logged out", http.StatusOK)
}

func (h *AuthHandler) RegisterUser(c *gin.Context) {
	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.RegisterForm
	if err := c.ShouldBindJSON(&form); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := utils.SanitizeStruct(&form); err != nil {
		utils.ErrorMessage(c, "Sanitization error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	err := h.Service.RegisterUser(form.Email, form.Password, form.PasswordConfirmation, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Registration failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "200", "Account created", http.StatusOK)
}
