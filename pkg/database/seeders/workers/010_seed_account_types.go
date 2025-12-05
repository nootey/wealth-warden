package workers

import (
	"context"
	"fmt"
	"wealth-warden/pkg/config"

	"gorm.io/gorm"
)

type AccountType struct {
	Type    string `gorm:"column:type"`
	Subtype string `gorm:"column:sub_type"`
}

func SeedAccountTypes(ctx context.Context, db *gorm.DB, cfg *config.Config) error {

	var accountTypeSeeds = []AccountType{
		// Cash
		{Type: "cash", Subtype: "checking"},
		{Type: "cash", Subtype: "savings"},
		{Type: "cash", Subtype: "health_savings"},
		{Type: "cash", Subtype: "money_market"},

		// Investment
		{Type: "investment", Subtype: "brokerage"},
		{Type: "investment", Subtype: "retirement"},
		{Type: "investment", Subtype: "pension"},
		{Type: "investment", Subtype: "mutual_fund"},

		// Crypto
		{Type: "crypto", Subtype: "wallet"},
		{Type: "crypto", Subtype: "exchange"},

		// Property
		{Type: "property", Subtype: "residential"},
		{Type: "property", Subtype: "commercial"},

		// Vehicle
		{Type: "vehicle", Subtype: "car"},
		{Type: "vehicle", Subtype: "motorcycle"},
		{Type: "vehicle", Subtype: "boat"},
		{Type: "vehicle", Subtype: "private_jet"},

		// Other Assets
		{Type: "other_asset", Subtype: "other"},

		// Liabilities
		{Type: "credit_card", Subtype: "credit"},
		{Type: "loan", Subtype: "mortgage"},
		{Type: "loan", Subtype: "student"},
		{Type: "loan", Subtype: "personal"},
		{Type: "other_liability", Subtype: "other"},
	}

	for _, seed := range accountTypeSeeds {
		var existing int64
		err := db.WithContext(ctx).Model(&AccountType{}).
			Where("type = ? AND sub_type = ?", seed.Type, seed.Subtype).
			Count(&existing).Error
		if err != nil {
			return fmt.Errorf("failed checking existing account_type: %w", err)
		}

		if existing == 0 {
			if err := db.WithContext(ctx).Create(&AccountType{
				Type:    seed.Type,
				Subtype: seed.Subtype,
			}).Error; err != nil {
				return fmt.Errorf("failed inserting account_type: %w", err)
			}
		}
	}

	return nil
}
