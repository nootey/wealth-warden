package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

func SeedInflowTable(ctx context.Context, db *gorm.DB) error {
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
	randomDescriptions := []string{
		"Salary payment", "Freelance gig", "Bonus received", "Investment return",
		"Side hustle income", "Gift money", "Stock dividend", "Tax refund",
		"Cashback", "Rental income", "Savings interest", "Other income",
	}
	currentYear := time.Now().Year()
	return db.Transaction(func(tx *gorm.DB) error {
		for _, orgID := range organizationIDs {
			var userID uint
			err := tx.Raw(`SELECT user_id FROM organization_users WHERE organization_id = ? LIMIT 1`, orgID).Scan(&userID).Error
			if err != nil {
				return fmt.Errorf("error retrieving user ID for organization %d: %w", orgID, err)
			}
			if userID == 0 {
				continue
			}
			var categoryIDs []uint
			err = tx.Raw(`SELECT id FROM inflow_categories WHERE organization_id = ?`, orgID).Scan(&categoryIDs).Error
			if err != nil {
				return fmt.Errorf("failed to retrieve categories for organization %d: %w", orgID, err)
			}
			if len(categoryIDs) == 0 {
				continue
			}
			// Insert 100 random inflow records.
			for i := 0; i < 100; i++ {
				randomCategory := categoryIDs[rand.Intn(len(categoryIDs))]
				randomAmount := 10.00 + rand.Float64()*(1000.00-10.00)
				randomMonth := rand.Intn(12) + 1
				randomDay := rand.Intn(28) + 1
				randomDate := time.Date(currentYear, time.Month(randomMonth), randomDay, 0, 0, 0, 0, time.UTC)
				var description *string
				if rand.Float64() < 0.5 {
					desc := randomDescriptions[rand.Intn(len(randomDescriptions))]
					description = &desc
				}
				err = tx.Exec(`
					INSERT INTO inflows (organization_id, user_id, inflow_category_id, amount, inflow_date, description, created_at, updated_at) 
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, orgID, userID, randomCategory, randomAmount, randomDate, description, time.Now(), time.Now()).Error
				if err != nil {
					return fmt.Errorf("failed to insert inflow record for organization %d: %w", orgID, err)
				}
			}

			// Insert one guaranteed record per category for the current month.
			currentDate := time.Now()
			// Use current date with the current year and month (time set to midnight UTC)
			guaranteedDate := time.Date(currentYear, currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
			for _, categoryID := range categoryIDs {
				randomAmount := 10.00 + rand.Float64()*(1000.00-10.00)
				var description *string
				if rand.Float64() < 0.5 {
					desc := randomDescriptions[rand.Intn(len(randomDescriptions))]
					description = &desc
				}
				err = tx.Exec(`
					INSERT INTO inflows (organization_id, user_id, inflow_category_id, amount, inflow_date, description, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, orgID, userID, categoryID, randomAmount, guaranteedDate, description, time.Now(), time.Now()).Error
				if err != nil {
					return fmt.Errorf("failed to insert guaranteed inflow record for organization %d, category %d: %w", orgID, categoryID, err)
				}
			}
		}
		return nil
	})
}
