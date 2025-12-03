package repositories

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RolePermissionRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	CountActiveUsersForRole(ctx context.Context, tx *gorm.DB, roleID int64) (int64, error)
	FindAllRoles(ctx context.Context, tx *gorm.DB, withPermissions bool) ([]models.Role, error)
	FindAllPermissions(ctx context.Context, tx *gorm.DB) ([]models.Permission, error)
	FindRoleByID(ctx context.Context, tx *gorm.DB, id int64, withPermissions bool) (*models.Role, error)
	FindRoleByName(ctx context.Context, tx *gorm.DB, roleName string) (*models.Role, error)
	InsertRole(ctx context.Context, tx *gorm.DB, record *models.Role) (int64, error)
	EnsurePermissionsExist(ctx context.Context, tx *gorm.DB, ids []int64) error
	AttachPermissionIDs(ctx context.Context, tx *gorm.DB, roleID int64, permIDs []int64) error
	ReplaceRolePermissions(ctx context.Context, tx *gorm.DB, roleID int64, permIDs []int64) error
	UpdateRole(ctx context.Context, tx *gorm.DB, record models.Role) (int64, error)
	DeleteRole(ctx context.Context, tx *gorm.DB, id int64) error
}

type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepositoryRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

var _ RolePermissionRepositoryInterface = (*RolePermissionRepository)(nil)

func (r *RolePermissionRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *RolePermissionRepository) CountActiveUsersForRole(ctx context.Context, tx *gorm.DB, roleID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var cnt int64
	err := db.Model(&models.User{}).
		Where("role_id = ?", roleID).
		Count(&cnt).Error
	return cnt, err
}

func (r *RolePermissionRepository) FindAllRoles(ctx context.Context, tx *gorm.DB, withPermissions bool) ([]models.Role, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Role
	query := db.Model(&models.Role{})

	if withPermissions {
		query = query.Preload("Permissions")
	}

	query = query.Where("name != ?", "super-admin")

	result := query.Find(&records)
	return records, result.Error
}

func (r *RolePermissionRepository) FindAllPermissions(ctx context.Context, tx *gorm.DB) ([]models.Permission, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Permission
	query := db.Model(&models.Permission{}).
		Where("name != ?", "root_access").
		Find(&records)

	return records, query.Error
}

func (r *RolePermissionRepository) FindRoleByID(ctx context.Context, tx *gorm.DB, id int64, withPermissions bool) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Role
	q := db.Where("id =?", id)

	if withPermissions {
		q.Preload("Permissions")
	}

	q.Find(&record)
	return &record, q.Error
}

func (r *RolePermissionRepository) FindRoleByName(ctx context.Context, tx *gorm.DB, roleName string) (*models.Role, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Role
	result := db.Where("name =?", roleName).Find(&record)
	return &record, result.Error
}

func (r *RolePermissionRepository) InsertRole(ctx context.Context, tx *gorm.DB, record *models.Role) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *RolePermissionRepository) EnsurePermissionsExist(ctx context.Context, tx *gorm.DB, ids []int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if len(ids) == 0 {
		return fmt.Errorf("at least one permission is required")
	}
	var count int64
	if err := db.Model(&models.Permission{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}

	if count != int64(len(ids)) {
		return fmt.Errorf("some permissions do not exist")
	}

	return nil
}

func (r *RolePermissionRepository) AttachPermissionIDs(ctx context.Context, tx *gorm.DB, roleID int64, permIDs []int64) error {

	if len(permIDs) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	rows := make([]models.RolePermission, 0, len(permIDs))
	for _, pid := range permIDs {
		rows = append(rows, models.RolePermission{RoleID: roleID, PermissionID: pid})
	}

	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *RolePermissionRepository) ReplaceRolePermissions(ctx context.Context, tx *gorm.DB, roleID int64, permIDs []int64) error {

	if len(permIDs) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	// Remove permissions that are no longer present.
	if err := db.Where("role_id = ? AND permission_id NOT IN ?", roleID, permIDs).
		Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}

	// Add any missing permissions.
	rows := make([]models.RolePermission, 0, len(permIDs))
	for _, pid := range permIDs {
		rows = append(rows, models.RolePermission{RoleID: roleID, PermissionID: pid})
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *RolePermissionRepository) UpdateRole(ctx context.Context, tx *gorm.DB, record models.Role) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.Role{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":        record.Name,
			"description": record.Description,
			"updated_at":  time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *RolePermissionRepository) DeleteRole(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("id = ?", id).
		Delete(&models.Role{}).Error
}
