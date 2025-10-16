package repositories

import (
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type ImportRepository struct {
	DB *gorm.DB
}

func NewImportRepository(db *gorm.DB) *ImportRepository {
	return &ImportRepository{DB: db}
}

func (r *ImportRepository) FindImportsByImportType(tx *gorm.DB, userID int64, importType string) ([]models.Import, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.Import

	q := db.Model(&models.Import{}).
		Where("user_id = ? AND import_type = ?", userID, importType)

	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ImportRepository) FindImportByID(tx *gorm.DB, id, userID int64, importType string) (*models.Import, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Import

	q := db.Model(&models.Import{}).
		Where("id= ? AND user_id = ? AND import_type = ?", id, userID, importType)

	err := q.First(&record).Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *ImportRepository) InsertImport(tx *gorm.DB, record models.Import) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *ImportRepository) UpdateImport(tx *gorm.DB, id int64, fields map[string]interface{}) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	return db.Model(&models.Import{}).Where("id = ?", id).Updates(fields).Error
}
