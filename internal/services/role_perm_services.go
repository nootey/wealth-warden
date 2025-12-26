package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type RolePermissionServiceInterface interface {
	FetchAllRoles(ctx context.Context, withPermissions bool) ([]models.Role, error)
	FetchAllPermissions(ctx context.Context) ([]models.Permission, error)
	FetchRoleByID(ctx context.Context, ID int64, withPermissions bool) (*models.Role, error)
	InsertRole(ctx context.Context, userID int64, req models.RoleReq) (int64, error)
	UpdateRole(ctx context.Context, userID, id int64, req *models.RoleReq) (int64, error)
	DeleteRole(ctx context.Context, userID, id int64) error
}
type RolePermissionService struct {
	repo          repositories.RolePermissionRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
}

func NewRolePermissionService(
	repo *repositories.RolePermissionRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
) *RolePermissionService {
	return &RolePermissionService{
		repo:          repo,
		loggingRepo:   loggingRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ RolePermissionServiceInterface = (*RolePermissionService)(nil)

func (s *RolePermissionService) FetchAllRoles(ctx context.Context, withPermissions bool) ([]models.Role, error) {
	return s.repo.FindAllRoles(ctx, nil, withPermissions)
}

func (s *RolePermissionService) FetchAllPermissions(ctx context.Context) ([]models.Permission, error) {
	return s.repo.FindAllPermissions(ctx, nil)
}

func (s *RolePermissionService) FetchRoleByID(ctx context.Context, ID int64, withPermissions bool) (*models.Role, error) {
	record, err := s.repo.FindRoleByID(ctx, nil, ID, withPermissions)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RolePermissionService) InsertRole(ctx context.Context, userID int64, req models.RoleReq) (int64, error) {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	role := models.Role{
		Name:        req.Name,
		Description: &req.Description,
		IsDefault:   false,
	}

	roleID, err := s.repo.InsertRole(ctx, tx, &role)
	if err != nil {
		return 0, err
	}

	permIDs := make([]int64, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		if p.ID > 0 {
			permIDs = append(permIDs, p.ID)
		}
	}
	if err = s.repo.EnsurePermissionsExist(ctx, tx, permIDs); err != nil {
		return 0, err
	}

	if err = s.repo.AttachPermissionIDs(ctx, tx, role.ID, permIDs); err != nil {
		return 0, err
	}

	err = tx.Commit().Error
	if err != nil {
		return 0, err
	}

	changes := utils.InitChanges()

	var desc string
	if role.Description != nil {
		desc = *role.Description
	}

	utils.CompareChanges("", strconv.FormatInt(roleID, 10), changes, "id")
	utils.CompareChanges("", role.Name, changes, "name")
	utils.CompareChanges("", desc, changes, "description")

	addedPerms := make([]map[string]interface{}, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		addedPerms = append(addedPerms, map[string]interface{}{
			"name": p.Name,
		})
	}

	permBytes, _ := json.Marshal(addedPerms)
	permString := string(permBytes)

	utils.CompareChanges("", permString, changes, "permissions")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "role",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

	return roleID, nil
}

func (s *RolePermissionService) UpdateRole(ctx context.Context, userID, id int64, req *models.RoleReq) (int64, error) {

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

	// Load existing user
	exRole, err := s.repo.FindRoleByID(ctx, tx, id, true)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find user with given id %w", err)
	}

	role := models.Role{
		ID:          exRole.ID,
		Name:        req.Name,
		Description: &req.Description,
	}

	roleID, err := s.repo.UpdateRole(ctx, tx, role)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if !role.IsDefault {
		permIDs := make([]int64, 0, len(req.Permissions))
		for _, p := range req.Permissions {
			if p.ID > 0 {
				permIDs = append(permIDs, p.ID)
			}
		}
		if err := s.repo.EnsurePermissionsExist(ctx, tx, permIDs); err != nil {
			tx.Rollback()
			return 0, err
		}
		if err := s.repo.ReplaceRolePermissions(ctx, tx, role.ID, permIDs); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(roleID, 10), changes, "id")
	utils.CompareChanges(exRole.Name, role.Name, changes, "name")
	utils.CompareChanges(utils.SafeString(exRole.Description), utils.SafeString(role.Description), changes, "description")

	if !role.IsDefault {
		prevPerms := make([]map[string]interface{}, 0, len(exRole.Permissions))
		for _, p := range exRole.Permissions {
			prevPerms = append(prevPerms, map[string]interface{}{"name": p.Name})
		}
		newPerms := make([]map[string]interface{}, 0, len(req.Permissions))
		for _, p := range req.Permissions {
			newPerms = append(newPerms, map[string]interface{}{"name": p.Name})
		}

		prevBytes, _ := json.Marshal(prevPerms)
		newBytes, _ := json.Marshal(newPerms)
		utils.CompareChanges(string(prevBytes), string(newBytes), changes, "permissions")
	}

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "role",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return 0, err
		}
	}

	return roleID, nil
}

func (s *RolePermissionService) DeleteRole(ctx context.Context, userID, id int64) error {

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

	role, err := s.repo.FindRoleByID(ctx, tx, id, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find user with given id %w", err)
	}

	if role.IsDefault {
		tx.Rollback()
		return errors.New("default roles can not be deleted")
	}

	cnt, err := s.repo.CountActiveUsersForRole(ctx, tx, role.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if cnt > 0 {
		tx.Rollback()
		return fmt.Errorf("cannot permanently delete category: %d active transactions still reference it", cnt)
	}

	if err := s.repo.DeleteRole(ctx, tx, role.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(role.Name, "", changes, "name")
	utils.CompareChanges(utils.SafeString(role.Description), "", changes, "description")

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
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
