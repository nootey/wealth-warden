package repositories

import (
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	query := r.DB.Model(&models.Permission{})

	query = query.Where("name != ?", "root_access")

	query = query.Find(&records)

	return records, query.Error
}

func (r *RolePermissionRepository) FindRoleByID(tx *gorm.DB, id int64, withPermissions bool) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Role
	q := db.Where("id =?", id)

	if withPermissions {
		q.Preload("Permissions")
	}

	q.Find(&record)
	return &record, q.Error
}

func (r *RolePermissionRepository) FindRoleByName(tx *gorm.DB, roleName string) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Role
	result := db.Where("name =?", roleName).Find(&record)
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

func (r *RolePermissionRepository) EnsurePermissionsExist(tx *gorm.DB, ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("at least one permission is required")
	}
	var count int64
	if err := tx.Model(&models.Permission{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return fmt.Errorf("some permissions do not exist")
	}
	return nil
}

func (r *RolePermissionRepository) AttachPermissionIDs(tx *gorm.DB, roleID int64, permIDs []int64) error {
	if len(permIDs) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	rows := make([]models.RolePermission, 0, len(permIDs))
	for _, pid := range permIDs {
		rows = append(rows, models.RolePermission{RoleID: roleID, PermissionID: pid})
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *RolePermissionRepository) ReplaceRolePermissions(tx *gorm.DB, roleID int64, permIDs []int64) error {
	if len(permIDs) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	// Remove permissions that are no longer present.
	if err := tx.
		Where("role_id = ? AND permission_id NOT IN ?", roleID, permIDs).
		Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}

	// Add any missing permissions.
	rows := make([]models.RolePermission, 0, len(permIDs))
	for _, pid := range permIDs {
		rows = append(rows, models.RolePermission{RoleID: roleID, PermissionID: pid})
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
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
