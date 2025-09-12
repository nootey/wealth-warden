package workers

import (
	"context"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/rand"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
)

func SeedTransactions(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	var users []models.User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		return err
	}

	accRepo := repositories.NewAccountRepository(db)
	txnRepo := repositories.NewTransactionRepository(db)

	accService := services.NewAccountService(
		nil,
		nil,
		accRepo,
		txnRepo,
	)

	for _, u := range users {

		var accounts []models.Account
		if err := db.WithContext(ctx).
			Preload("AccountType").
			Where("accounts.user_id = ?", u.ID).
			Find(&accounts).Error; err != nil {
			return err
		}
		if len(accounts) == 0 {
			logger.Info("no accounts for user", zap.Int64("user_id", u.ID))
			continue
		}

		totalTxns := 250
		perAcc := totalTxns / len(accounts)

		for _, acc := range accounts {
			// fetch latest balance row for starting balance
			var bal models.Balance
			if err := db.WithContext(ctx).
				Where("account_id = ?", acc.ID).
				Order("as_of DESC").
				First(&bal).Error; err != nil {
				return err
			}

			currBal := bal.StartBalance

			for i := 0; i < perAcc; i++ {
				daysAgo := rng.Intn(365)
				date := today.AddDate(0, 0, -daysAgo)

				var ttype string
				if rng.Float64() < 0.4 {
					ttype = "income"
				} else {
					ttype = "expense"
				}

				amt := decimal.NewFromFloat(10 + rng.Float64()*1000).Round(2)

				// prevent asset accounts from going negative
				if acc.AccountType.Classification == "asset" && ttype == "expense" {
					if currBal.LessThanOrEqual(decimal.Zero) {
						// no money left, force income txn instead
						ttype = "income"
					} else if amt.GreaterThan(currBal) {
						// shrink expense to available balance
						amt = currBal
					}
				}

				desc := "Random " + ttype

				var category models.Category
				_ = db.Model(&models.Category{}).
					Where("classification = ?", "uncategorized").
					Order("name").
					First(&category)

				t := models.Transaction{
					UserID:          u.ID,
					AccountID:       acc.ID,
					TransactionType: ttype,
					CategoryID:      &category.ID,
					Amount:          amt,
					Currency:        acc.Currency,
					TxnDate:         date,
					Description:     &desc,
					IsAdjustment:    false,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				if err := db.WithContext(ctx).Create(&t).Error; err != nil {
					return err
				}

				if err := accService.UpdateAccountCashBalance(db, &acc, t.TxnDate, ttype, amt); err != nil {
					return err
				}

				// adjust running balance
				if ttype == "income" {
					currBal = currBal.Add(amt)
				} else {
					currBal = currBal.Sub(amt)
				}
			}

			logger.Info("seeded transactions",
				zap.Int64("user_id", u.ID),
				zap.Int("count", perAcc),
				zap.String("account", acc.Name),
				zap.String("ending_balance", currBal.StringFixed(2)),
			)
		}
	}

	return nil
}
