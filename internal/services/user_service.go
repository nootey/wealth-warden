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

func (s *UserService) FetchUserByID(ID uint) (*models.User, error) {
	record, err := s.UserRepo.GetUserByID(ID, false)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *UserService) CreateUser(user *models.User) error {
	err := s.UserRepo.CreateUser(user)
	if err != nil {
		return err
	}
	return nil
}
