package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type AccountRepository struct {
	DB *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{DB: db}
}

func (r *AccountRepository) FindAllAccountTypes(user *models.User) ([]models.AccountType, error) {
	var records []models.AccountType
	result := r.DB.Find(&records)
	return records, result.Error
}
