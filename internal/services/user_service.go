package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type UserService struct {
	Config      *config.Config
	Ctx         *DefaultServiceContext
	Repo        *repositories.UserRepository
	RoleService *RolePermissionService
}

func NewUserService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.UserRepository,
	roleService *RolePermissionService,
) *UserService {
	return &UserService{
		Ctx:         ctx,
		Config:      cfg,
		Repo:        repo,
		RoleService: roleService,
	}
}

func (s *UserService) FetchUsersPaginated(p utils.PaginationParams, includeDeleted bool) ([]models.User, *utils.Paginator, error) {

	totalRecords, err := s.Repo.CountUsers(p.Filters, includeDeleted)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.Repo.FindUsers(offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeDeleted)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
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

func (s *UserService) InsertInvitation(invitation *models.Invitation) error {

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

	name := utils.EmailToName(invitation.Email)

	err = s.Ctx.AuthService.mailer.SendRegistrationEmail(invitation.Email, name, hash)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) UpdateUser(userID, id int64, req *models.UserReq) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load existing user
	exUsr, err := s.Repo.FindUserByID(tx, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find user with given id %w", err)
	}

	// Load old relations
	oldRole, err := s.RoleService.Repo.FindRoleByID(tx, exUsr.RoleID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find existing role: %w", err)
	}

	// Resolve new relations
	newRole, err := s.RoleService.Repo.FindRoleByID(tx, req.RoleID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find role wit given id: %w", err)
	}

	usr := models.User{
		ID:          exUsr.ID,
		DisplayName: req.DisplayName,
		RoleID:      newRole.ID,
	}

	if req.Password != nil {
		if req.Password != req.PasswordConfirmation {
			tx.Rollback()
			return errors.New("password confirmation must match provided password")
		}

		hashedPassword, err := utils.HashAndSaltPassword(*req.Password)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to hash password: %w", err)
		}

		usr.Password = hashedPassword
	}

	_, err = s.Repo.UpdateUser(tx, usr)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(oldRole.Name, newRole.Name, changes, "role")
	utils.CompareChanges(exUsr.DisplayName, usr.DisplayName, changes, "display_name")

	if !changes.IsEmpty() {
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "user",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *UserService) DeleteUser(userID, id int64) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	usr, err := s.Repo.FindUserByID(tx, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find user with given id %w", err)
	}

	role, err := s.RoleService.Repo.FindRoleByID(tx, usr.RoleID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find role wit given id: %w", err)
	}

	if err := s.Repo.DeleteUser(tx, usr.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(role.Name, "", changes, "role")
	utils.CompareChanges(usr.DisplayName, "", changes, "display_name")

	if !changes.IsEmpty() {
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "delete",
			Category:    "user",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}
