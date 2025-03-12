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

func (r *BudgetRepository) InsertBudget(tx *gorm.DB, user *models.User, record *models.MonthlyBudget) (uint, error) {
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}
