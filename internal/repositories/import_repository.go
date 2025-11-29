package repositories

import (
	"context"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type ImportRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindImportsByImportType(ctx context.Context, tx *gorm.DB, userID int64, importType string) ([]models.Import, error)
	FindImportByID(ctx context.Context, tx *gorm.DB, id, userID int64, importType string) (*models.Import, error)
	InsertImport(ctx context.Context, tx *gorm.DB, record models.Import) (int64, error)
	UpdateImport(ctx context.Context, tx *gorm.DB, id int64, fields map[string]interface{}) error
	DeleteImport(ctx context.Context, tx *gorm.DB, id, userID int64) error
}

type ImportRepository struct {
	db *gorm.DB
}

func NewImportRepository(db *gorm.DB) *ImportRepository {
	return &ImportRepository{db: db}
}

var _ ImportRepositoryInterface = (*ImportRepository)(nil)

func (r *ImportRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *ImportRepository) FindImportsByImportType(ctx context.Context, tx *gorm.DB, userID int64, importType string) ([]models.Import, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Import
	q := db.Model(&models.Import{}).
		Where("user_id = ? AND type = ?", userID, importType)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ImportRepository) FindImportByID(ctx context.Context, tx *gorm.DB, id, userID int64, importType string) (*models.Import, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Import
	q := db.Model(&models.Import{}).
		Where("id= ? AND user_id = ? AND type = ?", id, userID, importType)

	err := q.First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *ImportRepository) InsertImport(ctx context.Context, tx *gorm.DB, record models.Import) (int64, error) {

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

func (r *ImportRepository) UpdateImport(ctx context.Context, tx *gorm.DB, id int64, fields map[string]interface{}) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)
	return db.Model(&models.Import{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ImportRepository) DeleteImport(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Import{}).Error; err != nil {
		return err
	}

	return nil
}
