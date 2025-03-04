package repositories

import "gorm.io/gorm"

type SavingsRepository struct {
	Db *gorm.DB
}

func NewSavingsRepository(db *gorm.DB) *SavingsRepository {
	return &SavingsRepository{Db: db}
}
