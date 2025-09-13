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
