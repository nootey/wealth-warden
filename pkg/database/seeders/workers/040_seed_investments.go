package workers

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func SeedInvestments(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Asset creation and trades need live prices and exchange rates
	priceClient, err := finance.NewPriceFetchClient(cfg.FinanceAPIBaseURL)
	if err != nil {
		return fmt.Errorf("investment seeder requires the finance API: %w", err)
	}

	invRepo := repositories.NewInvestmentRepository(db)
	accRepo := repositories.NewAccountRepository(db)
	txnRepo := repositories.NewTransactionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	loggingRepo := repositories.NewLoggingRepository(db)
	invService := services.NewInvestmentService(zap.NewNop(), invRepo, accRepo, txnRepo, settingsRepo, loggingRepo, queue.NoopDispatcher{}, priceClient)

	var users []models.User
	if err := db.WithContext(ctx).Find(&users).Error; err != nil {
		return err
	}

	type assetSeed struct {
		AccountName    string
		InvestmentType models.InvestmentType
		Name           string
		Ticker         string
		Currency       string
		Trades         int
	}

	seeds := []assetSeed{
		{AccountName: "Crypto Exchange", InvestmentType: models.InvestmentCrypto, Name: "Bitcoin", Ticker: "BTC-USD", Currency: "USD", Trades: 13},
		{AccountName: "Investment account", InvestmentType: models.InvestmentETF, Name: "iShares Core MSCI World", Ticker: "IWDA.AS", Currency: "EUR", Trades: 12},
	}

	for _, u := range users {
		for _, s := range seeds {
			var acc models.Account
			err := db.WithContext(ctx).
				Where("user_id = ? AND name = ?", u.ID, s.AccountName).
				First(&acc).Error
			if err == gorm.ErrRecordNotFound {
				continue
			}
			if err != nil {
				return err
			}

			assetID, err := invService.InsertAsset(ctx, u.ID, &models.InvestmentAssetReq{
				AccountID:      acc.ID,
				InvestmentType: s.InvestmentType,
				Name:           s.Name,
				Ticker:         s.Ticker,
				Quantity:       decimal.Zero,
				Currency:       s.Currency,
			})
			if err != nil {
				return err
			}

			if err := seedTradesForAsset(ctx, rng, today, invRepo, accRepo, invService, u.ID, assetID, s.Trades, s.Currency); err != nil {
				return err
			}
		}
	}

	return nil
}

func seedTradesForAsset(
	ctx context.Context,
	rng *rand.Rand,
	today time.Time,
	invRepo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	invService *services.InvestmentService,
	userID, assetID int64,
	numTrades int,
	tradeCurrency string,
) error {
	// Spread trades over the last year, oldest first, so sells always
	// happen against quantity accumulated by earlier buys
	const daysSpan = 360
	step := daysSpan / numTrades

	soldOnce := false

	for i := 0; i < numTrades; i++ {
		daysAgo := daysSpan - i*step - rng.Intn(step)
		date := today.AddDate(0, 0, -daysAgo)

		// Refetch each iteration — quantity and current price move with every trade
		asset, err := invRepo.FindInvestmentAssetByID(ctx, nil, assetID, userID)
		if err != nil {
			return err
		}

		basePrice := decimal.NewFromInt(100)
		if asset.CurrentPrice != nil && asset.CurrentPrice.IsPositive() {
			basePrice = *asset.CurrentPrice
		}
		price := basePrice.Mul(decimal.NewFromFloat(0.8 + rng.Float64()*0.4)).Round(4)

		isCrypto := asset.InvestmentType == models.InvestmentCrypto

		canSell := asset.Quantity.IsPositive() && i > 1
		isSell := canSell && (rng.Float64() < 0.3 || (i == numTrades-1 && !soldOnce))

		var quantity, fee decimal.Decimal

		if isSell {
			fraction := decimal.NewFromFloat(0.1 + rng.Float64()*0.3)
			quantity = asset.Quantity.Mul(fraction)
		} else {
			// Buy: cap spend so the affordability check never fails
			bal, err := accRepo.FindLatestBalance(ctx, nil, asset.AccountID, userID)
			if err != nil {
				return err
			}
			// Same rate (and cache entry) the trade insert will use
			rate, err := invService.GetExchangeRate(ctx, tradeCurrency, asset.Account.Currency, &date)
			if err != nil {
				return err
			}
			maxSpend := bal.EndBalance.Mul(decimal.NewFromFloat(0.3)).Div(rate)

			spend := decimal.NewFromFloat(200 + rng.Float64()*1800)
			if spend.GreaterThan(maxSpend) {
				spend = maxSpend
			}
			if spend.LessThan(decimal.NewFromInt(10)) {
				continue // not enough cash left for a meaningful buy
			}
			quantity = spend.Div(price)
		}

		if isCrypto {
			quantity = quantity.RoundDown(6)
			fee = quantity.Mul(decimal.NewFromFloat(0.001)).RoundDown(8) // fee in coin units
		} else {
			quantity = quantity.RoundDown(0) // whole shares
			fee = decimal.NewFromFloat(1 + rng.Float64()*2).Round(2)
		}
		if !quantity.IsPositive() {
			continue
		}

		req := &models.InvestmentTradeReq{
			AssetID:      assetID,
			TradeType:    models.InvestmentBuy,
			TxnDate:      date,
			Quantity:     quantity,
			PricePerUnit: price,
			Currency:     tradeCurrency,
			Fee:          &fee,
		}
		if isSell {
			req.TradeType = models.InvestmentSell
		}

		if _, err := invService.InsertInvestmentTrade(ctx, userID, req); err != nil {
			return err
		}

		if isSell {
			soldOnce = true
		}
	}

	return nil
}
