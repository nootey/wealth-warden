package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"math/rand"
	"time"
)

func init() {
	goose.AddMigrationContext(upSeedOutflowTable, downSeedOutflowTable)
}

func upSeedOutflowTable(ctx context.Context, tx *sql.Tx) error {

	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}
	userIDs, err := getUserIDs(ctx, tx, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	randomDescriptions := []string{
		"Sushi", "Went out", "Contacts", "Gift",
		"Gas", "Car part", "Hood burger", "Taxes",
		"Toothbrush", "Laptop", "Phone", "Rice cooker",
	}

	currentYear := time.Now().Year()

	for _, userID := range userIDs {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM outflow_categories WHERE user_id = ?`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		var categoryIDs []uint
		for rows.Next() {
			var categoryID uint
			if err := rows.Scan(&categoryID); err != nil {
				return err
			}
			categoryIDs = append(categoryIDs, categoryID)
		}

		if len(categoryIDs) == 0 {
			continue
		}

		for i := 0; i < 20; i++ {
			randomCategory := categoryIDs[rand.Intn(len(categoryIDs))]
			randomAmount := 10.00 + rand.Float64()*(1000.00-10.00)

			randomMonth := rand.Intn(12) + 1
			randomDay := rand.Intn(28) + 1
			randomDate := time.Date(currentYear, time.Month(randomMonth), randomDay, 0, 0, 0, 0, time.UTC)

			var description *string
			if rand.Float64() < 0.5 { // 50% chance of having a description
				desc := randomDescriptions[rand.Intn(len(randomDescriptions))]
				description = &desc
			}

			_, err := tx.ExecContext(ctx, `
				INSERT INTO outflows (user_id, outflow_category_id, amount, outflow_date, description, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, userID, randomCategory, randomAmount, randomDate, description, time.Now(), time.Now())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downSeedOutflowTable(ctx context.Context, tx *sql.Tx) error {
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

	for _, userID := range userIDs {
		_, err := tx.ExecContext(ctx, `DELETE FROM outflows WHERE user_id = ?`, userID)
		if err != nil {
			return err
		}
	}

	return nil
}
