package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

func SeedDynamicCategories(ctx context.Context, db *gorm.DB) error {
	userID := 1 // Seeding for user with ID 1

	return db.Transaction(func(tx *gorm.DB) error {
		// Insert "True Salary" Dynamic Category
		trueSalary := struct {
			ID        int64
			UserID    int
			Name      string
			CreatedAt time.Time
			UpdatedAt time.Time
		}{UserID: userID, Name: "True Salary", CreatedAt: time.Now(), UpdatedAt: time.Now()}

		if err := tx.Table("dynamic_categories").Create(&trueSalary).Error; err != nil {
			return fmt.Errorf("failed to insert 'True Salary' category: %w", err)
		}

		trueSalaryCategoryID := trueSalary.ID

		// Fetch IDs for "Salary", "Food and transport", and "SP" inflow categories
		inflowCategories := []string{"Salary", "Food and transport", "SP"}
		var inflowCategoryIDs []int64

		err := tx.Table("inflow_categories").
			Select("id").
			Where("user_id = ? AND name IN ?", userID, inflowCategories).
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

		// Insert "Effective Salary" Dynamic Category
		effectiveSalary := struct {
			ID        int64
			UserID    int
			Name      string
			CreatedAt time.Time
			UpdatedAt time.Time
		}{UserID: userID, Name: "Effective salary", CreatedAt: time.Now(), UpdatedAt: time.Now()}

		if err := tx.Table("dynamic_categories").Create(&effectiveSalary).Error; err != nil {
			return fmt.Errorf("failed to insert 'Effective Salary' category: %w", err)
		}

		expenseCoverCategoryID := effectiveSalary.ID

		// Link "Effective Salary" to "True Salary"
		err = tx.Exec(`
			INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at) 
			VALUES (?, 'dynamic', ?, ?, ?)
		`, expenseCoverCategoryID, trueSalaryCategoryID, time.Now(), time.Now()).Error
		if err != nil {
			return fmt.Errorf("failed to link 'Effective Salary' to 'True Salary': %w", err)
		}

		// Fetch "SP" outflow category ID
		var outflowCategoryID int64
		err = tx.Raw(`
			SELECT id FROM outflow_categories WHERE user_id = ? AND name = ?
		`, userID, "SP").Scan(&outflowCategoryID).Error

		// If "SP" outflow category exists, link it
		if err == nil {
			err = tx.Exec(`
				INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at) 
				VALUES (?, 'outflow', ?, ?, ?)
			`, expenseCoverCategoryID, outflowCategoryID, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("failed to link 'Effective Salary' to 'SP' outflow category: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to retrieve 'SP' outflow category: %w", err)
		}

		return nil
	})
}

func UnseedDynamicCategories(ctx context.Context, db *gorm.DB) error {
	userID := 1

	return db.Transaction(func(tx *gorm.DB) error {
		// Get category IDs
		var trueSalaryCategoryID, expenseCoverCategoryID int64
		err := tx.Raw(`
			SELECT id FROM dynamic_categories WHERE user_id = ? AND name = ?
		`, userID, "True Salary").Scan(&trueSalaryCategoryID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find 'True Salary' category: %w", err)
		}

		err = tx.Raw(`
			SELECT id FROM dynamic_categories WHERE user_id = ? AND name = ?
		`, userID, "Effective salary").Scan(&expenseCoverCategoryID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find 'Effective Salary' category: %w", err)
		}

		// Delete mappings first
		if trueSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, trueSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete mappings for 'True Salary': %w", err)
			}
		}
		if expenseCoverCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, expenseCoverCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete mappings for 'Effective Salary': %w", err)
			}
		}

		// Delete categories
		if trueSalaryCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_categories WHERE id = ?`, trueSalaryCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete 'True Salary' category: %w", err)
			}
		}
		if expenseCoverCategoryID > 0 {
			err = tx.Exec(`DELETE FROM dynamic_categories WHERE id = ?`, expenseCoverCategoryID).Error
			if err != nil {
				return fmt.Errorf("failed to delete 'Effective Salary' category: %w", err)
			}
		}

		return nil
	})
}
