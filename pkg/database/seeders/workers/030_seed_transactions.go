package workers

import (
	"context"
	"math"
	"math/rand"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	accService := services.NewAccountService(nil, nil, accRepo, txnRepo)

	var incCats, expCats []models.Category
	_ = db.WithContext(ctx).Where("classification = ?", "income").Find(&incCats).Error
	_ = db.WithContext(ctx).Where("classification = ?", "expense").Find(&expCats).Error
	var uncategorized models.Category
	_ = db.WithContext(ctx).Where("classification = ?", "uncategorized").First(&uncategorized).Error

	pick := func(cs []models.Category) *int64 {
		if len(cs) == 0 {
			if uncategorized.ID == 0 {
				return nil
			}
			return &uncategorized.ID
		}
		id := cs[rng.Intn(len(cs))].ID
		return &id
	}

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

		const yearsSpan = 5
		const txnsPerYear = 100
		totalTxns := yearsSpan * txnsPerYear
		perAcc := int(math.Max(1, float64(totalTxns)/float64(len(accounts))))

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

			openDays := int(today.Sub(acc.OpenedAt.UTC().Truncate(24*time.Hour)).Hours() / 24)
			maxBack := int(math.Min(float64(openDays), float64(365*yearsSpan+7)))
			if maxBack < 1 {
				maxBack = 1
			}

			class := acc.AccountType.Classification
			var incomeProb float64
			var dampWrong decimal.Decimal
			var boostRight decimal.Decimal

			switch class {
			case "asset":
				incomeProb = 0.65
				dampWrong = decimal.NewFromFloat(0.5)
				boostRight = decimal.NewFromFloat(1.0)
			case "liability":
				incomeProb = 0.30
				dampWrong = decimal.NewFromFloat(0.5)
				boostRight = decimal.NewFromFloat(1.15)
			default:
				incomeProb = 0.50
				dampWrong = decimal.NewFromFloat(0.9)
				boostRight = decimal.NewFromFloat(1.0)
			}

			for i := 0; i < perAcc; i++ {
				daysAgo := rng.Intn(maxBack)
				date := today.AddDate(0, 0, -daysAgo)

				ttype := "expense"
				if rng.Float64() < incomeProb {
					ttype = "income"
				}

				amt := decimal.NewFromFloat(10 + rng.Float64()*5000).Round(2)
				isRightDir := (class == "asset" && ttype == "income") || (class == "liability" && ttype == "expense")
				if isRightDir {
					amt = amt.Mul(boostRight).Round(2)
				} else {
					amt = amt.Mul(dampWrong).Round(2)
				}
				if amt.LessThanOrEqual(decimal.Zero) {
					amt = decimal.NewFromInt(1)
				}

				// prevent asset accounts from going negative
				if class == "asset" && ttype == "expense" {
					if currBal.LessThanOrEqual(decimal.Zero) {
						ttype = "income" // force inflow if nothing left
					} else if amt.GreaterThan(currBal) {
						amt = currBal // shrink expense to available balance
					}
				}
				// liabilities should not go positive
				if class == "liability" && ttype == "income" {
					next := currBal.Add(amt)
					if next.GreaterThan(decimal.Zero) {
						capAmt := currBal.Abs().Mul(decimal.NewFromFloat(0.8)).Round(2)
						if capAmt.LessThan(decimal.NewFromInt(1)) {
							ttype = "expense"
						} else {
							amt = capAmt
						}
					}
				}

				var catID *int64
				if ttype == "income" {
					catID = pick(incCats)
				} else {
					catID = pick(expCats)
				}

				t := models.Transaction{
					UserID:          u.ID,
					AccountID:       acc.ID,
					TransactionType: ttype,
					CategoryID:      catID,
					Amount:          amt,
					Currency:        acc.Currency,
					TxnDate:         date,
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
