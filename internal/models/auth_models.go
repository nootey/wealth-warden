package models

type AuthForm struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginForm struct {
	AuthForm
	RememberMe bool `json:"remember_me"`
}

type RegisterForm struct {
	AuthForm
	DisplayName          string `json:"display_name" validate:"required"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
}
