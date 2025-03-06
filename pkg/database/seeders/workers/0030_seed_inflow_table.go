package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

func SeedInflowTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	userIDs, err := GetUserIDs(ctx, db, emails)
	if err != nil || len(userIDs) == 0 {
		return err
	}

	randomDescriptions := []string{
		"Salary payment", "Freelance gig", "Bonus received", "Investment return",
		"Side hustle income", "Gift money", "Stock dividend", "Tax refund",
		"Cashback", "Rental income", "Savings interest", "Other income",
	}

	currentYear := time.Now().Year()

	// Use a transaction for safe execution
	return db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			var categoryIDs []uint
			err := tx.Raw(`SELECT id FROM inflow_categories WHERE user_id = ?`, userID).Scan(&categoryIDs).Error
			if err != nil {
				return fmt.Errorf("failed to retrieve categories for user %d: %w", userID, err)
			}

			if len(categoryIDs) == 0 {
				continue
			}

			for i := 0; i < 100; i++ {
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
					INSERT INTO inflows (user_id, inflow_category_id, amount, inflow_date, description, created_at, updated_at) 
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, userID, randomCategory, randomAmount, randomDate, description, time.Now(), time.Now()).Error
				if err != nil {
					return fmt.Errorf("failed to insert inflow record for user %d: %w", userID, err)
				}
			}
		}
		return nil
	})
}
