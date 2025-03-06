package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

func SeedInflowCategoryTable(ctx context.Context, db *gorm.DB) error {
	// Define the organization names that should have inflow categories.
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

	// Define the inflow categories.
	categories := []string{"Salary", "Food and transport", "SP", "Bonus", "Other"}

	// Insert each category for each organization.
	for _, orgID := range organizationIDs {
		for _, category := range categories {
			err := db.Exec(`
				INSERT INTO inflow_categories (organization_id, name, created_at, updated_at) 
				VALUES (?, ?, ?, ?)
			`, orgID, category, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("error inserting category %s for organization %d: %w", category, orgID, err)
			}
		}
	}

	return nil
}
