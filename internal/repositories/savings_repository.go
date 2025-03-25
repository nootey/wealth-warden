package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type SavingsRepository struct {
	Db *gorm.DB
}

func NewSavingsRepository(db *gorm.DB) *SavingsRepository {
	return &SavingsRepository{Db: db}
}

func (r *SavingsRepository) FindAllSavingCategories(user *models.User) ([]models.SavingsCategory, error) {
	var records []models.SavingsCategory
	result := r.Db.Where("organization_id = ?", *user.PrimaryOrganizationID).Find(&records)
	return records, result.Error
}

func (r *SavingsRepository) FindSavings(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.SavingsAllocation, error) {
	var records []models.SavingsAllocation
	orderBy := sortField + " " + sortOrder

	query := r.Db.
		Preload("SavingsCategory").
		Where("savings_allocations.organization_id = ? AND YEAR(savings_allocations.savings_date) = ?", *user.PrimaryOrganizationID, year)

	if utils.NeedsJoin(filters, "savings_category") {
		query = query.Joins("JOIN savings_categories ON savings_categories.id = savings_allocations.savings_category_id")
	}

	query = utils.ApplyFilters(query, filters)

	err := query.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *SavingsRepository) CountSavings(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.Db.Model(&models.SavingsAllocation{}).
		Where("savings_allocations.organization_id = ? AND YEAR(savings_allocations.savings_date) = ?", *user.PrimaryOrganizationID, year)

	if utils.NeedsJoin(filters, "savings_category") {
		query = query.Joins("JOIN savings_categories ON savings_categories.id = savings_allocations.savings_category_id")
	}

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}
