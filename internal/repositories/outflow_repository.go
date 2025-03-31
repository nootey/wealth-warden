package repositories

import (
	"errors"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type OutflowRepository struct {
	Db *gorm.DB
}

func NewOutflowRepository(db *gorm.DB) *OutflowRepository {
	return &OutflowRepository{Db: db}
}

func (r *OutflowRepository) CountOutflowsByCategory(user *models.User, categoryID uint, count *int64) error {
	return r.Db.Model(&models.Outflow{}).
		Where("outflow_category_id = ?", categoryID).
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Count(count).Error
}

func (r *OutflowRepository) CountOutflows(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.Db.Model(&models.Outflow{}).
		Where("outflows.organization_id = ? AND YEAR(outflows.outflow_date) = ?", *user.PrimaryOrganizationID, year)

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

func (r *OutflowRepository) SumOutflowsByCategory(user *models.User, categoryID uint, year, month int) (float64, error) {
	var total float64

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0) // Moves to the first day of the next month

	err := r.Db.Model(&models.Outflow{}).
		Where("outflow_category_id = ? AND outflow_date >= ? AND outflow_date < ?", categoryID, startDate, endDate).
		Where("organization_id = ?", &user.PrimaryOrganizationID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *OutflowRepository) FindOutflows(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Outflow, error) {
	var records []models.Outflow

	query := r.Db.
		Preload("OutflowCategory").
		Where("outflows.organization_id = ? AND YEAR(outflows.outflow_date) = ?", *user.PrimaryOrganizationID, year)

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

func (r *OutflowRepository) GetOutflowByID(user *models.User, outflowID uint) (*models.Outflow, error) {
	var outflow models.Outflow
	err := r.Db.Preload("OutflowCategory").Where("id = ? AND organization_id = ?", outflowID, *user.PrimaryOrganizationID).First(&outflow).Error
	if err != nil {
		return nil, err
	}
	return &outflow, nil
}

func (r *OutflowRepository) GetOutflowCategoryByID(user *models.User, categoryID uint) (*models.OutflowCategory, error) {
	var record models.OutflowCategory
	err := r.Db.Where("id = ? AND organization_id = ?", categoryID, *user.PrimaryOrganizationID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *OutflowRepository) FindTotalForGroupedOutflows(user *models.User, year int) ([]models.OutflowSummary, error) {
	var total []models.OutflowSummary
	err := r.Db.
		Model(&models.Outflow{}).
		Select("MONTH(outflow_date) AS month, 0 AS category_id, 'Total' AS category_name, SUM(amount) AS total_amount, 'fixed' AS category_type").
		Where("organization_id = ? AND YEAR(outflow_date) = ? AND deleted_at IS NULL", *user.PrimaryOrganizationID, year).
		Group("MONTH(outflow_date)").
		Scan(&total).Error
	if err != nil {
		return nil, err
	}

	return total, nil
}

func (r *OutflowRepository) FetchGroupedOutflowsByCategoryAndMonth(user *models.User, year int) ([]models.OutflowSummary, error) {
	var results []models.OutflowSummary
	orgID := *user.PrimaryOrganizationID

	err := r.Db.
		Table("outflows o").
		Select(`
			MONTH(o.outflow_date) as month,
			oc.id as category_id,
			oc.name as category_name,
			oc.outflow_type as category_type,
			SUM(o.amount) as total_amount,
			oc.spending_limit
		`).
		Joins("JOIN outflow_categories oc ON o.outflow_category_id = oc.id").
		Where("o.deleted_at IS NULL").
		Where("o.organization_id = ?", orgID).
		Where("YEAR(o.outflow_date) = ?", year).
		Group("MONTH(o.outflow_date), oc.id, oc.name, oc.outflow_type, oc.spending_limit").
		Order("month, category_type, category_name").
		Scan(&results).Error

	return results, err
}

func (r *OutflowRepository) GetAllOutflowCategories(user *models.User) ([]models.OutflowCategory, error) {
	var outflowCategories []models.OutflowCategory
	result := r.Db.Where("organization_id = ?", *user.PrimaryOrganizationID).Find(&outflowCategories)
	return outflowCategories, result.Error
}

func (r *OutflowRepository) InsertOutflow(tx *gorm.DB, user *models.User, record *models.Outflow) (uint, error) {
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *OutflowRepository) UpdateOutflow(tx *gorm.DB, user *models.User, record *models.Outflow) (uint, error) {
	record.UserID = user.ID
	if err := tx.Model(&models.Outflow{}).
		Where("id = ? AND organization_id = ?", record.ID, *user.PrimaryOrganizationID).
		Updates(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *OutflowRepository) InsertOutflowCategory(tx *gorm.DB, user *models.User, record *models.OutflowCategory) error {

	var existing models.OutflowCategory
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

func (r *OutflowRepository) UpdateOutflowCategory(tx *gorm.DB, user *models.User, record *models.OutflowCategory) error {
	record.UserID = user.ID
	if err := tx.Model(&models.OutflowCategory{}).Where("organization_id = ? AND id = ?", *user.PrimaryOrganizationID, record.ID).Updates(record).Error; err != nil {
		return err
	}
	return nil
}

func (r *OutflowRepository) DropOutflow(tx *gorm.DB, user *models.User, recordID uint) error {
	return tx.Where("id = ? AND organization_id = ?", recordID, *user.PrimaryOrganizationID).Delete(&models.Outflow{}).Error
}

func (r *OutflowRepository) DropOutflowCategory(tx *gorm.DB, user *models.User, recordID uint) error {
	result := tx.Where("organization_id = ?", *user.PrimaryOrganizationID).Delete(&models.OutflowCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
