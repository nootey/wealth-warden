package handlers

import (
	"fmt"
	"net/http"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/constants"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg        *config.Config
	middleware *middleware.WebClientMiddleware
	Service    *services.AuthService
}

func NewAuthHandler(
	cfg *config.Config,
	middleware *middleware.WebClientMiddleware,
	service *services.AuthService,
) *AuthHandler {
	return &AuthHandler{
		cfg:        cfg,
		middleware: middleware,
		Service:    service,
	}
}

func (h *AuthHandler) LoginUser(c *gin.Context) {

	ctx := c.Request.Context()
	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.LoginForm
	if err := c.ShouldBindJSON(&form); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	user, err := h.Service.ValidateLogin(ctx, form.Email, form.Password, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Login failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	accessToken, refreshToken, err := h.middleware.GenerateLoginTokens(user.ID, form.RememberMe)
	if err != nil {
		utils.ErrorMessage(c, "Failed to generate tokens", err.Error(), http.StatusInternalServerError, err)
		return
	}

	// Calculate expiration
	var expiresAt int
	if form.RememberMe {
		expiresAt = int(constants.RefreshCookieTTLLong.Seconds())
	} else {
		expiresAt = int(constants.RefreshCookieTTLShort.Seconds())
	}

	// Set cookies
	c.SetSameSite(http.SameSiteLaxMode)
	domain := h.middleware.CookieDomainForEnv()
	secure := h.middleware.CookieSecure()
	c.SetCookie("access", accessToken, int(constants.AccessCookieTTL.Seconds()), "/", domain, secure, true)
	c.SetCookie("refresh", refreshToken, expiresAt, "/", domain, secure, true)

	utils.SuccessMessage(c, "", "Logged in", http.StatusOK)
}

func (h *AuthHandler) GetAuthUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	user, err := h.Service.GetCurrentUser(ctx, userID)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) LogoutUser(c *gin.Context) {
	domain := h.middleware.CookieDomainForEnv()
	secure := h.middleware.CookieSecure()
	c.SetCookie("access", "", -1, "/", domain, secure, true)
	c.SetCookie("refresh", "", -1, "/", domain, secure, true)
	utils.SuccessMessage(c, "", "Logged out", http.StatusOK)
}

func (h *AuthHandler) SignUp(c *gin.Context) {

	ctx := c.Request.Context()
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

	err := h.Service.SignUp(ctx, form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Registration failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Account created successfully!", http.StatusOK)
}

func (h *AuthHandler) ValidateInvitationEmail(c *gin.Context) {

	ctx := c.Request.Context()
	queryParams := c.Request.URL.Query()
	hash := queryParams.Get("token")

	err := h.Service.ValidateInvitation(ctx, hash)
	if err == nil {
		redirectUrl := utils.GenerateWebClientReleaseLink(h.cfg, "")
		c.Redirect(http.StatusFound, fmt.Sprintf("%s%s?token=%s", redirectUrl, "signup", hash))
	} else {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "Email has been validated", "Success", http.StatusOK)
}

func (h *AuthHandler) ResendConfirmationEmail(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	err := h.Service.ResendConfirmationEmail(ctx, req.Email, userAgent, reqIP)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Email dispatched", http.StatusOK)
}

func (h *AuthHandler) ConfirmEmail(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	queryParams := c.Request.URL.Query()
	token := queryParams.Get("token")

	err := h.Service.ConfirmEmail(ctx, token, userAgent, reqIP)
	if err == nil {
		redirectUrl := utils.GenerateWebClientReleaseLink(h.cfg, "")
		c.Redirect(http.StatusFound, redirectUrl)
	} else {
		utils.ErrorMessage(c, "Error confirming email", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "", "Email confirmed", http.StatusOK)
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	err := h.Service.RequestPasswordReset(ctx, req.Email, userAgent, reqIP)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Email dispatched", http.StatusOK)
}

func (h *AuthHandler) ValidatePasswordReset(c *gin.Context) {

	ctx := c.Request.Context()
	queryParams := c.Request.URL.Query()
	tokenValue := queryParams.Get("token")

	token, err := h.Service.ValidatePasswordReset(ctx, tokenValue)
	if err != nil {
		utils.ErrorMessage(c, "Dispatch failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	redirectUrl := utils.GenerateWebClientReleaseLink(h.cfg, "")
	c.Redirect(http.StatusFound, fmt.Sprintf("%sreset-password/%s", redirectUrl, token))
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {

	ctx := c.Request.Context()
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

	err := h.Service.ResetPassword(ctx, form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Password reset failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Password reset complete", http.StatusOK)
}

func (h *AuthHandler) RegisterUser(c *gin.Context) {

	ctx := c.Request.Context()
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

	err := h.Service.RegisterUser(ctx, form, userAgent, loginIP)
	if err != nil {
		utils.ErrorMessage(c, "Registration failed", err.Error(), http.StatusUnauthorized, err)
		return
	}

	utils.SuccessMessage(c, "", "Registration complete", http.StatusOK)
}
