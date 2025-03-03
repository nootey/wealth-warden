package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"time"
)

func init() {
	goose.AddMigrationContext(upSeedOutflowCategoryTable, downSeedOutflowCategoryTable)
}

type Category struct {
	Name          string
	SpendingLimit float64
}

func getUserIDs(ctx context.Context, tx *sql.Tx, emails []string) ([]uint, error) {
	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

func insertCategories(ctx context.Context, tx *sql.Tx, userIDs []uint, categories []Category, outflowType string) error {
	for _, userID := range userIDs {
		for _, category := range categories {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO outflow_categories (user_id, name, spending_limit, outflow_type, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?)
			`, userID, category.Name, category.SpendingLimit, outflowType, time.Now(), time.Now())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteCategories(ctx context.Context, tx *sql.Tx, userIDs []uint, categories []Category) error {
	for _, userID := range userIDs {
		for _, category := range categories {
			_, err := tx.ExecContext(ctx, `
				DELETE FROM outflow_categories WHERE user_id = ? AND name = ?
			`, userID, category.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func upSeedOutflowCategoryTable(ctx context.Context, tx *sql.Tx) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}
	userIDs, err := getUserIDs(ctx, tx, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	fixedCategories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
	}

	err = insertCategories(ctx, tx, userIDs, fixedCategories, "fixed")
	if err != nil {
		return err
	}

	variableCategories := []Category{
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00}, {"SP", 160.00},
	}

	err = insertCategories(ctx, tx, userIDs, variableCategories, "variable")
	if err != nil {
		return err
	}

	return nil
}

func downSeedOutflowCategoryTable(ctx context.Context, tx *sql.Tx) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}
	userIDs, err := getUserIDs(ctx, tx, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	categories := []Category{
		{"Rent", 600.00}, {"Utility", 200.00}, {"Car loan", 500.00}, {"Phone plan", 15.00},
		{"Car - gas", 130.00}, {"Car - general", 500.00}, {"Food", 250.00}, {"Health", 250.00},
		{"Hygiene", 100.00}, {"Socialization", 100.00}, {"Tech", 250.00}, {"Entertainment", 100.00},
		{"eCommerce", 100.00}, {"Gifts", 300.00}, {"Random", 150.00},
	}

	return deleteCategories(ctx, tx, userIDs, categories)
}
