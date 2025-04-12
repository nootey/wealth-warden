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

func (r *SavingsRepository) CountSavingsTransactions(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.Db.Model(&models.SavingsTransaction{}).
		Where("savings_transactions.organization_id = ? AND YEAR(savings_transactions.transaction_date) = ?", *user.PrimaryOrganizationID, year)

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *SavingsRepository) FindAllSavingCategories(user *models.User) ([]models.SavingsCategory, error) {
	var records []models.SavingsCategory
	result := r.Db.Where("organization_id = ?", *user.PrimaryOrganizationID).Find(&records)
	return records, result.Error
}

func (r *SavingsRepository) FindTotalForTransactionForCategory(user *models.User, transaction string, categoryID uint, year int) (float64, error) {
	var total float64

	err := r.Db.
		Table("savings_transactions").
		Select("COALESCE(SUM(adjusted_amount), 0)").
		Where("transaction_type =?", transaction).
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Where("savings_category_id = ?", categoryID).
		Where("YEAR(transaction_date) = ?", year).
		Scan(&total).Error

	return total, err
}

func (r *SavingsRepository) FindSavingsTransactions(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.SavingsTransaction, error) {
	var records []models.SavingsTransaction

	query := r.Db.
		Preload("SavingsCategory").
		Where("savings_transactions.organization_id = ? AND YEAR(savings_transactions.transaction_date) = ?", *user.PrimaryOrganizationID, year)

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, sortField, sortOrder)

	for _, join := range joins {
		query = query.Joins(join)
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
		Model(&models.SavingsTransaction{}).
		Select("MONTH(transaction_date) AS month, 0 AS category_id, 'Total' AS category_name, SUM(adjusted_amount) AS total_amount, 'fixed' AS category_type").
		Where("organization_id = ? AND YEAR(transaction_date) = ?", *user.PrimaryOrganizationID, year).
		Group("MONTH(transaction_date)").
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
		Table("savings_transactions s").
		Select(`
			MONTH(s.transaction_date) AS month,
			sc.id AS category_id,
			sc.name AS category_name,
			sc.savings_type AS category_type,
			COALESCE(SUM(CASE WHEN s.transaction_type = 'allocation' THEN s.adjusted_amount ELSE 0 END), 0) AS total_allocated,
			COALESCE(SUM(CASE WHEN s.transaction_type = 'deduction' THEN s.adjusted_amount ELSE 0 END), 0) AS total_deducted,
			GREATEST(
				COALESCE(SUM(CASE WHEN s.transaction_type = 'allocation' THEN s.adjusted_amount ELSE 0 END), 0) -
				COALESCE(SUM(CASE WHEN s.transaction_type = 'deduction' THEN s.adjusted_amount ELSE 0 END), 0), 
			0) AS goal_progress,
			sc.goal_target AS goal_target
		`).
		Joins("JOIN savings_categories sc ON s.savings_category_id = sc.id").
		Where("s.organization_id = ?", orgID).
		Where("YEAR(s.transaction_date) = ?", year).
		Group("MONTH(s.transaction_date), sc.id, sc.name, sc.savings_type, sc.goal_target").
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

func (r *SavingsRepository) InsertSavingsAllocation(tx *gorm.DB, user *models.User, record *models.SavingsTransaction) error {

	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *SavingsRepository) InsertSavingsDeduction(tx *gorm.DB, user *models.User, record *models.SavingsTransaction) error {

	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *SavingsRepository) InsertSavingsCategory(tx *gorm.DB, user *models.User, record *models.SavingsCategory) (uint, error) {

	var existing models.SavingsCategory
	if err := tx.Where("organization_id = ? AND name = ?", *user.PrimaryOrganizationID, record.Name).First(&existing).Error; err == nil {
		return 0, errors.New("category with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Insert new category
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *SavingsRepository) UpdateCategoryGoalProgress(tx *gorm.DB, user *models.User, categoryID uint, amount float64, modifier int) error {

	category, err := r.GetSavingsCategoryByID(user, categoryID)
	if err != nil {
		return err
	}

	category.UserID = user.ID
	category.GoalProgress += (amount) * float64(modifier)

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

func (r *SavingsRepository) DropSavingsCategory(tx *gorm.DB, user *models.User, recordID uint) error {
	return tx.Where("id = ? AND organization_id = ?", recordID, *user.PrimaryOrganizationID).Delete(&models.SavingsCategory{}).Error
}
