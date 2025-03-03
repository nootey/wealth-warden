package repositories

import (
	"errors"
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type OutflowRepository struct {
	Db *gorm.DB
}

func NewOutflowRepository(db *gorm.DB) *OutflowRepository {
	return &OutflowRepository{Db: db}
}

func (r *OutflowRepository) CountOutflowsByCategory(userID, categoryID uint, count *int64) error {
	return r.Db.Model(&models.Outflow{}).
		Where("outflow_category_id = ?", categoryID).
		Where("user_id = ?", userID).
		Count(count).Error
}

func (r *OutflowRepository) CountOutflows(userID uint, year int) (int64, error) {
	var totalRecords int64
	err := r.Db.Model(&models.Outflow{}).
		Where("user_id = ? AND YEAR(outflow_date) = ?", userID, year).
		Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *OutflowRepository) FindOutflows(userID uint, year, offset, limit int, sortField, sortOrder string) ([]models.Outflow, error) {
	var outflows []models.Outflow
	orderBy := sortField + " " + sortOrder

	err := r.Db.
		Preload("OutflowCategory").
		Where("user_id = ? AND YEAR(outflow_date) = ?", userID, year).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&outflows).Error

	if err != nil {
		return nil, err
	}

	return outflows, nil
}

func (r *OutflowRepository) GetOutflowByID(userID, outflowID uint) (*models.Outflow, error) {
	var outflow models.Outflow
	err := r.Db.Preload("OutflowCategory").Where("id = ? AND user_id = ?", outflowID, userID).First(&outflow).Error
	if err != nil {
		return nil, err
	}
	return &outflow, nil
}

func (r *OutflowRepository) GetOutflowCategoryByID(userID, outflowCategoryID uint) (*models.OutflowCategory, error) {
	var outflowCategory models.OutflowCategory
	err := r.Db.Where("id = ? AND user_id = ?", outflowCategoryID, userID).First(&outflowCategory).Error
	if err != nil {
		return nil, err
	}
	return &outflowCategory, nil
}

func (r *OutflowRepository) FindAllOutflowsGroupedByMonth(userID uint, year int) ([]models.OutflowSummary, error) {
	var results []models.OutflowSummary

	err := r.Db.Raw(`
        SELECT * FROM (
            -- Regular category rows
            SELECT
                MONTH(o.outflow_date) AS month,
                oc.id AS category_id,
                oc.name AS category_name,
                SUM(o.amount) AS total_amount,
                oc.spending_limit AS spending_limit,
                oc.outflow_type AS category_type
            FROM outflows o
            JOIN outflow_categories oc ON o.outflow_category_id = oc.id
            WHERE o.deleted_at IS NULL
              AND o.user_id = ?
              AND YEAR(o.outflow_date) = ?
            GROUP BY oc.id, oc.name, month, oc.spending_limit, category_type

            UNION ALL

            -- "Total" row for each month (sums all categories)
            SELECT
                MONTH(o.outflow_date) AS month,
                0 AS category_id,
                'Total' AS category_name,
                SUM(o.amount) AS total_amount,
                NULL AS spending_limit,
                NULL AS category_type
            FROM outflows o
            WHERE o.deleted_at IS NULL
              AND o.user_id = ?
              AND YEAR(o.outflow_date) = ?
            GROUP BY MONTH(o.outflow_date)
        ) AS combined
        ORDER BY 
            (CASE WHEN category_name = 'Total' THEN 0 ELSE 1 END),
            category_type,
            category_name,
            month
    `, userID, year, userID, year).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *OutflowRepository) GetAllOutflowCategories(userID uint) ([]models.OutflowCategory, error) {
	var outflowCategories []models.OutflowCategory
	result := r.Db.Where("user_id = ?", userID).Find(&outflowCategories)
	return outflowCategories, result.Error
}

func (r *OutflowRepository) InsertOutflow(tx *gorm.DB, userID uint, record *models.Outflow) (uint, error) {
	record.UserID = userID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *OutflowRepository) UpdateOutflow(tx *gorm.DB, userID uint, record *models.Outflow) (uint, error) {
	if err := tx.Model(&models.Outflow{}).Where("id = ? AND user_id = ?", record.ID, userID).Updates(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *OutflowRepository) InsertOutflowCategory(tx *gorm.DB, userID uint, outflowCategory *models.OutflowCategory) error {

	var existing models.OutflowCategory
	if err := tx.Where("user_id = ? AND name = ?", userID, outflowCategory.Name).First(&existing).Error; err == nil {
		return errors.New("category with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Insert new category
	outflowCategory.UserID = userID
	if err := tx.Create(&outflowCategory).Error; err != nil {
		return err
	}
	return nil
}

func (r *OutflowRepository) UpdateOutflowCategory(tx *gorm.DB, userID uint, record *models.OutflowCategory) error {

	if err := tx.Model(&models.OutflowCategory{}).Where("user_id = ? AND id = ?", userID, record.ID).Updates(record).Error; err != nil {
		return err
	}
	return nil
}

func (r *OutflowRepository) DropOutflow(tx *gorm.DB, userID uint, recordID uint) error {
	return tx.Where("id = ? AND user_id = ?", recordID, userID).Delete(&models.Outflow{}).Error
}

func (r *OutflowRepository) DropOutflowCategory(tx *gorm.DB, userID uint, recordID uint) error {
	result := tx.Where("user_id = ?", userID).Delete(&models.OutflowCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
