package repositories

import (
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type ExportRepository struct {
	DB *gorm.DB
}

func NewExportRepository(db *gorm.DB) *ExportRepository {
	return &ExportRepository{DB: db}
}

func (r *ExportRepository) FindExports(tx *gorm.DB, userID int64) ([]models.Export, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.Export

	q := db.Model(&models.Export{}).
		Where("user_id = ?", userID)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ExportRepository) FindExportByID(tx *gorm.DB, id, userID int64) (*models.Export, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record *models.Export

	q := db.Model(&models.Export{}).
		Where("id = ? AND user_id = ?", id, userID)

	err := q.First(&record).Error
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r *ExportRepository) FindExportsByExportType(tx *gorm.DB, userID int64, ExportType string) ([]models.Export, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.Export

	q := db.Model(&models.Export{}).
		Where("user_id = ? AND export_type = ?", userID, ExportType)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ExportRepository) InsertExport(tx *gorm.DB, record *models.Export) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	return db.Create(record).Error
}

func (r *ExportRepository) UpdateExport(tx *gorm.DB, id int64, fields map[string]interface{}) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	return db.Model(&models.Export{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ExportRepository) DeleteExport(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Export{}).Error; err != nil {
		return err
	}

	return nil
}
