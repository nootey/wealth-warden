package services

import (
	"wealth-warden/server/internal/models"
	"wealth-warden/server/internal/repositories"
)

type UserService struct {
	UserRepo *repositories.UserRepository
}

func NewUserService(
	userRepo *repositories.UserRepository,
) *UserService {
	return &UserService{
		UserRepo: userRepo,
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
