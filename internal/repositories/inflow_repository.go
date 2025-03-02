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

func (r *InflowRepository) CountDynamicCategoryByID(categoryID uint, count *int64) error {
	return r.Db.Model(&models.DynamicCategoryMapping{}).
		Where("related_id = ?", categoryID).
		Where("related_type = ?", "dynamic").
		Count(count).Error
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

func (r *InflowRepository) GetDynamicCategoryByID(userID, ID uint) (*models.DynamicCategory, error) {
	var record models.DynamicCategory
	err := r.Db.Where("id = ? AND user_id = ?", ID, userID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *InflowRepository) FindAllInflowsGroupedByMonth(userID uint) ([]models.InflowSummary, error) {
	var results []models.InflowSummary

	// Define the recursive CTE for dynamic categories
	dynamicFlatCTE := `
	WITH RECURSIVE dynamic_flat AS (
    SELECT
        dcm.dynamic_category_id AS root_dynamic_category_id,
        dcm.related_type,
        dcm.related_id,
        CASE 
          WHEN dcm.related_type = 'inflow' THEN 1
          WHEN dcm.related_type = 'outflow' THEN -1
          WHEN dcm.related_type = 'dynamic' THEN 1
        END AS sign,
        0 AS level
    FROM dynamic_category_mappings dcm
    UNION ALL
    SELECT
        df.root_dynamic_category_id,
        dcm.related_type,
        dcm.related_id,
        df.sign * (CASE 
                    WHEN dcm.related_type = 'inflow' THEN 1
                    WHEN dcm.related_type = 'outflow' THEN -1
                    WHEN dcm.related_type = 'dynamic' THEN 1
                   END) AS sign,
        df.level + 1 AS level
    FROM dynamic_flat df
    JOIN dynamic_category_mappings dcm 
      ON df.related_type = 'dynamic' 
     AND dcm.dynamic_category_id = df.related_id
	)
	`

	// Query for static inflow categories
	staticInflowsQuery := `
    SELECT
        MONTH(i.inflow_date) AS month,
        ic.id AS category_id,
        ic.name AS category_name,
        SUM(i.amount) AS total_amount,
        'static' AS category_type
    FROM inflows i
    JOIN inflow_categories ic ON i.inflow_category_id = ic.id
    WHERE i.deleted_at IS NULL
      AND i.user_id = ?
      AND YEAR(i.inflow_date) = YEAR(CURDATE())
    GROUP BY ic.id, ic.name, MONTH(i.inflow_date)
	`

	// Query for dynamic category calculations
	dynamicCategoriesQuery := `
    SELECT
        m.month,
        dc.id AS category_id,
        dc.name AS category_name,
        SUM( df.sign * COALESCE(inf.total_amount, outf.total_amount, 0) ) AS total_amount,
        'dynamic' AS category_type
    FROM dynamic_categories dc
    CROSS JOIN (
      SELECT 1 AS month UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 
      UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 
      UNION SELECT 9 UNION SELECT 10 UNION SELECT 11 UNION SELECT 12
    ) m
    LEFT JOIN dynamic_flat df ON df.root_dynamic_category_id = dc.id
    LEFT JOIN (
        SELECT 
            MONTH(inflow_date) AS month, 
            inflow_category_id, 
            SUM(amount) AS total_amount
        FROM inflows
        JOIN inflow_categories ic ON inflows.inflow_category_id = ic.id
        WHERE inflows.deleted_at IS NULL 
          AND inflows.user_id = ?
          AND YEAR(inflow_date) = YEAR(CURDATE())
        GROUP BY MONTH(inflow_date), inflow_category_id
    ) inf ON df.related_type = 'inflow' 
       AND inf.inflow_category_id = df.related_id 
       AND inf.month = m.month
    LEFT JOIN (
        SELECT 
            MONTH(outflow_date) AS month, 
            outflow_category_id, 
            SUM(amount) AS total_amount
        FROM outflows
        WHERE deleted_at IS NULL 
          AND user_id = ?
          AND YEAR(outflow_date) = YEAR(CURDATE())
        GROUP BY MONTH(outflow_date), outflow_category_id
    ) outf ON df.related_type = 'outflow' 
       AND outf.outflow_category_id = df.related_id 
       AND outf.month = m.month
    WHERE dc.user_id = ?
      AND df.related_type IN ('inflow', 'outflow')
    GROUP BY m.month, dc.id, dc.name
	`

	// Query for the "Total" row for static inflows only
	totalRowQuery := `
    SELECT
        MONTH(i.inflow_date) AS month,
        0 AS category_id,
        'Total' AS category_name,
        SUM(i.amount) AS total_amount,
        'static' AS category_type
    FROM inflows i
    JOIN inflow_categories ic ON i.inflow_category_id = ic.id
    WHERE i.deleted_at IS NULL
      AND i.user_id = ?
      AND YEAR(i.inflow_date) = YEAR(CURDATE())
    GROUP BY MONTH(i.inflow_date)
	`

	// Combine all the fragments into the final query
	finalQuery := dynamicFlatCTE + `
	SELECT * FROM (
    ` + staticInflowsQuery + `
    UNION ALL
    ` + dynamicCategoriesQuery + `
    UNION ALL
    ` + totalRowQuery + `
	) AS combined
	ORDER BY 
    (CASE WHEN category_name = 'Total' THEN 1 ELSE 0 END),
    category_name, 
    month;
	`

	// Execute the final query with the required parameters
	// TODO: this query is quite disgusting, brainstorm some options to improve it
	err := r.Db.Raw(finalQuery, userID, userID, userID, userID, userID).Scan(&results).Error
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
	result := r.Db.Preload("Mappings").Where("user_id = ?", userID).Find(&records)
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

func (r *InflowRepository) InsertDynamicCategoryMapping(tx *gorm.DB, mapping models.DynamicCategoryMapping) error {

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

func (r *InflowRepository) DropDynamicCategory(tx *gorm.DB, userID uint, recordID uint) error {
	// Delete related mappings first
	if err := tx.Where("dynamic_category_id = ?", recordID).Delete(&models.DynamicCategoryMapping{}).Error; err != nil {
		return err
	}

	// Delete the dynamic category
	result := tx.Where("user_id = ?", userID).Delete(&models.DynamicCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
