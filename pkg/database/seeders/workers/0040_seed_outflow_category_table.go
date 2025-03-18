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

func insertCategories(ctx context.Context, tx *gorm.DB, orgIDs []uint, categories []Category, outflowType string) error {
	for _, orgID := range orgIDs {
		var userID uint
		err := tx.Raw(`SELECT user_id FROM organization_users WHERE organization_id = ? LIMIT 1`, orgID).Scan(&userID).Error
		if err != nil {
			return fmt.Errorf("error retrieving user ID for organization %d: %w", orgID, err)
		}
		if userID == 0 {
			continue
		}
		for _, category := range categories {
			err := tx.Exec(`
				INSERT INTO outflow_categories (organization_id, user_id, name, spending_limit, outflow_type, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, orgID, userID, category.Name, category.SpendingLimit, outflowType, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("failed to insert category %s for organization %d: %w", category.Name, orgID, err)
			}
		}
	}
	return nil
}

func deleteCategories(ctx context.Context, tx *gorm.DB, orgIDs []uint, categories []Category) error {
	for _, orgID := range orgIDs {
		for _, category := range categories {
			err := tx.Exec(`
				DELETE FROM outflow_categories WHERE organization_id = ? AND name = ?
			`, orgID, category.Name).Error
			if err != nil {
				return fmt.Errorf("failed to delete category %s for organization %d: %w", category.Name, orgID, err)
			}
		}
	}
	return nil
}

func SeedOutflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	orgNames := []string{"Super Admin", "Member"}
	var organizationIDs []uint
	for _, orgName := range orgNames {
		var orgID uint
		err := db.Raw(`SELECT id FROM organizations WHERE name = ?`, orgName).Scan(&orgID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return fmt.Errorf("error retrieving organization ID for %s: %w", orgName, err)
		}
		organizationIDs = append(organizationIDs, orgID)
	}
	if len(organizationIDs) == 0 {
		return nil
	}
	fixedCategories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
	}
	variableCategories := []Category{
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00}, {"SP - Taxes", 160.00},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := insertCategories(ctx, tx, organizationIDs, fixedCategories, "fixed"); err != nil {
			return err
		}
		if err := insertCategories(ctx, tx, organizationIDs, variableCategories, "variable"); err != nil {
			return err
		}
		return nil
	})
}

func UnseedOutflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	orgNames := []string{"Super Admin", "Member"}
	var organizationIDs []uint
	for _, orgName := range orgNames {
		var orgID uint
		err := db.Raw(`SELECT id FROM organizations WHERE name = ?`, orgName).Scan(&orgID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return fmt.Errorf("error retrieving organization ID for %s: %w", orgName, err)
		}
		organizationIDs = append(organizationIDs, orgID)
	}
	if len(organizationIDs) == 0 {
		return nil
	}
	categories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00}, {"SP - Taxes", 160.00},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return deleteCategories(ctx, tx, organizationIDs, categories)
	})
}
