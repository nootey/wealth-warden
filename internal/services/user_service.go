package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
)

type UserService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.UserRepository
}

func NewUserService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.UserRepository,
) *UserService {
	return &UserService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	users, err := s.Repo.GetAllUsers()
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) FetchUserByID(ID uint) (*models.User, error) {
	record, err := s.Repo.GetUserByID(ID)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *UserService) CreateUser(user *models.User) error {
	err := s.Repo.CreateUser(user)
	if err != nil {
		return err
	}
	return nil
}
