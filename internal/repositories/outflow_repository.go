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

func (r *OutflowRepository) CountOutflows(userID uint) (int64, error) {
	var totalRecords int64
	err := r.Db.Model(&models.Outflow{}).Where("user_id = ?", userID).Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *OutflowRepository) FindOutflows(userID uint, offset, limit int, sortField, sortOrder string) ([]models.Outflow, error) {
	var outflows []models.Outflow
	orderBy := sortField + " " + sortOrder

	err := r.Db.
		Preload("OutflowCategory").
		Where("user_id = ?", userID).
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

func (r *OutflowRepository) FindAllOutflowsGroupedByMonth(userID uint) ([]models.OutflowSummary, error) {
	var results []models.OutflowSummary

	err := r.Db.Raw(`
        SELECT * FROM (
            -- Regular category rows
            SELECT
                MONTH(i.outflow_date) AS month,
                ic.id AS outflow_category_id,
                ic.name AS outflow_category_name,
                SUM(i.amount) AS total_amount
            FROM outflows i
            JOIN outflow_categories ic ON i.outflow_category_id = ic.id
            WHERE i.deleted_at IS NULL
            AND i.user_id = ?
            AND YEAR(i.outflow_date) = YEAR(CURDATE())
            GROUP BY ic.id, ic.name, month

            UNION ALL

            -- "Total" row for each month (sums all categories)
            SELECT
                MONTH(i.outflow_date) AS month,
                0 AS outflow_category_id,
                'Total' AS outflow_category_name,
                SUM(i.amount) AS total_amount
            FROM outflows i
            WHERE i.deleted_at IS NULL
            AND i.user_id = ?
            AND YEAR(i.outflow_date) = YEAR(CURDATE())
            GROUP BY MONTH(i.outflow_date)
        ) AS combined
        ORDER BY 
            (CASE WHEN outflow_category_name = 'Total' THEN 1 ELSE 0 END),
            outflow_category_name, 
            month`, userID, userID).Scan(&results).Error

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

func (r *OutflowRepository) InsertOutflow(tx *gorm.DB, userID uint, outflow *models.Outflow) (uint, error) {
	outflow.UserID = userID
	if err := tx.Create(&outflow).Error; err != nil {
		return 0, err
	}
	return outflow.ID, nil
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

func (r *OutflowRepository) DropOutflow(tx *gorm.DB, userID uint, outflowID uint) error {
	return tx.Where("id = ? AND user_id = ?", outflowID, userID).Delete(&models.Outflow{}).Error
}

func (r *OutflowRepository) DropOutflowCategory(tx *gorm.DB, userID uint, id uint) error {
	result := tx.Where("user_id = ?", userID).Delete(&models.OutflowCategory{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
