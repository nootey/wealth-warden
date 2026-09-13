package services

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/mailer"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = apperr.New(apperr.NotFound, "User not found")
	ErrInvitationNotFound = apperr.New(apperr.NotFound, "Invitation not found")
	ErrInvalidRoleID      = apperr.New(apperr.Validation, "The selected role does not exist")
	ErrInvalidLink        = apperr.New(apperr.Unauthorized, "This link is no longer valid")
)

type UserServiceInterface interface {
	GetAllActiveUserIDs(ctx context.Context) ([]int64, error)
	FetchUsersPaginated(ctx context.Context, p utils.PaginationParams, includeDeleted bool) ([]models.User, *utils.Paginator, error)
	FetchInvitationsPaginated(ctx context.Context, p utils.PaginationParams) ([]models.Invitation, *utils.Paginator, error)
	FetchUserByID(ctx context.Context, ID int64) (*models.User, error)
	SearchUsersByEmail(ctx context.Context, q string) ([]models.UserLookup, error)
	FetchUserByToken(ctx context.Context, tokenType, tokenValue string) (*models.User, error)
	FetchInvitationByHash(ctx context.Context, hash string) (*models.Invitation, error)
	InsertInvitation(ctx context.Context, userID int64, req models.InvitationReq) (int64, error)
	UpdateUser(ctx context.Context, userID, id int64, req *models.UserReq) (int64, error)
	DeleteUser(ctx context.Context, userID, id int64) error
	ResendInvitation(ctx context.Context, userID, id int64) (int64, error)
	DeleteInvitation(ctx context.Context, userID, id int64) error
}

type UserService struct {
	repo          *repositories.UserRepository
	roleRepo      repositories.RolePermissionRepositoryInterface
	jobDispatcher jobqueue.Dispatcher
	mailer        *mailer.Mailer
}

func NewUserService(
	repo *repositories.UserRepository,
	roleRepo *repositories.RolePermissionRepository,
	jobDispatcher jobqueue.Dispatcher,
	mailer *mailer.Mailer,
) *UserService {
	return &UserService{
		repo:          repo,
		roleRepo:      roleRepo,
		jobDispatcher: jobDispatcher,
		mailer:        mailer,
	}
}

var _ UserServiceInterface = (*UserService)(nil)

func (s *UserService) GetAllActiveUserIDs(ctx context.Context) ([]int64, error) {
	return s.repo.GetAllActiveUserIDs(ctx, nil)
}

func (s *UserService) SearchUsersByEmail(ctx context.Context, q string) ([]models.UserLookup, error) {

	q = strings.TrimSpace(q)
	if len(q) < 2 {
		return nil, apperr.New(apperr.Invalid, "Search needs at least 2 characters")
	}

	return s.repo.SearchUsersByEmail(ctx, nil, q, 20)
}

func (s *UserService) FetchUsersPaginated(ctx context.Context, p utils.PaginationParams, includeDeleted bool) ([]models.User, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountUsers(ctx, nil, p.Filters, includeDeleted)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindUsers(ctx, nil, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, includeDeleted)
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

func (s *UserService) FetchInvitationsPaginated(ctx context.Context, p utils.PaginationParams) ([]models.Invitation, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvitations(ctx, nil, p.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvitations(ctx, nil, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters)
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

func (s *UserService) FetchUserByID(ctx context.Context, ID int64) (*models.User, error) {
	record, err := s.repo.FindUserByID(ctx, nil, ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return record, nil
}

func (s *UserService) FetchUserByToken(ctx context.Context, tokenType, tokenValue string) (*models.User, error) {

	token, err := s.repo.FindTokenByValue(ctx, nil, tokenType, tokenValue)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidLink
		}
		return nil, err
	}

	raw, err := utils.UnwrapToken(token, "user_id")
	if err != nil {
		return nil, apperr.Wrap(apperr.Unauthorized, ErrInvalidLink.Message, err)
	}

	num := raw.(json.Number)
	userID, err := num.Int64()
	if err != nil {
		return nil, apperr.Wrap(apperr.Unauthorized, ErrInvalidLink.Message, err)
	}

	user, err := s.repo.FindUserByID(ctx, nil, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidLink
		}
		return nil, err
	}

	return user, nil
}

func (s *UserService) FetchInvitationByHash(ctx context.Context, hash string) (*models.Invitation, error) {
	record, err := s.repo.FindUserInvitationByHash(ctx, nil, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	return record, nil
}

func (s *UserService) InsertInvitation(ctx context.Context, userID int64, req models.InvitationReq) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	hash, err := utils.GenerateSecureToken(64)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	invitation := &models.Invitation{
		Email:  req.Email,
		RoleID: req.RoleID,
		Hash:   hash,
	}

	invID, err := s.repo.InsertInvitation(ctx, tx, invitation)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	changes := utils.InitChanges()

	role, err := s.roleRepo.FindRoleByID(ctx, tx, invitation.RoleID, false)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrInvalidRoleID
		}
		return 0, err
	}

	utils.CompareChanges("", strconv.FormatInt(invID, 10), changes, "id")
	utils.CompareChanges("", role.Name, changes, "role")
	utils.CompareChanges("", invitation.Email, changes, "email")

	err = tx.Commit().Error
	if err != nil {
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "create",
		Category:    "invitation",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	name := utils.EmailToName(invitation.Email)

	if s.mailer != nil {
		err = s.mailer.SendRegistrationEmail(invitation.Email, name, hash)
		if err != nil {
			return 0, err
		}
	}

	return invID, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID, id int64, req *models.UserReq) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	exUsr, err := s.repo.FindUserByID(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}

	// The role already on the user. A missing one is broken data, so it stays a generic 500.
	oldRole, err := s.roleRepo.FindRoleByID(ctx, tx, exUsr.RoleID, false)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// The role the client picked. A missing one is a bad field.
	newRole, err := s.roleRepo.FindRoleByID(ctx, tx, req.RoleID, false)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrInvalidRoleID
		}
		return 0, err
	}

	usr := models.User{
		ID:             exUsr.ID,
		DisplayName:    req.DisplayName,
		RoleID:         newRole.ID,
		EmailConfirmed: req.EmailConfirmed,
	}

	uID, err := s.repo.UpdateUser(ctx, tx, usr)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if req.Password != nil {
		if req.Password != req.PasswordConfirmation {
			tx.Rollback()
			return 0, apperr.New(apperr.Validation, "password confirmation must match provided password")
		}

		hashedPassword, err := utils.HashAndSaltPassword(*req.Password)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		err = s.repo.UpdateUserPassword(ctx, tx, exUsr.ID, hashedPassword)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(oldRole.Name, newRole.Name, changes, "role")
	utils.CompareChanges(exUsr.DisplayName, usr.DisplayName, changes, "display_name")
	utils.CompareDateChange(exUsr.EmailConfirmed, usr.EmailConfirmed, changes, "email_confirmed")

	if changes.HasChanges() {
		changes.Stamp("id", strconv.FormatInt(uID, 10))
		if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       "update",
			Category:    "user",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return 0, err
		}
	}

	return uID, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID, id int64) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	usr, err := s.repo.FindUserByID(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	role, err := s.roleRepo.FindRoleByID(ctx, tx, usr.RoleID, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	newEmail := usr.Email + "_" + strconv.FormatInt(usr.ID, 10)

	if err := s.repo.DeleteUser(ctx, tx, usr.ID, newEmail); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(usr.ID, 10), changes, "id")
	utils.CompareChanges(role.Name, "", changes, "role")
	utils.CompareChanges(usr.DisplayName, "", changes, "display_name")
	utils.CompareChanges(usr.Email, "", changes, "email")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *UserService) ResendInvitation(ctx context.Context, userID, id int64) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	invitation, err := s.repo.FindInvitationByID(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrInvitationNotFound
		}
		return 0, err
	}

	hash, err := utils.GenerateSecureToken(64)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	newInv := &models.Invitation{
		Email:  invitation.Email,
		RoleID: invitation.RoleID,
		Hash:   hash,
	}

	// Delete existing invitation
	err = s.repo.DeleteInvitation(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Insert new invitation
	invID, err := s.repo.InsertInvitation(ctx, tx, newInv)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	changes := utils.InitChanges()

	role, err := s.roleRepo.FindRoleByID(ctx, tx, newInv.RoleID, false)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	utils.CompareChanges("", role.Name, changes, "role")
	utils.CompareChanges("", newInv.Email, changes, "email")

	err = tx.Commit().Error
	if err != nil {
		return 0, err
	}

	name := utils.EmailToName(newInv.Email)

	if s.mailer != nil {
		err = s.mailer.SendRegistrationEmail(newInv.Email, name, hash)
		if err != nil {
			return 0, err
		}
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "resend",
		Category:    "invitation",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	return invID, nil
}

func (s *UserService) DeleteInvitation(ctx context.Context, userID, id int64) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	inv, err := s.repo.FindInvitationByID(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvitationNotFound
		}
		return err
	}

	role, err := s.roleRepo.FindRoleByID(ctx, tx, inv.RoleID, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := s.repo.DeleteInvitation(ctx, tx, inv.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(inv.Email, "", changes, "email")
	utils.CompareChanges(role.Name, "", changes, "role")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       "delete",
			Category:    "invitation",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}
