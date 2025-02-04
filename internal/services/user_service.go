package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type UserService struct {
	UserRepo *repositories.UserRepository
	Config   *config.Config
}

func NewUserService(cfg *config.Config, userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		UserRepo: userRepo,
		Config:   cfg,
	}
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	users, err := s.UserRepo.GetAllUsers()
	if err != nil {
		return nil, err
	}

	return users, nil
}

// CreateUser adds a new user through the repository.
func (s *UserService) CreateUser(user *models.User) error {
	err := s.UserRepo.CreateUser(user)
	if err != nil {
		return err
	}
	return nil
}
