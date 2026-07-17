package models

import "time"

type Session struct {
	ID        string
	UserAgent string
	IP        string
	CreatedAt time.Time
	LastSeen  time.Time
}

type SessionInfo struct {
	ID        string    `json:"id"`
	Device    string    `json:"device"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	Current   bool      `json:"current"`
}

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
	InvitationID         *int64 `json:"invitation_id"`
}

type ResetPasswordForm struct {
	AuthForm
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
}

type ReqID struct {
	ID string `json:"id" binding:"required"`
}

type ReqEmail struct {
	Email string `json:"email" binding:"required,email"`
}
