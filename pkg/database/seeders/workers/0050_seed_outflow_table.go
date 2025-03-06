package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

// SeedOutflowTable inserts random outflows for multiple users
func SeedOutflowTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Get user IDs
	userIDs, err := GetUserIDs(ctx, db, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	randomDescriptions := []string{
		"Sushi", "Went out", "Contacts", "Gift",
		"Gas", "Car part", "Hood burger", "Taxes",
		"Toothbrush", "Laptop", "Phone", "Rice cooker",
	}

	currentYear := time.Now().Year()

	// Wrap everything in a transaction
	return db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			// Get outflow category IDs for the user
			var categoryIDs []uint
			err := tx.Raw(`SELECT id FROM outflow_categories WHERE user_id = ?`, userID).Scan(&categoryIDs).Error
			if err != nil {
				return fmt.Errorf("failed to retrieve categories for user %d: %w", userID, err)
			}

			if len(categoryIDs) == 0 {
				continue
			}

			// Insert random outflows
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

				err := tx.Exec(`
					INSERT INTO outflows (user_id, outflow_category_id, amount, outflow_date, description, created_at, updated_at) 
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, userID, randomCategory, randomAmount, randomDate, description, time.Now(), time.Now()).Error
				if err != nil {
					return fmt.Errorf("failed to insert outflow for user %d: %w", userID, err)
				}
			}
		}
		return nil
	})
}

// UnseedOutflowTable deletes outflows for given users
func UnseedOutflowTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	// Get user IDs
	userIDs, err := GetUserIDs(ctx, db, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	// Wrap deletion in a transaction
	return db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			err := tx.Exec(`DELETE FROM outflows WHERE user_id = ?`, userID).Error
			if err != nil {
				return fmt.Errorf("failed to delete outflows for user %d: %w", userID, err)
			}
		}
		return nil
	})
}
