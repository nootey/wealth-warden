package workers

import (
	"context"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AccountType struct {
	Type    string
	Subtype string
}

func SeedAccountTypes(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {

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
			Where("type = ? AND subtype = ?", seed.Type, seed.Subtype).
			Count(&existing).Error
		if err != nil {
			logger.Error("failed checking existing account_type", zap.Error(err))
			return err
		}

		if existing == 0 {
			if err := db.WithContext(ctx).Create(&AccountType{
				Type:    seed.Type,
				Subtype: seed.Subtype,
			}).Error; err != nil {
				logger.Error("failed inserting account_type", zap.String("type", seed.Type), zap.String("subtype", seed.Subtype), zap.Error(err))
				return err
			}
			logger.Info("seeded account_type", zap.String("type", seed.Type), zap.String("subtype", seed.Subtype))
		}
	}

	return nil
}
