package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"time"
)

func init() {
	goose.AddMigrationContext(upSeedDynamicCategories, downSeedDynamicCategories)
}

func upSeedDynamicCategories(ctx context.Context, tx *sql.Tx) error {
	userID := 1 // Seeding for user with ID 1

	// Insert "True Salary" Dynamic Category
	res, err := tx.ExecContext(ctx, `
		INSERT INTO dynamic_categories (user_id, name, created_at, updated_at) 
		VALUES (?, ?, ?, ?)
	`, userID, "True Salary", time.Now(), time.Now())
	if err != nil {
		return err
	}

	trueSalaryCategoryID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// Fetch IDs for "Salary", "Food and transport", and "SP" inflow categories
	inflowCategories := []string{"Salary", "Food and transport", "SP"}
	for _, category := range inflowCategories {
		var categoryID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM inflow_categories WHERE user_id = ? AND name = ?`, userID, category).Scan(&categoryID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}

		// Insert mappings for "True Salary"
		_, err = tx.ExecContext(ctx, `
			INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at) 
			VALUES (?, 'inflow', ?, ?, ?)
		`, trueSalaryCategoryID, categoryID, time.Now(), time.Now())
		if err != nil {
			return err
		}
	}

	res, err = tx.ExecContext(ctx, `
		INSERT INTO dynamic_categories (user_id, name, created_at, updated_at) 
		VALUES (?, ?, ?, ?)
	`, userID, "Effective salary", time.Now(), time.Now())
	if err != nil {
		return err
	}

	expenseCoverCategoryID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at) 
		VALUES (?, 'dynamic', ?, ?, ?)
	`, expenseCoverCategoryID, trueSalaryCategoryID, time.Now(), time.Now())
	if err != nil {
		return err
	}

	var OutflowCategoryID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM outflow_categories WHERE user_id = ? AND name = ?`, userID, "SP").Scan(&OutflowCategoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // If no SP outflow exists, we skip it
		}
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO dynamic_category_mappings (dynamic_category_id, related_type, related_id, created_at, updated_at) 
		VALUES (?, 'outflow', ?, ?, ?)
	`, expenseCoverCategoryID, OutflowCategoryID, time.Now(), time.Now())

	return err
}

func downSeedDynamicCategories(ctx context.Context, tx *sql.Tx) error {
	userID := 1

	// Get category IDs
	var trueSalaryCategoryID, expenseCoverCategoryID int64
	_ = tx.QueryRowContext(ctx, `SELECT id FROM dynamic_categories WHERE user_id = ? AND name = ?`, userID, "True Salary").Scan(&trueSalaryCategoryID)
	_ = tx.QueryRowContext(ctx, `SELECT id FROM dynamic_categories WHERE user_id = ? AND name = ?`, userID, "Effective salary").Scan(&expenseCoverCategoryID)

	// Delete mappings first
	_, _ = tx.ExecContext(ctx, `DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, trueSalaryCategoryID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM dynamic_category_mappings WHERE dynamic_category_id = ?`, expenseCoverCategoryID)

	// Delete categories
	_, _ = tx.ExecContext(ctx, `DELETE FROM dynamic_categories WHERE id = ?`, trueSalaryCategoryID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM dynamic_categories WHERE id = ?`, expenseCoverCategoryID)

	return nil
}
