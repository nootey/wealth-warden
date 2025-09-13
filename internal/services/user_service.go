package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
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

func (s *UserService) FetchUserByID(ID int64) (*models.User, error) {
	record, err := s.Repo.GetUserByID(ID)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *UserService) CreateInvitation(invitation *models.Invitation) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	hash, err := utils.GenerateSecureToken(64)
	if err != nil {
		return err
	}

	invitation.Hash = hash

	_, err = s.Repo.InsertInvitation(tx, invitation)
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	//err = s.Mailer.SendRegistrationEmail(invitation.Email, hash)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (s *UserService) CreateUser(user *models.User) error {
	err := s.Repo.CreateUser(user)
	if err != nil {
		return err
	}
	return nil
}
