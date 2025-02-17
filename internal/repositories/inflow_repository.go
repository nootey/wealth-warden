package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type InflowRepository struct {
	db *gorm.DB
}

func NewInflowRepository(db *gorm.DB) *InflowRepository {
	return &InflowRepository{db: db}
}

func (r *InflowRepository) CountInflowsByCategory(userID, categoryID uint, count *int64) error {
	return r.db.Model(&models.Inflow{}).
		Where("inflow_category_id = ?", categoryID).
		Where("user_id = ?", userID).
		Count(count).Error
}

func (r *InflowRepository) CountInflows(userID uint) (int64, error) {
	var totalRecords int64
	err := r.db.Model(&models.Inflow{}).Where("user_id = ?", userID).Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *InflowRepository) FindInflows(userID uint, offset, limit int, sortField, sortOrder string) ([]models.Inflow, error) {
	var inflows []models.Inflow
	orderBy := sortField + " " + sortOrder

	err := r.db.
		Preload("InflowCategory").
		Where("user_id = ?", userID).
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&inflows).Error

	if err != nil {
		return nil, err
	}

	return inflows, nil
}

func (r *InflowRepository) FindAllInflowsGroupedByMonth(userID uint) ([]models.InflowSummary, error) {
	var results []models.InflowSummary

	err := r.db.Raw(`
        SELECT * FROM (
            -- Regular category rows
            SELECT
                MONTH(i.inflow_date) AS month,
                ic.id AS inflow_category_id,
                ic.name AS inflow_category_name,
                SUM(i.amount) AS total_amount
            FROM inflows i
            JOIN inflow_categories ic ON i.inflow_category_id = ic.id
            WHERE i.deleted_at IS NULL
            AND i.user_id = ?
            AND YEAR(i.inflow_date) = YEAR(CURDATE())
            GROUP BY ic.id, ic.name, month

            UNION ALL

            -- "Total" row for each month (sums all categories)
            SELECT
                MONTH(i.inflow_date) AS month,
                0 AS inflow_category_id,
                'Total' AS inflow_category_name,
                SUM(i.amount) AS total_amount
            FROM inflows i
            WHERE i.deleted_at IS NULL
            AND i.user_id = ?
            AND YEAR(i.inflow_date) = YEAR(CURDATE())
            GROUP BY MONTH(i.inflow_date)
        ) AS combined
        ORDER BY 
            (CASE WHEN inflow_category_name = 'Total' THEN 1 ELSE 0 END),
            inflow_category_name, 
            month`, userID, userID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *InflowRepository) GetAllInflowCategories(userID uint) ([]models.InflowCategory, error) {
	var inflowCategories []models.InflowCategory
	result := r.db.Where("user_id = ?", userID).Find(&inflowCategories)
	return inflowCategories, result.Error
}

func (r *InflowRepository) InsertInflow(userID uint, inflow *models.Inflow) error {
	inflow.UserID = userID
	if err := r.db.Create(&inflow).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) InsertInflowCategory(userID uint, inflowCategory *models.InflowCategory) error {
	inflowCategory.UserID = userID
	if err := r.db.Create(&inflowCategory).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) DropInflow(userID uint, id uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&models.Inflow{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *InflowRepository) DropInflowCategory(userID uint, id uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&models.InflowCategory{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
