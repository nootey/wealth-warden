package services

import (
	"encoding/json"
	"errors"
	"fmt"
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
	record, err := s.Repo.FindUserByID(nil, ID)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *UserService) FetchUserByToken(tokenType, tokenValue string) (*models.User, error) {

	token, err := s.Repo.FindTokenByValue(nil, tokenType, tokenValue)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, errors.New("no valid token found")
	}

	raw, err := utils.UnwrapToken(token, "user_id")
	if err != nil {
		return nil, fmt.Errorf("no user_id in token data")
	}

	num := raw.(json.Number)
	userID, err := num.Int64()
	if err != nil {
		return nil, fmt.Errorf("invalid user_id in token data: %v", err)
	}

	user, err := s.Repo.FindUserByID(nil, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) FetchInvitationByHash(hash string) (*models.Invitation, error) {
	record, err := s.Repo.FindUserInvitationByHash(nil, hash)
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

	err = s.Ctx.AuthService.mailer.SendRegistrationEmail(invitation.Email, invitation.DisplayName, hash)
	if err != nil {
		return err
	}

	return nil
}
