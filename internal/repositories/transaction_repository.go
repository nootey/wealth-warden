package repositories

import (
	"gorm.io/gorm"
)

type TransactionRepository struct {
	DB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

func (r *TransactionRepository) FindAllCategories(user *models.User) ([]models.Category, error) {
	var records []models.Category
	result := r.DB.Find(&records)
	return records, result.Error
}
