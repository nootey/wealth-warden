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

type RolePermissionService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.RolePermissionRepository
}

func NewRolePermissionService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.RolePermissionRepository,
) *RolePermissionService {
	return &RolePermissionService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}

func (s *RolePermissionService) FetchAllRoles(withPermissions bool) ([]models.Role, error) {
	return s.Repo.FindAllRoles(withPermissions)
}

func (s *RolePermissionService) FetchAllPermissions() ([]models.Permission, error) {
	return s.Repo.FindAllPermissions()
}

func (s *RolePermissionService) FetchRoleByID(ID int64, withPermissions bool) (*models.Role, error) {
	record, err := s.Repo.FindRoleByID(nil, ID, withPermissions)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RolePermissionService) InsertRole(userID int64, req models.RoleReq) error {

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	role := models.Role{
		Name:        req.Name,
		Description: &req.Description,
		IsDefault:   false,
	}

	_, err := s.Repo.InsertRole(tx, &role)
	if err != nil {
		return err
	}

	permIDs := make([]int64, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		if p.ID > 0 {
			permIDs = append(permIDs, p.ID)
		}
	}
	if err = s.Repo.EnsurePermissionsExist(tx, permIDs); err != nil {
		return err
	}

	if err = s.Repo.AttachPermissionIDs(tx, role.ID, permIDs); err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	changes := utils.InitChanges()

	var desc string
	if role.Description != nil {
		desc = *role.Description
	}

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

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "role",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *RolePermissionService) UpdateRole(userID, id int64, req *models.RoleReq) error {
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
	exRole, err := s.Repo.FindRoleByID(tx, id, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find user with given id %w", err)
	}

	role := models.Role{
		ID:          exRole.ID,
		Name:        req.Name,
		Description: &req.Description,
	}

	_, err = s.Repo.UpdateRole(tx, role)
	if err != nil {
		tx.Rollback()
		return err
	}

	if !role.IsDefault {
		permIDs := make([]int64, 0, len(req.Permissions))
		for _, p := range req.Permissions {
			if p.ID > 0 {
				permIDs = append(permIDs, p.ID)
			}
		}
		if err := s.Repo.EnsurePermissionsExist(tx, permIDs); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.Repo.ReplaceRolePermissions(tx, role.ID, permIDs); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
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
		if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
			LoggingRepo: s.Ctx.LoggingService.Repo,
			Logger:      s.Ctx.Logger,
			Event:       "update",
			Category:    "role",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *RolePermissionService) DeleteRole(userID, id int64) error {

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

	role, err := s.Repo.FindRoleByID(tx, id, false)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find user with given id %w", err)
	}

	if role.IsDefault {
		tx.Rollback()
		return errors.New("default roles can not be deleted")
	}

	cnt, err := s.Repo.CountActiveUsersForRole(tx, role.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if cnt > 0 {
		tx.Rollback()
		return fmt.Errorf("cannot permanently delete category: %d active transactions still reference it", cnt)
	}

	if err := s.Repo.DeleteRole(tx, role.ID); err != nil {
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
