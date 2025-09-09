package workers

import (
	"context"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/rand"
	"strings"
	"time"
	"wealth-warden/internal/models"
)

func SeedAccounts(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	type acctSeed struct {
		Name           string
		Type           string
		Subtype        string `gorm:"column:sub_type"`
		Classification string
		Currency       string
		StartBalance   decimal.Decimal
	}

	seeds := []acctSeed{
		{Name: "Checking account", Type: "cash", Subtype: "checking", Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(1000)},
		{Name: "Savings account", Type: "cash", Subtype: "savings", Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(10000)},
		{Name: "Investment account", Type: "investment", Subtype: "brokerage", Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(2500)},
		{Name: "Crypto Exchange", Type: "crypto", Subtype: "exchange", Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(100000)},
		{Name: "Gambling debt", Type: "other_liability", Subtype: "other", Classification: "liability", Currency: "eur", StartBalance: decimal.NewFromInt(-50000)},
	}

	usernames := []string{"Support", "Member"}

	var users []models.User
	if err := db.WithContext(ctx).
		Where("username IN ?", usernames).
		Find(&users).Error; err != nil {
		return err
	}

	usersByName := map[string]models.User{}
	for _, u := range users {
		usersByName[u.Username] = u
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	const minDaysBack = 370 // strictly > 365
	const maxDaysBack = 430

	for _, uname := range usernames {
		u, ok := usersByName[uname]
		if !ok {
			logger.Warn("user not found, skipping", zap.String("username", uname))
			continue
		}

		for _, s := range seeds {
			// Find account type
			var at models.AccountType
			q := db.WithContext(ctx).Model(&models.AccountType{}).
				Where("LOWER(type) = ? AND LOWER(classification) = ?",
					strings.ToLower(s.Type), strings.ToLower(s.Classification))

			q = q.Where("LOWER(sub_type) = ?", strings.ToLower(s.Subtype))

			if err := q.First(&at).Error; err != nil {
				return err
			}

			// Skip if account already exists
			var existing models.Account
			err := db.WithContext(ctx).
				Where("user_id = ? AND name = ?", u.ID, s.Name).
				First(&existing).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}

			delta := minDaysBack + rng.Intn(maxDaysBack-minDaysBack+1)
			asOf := today.AddDate(0, 0, -delta)

			// Create account
			acc := models.Account{
				UserID:        u.ID,
				Name:          s.Name,
				AccountTypeID: at.ID,
				Currency:      strings.ToUpper(s.Currency),
				CreatedAt:     asOf,
				UpdatedAt:     asOf,
			}
			if err := db.WithContext(ctx).Create(&acc).Error; err != nil {
				return err
			}

			// Create balance with start balance
			bal := models.Balance{
				AccountID:    acc.ID,
				AsOf:         asOf,
				StartBalance: s.StartBalance,
				Currency:     strings.ToUpper(s.Currency),
				CreatedAt:    asOf,
				UpdatedAt:    asOf,
			}
			if err := db.WithContext(ctx).Create(&bal).Error; err != nil {
				return err
			}

			logger.Info("seeded account",
				zap.String("username", uname),
				zap.String("name", s.Name),
				zap.String("start_balance", s.StartBalance.StringFixed(2)),
			)
		}
	}

	return nil
}
