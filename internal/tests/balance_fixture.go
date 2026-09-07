package tests

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// The bulk seeder draws from a clock-seeded rand, so it can never prove that a
// refactor left the daily balances untouched. This fixture is the deterministic
// stand-in: two users, three accounts each, fixed transactions and fixed trades.
// Every date comes from the caller's anchor, so two runs over the same anchor
// must produce byte-identical daily balances.

// Ids follow the insert order in 010_seed_account_types.go, on a fresh database.
const (
	fixtureCheckingTypeID   = 1  // cash / checking
	fixtureBrokerageTypeID  = 5  // investment / brokerage
	fixtureCreditCardTypeID = 18 // credit_card / credit, classified as a liability
)

// DailyBalanceRow is one row of the derived daily balance table.
type DailyBalanceRow struct {
	AccountID  int64
	AsOf       time.Time
	EndBalance decimal.Decimal
}

type fixtureUser struct {
	email     string
	checking  string
	broker    string
	card      string
	opening   int64
	brokerage int64
	cardDebt  int64
	income    int64
	expense1  int64
	expense2  int64
	charge    int64
	repayment int64
	buyQty    int64
	sellQty   int64
}

var fixtureUsers = []fixtureUser{
	{
		email:    "", // the root user from the basic seed
		checking: "Fixture Checking A", broker: "Fixture Brokerage A", card: "Fixture Card A",
		opening: 1000, brokerage: 5000, cardDebt: 500,
		income: 250, expense1: 80, expense2: 40,
		charge: 150, repayment: 200,
		buyQty: 10, sellQty: 4,
	},
	{
		email:    "fixture-b@local.test",
		checking: "Fixture Checking B", broker: "Fixture Brokerage B", card: "Fixture Card B",
		opening: 2500, brokerage: 8000, cardDebt: 1200,
		income: 400, expense1: 125, expense2: 60,
		charge: 75, repayment: 310,
		buyQty: 25, sellQty: 9,
	},
}

// EnsureFixtureUser inserts a member user and its settings row if absent, then
// returns its id. Users survive the test truncate, so the id is stable.
func EnsureFixtureUser(ctx context.Context, db *gorm.DB, email string) (int64, error) {
	var roleID int64
	if err := db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&roleID).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch member role: %w", err)
	}
	if roleID == 0 {
		return 0, fmt.Errorf("global role 'member' does not exist")
	}

	now := time.Now().UTC()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO users (email, password, display_name, role_id, email_confirmed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (email) DO NOTHING
	`, email, "x", "Fixture", roleID, now, now, now).Error; err != nil {
		return 0, fmt.Errorf("failed to insert fixture user: %w", err)
	}

	var userID int64
	if err := db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID).Error; err != nil {
		return 0, fmt.Errorf("failed to read fixture user id: %w", err)
	}

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO settings_user (user_id, theme, accent, language, timezone, created_at, updated_at)
		VALUES (?, 'system', NULL, 'en', 'UTC', ?, ?)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, now, now).Error; err != nil {
		return 0, fmt.Errorf("failed to insert fixture user settings: %w", err)
	}

	return userID, nil
}

// SeedBalanceFixture writes the whole fixture. today anchors every date, so the
// caller must pass the same value to both runs of a diff.
func SeedBalanceFixture(ctx context.Context, app *bootstrap.ServiceContainer, db *gorm.DB, today time.Time) ([]int64, error) {
	opened := today.AddDate(0, 0, -30)
	userIDs := make([]int64, 0, len(fixtureUsers))

	for _, u := range fixtureUsers {
		userID := int64(1)
		if u.email != "" {
			id, err := EnsureFixtureUser(ctx, db, u.email)
			if err != nil {
				return nil, err
			}
			userID = id
		}
		userIDs = append(userIDs, userID)

		checkingID, err := insertFixtureAccount(ctx, app, userID, u.checking, fixtureCheckingTypeID, u.opening, opened)
		if err != nil {
			return nil, err
		}
		brokerID, err := insertFixtureAccount(ctx, app, userID, u.broker, fixtureBrokerageTypeID, u.brokerage, opened)
		if err != nil {
			return nil, err
		}

		// InsertAccount negates the opening amount for a liability, so this account
		// starts at -cardDebt. It is here to keep that sign path in the diff.
		cardID, err := insertFixtureAccount(ctx, app, userID, u.card, fixtureCreditCardTypeID, u.cardDebt, opened)
		if err != nil {
			return nil, err
		}

		txns := []struct {
			account int64
			kind    string
			amount  int64
			day     int
		}{
			{checkingID, "income", u.income, -25},
			{checkingID, "expense", u.expense1, -20},
			{checkingID, "expense", u.expense2, -10},
			{cardID, "expense", u.charge, -18},
			{cardID, "income", u.repayment, -8},
		}
		for _, t := range txns {
			if _, err := app.TransactionService.InsertTransaction(ctx, userID, &models.TransactionReq{
				AccountID: t.account,
				Direction: t.kind,
				Amount:    decimal.NewFromInt(t.amount),
				TxnDate:   today.AddDate(0, 0, t.day),
			}); err != nil {
				return nil, fmt.Errorf("failed to insert fixture transaction: %w", err)
			}
		}

		assetID, err := app.InvestmentService.InsertAsset(ctx, userID, &models.InvestmentAssetReq{
			AccountID:      brokerID,
			InvestmentType: models.InvestmentETF,
			Name:           "iShares Core MSCI World",
			Ticker:         "IWDA.AS",
			Quantity:       decimal.Zero,
			Currency:       "EUR",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to insert fixture asset: %w", err)
		}

		fee := decimal.NewFromInt(2)
		trades := []struct {
			kind models.TradeType
			qty  int64
			unit int64
			day  int
		}{
			{models.InvestmentBuy, u.buyQty, 100, -22},
			{models.InvestmentSell, u.sellQty, 110, -12},
		}
		for _, t := range trades {
			if _, err := app.InvestmentService.InsertInvestmentTrade(ctx, userID, &models.InvestmentTradeReq{
				AssetID:      assetID,
				TradeType:    t.kind,
				TxnDate:      today.AddDate(0, 0, t.day),
				Quantity:     decimal.NewFromInt(t.qty),
				PricePerUnit: decimal.NewFromInt(t.unit),
				Currency:     "EUR",
				Fee:          &fee,
			}); err != nil {
				return nil, fmt.Errorf("failed to insert fixture trade: %w", err)
			}
		}
	}

	return userIDs, nil
}

func insertFixtureAccount(ctx context.Context, app *bootstrap.ServiceContainer, userID int64, name string, typeID, opening int64, opened time.Time) (int64, error) {
	balance := decimal.NewFromInt(opening)
	id, err := app.AccountService.InsertAccount(ctx, userID, &models.AccountReq{
		Name:          name,
		AccountTypeID: typeID,
		Balance:       &balance,
		OpenedAt:      opened,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to insert fixture account %q: %w", name, err)
	}
	return id, nil
}

// DumpDailyBalances reads the whole derived daily balance table in a stable order.
func DumpDailyBalances(ctx context.Context, db *gorm.DB) ([]DailyBalanceRow, error) {
	var rows []DailyBalanceRow
	err := db.WithContext(ctx).
		Table("account_daily_snapshots").
		Select("account_id, as_of, end_balance").
		Order("account_id ASC, as_of ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to dump daily balances: %w", err)
	}
	return rows, nil
}

// DiffDailyBalances returns "" when the two dumps agree to 4 decimal places, and
// a description of the first difference otherwise.
func DiffDailyBalances(want, got []DailyBalanceRow) string {
	n := len(want)
	if len(got) < n {
		n = len(got)
	}

	for i := 0; i < n; i++ {
		w, g := want[i], got[i]
		if w.AccountID != g.AccountID || !w.AsOf.Equal(g.AsOf) {
			return fmt.Sprintf("row %d: want account %d on %s, got account %d on %s",
				i, w.AccountID, w.AsOf.Format(time.DateOnly), g.AccountID, g.AsOf.Format(time.DateOnly))
		}
		if !w.EndBalance.Round(4).Equal(g.EndBalance.Round(4)) {
			return fmt.Sprintf("row %d: account %d on %s: want end_balance %s, got %s",
				i, w.AccountID, w.AsOf.Format(time.DateOnly),
				w.EndBalance.StringFixed(4), g.EndBalance.StringFixed(4))
		}
	}

	if len(want) != len(got) {
		return fmt.Sprintf("row count: want %d, got %d (first %d rows match)", len(want), len(got), n)
	}
	return ""
}
