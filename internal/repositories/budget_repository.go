package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type BudgetRepository struct {
	Db *gorm.DB
}

func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{Db: db}
}

func (r *BudgetRepository) GetBudgetForMonth(user *models.User, year, month int) (*models.MonthlyBudget, error) {
	var record *models.MonthlyBudget
	result := r.Db.Where("organization_id = ? AND year = ? AND month = ?", *user.PrimaryOrganizationID, year, month).Find(&record)
	return record, result.Error
}
