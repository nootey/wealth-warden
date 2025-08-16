package workers

import (
	"context"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"time"
	"wealth-warden/internal/models"
)

func strptr(s string) *string { return &s }

func SeedAccounts(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	type acctSeed struct {
		Name           string
		Type           string
		Subtype        *string
		Classification string
		Currency       string
		StartBalance   decimal.Decimal
	}

	seeds := []acctSeed{
		{Name: "Checking account", Type: "cash", Subtype: strptr("checking"), Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(1000)},
		{Name: "Savings account", Type: "cash", Subtype: strptr("savings"), Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(10000)},
		{Name: "Investment account", Type: "investment", Subtype: strptr("brokerage"), Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(2500)},
		{Name: "Crypto Exchange", Type: "crypto", Subtype: strptr("exchange"), Classification: "asset", Currency: "eur", StartBalance: decimal.NewFromInt(100000)},
		{Name: "Gambling debt", Type: "other_liability", Subtype: strptr("other"), Classification: "liability", Currency: "eur", StartBalance: decimal.NewFromInt(-50000)},
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

			if s.Subtype != nil {
				q = q.Where("LOWER(subtype) = ?", strings.ToLower(*s.Subtype))
			} else {
				q = q.Where("subtype IS NULL")
			}

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

			// Create account
			acc := models.Account{
				UserID:        u.ID,
				Name:          s.Name,
				AccountTypeID: uint(at.ID),
				Currency:      strings.ToUpper(s.Currency),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if err := db.WithContext(ctx).Create(&acc).Error; err != nil {
				return err
			}

			// Create balance with start balance
			bal := models.Balance{
				AccountID:    acc.ID,
				AsOf:         time.Now().Truncate(24 * time.Hour),
				StartBalance: s.StartBalance,
				Currency:     strings.ToUpper(s.Currency),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
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
