package tests

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionTypeMigration(t *testing.T) {
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

	// Stop one short of the new migration, so the boolean columns still exist.
	require.NoError(t, goose.UpTo(sqlDB, path, 20260908120000))

	require.NoError(t, db.Exec(`
		INSERT INTO roles (name) VALUES ('tester');
		INSERT INTO users (password, email, display_name, role_id)
			VALUES ('x', 'a@b.c', 'Tester', (SELECT id FROM roles WHERE name='tester'));
		INSERT INTO account_types (type, sub_type) VALUES ('cash', 'checking');
		INSERT INTO accounts (user_id, name, account_type_id)
			VALUES ((SELECT id FROM users WHERE email='a@b.c'), 'Acc',
			        (SELECT id FROM account_types WHERE type='cash'));`).Error)

	// A live ledger row, a live merged transfer carrying both flags, and a
	// soft-deleted transfer. The soft-deleted one is what broke on the prod dump.
	require.NoError(t, db.Exec(`
		INSERT INTO transactions
			(user_id, account_id, direction, amount, currency, txn_date,
			 is_adjustment, is_transfer, is_system, deleted_at)
		SELECT u.id, a.id, 'expense', 1, 'EUR', CURRENT_DATE, v.adj, v.tr, v.sys, v.del
		FROM users u, accounts a, (VALUES
			(false, false, false, NULL::timestamptz),
			(true,  true,  false, NULL::timestamptz),
			(false, true,  false, NOW()),
			(false, false, true,  NOW())
		) AS v(adj, tr, sys, del)
		WHERE u.email='a@b.c'`).Error)

	require.NoError(t, goose.Up(sqlDB, path))

	type row struct {
		TransactionType string
		N               int64
	}
	var got []row
	require.NoError(t, db.Raw(`SELECT transaction_type, count(*) AS n
		FROM transactions GROUP BY 1 ORDER BY 1`).Scan(&got).Error)
	// ORDER BY on an enum follows declaration order.
	require.Equal(t, []row{
		{"ledger", 1},
		{"transfer", 1},          // soft-deleted transfer still backfilled
		{"adjustment", 1},        // both flags set -> adjustment, not transfer
		{"investment_income", 1}, // soft-deleted is_system row still backfilled
	}, got)

	// The guard trigger must be back on, or every later write to a deleted row passes.
	err = db.Exec(`UPDATE transactions SET amount = 99 WHERE deleted_at IS NOT NULL`).Error
	require.ErrorContains(t, err, "soft-deleted", "the guard trigger was left disabled")

	require.NoError(t, goose.Down(sqlDB, path))

	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM transactions WHERE is_transfer`).Scan(&n).Error)
	require.Equal(t, int64(1), n, "down did not restore is_transfer")

	err = db.Exec(`UPDATE transactions SET amount = 99 WHERE deleted_at IS NOT NULL`).Error
	require.ErrorContains(t, err, "soft-deleted", "down left the guard trigger disabled")

	require.NoError(t, goose.Up(sqlDB, path))
}
