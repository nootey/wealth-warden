package repositories

import (
	"errors"
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type InflowRepository struct {
	Db *gorm.DB
}

func NewInflowRepository(db *gorm.DB) *InflowRepository {
	return &InflowRepository{Db: db}
}

func (r *InflowRepository) CountInflowsByCategory(userID, categoryID uint, count *int64) error {
	return r.Db.Model(&models.Inflow{}).
		Where("inflow_category_id = ?", categoryID).
		Where("user_id = ?", userID).
		Count(count).Error
}

func (r *InflowRepository) CountInflows(userID uint) (int64, error) {
	var totalRecords int64
	err := r.Db.Model(&models.Inflow{}).Where("user_id = ?", userID).Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *InflowRepository) FindInflows(userID uint, offset, limit int, sortField, sortOrder string) ([]models.Inflow, error) {
	var inflows []models.Inflow
	orderBy := sortField + " " + sortOrder

	err := r.Db.
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

func (r *InflowRepository) GetInflowByID(userID, inflowID uint) (*models.Inflow, error) {
	var inflow models.Inflow
	err := r.Db.Preload("InflowCategory").Where("id = ? AND user_id = ?", inflowID, userID).First(&inflow).Error
	if err != nil {
		return nil, err
	}
	return &inflow, nil
}

func (r *InflowRepository) GetInflowCategoryByID(userID, inflowCategoryID uint) (*models.InflowCategory, error) {
	var inflowCategory models.InflowCategory
	err := r.Db.Where("id = ? AND user_id = ?", inflowCategoryID, userID).First(&inflowCategory).Error
	if err != nil {
		return nil, err
	}
	return &inflowCategory, nil
}

func (r *InflowRepository) FindAllInflowsGroupedByMonth(userID uint) ([]models.InflowSummary, error) {
	var results []models.InflowSummary

	err := r.Db.Raw(`
        SELECT * FROM (
            -- Regular category rows
            SELECT
                MONTH(i.inflow_date) AS month,
                ic.id AS category_id,
                ic.name AS category_name,
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
                0 AS category_id,
                'Total' AS category_name,
                SUM(i.amount) AS total_amount
            FROM inflows i
            WHERE i.deleted_at IS NULL
            AND i.user_id = ?
            AND YEAR(i.inflow_date) = YEAR(CURDATE())
            GROUP BY MONTH(i.inflow_date)
        ) AS combined
        ORDER BY 
            (CASE WHEN category_name = 'Total' THEN 1 ELSE 0 END),
            category_name, 
            month`, userID, userID).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *InflowRepository) GetAllInflowCategories(userID uint) ([]models.InflowCategory, error) {
	var inflowCategories []models.InflowCategory
	result := r.Db.Where("user_id = ?", userID).Find(&inflowCategories)
	return inflowCategories, result.Error
}

func (r *InflowRepository) GetAllDynamicCategories(userID uint) ([]models.DynamicCategory, error) {
	var records []models.DynamicCategory
	result := r.Db.Where("user_id = ?", userID).Find(&records)
	return records, result.Error
}

func (r *InflowRepository) InsertInflow(tx *gorm.DB, userID uint, record *models.Inflow) (uint, error) {
	record.UserID = userID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *InflowRepository) UpdateInflow(tx *gorm.DB, userID uint, record *models.Inflow) (uint, error) {
	record.UserID = userID
	if err := tx.Model(&models.Inflow{}).Where("id = ? AND user_id = ?", record.ID, userID).Updates(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *InflowRepository) InsertInflowCategory(tx *gorm.DB, userID uint, record *models.InflowCategory) error {

	var existing models.InflowCategory
	if err := tx.Where("user_id = ? AND name = ?", userID, record.Name).First(&existing).Error; err == nil {
		return errors.New("category with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Insert new category
	record.UserID = userID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) InsertDynamicCategory(tx *gorm.DB, userID uint, record *models.DynamicCategory) (uint, error) {

	var existing models.DynamicCategory
	if err := tx.Where("user_id = ? AND name = ?", userID, record.Name).First(&existing).Error; err == nil {
		return 0, errors.New("category with this name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// Insert new category
	record.UserID = userID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *InflowRepository) InsertDynamicCategoryMapping(tx *gorm.DB, categoryID uint, mapping models.DynamicCategoryMapping) error {

	mapping.DynamicCategoryID = categoryID
	if err := tx.Create(&mapping).Error; err != nil {
		return err
	}

	return nil
}

func (r *InflowRepository) UpdateInflowCategory(tx *gorm.DB, userID uint, record *models.InflowCategory) error {

	if err := tx.Model(&models.InflowCategory{}).Where("user_id = ? AND id = ?", userID, record.ID).Updates(record).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) DropInflow(tx *gorm.DB, userID uint, recordID uint) error {
	return tx.Where("id = ? AND user_id = ?", recordID, userID).Delete(&models.Inflow{}).Error
}

func (r *InflowRepository) DropInflowCategory(tx *gorm.DB, userID uint, recordID uint) error {
	result := tx.Where("user_id = ?", userID).Delete(&models.InflowCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
