package repositories

import "gorm.io/gorm"

type BudgetRepository struct {
	Db *gorm.DB
}

func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{Db: db}
}
