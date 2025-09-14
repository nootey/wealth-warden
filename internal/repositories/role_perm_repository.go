package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type RolePermissionRepository struct {
	DB *gorm.DB
}

func NewRolePermissionRepositoryRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{DB: db}
}

func (r *RolePermissionRepository) FindAllRoles(withPermissions bool) ([]models.Role, error) {
	var records []models.Role
	query := r.DB.Model(&models.Role{})

	if withPermissions {
		query = query.Preload("Permissions")
	}

	query = query.Where("name != ?", "super-admin")

	result := query.Find(&records)
	return records, result.Error
}

func (r *RolePermissionRepository) FindAllPermissions() ([]models.Permission, error) {
	var records []models.Permission
	result := r.DB.Find(&records)
	return records, result.Error
}

func (r *RolePermissionRepository) FindRoleByID(id int64) (*models.Role, error) {
	var record models.Role
	result := r.DB.Where("id =?", id).Find(&record)
	return &record, result.Error
}

func (r *RolePermissionRepository) FindRoleByName(roleName string) (*models.Role, error) {
	var record models.Role
	result := r.DB.Where("name =?", roleName).Find(&record)
	return &record, result.Error
}
