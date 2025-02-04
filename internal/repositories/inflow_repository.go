package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type InflowRepository struct {
	db *gorm.DB
}

func NewInflowRepository(db *gorm.DB) *InflowRepository {
	return &InflowRepository{db: db}
}

func (r *InflowRepository) GetAllInflowTypes() ([]models.InflowType, error) {
	var inflowTypes []models.InflowType
	result := r.db.Find(&inflowTypes)
	return inflowTypes, result.Error
}

func (r *InflowRepository) SaveInflowType(inflowType *models.InflowType) error {

	if err := r.db.Create(&inflowType).Error; err != nil {
		return err
	}
	return nil
}
