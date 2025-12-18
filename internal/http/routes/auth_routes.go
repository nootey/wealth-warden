package routes

import (
	"wealth-warden/internal/http/handlers"

	"github.com/gin-gonic/gin"
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
	apiGroup.GET("/confirm-email", handler.ConfirmEmail)
}

func AuthRoutes(apiGroup *gin.RouterGroup, handler *handlers.AuthHandler) {
	apiGroup.GET("/current", handler.GetAuthUser)
	apiGroup.POST("/resend-confirmation-email", handler.ResendConfirmationEmail)
}
