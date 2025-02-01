package services

import "wealth-warden/internal/repositories"

type AuthService struct {
	UserRepo *repositories.AuthRepository
}

func NewAuthService(
	authRepo *repositories.AuthRepository,
) *AuthService {
	return &AuthService{
		UserRepo: authRepo,
	}
}
