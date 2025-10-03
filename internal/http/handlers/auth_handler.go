package handlers

import (
	"fmt"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
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
	domain := h.Service.WebClientMiddleware.CookieDomainForEnv()
	secure := h.Service.WebClientMiddleware.CookieSecure()
	c.SetCookie("access", accessToken, int(constants.AccessCookieTTL.Seconds()), "/", domain, secure, true)
	c.SetCookie("refresh", refreshToken, expiresAt, "/", domain, secure, true)

	utils.SuccessMessage(c, "", "Logged in", http.StatusOK)
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
	domain := h.Service.WebClientMiddleware.CookieDomainForEnv()
	secure := h.Service.WebClientMiddleware.CookieSecure()
	c.SetCookie("access", "", -1, "/", domain, secure, true)
	c.SetCookie("refresh", "", -1, "/", domain, secure, true)
	utils.SuccessMessage(c, "", "Logged out", http.StatusOK)
}

func (h *AuthHandler) SignUp(c *gin.Context) {
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

	err := h.Service.SignUp(form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Registration failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Account created successfully!", http.StatusOK)
}

func (h *AuthHandler) ValidateInvitationEmail(c *gin.Context) {
	queryParams := c.Request.URL.Query()
	hash := queryParams.Get("token")

	err := h.Service.ValidateInvitation(hash)
	if err == nil {
		redirectUrl := utils.GenerateWebClientReleaseLink(h.Service.Config, "")
		c.Redirect(http.StatusFound, fmt.Sprintf("%s%s?token=%s", redirectUrl, "register", hash))
	} else {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Email has been validated", "Success", http.StatusOK)
}

func (h *AuthHandler) ResendConfirmationEmail(c *gin.Context) {

	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	err := h.Service.ResendConfirmationEmail(req.Email, userAgent, reqIP)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Email dispatched", http.StatusOK)
}

func (h *AuthHandler) ConfirmEmail(c *gin.Context) {

	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	queryParams := c.Request.URL.Query()
	token := queryParams.Get("token")

	err := h.Service.ConfirmEmail(token, userAgent, reqIP)
	if err == nil {
		redirectUrl := utils.GenerateWebClientReleaseLink(h.Service.Config, "")
		c.Redirect(http.StatusFound, redirectUrl)
	} else {
		utils.ErrorMessage(c, "Error confirming email", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "", "Email confirmed", http.StatusOK)
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {

	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	err := h.Service.RequestPasswordReset(req.Email, userAgent, reqIP)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Email dispatched", http.StatusOK)
}

func (h *AuthHandler) ValidatePasswordReset(c *gin.Context) {

	queryParams := c.Request.URL.Query()
	tokenValue := queryParams.Get("token")

	token, err := h.Service.ValidatePasswordReset(tokenValue)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	redirectUrl := utils.GenerateWebClientReleaseLink(h.Service.Config, "")
	c.Redirect(http.StatusFound, fmt.Sprintf("%sreset-password/%s", redirectUrl, token))
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {

	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.ResetPasswordForm
	if err := c.ShouldBindJSON(&form); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := utils.SanitizeStruct(&form); err != nil {
		utils.ErrorMessage(c, "Sanitization error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	err := h.Service.ResetPassword(form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Password reset failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Password reset complete", http.StatusOK)
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

	err := h.Service.RegisterUser(form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Registration failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Registration complete", http.StatusOK)
}
