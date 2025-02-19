package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"time"
)

func init() {
	goose.AddMigrationContext(upSeedInflowCategoryTable, downSeedInflowCategoryTable)
}

func upSeedInflowCategoryTable(ctx context.Context, tx *sql.Tx) error {

	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 {
		return nil
	}

	categories := []string{"Salary", "Food and transport", "SP", "Bonus", "Other"}

	for _, userID := range userIDs {
		for _, category := range categories {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO inflow_categories (user_id, name, created_at, updated_at) 
				VALUES (?, ?, ?, ?)
			`, userID, category, time.Now(), time.Now())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downSeedInflowCategoryTable(ctx context.Context, tx *sql.Tx) error {

	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 {
		return nil
	}

	categories := []string{"Salary", "Food and transport", "SP", "Bonus", "Other"}

	for _, userID := range userIDs {
		for _, category := range categories {
			_, err := tx.ExecContext(ctx, `
				DELETE FROM inflow_categories WHERE user_id = ? AND name = ?
			`, userID, category)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
