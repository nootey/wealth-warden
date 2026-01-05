package jobqueue

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type RecalculateAccountBalancesJob struct {
	Repo             repositories.InvestmentRepositoryInterface
	AccRepo          repositories.AccountRepositoryInterface
	PriceFetchClient finance.PriceFetcher
	AccountID        int64
	UserID           int64
	Currency         string
	FromDate         time.Time
	ToDate           time.Time
}

func (j *RecalculateAccountBalancesJob) Process(ctx context.Context) error {
	tx, err := j.Repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	balances, err := j.AccRepo.GetBalancesInRange(ctx, tx, j.AccountID, j.FromDate, j.ToDate)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get balances: %w", err)
	}

	for _, balance := range balances {
		if err := j.recalculateBalanceForDate(ctx, tx, balance.AsOf); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (j *RecalculateAccountBalancesJob) recalculateBalanceForDate(ctx context.Context, tx *gorm.DB, asOf time.Time) error {
	assets, err := j.Repo.FindAssetsByAccountID(ctx, tx, j.AccountID, j.UserID)
	if err != nil {
		return err
	}

	if len(assets) == 0 {
		return nil
	}

	var totalPnLAsOf decimal.Decimal

	for _, asset := range assets {
		qty, spent, err := j.Repo.GetInvestmentTotalsUpToDate(ctx, tx, asset.ID, asOf)
		if err != nil || qty.IsZero() {
			continue
		}

		price, err := j.fetchPriceForDate(ctx, asset, asOf)
		if err != nil {
			continue
		}

		currentValueAtDate := qty.Mul(price)
		pnlAtDate := currentValueAtDate.Sub(spent)

		pnlInAccountCurrency := pnlAtDate
		if asset.Currency != j.Currency {
			exchangeRate := j.getExchangeRate(ctx, asset.Currency, j.Currency, &asOf)
			pnlInAccountCurrency = pnlAtDate.Mul(exchangeRate)
		}

		totalPnLAsOf = totalPnLAsOf.Add(pnlInAccountCurrency)
	}

	var previousPnL decimal.Decimal
	cumulativePnL, err := j.AccRepo.GetCumulativeNonCashPnLBeforeDate(ctx, tx, j.AccountID, asOf)
	if err == nil {
		previousPnL = cumulativePnL
	}

	delta := totalPnLAsOf.Sub(previousPnL)

	if err := j.AccRepo.EnsureDailyBalanceRow(ctx, tx, j.AccountID, asOf, j.Currency); err != nil {
		return err
	}

	if err := j.AccRepo.SetDailyBalance(ctx, tx, j.AccountID, asOf, "non_cash_inflows", decimal.Zero); err != nil {
		return err
	}
	if err := j.AccRepo.SetDailyBalance(ctx, tx, j.AccountID, asOf, "non_cash_outflows", decimal.Zero); err != nil {
		return err
	}

	if delta.GreaterThan(decimal.Zero) {
		if err := j.AccRepo.AddToDailyBalance(ctx, tx, j.AccountID, asOf, "non_cash_inflows", delta); err != nil {
			return err
		}
	} else if delta.LessThan(decimal.Zero) {
		if err := j.AccRepo.AddToDailyBalance(ctx, tx, j.AccountID, asOf, "non_cash_outflows", delta.Abs()); err != nil {
			return err
		}
	}

	if err := j.AccRepo.FrontfillBalances(ctx, tx, j.AccountID, j.Currency, asOf); err != nil {
		return err
	}

	return j.AccRepo.UpsertSnapshotsFromBalances(
		ctx, tx, j.UserID, j.AccountID, j.Currency,
		asOf.UTC().Truncate(24*time.Hour),
		time.Now().UTC().Truncate(24*time.Hour),
	)
}

func (j *RecalculateAccountBalancesJob) fetchPriceForDate(ctx context.Context, asset models.InvestmentAsset, asOf time.Time) (decimal.Decimal, error) {
	if j.PriceFetchClient == nil {
		return decimal.Zero, fmt.Errorf("price fetch client not initialized")
	}
	priceData, err := j.PriceFetchClient.GetAssetPriceOnDate(ctx, asset.Ticker, asset.InvestmentType, asOf)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromFloat(priceData.Price), nil
}

func (j *RecalculateAccountBalancesJob) getExchangeRate(ctx context.Context, fromCurrency, toCurrency string, date *time.Time) decimal.Decimal {
	if j.PriceFetchClient != nil {
		var rate float64
		var err error
		if date != nil {
			rate, err = j.PriceFetchClient.GetExchangeRateOnDate(ctx, fromCurrency, toCurrency, *date)
		} else {
			rate, err = j.PriceFetchClient.GetExchangeRate(ctx, fromCurrency, toCurrency)
		}
		if err == nil {
			return decimal.NewFromFloat(rate)
		}
	}
	return decimal.NewFromFloat(1.0)
}
