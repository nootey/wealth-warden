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
			randomAmount := 10.00 + rand.Float64()*(10000.00-10.00)
			randomDate := time.Now().AddDate(0, -rand.Intn(6), -rand.Intn(30))

			_, err := tx.ExecContext(ctx, `
				INSERT INTO outflows (user_id, outflow_category_id, amount, outflow_date, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?)
			`, userID, randomCategory, randomAmount, randomDate, time.Now(), time.Now())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downSeedOutflowTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
