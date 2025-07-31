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

func (r *AccountRepository) InsertAccount(tx *gorm.DB, newRecord *models.Account) (uint, error) {
	if err := tx.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) InsertBalance(tx *gorm.DB, newRecord *models.Balance) (uint, error) {
	if err := tx.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}
