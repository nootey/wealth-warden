package tests

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOpeningTransactionMigration(t *testing.T) {
	ctx := context.Background()
	c, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)))
	require.NoError(t, err)
	defer func() { _ = c.Terminate(ctx) }()

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	require.NoError(t, goose.SetDialect("postgres"))
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	path := filepath.Join(root, "storage", "migrations")

	// Stop one short of the new pair, so the opening amount still lives in start_balance.
	require.NoError(t, goose.UpTo(sqlDB, path, 20260909120000))

	// An asset account, a liability, and a closed one. The closed account is the
	// reason the backfill has to suspend the posting trigger.
	require.NoError(t, db.Exec(`
		INSERT INTO roles (name) VALUES ('tester');
		INSERT INTO users (password, email, display_name, role_id)
			VALUES ('x', 'a@b.c', 'Tester', (SELECT id FROM roles WHERE name='tester'));
		INSERT INTO account_types (type, sub_type)
			VALUES ('cash', 'checking'), ('credit_card', 'credit');
		INSERT INTO categories (name, display_name, classification, is_default)
			VALUES ('(Uncategorized)', '(Uncategorized)', 'uncategorized', true);
		INSERT INTO accounts (user_id, name, account_type_id, currency, opened_at, is_active, closed_at)
		SELECT u.id, v.name, at.id, 'EUR', v.opened::timestamptz, v.active, v.closed
		FROM users u,
		     account_types at,
		     (VALUES
		        ('Checking', 'cash',        DATE '2026-01-01', true,  NULL::timestamptz),
		        ('Card',     'credit_card', DATE '2026-01-01', true,  NULL::timestamptz),
		        ('Closed',   'cash',        DATE '2026-02-01', false, NOW())
		     ) AS v(name, acc_type, opened, active, closed)
		WHERE u.email='a@b.c' AND at.type = v.acc_type`).Error)

	// Two days per account: the opening seed, then a day of flows chained off it.
	require.NoError(t, db.Exec(`
		INSERT INTO balances (account_id, as_of, start_balance, cash_inflows, cash_outflows, currency)
		SELECT a.id, DATE '2026-01-05', v.seed, 0, 0, 'EUR'
		FROM accounts a, (VALUES ('Checking', 1000.0), ('Card', -500.0), ('Closed', 250.0)) AS v(name, seed)
		WHERE a.name = v.name;

		INSERT INTO balances (account_id, as_of, start_balance, cash_inflows, cash_outflows, currency)
		SELECT a.id, DATE '2026-01-06', v.seed, 40, 15, 'EUR'
		FROM accounts a, (VALUES ('Checking', 1000.0), ('Card', -500.0), ('Closed', 250.0)) AS v(name, seed)
		WHERE a.name = v.name`).Error)

	type balRow struct {
		AccountID  int64
		AsOf       time.Time
		EndBalance decimal.Decimal
	}
	dump := func() []balRow {
		var rows []balRow
		require.NoError(t, db.Raw(`SELECT account_id, as_of, end_balance FROM balances
			ORDER BY account_id, as_of`).Scan(&rows).Error)
		return rows
	}
	before := dump()
	require.Len(t, before, 6)

	require.NoError(t, goose.Up(sqlDB, path))

	// The whole point: not one derived number moves.
	require.Equal(t, before, dump(), "the backfill changed a balance")

	type txnRow struct {
		Name       string
		Direction  string
		Amount     decimal.Decimal
		TxnDate    time.Time
		CategoryID *int64
	}
	var txns []txnRow
	require.NoError(t, db.Raw(`
		SELECT a.name, t.direction, t.amount, t.txn_date, t.category_id
		FROM transactions t JOIN accounts a ON a.id = t.account_id
		WHERE t.transaction_type = 'opening' ORDER BY a.name`).Scan(&txns).Error)
	require.Len(t, txns, 3, "every account needs an opening transaction, closed ones included")

	require.Equal(t, "Card", txns[0].Name)
	require.Equal(t, "expense", txns[0].Direction, "a liability opens negative")
	require.True(t, decimal.NewFromInt(500).Equal(txns[0].Amount))
	require.Equal(t, "income", txns[1].Direction)
	require.True(t, decimal.NewFromInt(1000).Equal(txns[1].Amount))

	// Only the earliest row per account held a seed. The later rows carry the
	// chain, and phase 3 is what retires that.
	// Without a category the row cannot be opened in the edit form.
	for _, txn := range txns {
		require.NotNil(t, txn.CategoryID, "%s opening row has no category", txn.Name)
	}

	var seeds int64
	require.NoError(t, db.Raw(`
		SELECT count(*) FROM (
			SELECT DISTINCT ON (account_id) start_balance FROM balances ORDER BY account_id, as_of
		) o WHERE start_balance <> 0`).Scan(&seeds).Error)
	require.Zero(t, seeds, "start_balance is no longer where the opening amount lives")

	// The closed account opened after its earliest balance row, so opened_at moves
	// back onto it. Rebuilds start there, and the opening row must be inside them.
	var openedAt time.Time
	require.NoError(t, db.Raw(`SELECT opened_at FROM accounts WHERE name = 'Closed'`).Scan(&openedAt).Error)
	require.Equal(t, "2026-01-05", openedAt.UTC().Format("2006-01-02"))

	// The trigger must be back on, or every later write to a closed account passes.
	err = db.Exec(`INSERT INTO transactions (user_id, account_id, direction, amount, currency, txn_date)
		SELECT a.user_id, a.id, 'income', 1, 'EUR', CURRENT_DATE FROM accounts a WHERE a.name = 'Closed'`).Error
	require.ErrorContains(t, err, "not open", "the posting trigger was left disabled")

	// Down puts the amount back where it was.
	require.NoError(t, goose.Down(sqlDB, path))
	require.NoError(t, goose.Down(sqlDB, path))

	require.Equal(t, before, dump(), "down did not restore the balances")

	var left int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM transactions`).Scan(&left).Error)
	require.Zero(t, left, "down left opening transactions behind")

	require.NoError(t, goose.Up(sqlDB, path))
}
