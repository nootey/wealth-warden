package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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

	var loginForm models.LoginForm
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusBadRequest, err)
		return
	}

	userPassword, _ := h.Service.UserRepo.GetPasswordByEmail(loginForm.Email)
	if userPassword == "" {

		changes := utils.InitChanges()
		description := "user does not exist"
		utils.CompareChanges("", loginForm.Email, changes, "email")

		logErr := h.Service.LoggingService.LoggingRepo.InsertAccessLog(nil, "fail", "login", nil, &loginIP, &userAgent, nil, changes, &description)
		if logErr != nil {
			utils.ErrorMessage(c, "Error occurred", logErr.Error(), http.StatusBadRequest, logErr)
			return
		}

		utils.ErrorMessage(c, "Error occurred", logErr.Error(), http.StatusUnauthorized, logErr)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(loginForm.Password))
	if err != nil {

		changes := utils.InitChanges()
		description := "incorrect password"
		utils.CompareChanges("", loginForm.Email, changes, "email")

		logErr := h.Service.LoggingService.LoggingRepo.InsertAccessLog(nil, "fail", "login", nil, &loginIP, &userAgent, nil, changes, &description)
		if logErr != nil {
			utils.ErrorMessage(c, "Error occurred", logErr.Error(), http.StatusBadRequest, logErr)
			return
		}

		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusUnauthorized, err)
		return
	}

	user, _ := h.Service.UserRepo.GetUserByEmail(loginForm.Email, false)
	if user == nil {
		err = errors.New("user data unavailable")
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	accessToken, refreshToken, err := h.Service.WebClientMiddleware.GenerateLoginTokens(user.ID, loginForm.RememberMe)
	if err != nil {
		utils.ErrorMessage(c, "Authentication error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	var expiresAt int
	if loginForm.RememberMe {
		expiresAt = 3600 * 24 * 14
	} else {
		expiresAt = 3600 * 24
	}

	//if user.ValidatedAt.IsZero() {
	//	utils.SuccessMessage("401", "Logged in", http.StatusOK)(c.Writer, c.Request)
	//	return
	//}
	//
	//if secrets.IPLog == false {
	//	loginIP = ""
	//}

	logErr := h.Service.LoggingService.LoggingRepo.InsertAccessLog(nil, "success", "login", nil, &loginIP, &userAgent, nil, nil, nil)
	if logErr != nil {
		utils.ErrorMessage(c, "Error occurred", logErr.Error(), http.StatusBadRequest, logErr)
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
