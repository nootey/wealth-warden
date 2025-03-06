package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

func SeedInflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	emails := []string{"support@wealth-warden.com", "member@wealth-warden.com"}

	var userIDs []uint
	for _, email := range emails {
		var userID uint
		err := db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return fmt.Errorf("error retrieving user ID for %s: %w", email, err)
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 {
		return nil
	}

	categories := []string{"Salary", "Food and transport", "SP", "Bonus", "Other"}

	for _, userID := range userIDs {
		for _, category := range categories {
			err := db.Exec(`
				INSERT INTO inflow_categories (user_id, name, created_at, updated_at) 
				VALUES (?, ?, ?, ?)
			`, userID, category, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("error inserting category %s for user %d: %w", category, userID, err)
			}
		}
	}

	return nil
}
