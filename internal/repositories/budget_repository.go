package repositories

import (
	"fmt"
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
	result := r.Db.Preload("DynamicCategory.Mappings").
		Preload("Allocations").
		Where("organization_id = ? AND year = ? AND month = ?", *user.PrimaryOrganizationID, year, month).
		Find(&record)
	return record, result.Error
}

func (r *BudgetRepository) FindBudgetByID(id uint, user *models.User, withMappings bool) (*models.MonthlyBudget, error) {
	var record models.MonthlyBudget

	query := r.Db.Preload("DynamicCategory.Mappings").
		Where("id = ? AND organization_id = ?", id, *user.PrimaryOrganizationID)

	if withMappings {
		query = query.Preload("Allocations")
	}

	result := query.Find(&record)

	if result.Error != nil {
		return nil, result.Error
	}

	return &record, nil
}

func (r *BudgetRepository) InsertMonthlyBudget(tx *gorm.DB, user *models.User, record *models.MonthlyBudget) (uint, error) {
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *BudgetRepository) InsertMonthlyBudgetAllocation(tx *gorm.DB, record *models.MonthlyBudgetAllocation) error {
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *BudgetRepository) UpdateMonthlyBudget(tx *gorm.DB, user *models.User, record *models.MonthlyBudget) error {
	record.UserID = user.ID

	if err := tx.Model(&models.MonthlyBudget{}).
		Where("id = ? AND organization_id = ?", record.ID, *user.PrimaryOrganizationID).
		Updates(map[string]interface{}{
			"total_inflow":     record.TotalInflow,
			"total_outflow":    record.TotalOutflow,
			"budget_inflow":    record.BudgetInflow,
			"budget_outflow":   record.BudgetOutflow,
			"effective_budget": record.EffectiveBudget,
			"budget_snapshot":  record.BudgetSnapshot,
		}).Error; err != nil {
		return err
	}

	return nil
}

func (r *BudgetRepository) RecalculatePercentileAllocations(tx *gorm.DB, budget *models.MonthlyBudget) error {
	for i := range budget.Allocations {
		allocation := &budget.Allocations[i]
		fmt.Println(allocation)
		if allocation.Method == "percentage" {
			fmt.Println(allocation.AllocatedValue)
			allocation.AllocatedValue = budget.BudgetSnapshot * (allocation.Allocation / 100.0)
			fmt.Println(allocation.AllocatedValue)
			if err := tx.Model(&models.MonthlyBudgetAllocation{}).
				Where("id = ?", allocation.ID).
				Update("allocated_value", allocation.AllocatedValue).Error; err != nil {
				return fmt.Errorf("failed to update allocation ID %d: %w", allocation.ID, err)
			}
		}
	}

	return nil
}
