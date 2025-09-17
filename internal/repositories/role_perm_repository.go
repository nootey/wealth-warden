package repositories

import (
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
)

type RolePermissionRepository struct {
	DB *gorm.DB
}

func NewRolePermissionRepositoryRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{DB: db}
}

func (r *RolePermissionRepository) CountActiveUsersForRole(tx *gorm.DB, roleID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var cnt int64
	err := db.Model(&models.User{}).
		Where("role_id = ?", roleID).
		Count(&cnt).Error
	return cnt, err
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

func (r *RolePermissionRepository) FindRoleByID(tx *gorm.DB, id int64) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Role
	result := r.DB.Where("id =?", id).Find(&record)
	return &record, result.Error
}

func (r *RolePermissionRepository) FindRoleByName(tx *gorm.DB, roleName string) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Role
	result := r.DB.Where("name =?", roleName).Find(&record)
	return &record, result.Error
}

func (r *RolePermissionRepository) InsertRole(tx *gorm.DB, record *models.Role) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *RolePermissionRepository) UpdateRole(tx *gorm.DB, record models.Role) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.Role{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":        record.Name,
			"description": record.Description,
			"updated_at":  time.Now(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *RolePermissionRepository) DeleteRole(tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	return db.Where("id = ?", id).
		Delete(&models.Role{}).Error
}
