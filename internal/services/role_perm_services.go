package services

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
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
