package handlers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/middleware"
	"wealth-warden/pkg/utils"
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

	//loginIP := c.ClientIP()

	var loginForm models.LoginForm
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusBadRequest)(c, err)
		return
	}

	user, _ := h.Service.UserRepo.GetUserByEmail(loginForm.Email)
	if user == nil {
		utils.ErrorMessage("Error occurred", "Incorrect credentials", http.StatusBadRequest)(c, nil)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginForm.Password))
	if err != nil {
		utils.ErrorMessage("Error occurred", "Incorrect credentials", http.StatusBadRequest)(c, err)
		return
	}

	accessToken, refreshToken, err := middleware.GenerateLoginTokens(user.ID, loginForm.RememberMe)
	if err != nil {
		utils.ErrorMessage("Authentication error", err.Error(), http.StatusBadRequest)(c, err)
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

	// Set cookies and return success message as in your original function
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("wwa", accessToken, 60*15, "/", h.Config.ClientDomain, h.Config.Release, true)
	c.SetCookie("wwr", refreshToken, expiresAt, "/", h.Config.ClientDomain, h.Config.Release, true)

	utils.SuccessMessage("200", "Logged in", http.StatusOK)(c.Writer, c.Request)
}

func (h *AuthHandler) GetAuthUser(c *gin.Context) {
	user, err := h.Service.GetCurrentUser(c)
	if err != nil {
		utils.ErrorMessage("Error occurred", err.Error(), http.StatusInternalServerError)(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) LogoutUser(c *gin.Context) {
	c.SetCookie("wwa", "", -1, "/", h.Config.ClientDomain, h.Config.Release, true)
	c.SetCookie("wwr", "", -1, "/", h.Config.ClientDomain, h.Config.Release, true)
	utils.SuccessMessage("", "Logged out", http.StatusOK)(c.Writer, c.Request)
}
