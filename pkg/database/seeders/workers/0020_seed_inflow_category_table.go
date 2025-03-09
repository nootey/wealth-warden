package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

func SeedInflowCategoryTable(ctx context.Context, db *gorm.DB) error {
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

	categories := []string{"Salary", "Food and transport", "SP", "Bonus", "Other"}

	for _, orgID := range organizationIDs {
		var userID uint
		err := db.Raw(`
			SELECT user_id FROM organization_users 
			WHERE organization_id = ? 
			LIMIT 1
		`, orgID).Scan(&userID).Error
		if err != nil {
			return fmt.Errorf("error retrieving user ID for organization %d: %w", orgID, err)
		}
		if userID == 0 {
			return fmt.Errorf("user id can not be 0 for organization %d: %w", orgID, err)
		}

		for _, category := range categories {
			err := db.Exec(`
				INSERT INTO inflow_categories (organization_id, user_id, name, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?)
			`, orgID, userID, category, time.Now(), time.Now()).Error
			if err != nil {
				return fmt.Errorf("error inserting category %s for organization %d: %w", category, orgID, err)
			}
		}
	}

	return nil
}
