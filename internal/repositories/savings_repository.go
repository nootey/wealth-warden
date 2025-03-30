package repositories

import (
	"errors"
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
		Where("savings_allocations.organization_id = ? AND YEAR(savings_allocations.allocation_date) = ?", *user.PrimaryOrganizationID, year)

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

func (r *SavingsRepository) FindTotalForGroupedSavings(user *models.User, year int) ([]models.SavingsSummary, error) {
	var total []models.SavingsSummary
	err := r.Db.
		Model(&models.SavingsAllocation{}).
		Select("MONTH(allocation_date) AS month, 0 AS category_id, 'Total' AS category_name, SUM(adjusted_amount) AS total_amount, 'fixed' AS category_type").
		Where("organization_id = ? AND YEAR(allocation_date) = ?", *user.PrimaryOrganizationID, year).
		Group("MONTH(allocation_date)").
		Scan(&total).Error
	if err != nil {
		return nil, err
	}

	return total, nil
}

func (r *SavingsRepository) FetchGroupedSavingsByCategoryAndMonth(user *models.User, year int) ([]models.SavingsSummary, error) {
	var results []models.SavingsSummary
	orgID := *user.PrimaryOrganizationID

	err := r.Db.
		Table("savings_allocations s").
		Select(`
			MONTH(s.allocation_date) as month,
			sc.id as category_id,
			sc.name as category_name,
			sc.savings_type as category_type,
			SUM(s.adjusted_amount) as total_amount,
			sc.goal_progress as goal_progress,
			sc.goal_target as goal_target
		`).
		Joins("JOIN savings_categories sc ON s.savings_category_id = sc.id").
		Where("s.organization_id = ?", orgID).
		Where("YEAR(s.allocation_date) = ?", year).
		Group("MONTH(s.allocation_date), sc.id, sc.name, sc.savings_type, sc.goal_progress, sc.goal_target").
		Order("month, category_type, category_name").
		Scan(&results).Error

	return results, err
}

func (r *SavingsRepository) GetSavingsCategoryByID(user *models.User, categoryID uint) (*models.SavingsCategory, error) {
	var record models.SavingsCategory
	err := r.Db.Where("id = ? AND organization_id = ?", categoryID, *user.PrimaryOrganizationID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *SavingsRepository) CountSavings(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.Db.Model(&models.SavingsAllocation{}).
		Where("savings_allocations.organization_id = ? AND YEAR(savings_allocations.allocation_date) = ?", *user.PrimaryOrganizationID, year)

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

func (r *SavingsRepository) InsertSavingsAllocation(tx *gorm.DB, user *models.User, record *models.SavingsAllocation) error {

	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *SavingsRepository) InsertSavingsCategory(tx *gorm.DB, user *models.User, record *models.SavingsCategory) error {

	var existing models.SavingsCategory
	if err := tx.Where("organization_id = ? AND name = ?", *user.PrimaryOrganizationID, record.Name).First(&existing).Error; err == nil {
		return errors.New("category with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Insert new category
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *SavingsRepository) UpdateCategoryGoalProgress(tx *gorm.DB, user *models.User, record *models.SavingsAllocation, modifier int) error {

	category, err := r.GetSavingsCategoryByID(user, record.SavingsCategoryID)
	if err != nil {
		return err
	}

	category.UserID = user.ID
	category.GoalProgress += (*record.AdjustedAmount) * float64(modifier)

	if err := tx.Model(&models.SavingsCategory{}).
		Where("organization_id = ? AND id = ?", *user.PrimaryOrganizationID, category.ID).
		Update("goal_progress", category.GoalProgress).Error; err != nil {
		return err
	}

	return nil
}

func (r *SavingsRepository) UpdateSavingsCategory(tx *gorm.DB, user *models.User, record *models.SavingsCategory) error {
	record.UserID = user.ID
	if err := tx.Model(&models.SavingsCategory{}).Where("organization_id = ? AND id = ?", *user.PrimaryOrganizationID, record.ID).Updates(record).Error; err != nil {
		return err
	}
	return nil
}
