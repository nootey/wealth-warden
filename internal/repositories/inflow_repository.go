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

func (r *InflowRepository) CountInflows() (int64, error) {
	var totalRecords int64
	err := r.db.Model(&models.Inflow{}).Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *InflowRepository) GetInflows(offset, limit int, sortField, sortOrder string) ([]models.Inflow, error) {
	var inflows []models.Inflow
	orderBy := sortField + " " + sortOrder

	err := r.db.
		Preload("InflowCategory").
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&inflows).Error

	if err != nil {
		return nil, err
	}

	return inflows, nil
}

func (r *InflowRepository) GetAllInflowCategories() ([]models.InflowCategory, error) {
	var inflowCategories []models.InflowCategory
	result := r.db.Find(&inflowCategories)
	return inflowCategories, result.Error
}

func (r *InflowRepository) SaveInflow(inflow *models.Inflow) error {

	if err := r.db.Create(&inflow).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) SaveInflowCategory(inflowCategory *models.InflowCategory) error {

	if err := r.db.Create(&inflowCategory).Error; err != nil {
		return err
	}
	return nil
}
