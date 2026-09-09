package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	expectedBalanceSum  = `COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0)::numeric(19,4)`
	accountBalanceQuery = `
	SELECT a.id AS account_id,
	       a.currency,
	       COALESCE(ab.balance, 0)      AS balance,
	       COALESCE(mv.market_value, 0) AS market_value,
	       COALESCE(ab.balance, 0) + COALESCE(mv.market_value, 0) AS total_balance
	FROM accounts a
	LEFT JOIN balances ab ON ab.account_id = a.id
	LEFT JOIN LATERAL (
		SELECT s.market_value
		FROM balance_snapshots s
		WHERE s.account_id = a.id
		ORDER BY s.as_of DESC
		LIMIT 1
	) mv ON TRUE
`
)

type BalanceRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	ApplyDelta(ctx context.Context, tx *gorm.DB, accountID int64, amount decimal.Decimal) error
	GetBalance(ctx context.Context, tx *gorm.DB, accountID int64) (decimal.Decimal, error)
	RecomputeFromTransactions(ctx context.Context, tx *gorm.DB, accountID int64) error
	FindOpenAccountIDs(ctx context.Context, tx *gorm.DB, afterID int64, limit int) ([]int64, error)
	FindDriftedAccounts(ctx context.Context, tx *gorm.DB, accountIDs []int64) ([]models.BalanceDrift, error)
	RepairBalance(ctx context.Context, tx *gorm.DB, accountID int64) (models.BalanceDrift, bool, error)
	RebuildBalances(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from time.Time) error
	RebuildDailyRange(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from, to time.Time) error
	DeleteAccountSnapshots(ctx context.Context, tx *gorm.DB, accountID int64) error
	FindLatestBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error)
	FindAccountBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (*models.AccountBalance, error)
	ClearInvestmentSnapshots(ctx context.Context, tx *gorm.DB, userID int64) error
	UpdateSnapshotMarketValues(ctx context.Context, tx *gorm.DB, userID int64, from *time.Time) error
	UpdateSnapshotMarketValuesForUsers(ctx context.Context, tx *gorm.DB, userIDs []int64, from *time.Time) error
	HasSnapshotForDate(ctx context.Context, userID int64, date time.Time) (bool, error)
}

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

var _ BalanceRepositoryInterface = (*BalanceRepository)(nil)

func (r *BalanceRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *BalanceRepository) ApplyDelta(ctx context.Context, tx *gorm.DB, accountID int64, amount decimal.Decimal) error {
	if tx == nil {
		return errors.New("ApplyDelta requires a transaction")
	}

	return tx.WithContext(ctx).Exec(`
		INSERT INTO balances (account_id, user_id, currency, balance)
		SELECT a.id, a.user_id, a.currency, ?::numeric(19,4)
		FROM   accounts a
		
		WHERE  a.id = ?::bigint
		ON CONFLICT (account_id) DO UPDATE
		SET balance = balances.balance + EXCLUDED.balance;
	`, amount, accountID).Error
}

func (r *BalanceRepository) GetBalance(ctx context.Context, tx *gorm.DB, accountID int64) (decimal.Decimal, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Balance
	err := db.
		Where("account_id = ?", accountID).
		Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return decimal.Zero, nil
		}
		return decimal.Zero, err
	}
	return record.Balance, nil
}

func (r *BalanceRepository) RecomputeFromTransactions(ctx context.Context, tx *gorm.DB, accountID int64) error {
	if tx == nil {
		return errors.New("RecomputeFromTransactions requires a transaction")
	}
	if _, err := r.lockBalanceRow(ctx, tx, accountID); err != nil {
		return err
	}

	return tx.WithContext(ctx).Exec(`
		UPDATE balances b
		SET balance  = (
		        SELECT `+expectedBalanceSum+`
		        FROM   transactions t
		        WHERE  t.account_id = b.account_id AND t.deleted_at IS NULL
		    ),
		    user_id  = a.user_id,
		    currency = a.currency
		FROM   accounts a
		WHERE  a.id = b.account_id AND b.account_id = ?::bigint
	`, accountID).Error
}

func (r *BalanceRepository) lockBalanceRow(ctx context.Context, tx *gorm.DB, accountID int64) (models.Balance, error) {
	db := tx.WithContext(ctx)

	// An account with no balance row has nothing to lock, so it gets a zero row first.
	// Statements after the lock run on a fresh snapshot and see every committed write.

	if err := db.Exec(`
		INSERT INTO balances (account_id, user_id, currency, balance)
		SELECT a.id, a.user_id, a.currency, 0
		FROM   accounts a
		WHERE  a.id = ?::bigint
		ON CONFLICT (account_id) DO NOTHING
	`, accountID).Error; err != nil {
		return models.Balance{}, err
	}

	var row models.Balance
	err := db.Raw(`
		SELECT account_id, user_id, balance
		FROM   balances
		WHERE  account_id = ?::bigint
		FOR UPDATE
	`, accountID).Scan(&row).Error
	return row, err
}

func (r *BalanceRepository) FindOpenAccountIDs(ctx context.Context, tx *gorm.DB, afterID int64, limit int) ([]int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}

	var ids []int64
	err := db.WithContext(ctx).Raw(`
		SELECT id FROM accounts
		WHERE  closed_at IS NULL AND id > ?::bigint
		ORDER BY id
		LIMIT  ?::int
	`, afterID, limit).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *BalanceRepository) FindDriftedAccounts(ctx context.Context, tx *gorm.DB, accountIDs []int64) ([]models.BalanceDrift, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}

	db := tx
	if db == nil {
		db = r.db
	}

	var rows []models.BalanceDrift
	err := db.WithContext(ctx).Raw(`
		WITH expected AS (
			SELECT a.id AS account_id,
			       a.user_id,
			       `+expectedBalanceSum+` AS expected
			FROM   accounts a
			LEFT JOIN transactions t
			       ON t.account_id = a.id AND t.deleted_at IS NULL
			WHERE  a.id IN ?
			GROUP BY a.id, a.user_id
		)
		SELECT e.account_id,
		       e.user_id,
		       COALESCE(b.balance, 0) AS actual,
		       e.expected
		FROM   expected e
		LEFT JOIN balances b ON b.account_id = e.account_id
		WHERE  COALESCE(b.balance, 0) <> e.expected
		ORDER BY e.account_id
	`, accountIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *BalanceRepository) RepairBalance(ctx context.Context, tx *gorm.DB, accountID int64) (models.BalanceDrift, bool, error) {
	db := tx.WithContext(ctx)

	row, err := r.lockBalanceRow(ctx, tx, accountID)
	if err != nil {
		return models.BalanceDrift{}, false, err
	}
	drift := models.BalanceDrift{AccountID: row.AccountID, UserID: row.UserID, Actual: row.Balance}

	if err := db.Raw(`
		SELECT `+expectedBalanceSum+`
		FROM   transactions t
		WHERE  t.account_id = ?::bigint AND t.deleted_at IS NULL
	`, accountID).Scan(&drift.Expected).Error; err != nil {
		return models.BalanceDrift{}, false, err
	}

	// The scan saw a gap that a live write has since closed. Not drift.
	if drift.Actual.Equal(drift.Expected) {
		return models.BalanceDrift{}, false, nil
	}

	if err := db.Exec(`
		UPDATE balances SET balance = ?::numeric(19,4) WHERE account_id = ?::bigint
	`, drift.Expected, accountID).Error; err != nil {
		return models.BalanceDrift{}, false, err
	}
	return drift, true, nil
}

func (r *BalanceRepository) findAccountBalanceByID(ctx context.Context, db *gorm.DB, accountID int64) (models.AccountBalance, error) {
	var balance models.AccountBalance
	if err := db.WithContext(ctx).
		Raw(accountBalanceQuery+` WHERE a.id = ?`, accountID).
		Scan(&balance).Error; err != nil {
		return models.AccountBalance{}, err
	}
	return balance, nil
}

func (r *BalanceRepository) findAccountBalancesByIDs(ctx context.Context, db *gorm.DB, accountIDs []int64) (map[int64]models.AccountBalance, error) {
	out := make(map[int64]models.AccountBalance, len(accountIDs))
	if len(accountIDs) == 0 {
		return out, nil
	}

	var rows []models.AccountBalance
	if err := db.WithContext(ctx).
		Raw(accountBalanceQuery+` WHERE a.id IN ?`, accountIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.AccountID] = row
	}
	return out, nil
}

func (r *BalanceRepository) RebuildBalances(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from time.Time) error {
	if err := r.RecomputeFromTransactions(ctx, tx, accountID); err != nil {
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	return r.RebuildDailyRange(ctx, tx, userID, accountID, currency, from, today)
}

func (r *BalanceRepository) RebuildDailyRange(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from, to time.Time) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	from = from.UTC().Truncate(24 * time.Hour)
	to = to.UTC().Truncate(24 * time.Hour)

	return db.Exec(`
		WITH signed AS (
			SELECT t.txn_date::date AS day,
			       CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END AS delta
			FROM transactions t
			WHERE t.account_id = ?::bigint AND t.deleted_at IS NULL
		),
		seed AS (
			SELECT COALESCE(SUM(delta), 0) AS amount FROM signed WHERE day < ?::date
		),
		daily AS (
			SELECT day, SUM(delta) AS delta FROM signed
			WHERE day >= ?::date AND day <= ?::date
			GROUP BY day
		)
		INSERT INTO balance_snapshots (
			user_id, account_id, as_of, end_balance, currency, computed_at
		)
		SELECT
			?::bigint AS user_id,
			?::bigint AS account_id,
			d.day     AS as_of,
			((SELECT amount FROM seed)
			  + SUM(COALESCE(daily.delta, 0)) OVER (ORDER BY d.day))::numeric(19,4) AS end_balance,
			?::char(3) AS currency,
			NOW()      AS computed_at
		FROM generate_series(?::date, ?::date, '1 day') AS d(day)
		LEFT JOIN daily ON daily.day = d.day
		ON CONFLICT (account_id, as_of) DO UPDATE
		SET user_id     = EXCLUDED.user_id,
			currency    = EXCLUDED.currency,
			end_balance = EXCLUDED.end_balance,
			computed_at = NOW();
	`, accountID, from, from, to, userID, accountID, currency, from, to).Error
}

func (r *BalanceRepository) DeleteAccountSnapshots(ctx context.Context, tx *gorm.DB, accountID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("account_id = ?", accountID).
		Delete(&models.BalanceSnapshot{}).Error
}

func (r *BalanceRepository) FindLatestBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var row struct{ Balance decimal.Decimal }
	result := db.Raw(`
		SELECT COALESCE(ab.balance, 0) AS balance
		FROM accounts a
		LEFT JOIN balances ab ON ab.account_id = a.id
		WHERE a.id = ? AND a.user_id = ?
	`, accountID, userID).Scan(&row)
	if result.Error != nil {
		return decimal.Zero, result.Error
	}
	if result.RowsAffected == 0 {
		return decimal.Zero, gorm.ErrRecordNotFound
	}
	return row.Balance, nil
}

func (r *BalanceRepository) FindAccountBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (*models.AccountBalance, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var balance models.AccountBalance
	result := db.Raw(accountBalanceQuery+` WHERE a.id = ? AND a.user_id = ?`, accountID, userID).Scan(&balance)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &balance, nil
}

func (r *BalanceRepository) ClearInvestmentSnapshots(ctx context.Context, tx *gorm.DB, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)
	return db.Exec(`
        DELETE FROM balance_snapshots
        WHERE account_id IN (
            SELECT id FROM accounts WHERE user_id = ? AND closed_at IS NULL
        )
    `, userID).Error
}

func (r *BalanceRepository) UpdateSnapshotMarketValues(ctx context.Context, tx *gorm.DB, userID int64, from *time.Time) error {
	return r.UpdateSnapshotMarketValuesForUsers(ctx, tx, []int64{userID}, from)
}

func (r *BalanceRepository) UpdateSnapshotMarketValuesForUsers(ctx context.Context, tx *gorm.DB, userIDs []int64, from *time.Time) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if len(userIDs) == 0 {
		return nil
	}

	dateFilter := ""
	args := []interface{}{userIDs}
	if from != nil {
		dateFilter = "AND s.as_of >= ?"
		args = append(args, from.UTC().Truncate(24*time.Hour))
	}

	query := fmt.Sprintf(`
		UPDATE balance_snapshots s
		SET market_value = (
			SELECT COALESCE(SUM(
				ph_latest.price
				* COALESCE(erh_latest.rate, 1)
				* GREATEST(qty.held, 0)
			), 0)
			FROM investment_assets ia
			JOIN LATERAL (
				SELECT ph.price, ph.currency
				FROM ticker_price_history ph
				WHERE ph.ticker = ia.ticker
				  AND ph.as_of <= s.as_of
				ORDER BY ph.as_of DESC
				LIMIT 1
			) ph_latest ON true
			JOIN LATERAL (
				SELECT
					COALESCE(SUM(
						CASE WHEN it.trade_type = 'buy'  THEN  it.quantity
						     WHEN it.trade_type = 'sell' THEN -it.quantity
						END
					), 0)
					+ COALESCE((
						SELECT SUM(ii.quantity)
						FROM investment_income ii
						WHERE ii.asset_id    = ia.id
						  AND ii.income_type = 'staking_reward'
						  AND ii.txn_date   <= s.as_of
					), 0) AS held
				FROM investment_trades it
				WHERE it.asset_id  = ia.id
				  AND it.txn_date <= s.as_of
			) qty ON true
			LEFT JOIN LATERAL (
				SELECT erh.rate
				FROM exchange_rate_history erh
				WHERE erh.from_currency = ph_latest.currency
				  AND erh.to_currency   = a.currency
				  AND erh.as_of        <= s.as_of
				ORDER BY erh.as_of DESC
				LIMIT 1
			) erh_latest ON ph_latest.currency != a.currency
			WHERE ia.account_id = s.account_id
		)
		FROM accounts a
		JOIN account_types at ON at.id = a.account_type_id
		WHERE s.account_id = a.id
		  AND a.user_id IN ?
		  AND at.type IN ('investment', 'crypto')
		  %s;
	`, dateFilter)

	return db.Exec(query, args...).Error
}

func (r *BalanceRepository) HasSnapshotForDate(ctx context.Context, userID int64, date time.Time) (bool, error) {
	normalized := date.UTC().Truncate(24 * time.Hour)
	var exists bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM balance_snapshots s
			JOIN accounts a ON a.id = s.account_id
			WHERE a.user_id = ? AND s.as_of = ?
		)
	`, userID, normalized).Scan(&exists).Error
	return exists, err
}
