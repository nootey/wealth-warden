package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

func SeedDynamicCategories(ctx context.Context, db *gorm.DB) error {
	// Retrieve the organization ID for "Super Admin"
	var orgID uint
	err := db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Super Admin").Scan(&orgID).Error
	if err != nil {
		return fmt.Errorf("failed to retrieve organization id for 'Super Admin': %w", err)
	}
	if orgID == 0 {
		return fmt.Errorf("organization 'Super Admin' not found")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Insert "True Salary" Dynamic Category linked to the organization.
		trueSalary := struct {
			ID             int64
			OrganizationID int
			Name           string
			CreatedAt      time.Time
			UpdatedAt      time.Time
		}{
			OrganizationID: int(orgID),
			Name:           "True Salary",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := tx.Table("dynamic_categories").Create(&trueSalary).Error; err != nil {
			return fmt.Errorf("failed to insert 'True Salary' category: %w", err)
		}
		trueSalaryCategoryID := trueSalary.ID

		// Fetch IDs for inflow categories "Salary", "Food and transport", and "SP" linked to the organization.
		inflowCategories := []string{"Salary", "Food and transport", "SP"}
		var inflowCategoryIDs []int64
		err := tx.Table("inflow_categories").
			Select("id").
			Where("organization_id = ? AND name IN ?", orgID, inflowCategories).
			Scan(&inflowCategoryIDs).Error
		if err != nil {
			return fmt.Errorf("failed to retrieve inflow categories: %w", err)
		}

		// Insert mappings for "True Salary"
		for _, categoryID := range inflowCategoryIDs {
			err = tx.Exec(`
				INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at)
				VALUES (?, 'inflow', ?, ?, ?)
			`, trueSalaryCategoryID, categoryID, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("failed to insert mapping for inflow category %d: %w", categoryID, err)
			}
		}

		// Insert "Effective salary" Dynamic Category linked to the organization.
		effectiveSalary := struct {
			ID             int64
			OrganizationID int
			Name           string
			CreatedAt      time.Time
			UpdatedAt      time.Time
		}{
			OrganizationID: int(orgID),
			Name:           "Effective salary",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := tx.Table("dynamic_categories").Create(&effectiveSalary).Error; err != nil {
			return fmt.Errorf("failed to insert 'Effective Salary' category: %w", err)
		}
		effectiveSalaryCategoryID := effectiveSalary.ID

		// Link "Effective salary" to "True Salary" (dynamic mapping).
		err = tx.Exec(`
			INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at)
			VALUES (?, 'dynamic', ?, ?, ?)
		`, effectiveSalaryCategoryID, trueSalaryCategoryID, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to link 'Effective salary' to 'True Salary': %w", err)
		}

		// Fetch "SP" outflow category ID linked to the organization.
		var outflowCategoryID int64
		err = tx.Raw(`
			SELECT id FROM outflow_categories WHERE organization_id = ? AND name = ?
		`, orgID, "SP").Scan(&outflowCategoryID).Error
		if err == nil {
			// If "SP" outflow category exists, link it.
			err = tx.Exec(`
				INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at)
				VALUES (?, 'outflow', ?, ?, ?)
			`, effectiveSalaryCategoryID, outflowCategoryID, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("failed to link 'Effective salary' to 'SP' outflow category: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to retrieve 'SP' outflow category: %w", err)
		}

		return nil
	})
}

func UnseedDynamicCategories(ctx context.Context, db *gorm.DB) error {
	// Retrieve the organization ID for "Super Admin"
	var orgID uint
	err := db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Super Admin").Scan(&orgID).Error
	if err != nil {
		return fmt.Errorf("failed to retrieve organization id for 'Super Admin': %w", err)
	}
	if orgID == 0 {
		return fmt.Errorf("organization 'Super Admin' not found")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Retrieve dynamic category IDs linked to the organization.
		var trueSalaryCategoryID, effectiveSalaryCategoryID int64
		err := tx.Raw(`
			SELECT id FROM dynamic_categories WHERE organization_id = ? AND name = ?
		`, orgID, "True Salary").Scan(&trueSalaryCategoryID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find 'True Salary' category: %w", err)
		}
		err = tx.Raw(`
			SELECT id FROM dynamic_categories WHERE organization_id = ? AND name = ?
		`, orgID, "Effective salary").Scan(&effectiveSalaryCategoryID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find 'Effective salary' category: %w", err)
		}

		// Delete dynamic category mappings.
		if trueSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, trueSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete mappings for 'True Salary': %w", err)
			}
		}
		if effectiveSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, effectiveSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete mappings for 'Effective salary': %w", err)
			}
		}

		// Delete the dynamic categories.
		if trueSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_categories WHERE id = ?`, trueSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete 'True Salary' category: %w", err)
			}
		}
		if effectiveSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_categories WHERE id = ?`, effectiveSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete 'Effective salary' category: %w", err)
			}
		}

		return nil
	})
}
