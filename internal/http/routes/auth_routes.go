package routes

import (
	"github.com/gin-gonic/gin"
	"wealth-warden/internal/http/handlers"
)

func PublicAuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/validate-email", handler.ValidateInvitationEmail)
	apiGroup.POST("/login", handler.LoginUser)
	apiGroup.POST("/logout", handler.LogoutUser)
	apiGroup.POST("/signup", handler.SignUp)
	apiGroup.POST("/register", handler.RegisterUser)
	apiGroup.POST("/request-password-reset", handler.RequestPasswordReset)
	apiGroup.GET("/validate-password-reset", handler.ValidatePasswordReset)
	apiGroup.POST("/reset-password", handler.ResetPassword)
}

func AuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/current", handler.GetAuthUser)
	apiGroup.GET("/confirm-email", handler.ConfirmEmail)
	apiGroup.POST("/resend-confirmation-email", handler.ResendConfirmationEmail)
}
