package repositories

import (
	"context"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type ExportRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindExports(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Export, error)
	FindExportByID(ctx context.Context, tx *gorm.DB, id, userID int64) (*models.Export, error)
	FindExportsByExportType(ctx context.Context, tx *gorm.DB, userID int64, ExportType string) ([]models.Export, error)
	InsertExport(ctx context.Context, tx *gorm.DB, record *models.Export) error
	UpdateExport(ctx context.Context, tx *gorm.DB, id int64, fields map[string]interface{}) error
	DeleteExport(ctx context.Context, tx *gorm.DB, id, userID int64) error
}

type ExportRepository struct {
	db *gorm.DB
}

func NewExportRepository(db *gorm.DB) *ExportRepository {
	return &ExportRepository{db: db}
}

var _ ExportRepositoryInterface = (*ExportRepository)(nil)

func (r *ExportRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *ExportRepository) FindExports(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Export, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Export
	q := db.Model(&models.Export{}).
		Where("user_id = ?", userID)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ExportRepository) FindExportByID(ctx context.Context, tx *gorm.DB, id, userID int64) (*models.Export, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record *models.Export
	q := db.Model(&models.Export{}).
		Where("id = ? AND user_id = ?", id, userID)

	err := q.First(&record).Error
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r *ExportRepository) FindExportsByExportType(ctx context.Context, tx *gorm.DB, userID int64, ExportType string) ([]models.Export, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Export
	q := db.Model(&models.Export{}).
		Where("user_id = ? AND export_type = ?", userID, ExportType)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ExportRepository) InsertExport(ctx context.Context, tx *gorm.DB, record *models.Export) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Create(record).Error
}

func (r *ExportRepository) UpdateExport(ctx context.Context, tx *gorm.DB, id int64, fields map[string]interface{}) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Model(&models.Export{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ExportRepository) DeleteExport(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Export{}).Error; err != nil {
		return err
	}

	return nil
}
