package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type Category struct {
	Name          string
	SpendingLimit float64
}

func insertCategories(ctx context.Context, tx *gorm.DB, userIDs []uint, categories []Category, outflowType string) error {
	for _, userID := range userIDs {
		for _, category := range categories {
			err := tx.Exec(`
				INSERT INTO outflow_categories (user_id, name, spending_limit, outflow_type, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?)
			`, userID, category.Name, category.SpendingLimit, outflowType, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("failed to insert category %s for user %d: %w", category.Name, userID, err)
			}
		}
	}
	return nil
}

func deleteCategories(ctx context.Context, tx *gorm.DB, userIDs []uint, categories []Category) error {
	for _, userID := range userIDs {
		for _, category := range categories {
			err := tx.Exec(`
				DELETE FROM outflow_categories WHERE user_id = ? AND name = ?
			`, userID, category.Name).Error
			if err != nil {
				return fmt.Errorf("failed to delete category %s for user %d: %w", category.Name, userID, err)
			}
		}
	}
	return nil
}

func SeedOutflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Get user IDs
	userIDs, err := GetUserIDs(ctx, db, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	// Fixed and variable categories
	fixedCategories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
	}

	variableCategories := []Category{
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00}, {"SP", 160.00},
	}

	// Wrap in transaction
	return db.Transaction(func(tx *gorm.DB) error {
		if err := insertCategories(ctx, tx, userIDs, fixedCategories, "fixed"); err != nil {
			return err
		}
		if err := insertCategories(ctx, tx, userIDs, variableCategories, "variable"); err != nil {
			return err
		}
		return nil
	})
}

func UnseedOutflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Get user IDs
	userIDs, err := GetUserIDs(ctx, db, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	categories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00},
	}

	// Wrap deletion in transaction
	return db.Transaction(func(tx *gorm.DB) error {
		return deleteCategories(ctx, tx, userIDs, categories)
	})
}
