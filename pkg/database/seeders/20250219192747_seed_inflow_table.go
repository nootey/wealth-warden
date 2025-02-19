package migrations

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSeedInflowTable, downSeedInflowTable)
}

func upSeedInflowTable(ctx context.Context, tx *sql.Tx) error {
	// Define the target users by email
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Fetch user IDs for the specified emails
	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue // Skip if user not found
			}
			return err
		}
		userIDs = append(userIDs, userID)
	}

	// If no users found, return
	if len(userIDs) == 0 {
		return nil
	}

	// Seed inflows for each user
	for _, userID := range userIDs {
		// Fetch all category IDs for this user
		rows, err := tx.QueryContext(ctx, `SELECT id FROM inflow_categories WHERE user_id = ?`, userID)
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

		// If no categories exist for this user, skip seeding inflows
		if len(categoryIDs) == 0 {
			continue
		}

		// Insert 20 inflows for this user
		for i := 0; i < 20; i++ {
			randomCategory := categoryIDs[rand.Intn(len(categoryIDs))]         // Pick a random category
			randomAmount := 10.00 + rand.Float64()*(10000.00-10.00)            // Generate a random amount between 10 and 10,000
			randomDate := time.Now().AddDate(0, -rand.Intn(6), -rand.Intn(30)) // Random date in the last 6 months

			_, err := tx.ExecContext(ctx, `
				INSERT INTO inflows (user_id, inflow_category_id, amount, inflow_date, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?)
			`, userID, randomCategory, randomAmount, randomDate, time.Now(), time.Now())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downSeedInflowTable(ctx context.Context, tx *sql.Tx) error {
	// Define the target users by email
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Fetch user IDs for the specified emails
	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue // Skip if user not found
			}
			return err
		}
		userIDs = append(userIDs, userID)
	}

	// If no users found, return
	if len(userIDs) == 0 {
		return nil
	}

	// Delete inflows for the specified users
	for _, userID := range userIDs {
		_, err := tx.ExecContext(ctx, `DELETE FROM inflows WHERE user_id = ?`, userID)
		if err != nil {
			return err
		}
	}

	return nil
}
