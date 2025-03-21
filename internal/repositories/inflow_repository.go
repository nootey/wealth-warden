package repositories

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type InflowRepository struct {
	Db *gorm.DB
}

func NewInflowRepository(db *gorm.DB) *InflowRepository {
	return &InflowRepository{Db: db}
}

func (r *InflowRepository) CountInflowsByCategory(user *models.User, categoryID uint, count *int64) error {
	return r.Db.Model(&models.Inflow{}).
		Where("inflow_category_id = ?", categoryID).
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Count(count).Error
}

func (r *InflowRepository) CountInflows(user *models.User, year int, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.Db.Model(&models.Inflow{}).
		Where("inflows.organization_id = ? AND YEAR(inflows.inflow_date) = ?", *user.PrimaryOrganizationID, year)

	if utils.NeedsJoin(filters, "inflow_category") {
		query = query.Joins("JOIN inflow_categories ON inflow_categories.id = inflows.inflow_category_id")
	}

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
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

func (r *InflowRepository) SumInflowsByCategory(user *models.User, categoryID uint, year, month int) (float64, error) {
	var total float64

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0) // Moves to the first day of the next month

	err := r.Db.Model(&models.Inflow{}).
		Where("inflow_category_id = ? AND inflow_date >= ? AND inflow_date < ?", categoryID, startDate, endDate).
		Where("organization_id = ?", &user.PrimaryOrganizationID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *InflowRepository) FindInflows(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Inflow, error) {
	var records []models.Inflow
	orderBy := sortField + " " + sortOrder

	query := r.Db.
		Preload("InflowCategory").
		Where("inflows.organization_id = ? AND YEAR(inflows.inflow_date) = ?", *user.PrimaryOrganizationID, year)

	if utils.NeedsJoin(filters, "inflow_category") {
		query = query.Joins("JOIN inflow_categories ON inflow_categories.id = inflows.inflow_category_id")
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

func (r *InflowRepository) GetInflowByID(user *models.User, inflowID uint) (*models.Inflow, error) {
	var inflow models.Inflow
	err := r.Db.Preload("InflowCategory").
		Where("id = ? AND organization_id = ?", inflowID, *user.PrimaryOrganizationID).
		First(&inflow).Error
	if err != nil {
		return nil, err
	}
	return &inflow, nil
}

func (r *InflowRepository) FindInflowCategoryByID(user *models.User, inflowCategoryID uint) (*models.InflowCategory, error) {
	var inflowCategory models.InflowCategory
	err := r.Db.
		Where("id = ? AND organization_id = ?", inflowCategoryID, *user.PrimaryOrganizationID).
		First(&inflowCategory).Error
	if err != nil {
		return nil, err
	}
	return &inflowCategory, nil
}

func (r *InflowRepository) FindDynamicCategoryByID(user *models.User, ID uint) (*models.DynamicCategory, error) {
	var record models.DynamicCategory
	err := r.Db.
		Where("id = ? AND organization_id = ?", ID, *user.PrimaryOrganizationID).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *InflowRepository) FindAllInflowsGroupedByMonth(user *models.User, year int) ([]models.InflowSummary, error) {
	var results []models.InflowSummary
	orgID := *user.PrimaryOrganizationID

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
	      AND i.organization_id = ?
	      AND YEAR(i.inflow_date) = ?
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
	          AND inflows.organization_id = ?
	          AND YEAR(inflow_date) = ?
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
	          AND organization_id = ?
	          AND YEAR(outflow_date) = ?
	        GROUP BY MONTH(outflow_date), outflow_category_id
	    ) outf ON df.related_type = 'outflow' 
	       AND outf.outflow_category_id = df.related_id 
	       AND outf.month = m.month
	    WHERE dc.organization_id = ?
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
	      AND i.organization_id = ?
	      AND YEAR(i.inflow_date) = ?
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
	    (CASE category_type WHEN 'static' THEN 0 WHEN 'dynamic' THEN 1 ELSE 2 END), 
	    category_id,
	    month;
	`

	// Execute the final query with the required parameters
	err := r.Db.Raw(finalQuery,
		// staticInflowsQuery
		orgID, year,
		// dynamicCategoriesQuery: first subquery
		orgID, year,
		// dynamicCategoriesQuery: second subquery
		orgID, year,
		// dynamicCategoriesQuery: dc filter
		orgID,
		// totalRowQuery
		orgID, year,
	).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *InflowRepository) FindAllInflowCategories(user *models.User) ([]models.InflowCategory, error) {
	var inflowCategories []models.InflowCategory
	result := r.Db.Where("organization_id = ?", *user.PrimaryOrganizationID).Find(&inflowCategories)
	return inflowCategories, result.Error
}

func (r *InflowRepository) FindAllDynamicCategories(user *models.User) ([]models.DynamicCategory, error) {
	var records []models.DynamicCategory
	result := r.Db.Preload("Mappings").
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Find(&records)
	return records, result.Error
}

func (r *InflowRepository) FetchMappingsForDynamicCategory(id uint) ([]models.DynamicCategoryMapping, error) {
	var records []models.DynamicCategoryMapping
	result := r.Db.Where("dynamic_category_id = ?", id).
		Find(&records)
	return records, result.Error
}

func (r *InflowRepository) FindDynamicCategoryById(user *models.User, id uint) (*models.DynamicCategory, error) {
	var record *models.DynamicCategory
	result := r.Db.Preload("Mappings").
		Where("id = ?", id).
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		First(&record)
	return record, result.Error
}

func (r *InflowRepository) InsertInflow(tx *gorm.DB, user *models.User, record *models.Inflow) (uint, error) {
	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *InflowRepository) UpdateInflow(tx *gorm.DB, user *models.User, record *models.Inflow) (uint, error) {
	record.UserID = user.ID
	if err := tx.Model(&models.Inflow{}).
		Where("id = ? AND organization_id = ?", record.ID, *user.PrimaryOrganizationID).
		Updates(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *InflowRepository) InsertInflowCategory(tx *gorm.DB, user *models.User, record *models.InflowCategory) error {
	var existing models.InflowCategory
	if err := tx.Where("organization_id = ? AND name = ?", *user.PrimaryOrganizationID, record.Name).First(&existing).Error; err == nil {
		return fmt.Errorf("category with this name already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) InsertDynamicCategory(tx *gorm.DB, user *models.User, record *models.DynamicCategory) (uint, error) {
	var existing models.DynamicCategory
	if err := tx.Where("organization_id = ? AND name = ?", *user.PrimaryOrganizationID, record.Name).First(&existing).Error; err == nil {
		return 0, fmt.Errorf("category with this name already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	record.OrganizationID = *user.PrimaryOrganizationID
	record.UserID = user.ID
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

func (r *InflowRepository) UpdateInflowCategory(tx *gorm.DB, user *models.User, record *models.InflowCategory) error {
	record.UserID = user.ID
	if err := tx.Model(&models.InflowCategory{}).
		Where("organization_id = ? AND id = ?", *user.PrimaryOrganizationID, record.ID).
		Updates(record).Error; err != nil {
		return err
	}
	return nil
}

func (r *InflowRepository) DropInflow(tx *gorm.DB, user *models.User, recordID uint) error {
	return tx.Where("id = ? AND organization_id = ?", recordID, *user.PrimaryOrganizationID).
		Delete(&models.Inflow{}).Error
}

func (r *InflowRepository) DropInflowCategory(tx *gorm.DB, user *models.User, recordID uint) error {
	result := tx.Where("organization_id = ?", *user.PrimaryOrganizationID).
		Delete(&models.InflowCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *InflowRepository) DropDynamicCategory(tx *gorm.DB, user *models.User, recordID uint) error {
	if err := tx.Where("dynamic_category_id = ?", recordID).
		Delete(&models.DynamicCategoryMapping{}).Error; err != nil {
		return err
	}

	result := tx.Where("organization_id = ?", *user.PrimaryOrganizationID).
		Delete(&models.DynamicCategory{}, recordID)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
