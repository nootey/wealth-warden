package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

// SeedOutflowTable inserts random outflows for multiple organizations.
func SeedOutflowTable(ctx context.Context, db *gorm.DB) error {
	// Define organization names that should have outflow records.
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
		"Sushi", "Went out", "Contacts", "Gift",
		"Gas", "Car part", "Hood burger", "Taxes",
		"Toothbrush", "Laptop", "Phone", "Rice cooker",
	}

	currentYear := time.Now().Year()

	// Use a transaction for safe execution.
	return db.Transaction(func(tx *gorm.DB) error {
		for _, orgID := range organizationIDs {
			// Get outflow category IDs for the organization.
			var categoryIDs []uint
			err := tx.Raw(`SELECT id FROM outflow_categories WHERE organization_id = ?`, orgID).Scan(&categoryIDs).Error
			if err != nil {
				return fmt.Errorf("failed to retrieve categories for organization %d: %w", orgID, err)
			}

			if len(categoryIDs) == 0 {
				continue
			}

			// Insert random outflows.
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
					INSERT INTO outflows (organization_id, outflow_category_id, amount, outflow_date, description, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, orgID, randomCategory, randomAmount, randomDate, description, time.Now(), time.Now()).Error
				if err != nil {
					return fmt.Errorf("failed to insert outflow for organization %d: %w", orgID, err)
				}
			}
		}
		return nil
	})
}

// UnseedOutflowTable deletes outflows for the given organizations.
func UnseedOutflowTable(ctx context.Context, db *gorm.DB) error {
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

	// Wrap deletion in a transaction.
	return db.Transaction(func(tx *gorm.DB) error {
		for _, orgID := range organizationIDs {
			err := tx.Exec(`DELETE FROM outflows WHERE organization_id = ?`, orgID).Error
			if err != nil {
				return fmt.Errorf("failed to delete outflows for organization %d: %w", orgID, err)
			}
		}
		return nil
	})
}
