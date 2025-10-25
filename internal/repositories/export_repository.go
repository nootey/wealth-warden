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
