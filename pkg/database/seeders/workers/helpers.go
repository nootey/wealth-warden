package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

func GetUserIDs(ctx context.Context, db *gorm.DB, emails []string) ([]uint, error) {
	var userIDs []uint

	// Query all matching user IDs in a single query
	err := db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email IN (?)`, emails).Scan(&userIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user IDs: %w", err)
	}

	return userIDs, nil
}
