package handlers

import (
	"fmt"
	"net/http"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/config"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/internal/sessions"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg        *config.Config
	middleware middleware.WebClientMiddlewareInterface
	Service    services.AuthServiceInterface
}

func NewAuthHandler(
	cfg *config.Config,
	middleware middleware.WebClientMiddlewareInterface,
	service services.AuthServiceInterface,
) *AuthHandler {
	return &AuthHandler{
		cfg:        cfg,
		middleware: middleware,
		Service:    service,
	}
}

func (h *AuthHandler) PublicRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/validate-email", h.ValidateInvitationEmail)
	apiGroup.POST("/login", h.LoginUser)
	apiGroup.POST("/logout", h.LogoutUser)
	apiGroup.POST("/signup", h.SignUp)
	apiGroup.POST("/request-password-reset", h.RequestPasswordReset)
	apiGroup.GET("/validate-password-reset", h.ValidatePasswordReset)
	apiGroup.POST("/reset-password", h.ResetPassword)
	apiGroup.GET("/confirm-email", h.ConfirmEmail)
}

func (h *AuthHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/current", h.GetAuthUser)
	apiGroup.POST("/resend-confirmation-email", h.ResendConfirmationEmail)
	apiGroup.POST("/setup/complete", h.CompleteSetup)
}

func (h *AuthHandler) LoginUser(c *gin.Context) {

	ctx := c.Request.Context()
	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.LoginForm
	if err := c.ShouldBindJSON(&form); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	user, err := h.Service.ValidateLogin(ctx, form.Email, form.Password, userAgent, loginIP)
	if err != nil {
		_ = c.Error(err)
		return
	}

	sessionID, maxAge, err := h.middleware.CreateLoginSession(ctx, user.ID, form.RememberMe, userAgent, loginIP)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	domain := h.middleware.CookieDomainForEnv()
	secure := h.middleware.CookieSecure()
	c.SetCookie(sessions.CookieName, sessionID, maxAge, "/", domain, secure, true)

	utils.SuccessMessage(c, "", "Logged in", http.StatusOK)
}

func (h *AuthHandler) GetAuthUser(c *gin.Context) {

	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	user, err := h.Service.GetCurrentUser(ctx, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) LogoutUser(c *gin.Context) {
	if sessionID, err := c.Cookie(sessions.CookieName); err == nil && sessionID != "" {
		if err := h.middleware.DestroySession(c.Request.Context(), sessionID); err != nil {
			// Cookie clearing below still logs the client out; the session dies at TTL.
			_ = c.Error(err)
		}
	}

	domain := h.middleware.CookieDomainForEnv()
	secure := h.middleware.CookieSecure()
	c.SetCookie(sessions.CookieName, "", -1, "/", domain, secure, true)
	utils.SuccessMessage(c, "", "Logged out", http.StatusOK)
}

func (h *AuthHandler) SignUp(c *gin.Context) {

	ctx := c.Request.Context()
	loginIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var form models.RegisterForm
	if err := c.ShouldBindJSON(&form); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := utils.SanitizeStruct(&form); err != nil {
		_ = c.Error(err)
		return
	}

	_, err := h.Service.SignUp(ctx, form, userAgent, loginIP)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Account created successfully!", http.StatusOK)
}

func (h *AuthHandler) ValidateInvitationEmail(c *gin.Context) {

	ctx := c.Request.Context()
	queryParams := c.Request.URL.Query()
	hash := queryParams.Get("token")

	if err := h.Service.ValidateInvitation(ctx, hash); err != nil {
		_ = c.Error(err)
		return
	}

	redirectUrl := utils.GenerateWebClientReleaseLink(h.cfg, "")
	c.Redirect(http.StatusFound, fmt.Sprintf("%s%s?token=%s", redirectUrl, "signup", hash))
}

func (h *AuthHandler) ResendConfirmationEmail(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	err := h.Service.ResendConfirmationEmail(ctx, req.Email, userAgent, reqIP)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Email dispatched", http.StatusOK)
}

func (h *AuthHandler) CompleteSetup(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var req models.CompleteSetupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.Service.CompleteSetup(ctx, userID, req); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Setup complete", http.StatusOK)
}

func (h *AuthHandler) ConfirmEmail(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	queryParams := c.Request.URL.Query()
	token := queryParams.Get("token")

	if err := h.Service.ConfirmEmail(ctx, token, userAgent, reqIP); err != nil {
		_ = c.Error(err)
		return
	}

	redirectUrl := utils.GenerateWebClientReleaseLink(h.cfg, "")
	c.Redirect(http.StatusFound, redirectUrl)
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {

	ctx := c.Request.Context()
	reqIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req models.ReqEmail
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	err := h.Service.RequestPasswordReset(ctx, req.Email, userAgent, reqIP)
	if err != nil {
		_ = c.Error(err)
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
		_ = c.Error(err)
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
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := utils.SanitizeStruct(&form); err != nil {
		_ = c.Error(err)
		return
	}

	err := h.Service.ResetPassword(ctx, form, userAgent, loginIP)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "", "Password reset complete", http.StatusOK)
}
