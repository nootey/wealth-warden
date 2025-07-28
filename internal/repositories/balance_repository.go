package repositories

import (
	"gorm.io/gorm"
)

type BalanceRepository struct {
	DB *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{DB: db}
}
