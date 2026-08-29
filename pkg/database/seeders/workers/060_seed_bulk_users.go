package workers

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Writes rows directly and never calls the account services: those rebuild the
// whole day series per transaction, which does not survive contact with
// hundreds of thousands of rows.
const (
	bulkUsersPerTx  = 100
	bulkInsertBatch = 1000

	bulkNamePrefix    = "ww_user_"
	bulkNameSuffixLen = 8
	bulkNameCharset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Investments span one year: prices, the buy trades and the snapshot
	// series all sit inside this window.
	bulkInvHistoryDays    = 365
	bulkInvOpenBufferDays = 30 // the two accounts open this many days before the first trade
	bulkEquityPerUser     = 7  // picks from bulkNonCryptoPool, land in the EUR investment account
	bulkCryptoPerUser     = 3  // picks from bulkCryptoPool, land in the USD crypto account

	bulkInvAccountName    = "Investment account"
	bulkCryptoAccountName = "Crypto Exchange"

	// A small pause between the one-off price fetches, so a 10k-user seed does
	// not look like a scraper to Yahoo.
	bulkPriceFetchDelay = 400 * time.Millisecond
)

var (
	bulkInvOpeningCash    = decimal.NewFromInt(15000)  // EUR, investment account
	bulkCryptoOpeningCash = decimal.NewFromInt(8000)   // USD, crypto account
	bulkInvSpendFraction  = decimal.NewFromFloat(0.6)  // of opening cash, spread across the picks
	bulkInvSpendCeiling   = decimal.NewFromFloat(0.95) // never let buys pull an account below this share of opening cash
)

type bulkAssetSeed struct {
	Ticker string
	Name   string
	Type   models.InvestmentType
}

// Real Yahoo symbols only: the backoffice price jobs re-fetch these by ticker,
// so an invented symbol would fail every backfill run. EUR-denominated names
// first, then USD-denominated.
var bulkNonCryptoPool = []bulkAssetSeed{
	{Ticker: "VWCE.DE", Name: "Vanguard FTSE All-World UCITS ETF", Type: models.InvestmentETF},
	{Ticker: "SXR8.DE", Name: "iShares Core S&P 500 UCITS ETF", Type: models.InvestmentETF},
	{Ticker: "MEUD.PA", Name: "Amundi Stoxx Europe 600 UCITS ETF", Type: models.InvestmentETF},
	{Ticker: "ASML.AS", Name: "ASML Holding NV", Type: models.InvestmentStock},
	{Ticker: "MC.PA", Name: "LVMH Moet Hennessy Louis Vuitton", Type: models.InvestmentStock},
	{Ticker: "SAP.DE", Name: "SAP SE", Type: models.InvestmentStock},
	{Ticker: "SIE.DE", Name: "Siemens AG", Type: models.InvestmentStock},
	{Ticker: "AIR.PA", Name: "Airbus SE", Type: models.InvestmentStock},
	{Ticker: "ALV.DE", Name: "Allianz SE", Type: models.InvestmentStock},
	{Ticker: "OR.PA", Name: "L'Oreal SA", Type: models.InvestmentStock},
	{Ticker: "AAPL", Name: "Apple Inc.", Type: models.InvestmentStock},
	{Ticker: "MSFT", Name: "Microsoft Corporation", Type: models.InvestmentStock},
	{Ticker: "NVDA", Name: "NVIDIA Corporation", Type: models.InvestmentStock},
	{Ticker: "AMZN", Name: "Amazon.com Inc.", Type: models.InvestmentStock},
	{Ticker: "GOOGL", Name: "Alphabet Inc.", Type: models.InvestmentStock},
	{Ticker: "TSLA", Name: "Tesla Inc.", Type: models.InvestmentStock},
	{Ticker: "VOO", Name: "Vanguard S&P 500 ETF", Type: models.InvestmentETF},
	{Ticker: "SPY", Name: "SPDR S&P 500 ETF Trust", Type: models.InvestmentETF},
}

var bulkCryptoPool = []bulkAssetSeed{
	{Ticker: "BTC-USD", Name: "Bitcoin", Type: models.InvestmentCrypto},
	{Ticker: "ETH-USD", Name: "Ethereum", Type: models.InvestmentCrypto},
	{Ticker: "SOL-USD", Name: "Solana", Type: models.InvestmentCrypto},
	{Ticker: "XRP-USD", Name: "XRP", Type: models.InvestmentCrypto},
	{Ticker: "ADA-USD", Name: "Cardano", Type: models.InvestmentCrypto},
	{Ticker: "DOGE-USD", Name: "Dogecoin", Type: models.InvestmentCrypto},
	{Ticker: "LTC-USD", Name: "Litecoin", Type: models.InvestmentCrypto},
}

// bulkAssetPrices is one ticker's history, ascending by date, ready for a
// point-in-time lookup during the user loop.
type bulkAssetPrices struct {
	currency string
	dates    []time.Time
	prices   []decimal.Decimal
}

func (p *bulkAssetPrices) priceOn(date time.Time) decimal.Decimal {
	i := sort.Search(len(p.dates), func(i int) bool { return p.dates[i].After(date) })
	if i == 0 {
		return p.prices[0]
	}
	return p.prices[i-1]
}

// bulkFX holds the USD->EUR rate for every calendar day in the window,
// forward-filled so weekend trade and snapshot dates still resolve. The stored
// direction matches exchange_rate_history: rate = units of `to` per 1 of `from`.
type bulkFX struct {
	usdToEUR map[string]decimal.Decimal
}

func (f *bulkFX) rate(from, to string, date time.Time) decimal.Decimal {
	if from == to {
		return decimal.NewFromInt(1)
	}
	r, ok := f.usdToEUR[date.UTC().Format("2006-01-02")]
	if !ok || !r.IsPositive() {
		return decimal.NewFromInt(1)
	}
	switch {
	case from == "USD" && to == "EUR":
		return r
	case from == "EUR" && to == "USD":
		return decimal.NewFromInt(1).Div(r)
	default:
		return decimal.NewFromInt(1)
	}
}

// bulkPendingTrade carries a buy from the pricing pass to the batch insert,
// where the freshly generated asset ID is filled in.
type bulkPendingTrade struct {
	assetIdx   int
	date       time.Time
	price      decimal.Decimal
	qty        decimal.Decimal
	fee        decimal.Decimal
	valueAtBuy decimal.Decimal
	currency   string
	rateToUSD  decimal.Decimal
}

type bulkAccountSeed struct {
	Name         string
	Type         string
	Subtype      string
	StartBalance decimal.Decimal
}

var bulkAccountSeeds = []bulkAccountSeed{
	{Name: "Checking account", Type: "cash", Subtype: "checking", StartBalance: decimal.NewFromInt(2500)},
	{Name: "Savings account", Type: "cash", Subtype: "savings", StartBalance: decimal.NewFromInt(12000)},
	{Name: "Credit card", Type: "credit_card", Subtype: "credit", StartBalance: decimal.NewFromInt(-1500)},
}

// Investments are handled apart from the cash side: the 25-ticker price pool and
// the USD/EUR rate history are fetched once up front, then each user is just
// handed 10 holdings by direct insert inside seedBulkChunk.
func SeedBulkUsers(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	b := cfg.Seed.Bulk
	if b.Users <= 0 {
		fmt.Println("bulk user count is zero, skipping ...")
		return nil
	}
	if b.Years <= 0 {
		b.Years = 2
	}
	if b.TxnsPerUser <= 0 {
		b.TxnsPerUser = 500
	}
	if b.EmailDomain == "" {
		b.EmailDomain = "local.seed"
	}
	if b.Password == "" {
		b.Password = "password"
	}

	// Bcrypt costs ~60 ms, so one shared hash instead of one per user
	hashedPassword, err := utils.HashAndSaltPassword(b.Password)
	if err != nil {
		return fmt.Errorf("failed to hash bulk password: %w", err)
	}

	var roleID int64
	if err := db.WithContext(ctx).Raw(`SELECT id FROM roles WHERE name = ?`, "member").Scan(&roleID).Error; err != nil {
		return fmt.Errorf("failed to fetch global role member: %w", err)
	}
	if roleID == 0 {
		return fmt.Errorf("global role 'member' does not exist, please seed roles first")
	}

	accountTypeIDs, err := bulkAccountTypeIDs(ctx, db)
	if err != nil {
		return err
	}

	incCats, expCats, err := bulkCategoryIDs(ctx, db)
	if err != nil {
		return err
	}
	if len(incCats) == 0 || len(expCats) == 0 {
		return fmt.Errorf("no income or expense categories found, please seed categories first")
	}

	invTypeID, cryptoTypeID, err := bulkInvestmentAccountTypeIDs(ctx, db)
	if err != nil {
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Seed investments first: fetch the whole price pool + rate history once, so
	// the per-user loop stays plain inserts.
	pricePool, fx, err := seedBulkAssetPricePool(ctx, db, cfg, today)
	if err != nil {
		return fmt.Errorf("failed to seed asset price pool: %w", err)
	}

	var taken []string
	if err := db.WithContext(ctx).
		Raw(`SELECT email FROM users WHERE email LIKE ?`, "bulk%@"+b.EmailDomain).
		Scan(&taken).Error; err != nil {
		return fmt.Errorf("failed to read existing bulk users: %w", err)
	}
	skip := make(map[string]bool, len(taken))
	for _, e := range taken {
		skip[e] = true
	}

	var pending []int
	for i := 1; i <= b.Users; i++ {
		if !skip[bulkEmail(i, b.EmailDomain)] {
			pending = append(pending, i)
		}
	}
	if len(pending) == 0 {
		fmt.Println("all bulk users already exist, skipping ...")
		return nil
	}

	rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	accRepo := repositories.NewAccountRepository(db)
	txnRepo := repositories.NewTransactionRepository(db)

	for start := 0; start < len(pending); start += bulkUsersPerTx {
		end := min(start+bulkUsersPerTx, len(pending))
		chunk := pending[start:end]

		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return seedBulkChunk(ctx, tx, accRepo, txnRepo, rng, today, b, chunk,
				hashedPassword, roleID, accountTypeIDs, incCats, expCats,
				pricePool, fx, invTypeID, cryptoTypeID)
		})
		if err != nil {
			return fmt.Errorf("bulk chunk starting at user %d failed: %w", chunk[0], err)
		}

		fmt.Printf("bulk users seeded: %d/%d\n", end, len(pending))
	}

	// One set-based pass over every bulk user's investment/crypto snapshots,
	// bounded to the investment window. Runs after the chunks commit.
	if err := recomputeBulkInvestmentSnapshots(ctx, db, accRepo, b.EmailDomain, today); err != nil {
		return err
	}

	fmt.Printf("bulk credentials: %s .. %s, password %q\n",
		bulkEmail(pending[0], b.EmailDomain),
		bulkEmail(pending[len(pending)-1], b.EmailDomain),
		b.Password)

	return nil
}

func bulkEmail(i int, domain string) string {
	return fmt.Sprintf("bulk%05d@%s", i, domain)
}

func bulkAccountTypeIDs(ctx context.Context, db *gorm.DB) (map[string]int64, error) {
	ids := make(map[string]int64, len(bulkAccountSeeds))
	for _, s := range bulkAccountSeeds {
		var at models.AccountType
		if err := db.WithContext(ctx).
			Where("type = ? AND sub_type = ?", s.Type, s.Subtype).
			First(&at).Error; err != nil {
			return nil, fmt.Errorf("account type %s/%s not found: %w", s.Type, s.Subtype, err)
		}
		ids[s.Name] = at.ID
	}
	return ids, nil
}

func bulkCategoryIDs(ctx context.Context, db *gorm.DB) (inc, exp []int64, err error) {
	q := func(classification string) ([]int64, error) {
		var ids []int64
		e := db.WithContext(ctx).
			Model(&models.Category{}).
			Where("classification = ? AND user_id IS NULL", classification).
			Pluck("id", &ids).Error
		return ids, e
	}
	if inc, err = q("income"); err != nil {
		return nil, nil, err
	}
	exp, err = q("expense")
	return inc, exp, err
}

func seedBulkChunk(
	ctx context.Context,
	tx *gorm.DB,
	accRepo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	rng *rand.Rand,
	today time.Time,
	b config.BulkSeedConfig,
	indexes []int,
	hashedPassword string,
	roleID int64,
	accountTypeIDs map[string]int64,
	incCats, expCats []int64,
	pricePool map[string]*bulkAssetPrices,
	fx *bulkFX,
	invTypeID, cryptoTypeID int64,
) error {
	now := time.Now().UTC()

	users := make([]models.User, 0, len(indexes))
	for _, i := range indexes {
		confirmed := now
		users = append(users, models.User{
			Email:             bulkEmail(i, b.EmailDomain),
			Password:          hashedPassword,
			DisplayName:       bulkDisplayName(rng),
			RoleID:            roleID,
			EmailConfirmed:    &confirmed,
			HasCompletedSetup: true,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	if err := tx.WithContext(ctx).CreateInBatches(&users, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert users: %w", err)
	}

	settings := make([]models.SettingsUser, 0, len(users))
	for _, u := range users {
		settings = append(settings, models.SettingsUser{
			UserID:                u.ID,
			Theme:                 "system",
			Language:              "en",
			Timezone:              "UTC",
			DefaultCurrency:       "EUR",
			DefaultSheetSeparator: ";",
			CreatedAt:             now,
			UpdatedAt:             now,
		})
	}
	if err := tx.WithContext(ctx).CreateInBatches(&settings, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert user settings: %w", err)
	}

	// The opening balance row anchors the frontfill, so it has to exist before
	// any cash delta lands on the account
	accounts := make([]models.Account, 0, len(users)*len(bulkAccountSeeds))
	openedAt := make([]time.Time, 0, cap(accounts))
	for _, u := range users {
		opened := today.AddDate(0, 0, -(b.Years*365 + rng.Intn(60)))
		for _, s := range bulkAccountSeeds {
			accounts = append(accounts, models.Account{
				UserID:            u.ID,
				Name:              s.Name,
				AccountTypeID:     accountTypeIDs[s.Name],
				Currency:          "EUR",
				BalanceProjection: "fixed",
				ExpectedBalance:   decimal.Zero,
				OpenedAt:          opened,
				UpdatedAt:         opened,
			})
			openedAt = append(openedAt, opened)
		}
	}
	if err := tx.WithContext(ctx).CreateInBatches(&accounts, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert accounts: %w", err)
	}

	balances := make([]models.Balance, 0, len(accounts))
	for i, acc := range accounts {
		balances = append(balances, models.Balance{
			AccountID:    acc.ID,
			AsOf:         openedAt[i],
			StartBalance: bulkAccountSeeds[i%len(bulkAccountSeeds)].StartBalance,
			Currency:     acc.Currency,
			CreatedAt:    openedAt[i],
			UpdatedAt:    openedAt[i],
		})
	}
	if err := tx.WithContext(ctx).CreateInBatches(&balances, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert opening balances: %w", err)
	}

	perAcc := max(1, b.TxnsPerUser/len(bulkAccountSeeds))
	txns := make([]models.Transaction, 0, len(accounts)*perAcc)
	deltas := make(map[int64][]models.DailyCashDelta, len(accounts))

	for i, acc := range accounts {
		seed := bulkAccountSeeds[i%len(bulkAccountSeeds)]
		accTxns, accDeltas := bulkTransactionsForAccount(
			rng, today, openedAt[i], acc, seed, perAcc, incCats, expCats)
		txns = append(txns, accTxns...)
		deltas[acc.ID] = accDeltas
	}

	if err := txnRepo.InsertTransactionsBatch(ctx, tx, txns, bulkInsertBatch); err != nil {
		return fmt.Errorf("failed to insert transactions: %w", err)
	}

	// Once per account, not once per transaction
	for i, acc := range accounts {
		if err := accRepo.UpsertDailyCashBatch(ctx, tx, acc.ID, acc.Currency, deltas[acc.ID]); err != nil {
			return fmt.Errorf("failed to write daily balances: %w", err)
		}
		if err := accRepo.FrontfillBalances(ctx, tx, acc.ID, acc.Currency, openedAt[i]); err != nil {
			return fmt.Errorf("failed to frontfill balances: %w", err)
		}
		if err := accRepo.UpsertSnapshotsFromBalances(ctx, tx, acc.UserID, acc.ID, acc.Currency, openedAt[i], today); err != nil {
			return fmt.Errorf("failed to build snapshots: %w", err)
		}
	}

	if err := seedBulkSavingGoals(ctx, tx, rng, today, accounts); err != nil {
		return err
	}

	return seedBulkInvestments(ctx, tx, accRepo, rng, today, users, pricePool, fx, invTypeID, cryptoTypeID)
}

func bulkDisplayName(rng *rand.Rand) string {
	b := make([]byte, bulkNameSuffixLen)
	for i := range b {
		b[i] = bulkNameCharset[rng.Intn(len(bulkNameCharset))]
	}
	return bulkNamePrefix + string(b)
}

func bulkTransactionsForAccount(
	rng *rand.Rand,
	today, opened time.Time,
	acc models.Account,
	seed bulkAccountSeed,
	count int,
	incCats, expCats []int64,
) ([]models.Transaction, []models.DailyCashDelta) {
	isLiability := seed.StartBalance.IsNegative()

	incomeProb := 0.62
	if isLiability {
		incomeProb = 0.35
	}

	maxBack := max(1, int(today.Sub(opened).Hours()/24))
	now := time.Now().UTC()

	// Drawn at random but walked in order, so currBal below tracks the chain
	// the database will actually build
	dates := make([]time.Time, count)
	for i := range dates {
		dates[i] = today.AddDate(0, 0, -rng.Intn(maxBack))
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	currBal := seed.StartBalance
	txns := make([]models.Transaction, 0, count)
	byDay := make(map[time.Time]*models.DailyCashDelta, count)
	deltas := make([]models.DailyCashDelta, 0, count)

	for _, date := range dates {
		ttype := "expense"
		if rng.Float64() < incomeProb {
			ttype = "income"
		}

		amt := decimal.NewFromFloat(10 + rng.Float64()*900).Round(2)

		// keep asset accounts solvent
		if !isLiability && ttype == "expense" {
			if !currBal.IsPositive() {
				ttype = "income"
			} else if amt.GreaterThan(currBal) {
				amt = currBal
			}
		}
		// keep liability accounts in the red
		if isLiability && ttype == "income" && currBal.Add(amt).IsPositive() {
			amt = currBal.Abs().Mul(decimal.NewFromFloat(0.8)).Round(2)
			if amt.LessThan(decimal.NewFromInt(1)) {
				ttype = "expense"
			}
		}
		if !amt.IsPositive() {
			continue
		}

		cats := expCats
		if ttype == "income" {
			cats = incCats
		}
		catID := cats[rng.Intn(len(cats))]

		txns = append(txns, models.Transaction{
			UserID:          acc.UserID,
			AccountID:       acc.ID,
			TransactionType: ttype,
			CategoryID:      &catID,
			Amount:          amt,
			Currency:        acc.Currency,
			TxnDate:         date,
			CreatedAt:       now,
			UpdatedAt:       now,
		})

		d, ok := byDay[date]
		if !ok {
			d = &models.DailyCashDelta{AsOf: date}
			byDay[date] = d
		}
		if ttype == "income" {
			d.Inflows = d.Inflows.Add(amt)
			currBal = currBal.Add(amt)
		} else {
			d.Outflows = d.Outflows.Add(amt)
			currBal = currBal.Sub(amt)
		}
	}

	for _, d := range byDay {
		deltas = append(deltas, *d)
	}
	return txns, deltas
}

func seedBulkSavingGoals(
	ctx context.Context,
	tx *gorm.DB,
	rng *rand.Rand,
	today time.Time,
	accounts []models.Account,
) error {
	savingsIDs := make([]int64, 0, len(accounts)/len(bulkAccountSeeds))
	savingsUser := make(map[int64]int64, cap(savingsIDs))
	for _, acc := range accounts {
		if acc.Name == "Savings account" {
			savingsIDs = append(savingsIDs, acc.ID)
			savingsUser[acc.ID] = acc.UserID
		}
	}
	if len(savingsIDs) == 0 {
		return nil
	}

	type accBalance struct {
		AccountID  int64
		EndBalance decimal.Decimal
	}
	var latest []accBalance
	if err := tx.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (account_id) account_id, end_balance
		FROM balances
		WHERE account_id IN ?
		ORDER BY account_id, as_of DESC
	`, savingsIDs).Scan(&latest).Error; err != nil {
		return fmt.Errorf("failed to read savings balances: %w", err)
	}

	currentMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	names := []string{"Emergency fund", "Vacation"}

	var goals []models.SavingGoal

	// Contributions need goal IDs, which only exist after the goals are written
	type pendingContrib struct {
		goalIndex int
		amount    decimal.Decimal
		month     time.Time
		source    models.SavingContributionSource
	}
	var contribs []pendingContrib

	for _, lb := range latest {
		// Leave most of the account unallocated, or the uncategorized balance
		// goes negative
		budget := lb.EndBalance.Mul(decimal.NewFromFloat(0.4))
		if budget.LessThan(decimal.NewFromInt(200)) {
			continue
		}

		userID := savingsUser[lb.AccountID]
		for gi, name := range names {
			months := 4 + rng.Intn(5)
			share := budget.Div(decimal.NewFromInt(int64(len(names))))
			monthly := share.Div(decimal.NewFromInt(int64(months))).Round(2)
			if !monthly.IsPositive() {
				continue
			}

			createdAt := currentMonth.AddDate(0, -months, 0)
			targetDate := currentMonth.AddDate(0, 6+rng.Intn(12), 0)
			fundDay := 1

			goals = append(goals, models.SavingGoal{
				UserID:            userID,
				AccountID:         lb.AccountID,
				Name:              name,
				TargetAmount:      share.Mul(decimal.NewFromFloat(1.5)).Round(2),
				CurrentAmount:     monthly.Mul(decimal.NewFromInt(int64(months))),
				TargetDate:        &targetDate,
				Status:            models.SavingGoalStatusActive,
				Priority:          gi,
				MonthlyAllocation: &monthly,
				FundDayOfMonth:    &fundDay,
				CreatedAt:         createdAt,
				UpdatedAt:         createdAt,
			})

			goalIndex := len(goals) - 1
			for m := months; m > 0; m-- {
				month := currentMonth.AddDate(0, -m, 0)
				contribs = append(contribs, pendingContrib{
					goalIndex: goalIndex,
					amount:    monthly,
					month:     month,
					source:    models.SavingContributionSourceAuto,
				})
			}
		}
	}

	if len(goals) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).CreateInBatches(&goals, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert saving goals: %w", err)
	}

	rows := make([]models.SavingContribution, 0, len(contribs))
	for _, c := range contribs {
		g := goals[c.goalIndex]
		rows = append(rows, models.SavingContribution{
			UserID:    g.UserID,
			GoalID:    g.ID,
			Amount:    c.amount,
			Month:     c.month,
			Source:    c.source,
			CreatedAt: c.month,
			UpdatedAt: c.month,
		})
	}
	if err := tx.WithContext(ctx).CreateInBatches(&rows, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert saving contributions: %w", err)
	}

	return nil
}

func bulkInvestmentAccountTypeIDs(ctx context.Context, db *gorm.DB) (invID, cryptoID int64, err error) {
	lookup := func(t, sub string) (int64, error) {
		var at models.AccountType
		if e := db.WithContext(ctx).
			Where("type = ? AND sub_type = ?", t, sub).
			First(&at).Error; e != nil {
			return 0, fmt.Errorf("account type %s/%s not found: %w", t, sub, e)
		}
		return at.ID, nil
	}
	if invID, err = lookup("investment", "brokerage"); err != nil {
		return
	}
	cryptoID, err = lookup("crypto", "exchange")
	return
}

// bulkSleep pauses, but gives up early if the seed context is cancelled.
func bulkSleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// seedBulkAssetPricePool fetches real Yahoo history for every pool ticker and
// the USD/EUR rate, writes it to ticker_price_history / exchange_rate_history,
// and returns it in memory for the per-user loop. Fetches run once per seed.
func seedBulkAssetPricePool(ctx context.Context, db *gorm.DB, cfg *config.Config, today time.Time) (map[string]*bulkAssetPrices, *bulkFX, error) {
	priceClient, err := finance.NewPriceFetchClient(cfg.FinanceAPIBaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("bulk investment seeding requires the finance API: %w", err)
	}

	invRepo := repositories.NewInvestmentRepository(db)
	accRepo := repositories.NewAccountRepository(db)
	txnRepo := repositories.NewTransactionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	invService := services.NewInvestmentService(
		zap.NewNop(), invRepo, accRepo, txnRepo, settingsRepo, jobqueue.NoopDispatcher{}, priceClient)

	// A few days of slack before the earliest account open, so priceOn always
	// has a row to land on.
	from := today.AddDate(0, 0, -(bulkInvHistoryDays + bulkInvOpenBufferDays + 5))

	all := make([]bulkAssetSeed, 0, len(bulkNonCryptoPool)+len(bulkCryptoPool))
	all = append(all, bulkNonCryptoPool...)
	all = append(all, bulkCryptoPool...)

	pool := make(map[string]*bulkAssetPrices, len(all))
	for _, s := range all {
		if err := invService.BackfillTickerPriceHistory(ctx, s.Ticker, from, today); err != nil {
			return nil, nil, fmt.Errorf("price history for %s: %w", s.Ticker, err)
		}
		bulkSleep(ctx, bulkPriceFetchDelay)

		hist, err := invRepo.GetTickerPriceHistory(ctx, nil, s.Ticker)
		if err != nil {
			return nil, nil, err
		}
		if len(hist) == 0 {
			return nil, nil, fmt.Errorf("no price history returned for %s", s.Ticker)
		}

		ap := &bulkAssetPrices{
			currency: hist[len(hist)-1].Currency,
			dates:    make([]time.Time, len(hist)),
			prices:   make([]decimal.Decimal, len(hist)),
		}
		for i, h := range hist {
			ap.dates[i] = h.AsOf.UTC().Truncate(24 * time.Hour)
			ap.prices[i] = h.Price
		}
		pool[s.Ticker] = ap
	}

	fx, err := buildBulkFX(ctx, priceClient, invRepo, from, today)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("bulk asset price pool ready: %d tickers, rates %s..%s\n",
		len(pool), from.Format(time.DateOnly), today.Format(time.DateOnly))
	return pool, fx, nil
}

// buildBulkFX pulls one year of USD/EUR closes, forward-fills every calendar
// day, and caches both directions in exchange_rate_history.
func buildBulkFX(ctx context.Context, priceClient finance.PriceFetcher, invRepo *repositories.InvestmentRepository, from, to time.Time) (*bulkFX, error) {
	// Yahoo "EUR=X" close is EUR per 1 USD, i.e. the USD->EUR rate.
	raw, err := priceClient.GetAssetPriceRange(ctx, "EUR=X", from, to)
	if err != nil {
		return nil, fmt.Errorf("USD/EUR rate history: %w", err)
	}
	bulkSleep(ctx, bulkPriceFetchDelay)

	byDate := make(map[string]decimal.Decimal, len(raw))
	var firstKnown decimal.Decimal
	for _, p := range raw {
		if p.Price <= 0 {
			continue
		}
		v := decimal.NewFromFloat(p.Price)
		byDate[p.Date.UTC().Format("2006-01-02")] = v
		if firstKnown.IsZero() {
			firstKnown = v
		}
	}
	if firstKnown.IsZero() {
		return nil, fmt.Errorf("no usable USD/EUR rate history returned")
	}

	one := decimal.NewFromInt(1)
	filled := make(map[string]decimal.Decimal, int(to.Sub(from).Hours()/24)+2)
	last := firstKnown
	for d := from.UTC().Truncate(24 * time.Hour); !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if v, ok := byDate[key]; ok {
			last = v
		}
		filled[key] = last

		if err := invRepo.UpsertExchangeRate(ctx, nil, models.ExchangeRateHistory{
			FromCurrency: "USD", ToCurrency: "EUR", AsOf: d, Rate: last,
		}); err != nil {
			return nil, err
		}
		if err := invRepo.UpsertExchangeRate(ctx, nil, models.ExchangeRateHistory{
			FromCurrency: "EUR", ToCurrency: "USD", AsOf: d, Rate: one.Div(last),
		}); err != nil {
			return nil, err
		}
	}

	return &bulkFX{usdToEUR: filled}, nil
}

// seedBulkInvestments gives each user in the chunk an EUR investment account and
// a USD crypto account, then 7 equity + 3 crypto holdings from the pool, each
// with one buy trade. All plain batch inserts — no service calls.
func seedBulkInvestments(
	ctx context.Context,
	tx *gorm.DB,
	accRepo *repositories.AccountRepository,
	rng *rand.Rand,
	today time.Time,
	users []models.User,
	pricePool map[string]*bulkAssetPrices,
	fx *bulkFX,
	invTypeID, cryptoTypeID int64,
) error {
	tradeAnchor := today.AddDate(0, 0, -bulkInvHistoryDays)
	openedAt := tradeAnchor.AddDate(0, 0, -bulkInvOpenBufferDays)

	type accMeta struct {
		userID   int64
		currency string
	}

	// Two accounts per user, interleaved: even index = investment, odd = crypto.
	accounts := make([]models.Account, 0, len(users)*2)
	for _, u := range users {
		accounts = append(accounts,
			models.Account{
				UserID: u.ID, Name: bulkInvAccountName, AccountTypeID: invTypeID,
				Currency: "EUR", BalanceProjection: "fixed", ExpectedBalance: decimal.Zero,
				OpenedAt: openedAt, UpdatedAt: openedAt,
			},
			models.Account{
				UserID: u.ID, Name: bulkCryptoAccountName, AccountTypeID: cryptoTypeID,
				Currency: "USD", BalanceProjection: "fixed", ExpectedBalance: decimal.Zero,
				OpenedAt: openedAt, UpdatedAt: openedAt,
			},
		)
	}
	if err := tx.WithContext(ctx).CreateInBatches(&accounts, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert investment accounts: %w", err)
	}

	balances := make([]models.Balance, 0, len(accounts))
	meta := make(map[int64]accMeta, len(accounts))
	for _, acc := range accounts {
		start := bulkInvOpeningCash
		if acc.Currency == "USD" {
			start = bulkCryptoOpeningCash
		}
		balances = append(balances, models.Balance{
			AccountID: acc.ID, AsOf: openedAt, StartBalance: start, Currency: acc.Currency,
			CreatedAt: openedAt, UpdatedAt: openedAt,
		})
		meta[acc.ID] = accMeta{userID: acc.UserID, currency: acc.Currency}
	}
	if err := tx.WithContext(ctx).CreateInBatches(&balances, bulkInsertBatch).Error; err != nil {
		return fmt.Errorf("failed to insert investment opening balances: %w", err)
	}

	var assetRows []models.InvestmentAsset
	var pend []bulkPendingTrade
	deltas := make(map[int64][]models.DailyCashDelta)

	for ui, u := range users {
		invAccID := accounts[ui*2].ID
		cryptoAccID := accounts[ui*2+1].ID

		equities := bulkPickAssets(rng, bulkNonCryptoPool, bulkEquityPerUser)
		cryptos := bulkPickAssets(rng, bulkCryptoPool, bulkCryptoPerUser)

		eqBudget := bulkInvOpeningCash.Mul(bulkInvSpendFraction).Div(decimal.NewFromInt(int64(len(equities))))
		crBudget := bulkCryptoOpeningCash.Mul(bulkInvSpendFraction).Div(decimal.NewFromInt(int64(len(cryptos))))
		eqCeiling := bulkInvOpeningCash.Mul(bulkInvSpendCeiling)
		crCeiling := bulkCryptoOpeningCash.Mul(bulkInvSpendCeiling)

		var eqSpent, crSpent decimal.Decimal

		place := func(seeds []bulkAssetSeed, accID int64, currency string, budget, ceiling decimal.Decimal, spent *decimal.Decimal) {
			for _, s := range seeds {
				row, trade, cost, ok := bulkBuildBuy(rng, tradeAnchor, u.ID, accID, currency, s, budget, pricePool, fx)
				if !ok || spent.Add(cost).GreaterThan(ceiling) {
					continue
				}
				*spent = spent.Add(cost)
				assetRows = append(assetRows, row)
				trade.assetIdx = len(assetRows) - 1
				pend = append(pend, trade)
				deltas[accID] = append(deltas[accID], models.DailyCashDelta{AsOf: trade.date, Outflows: cost})
			}
		}

		place(equities, invAccID, "EUR", eqBudget, eqCeiling, &eqSpent)
		place(cryptos, cryptoAccID, "USD", crBudget, crCeiling, &crSpent)
	}

	if len(assetRows) > 0 {
		if err := tx.WithContext(ctx).CreateInBatches(&assetRows, bulkInsertBatch).Error; err != nil {
			return fmt.Errorf("failed to insert investment assets: %w", err)
		}

		trades := make([]models.InvestmentTrade, 0, len(pend))
		for _, p := range pend {
			a := assetRows[p.assetIdx]
			trades = append(trades, models.InvestmentTrade{
				UserID: a.UserID, AssetID: a.ID, TxnDate: p.date, TradeType: models.InvestmentBuy,
				Quantity: p.qty, Fee: p.fee, PricePerUnit: p.price, ValueAtBuy: p.valueAtBuy,
				RealizedValue: decimal.Zero, Currency: p.currency, ExchangeRateToUSD: p.rateToUSD,
				CreatedAt: p.date, UpdatedAt: p.date,
			})
		}
		if err := tx.WithContext(ctx).CreateInBatches(&trades, bulkInsertBatch).Error; err != nil {
			return fmt.Errorf("failed to insert investment trades: %w", err)
		}
	}

	// Every account needs a snapshot series so its opening cash shows in net
	// worth, whether or not any buy landed on it.
	for _, acc := range accounts {
		m := meta[acc.ID]
		if ds := deltas[acc.ID]; len(ds) > 0 {
			if err := accRepo.UpsertDailyCashBatch(ctx, tx, acc.ID, m.currency, ds); err != nil {
				return fmt.Errorf("failed to write investment cash flows: %w", err)
			}
		}
		if err := accRepo.FrontfillBalances(ctx, tx, acc.ID, m.currency, openedAt); err != nil {
			return fmt.Errorf("failed to frontfill investment balances: %w", err)
		}
		if err := accRepo.UpsertSnapshotsFromBalances(ctx, tx, m.userID, acc.ID, m.currency, openedAt, today); err != nil {
			return fmt.Errorf("failed to build investment snapshots: %w", err)
		}
	}

	return nil
}

func bulkPickAssets(rng *rand.Rand, pool []bulkAssetSeed, n int) []bulkAssetSeed {
	if n >= len(pool) {
		out := make([]bulkAssetSeed, len(pool))
		copy(out, pool)
		return out
	}
	perm := rng.Perm(len(pool))
	out := make([]bulkAssetSeed, n)
	for i := 0; i < n; i++ {
		out[i] = pool[perm[i]]
	}
	return out
}

// bulkBuildBuy turns a per-asset budget (in the account currency) into one buy:
// the asset row plus the pending trade and the cash cost in account currency.
// Returns ok=false when the ticker has no usable price.
func bulkBuildBuy(
	rng *rand.Rand,
	tradeAnchor time.Time,
	userID, accID int64,
	accCurrency string,
	s bulkAssetSeed,
	budget decimal.Decimal,
	pricePool map[string]*bulkAssetPrices,
	fx *bulkFX,
) (models.InvestmentAsset, bulkPendingTrade, decimal.Decimal, bool) {
	ap := pricePool[s.Ticker]
	if ap == nil || len(ap.prices) == 0 {
		return models.InvestmentAsset{}, bulkPendingTrade{}, decimal.Zero, false
	}

	date := tradeAnchor.AddDate(0, 0, rng.Intn(bulkInvOpenBufferDays))
	price := ap.priceOn(date)
	if !price.IsPositive() {
		return models.InvestmentAsset{}, bulkPendingTrade{}, decimal.Zero, false
	}

	tradeCcy := ap.currency
	isCrypto := s.Type == models.InvestmentCrypto
	spend := budget.Mul(fx.rate(accCurrency, tradeCcy, date))

	stockFee := decimal.NewFromFloat(1 + rng.Float64()*2).Round(2)
	holdingsQty, valueAtBuy, avgBuyPrice, fee, ok := bulkPositionFromSpend(isCrypto, price, spend, stockFee)
	if !ok {
		return models.InvestmentAsset{}, bulkPendingTrade{}, decimal.Zero, false
	}

	gross := valueAtBuy
	if !isCrypto {
		gross = gross.Add(fee)
	}
	costAcc := gross.Mul(fx.rate(tradeCcy, accCurrency, date)).Round(4)

	row := models.InvestmentAsset{
		UserID: userID, AccountID: accID, InvestmentType: s.Type, Name: s.Name, Ticker: s.Ticker,
		Quantity: holdingsQty, AverageBuyPrice: avgBuyPrice, ValueAtBuy: valueAtBuy, TotalFees: fee,
		Currency: tradeCcy, CreatedAt: date, UpdatedAt: date,
	}
	trade := bulkPendingTrade{
		date: date, price: price, qty: holdingsQty, fee: fee, valueAtBuy: valueAtBuy,
		currency: tradeCcy, rateToUSD: fx.rate(tradeCcy, "USD", date),
	}
	return row, trade, costAcc, true
}

// bulkPositionFromSpend mirrors InvestmentService.calculateTradeValue for a buy:
// crypto quantity carries the fee in coin units, stock/ETF quantity is whole
// shares with the fee tracked apart.
func bulkPositionFromSpend(isCrypto bool, price, spend, stockFee decimal.Decimal) (holdingsQty, valueAtBuy, avgBuyPrice, fee decimal.Decimal, ok bool) {
	if !price.IsPositive() || !spend.IsPositive() {
		return
	}

	if isCrypto {
		gross := spend.Div(price).RoundDown(6)
		feeCoins := gross.Mul(decimal.NewFromFloat(0.001)).RoundDown(8)
		holdingsQty = gross.Sub(feeCoins)
		if !holdingsQty.IsPositive() {
			return
		}
		valueAtBuy = holdingsQty.Mul(price)
		fee = feeCoins
	} else {
		qty := spend.Div(price).RoundDown(0)
		if !qty.IsPositive() {
			qty = decimal.NewFromInt(1)
		}
		holdingsQty = qty
		valueAtBuy = qty.Mul(price)
		fee = stockFee
	}

	avgBuyPrice = valueAtBuy.Div(holdingsQty)
	ok = true
	return
}

// recomputeBulkInvestmentSnapshots runs one set-based market-value pass over
// every bulk user's investment/crypto snapshots, bounded to the investment
// window. Scales to a 10k-user seed without a per-user round trip.
func recomputeBulkInvestmentSnapshots(ctx context.Context, db *gorm.DB, accRepo *repositories.AccountRepository, emailDomain string, today time.Time) error {
	var ids []int64
	if err := db.WithContext(ctx).
		Raw(`SELECT id FROM users WHERE email LIKE ?`, "bulk%@"+emailDomain).
		Scan(&ids).Error; err != nil {
		return fmt.Errorf("failed to list bulk users for snapshot recompute: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}

	from := today.AddDate(0, 0, -(bulkInvHistoryDays + bulkInvOpenBufferDays))
	fmt.Printf("recomputing investment snapshot market values for %d bulk users ...\n", len(ids))

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return accRepo.UpdateSnapshotMarketValuesForUsers(ctx, tx, ids, &from)
	})
	if err != nil {
		return fmt.Errorf("failed to recompute investment snapshot market values: %w", err)
	}

	fmt.Println("investment snapshot market values recomputed")
	return nil
}
