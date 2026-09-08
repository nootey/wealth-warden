package repositories

import (
	"context"
	"errors"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BalanceRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	ApplyDelta(ctx context.Context, tx *gorm.DB, accountID int64, amount decimal.Decimal) error
	GetBalance(ctx context.Context, tx *gorm.DB, accountID int64) (decimal.Decimal, error)
	RecomputeFromTransactions(ctx context.Context, tx *gorm.DB, accountID int64) error
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

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Exec(`
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
	return record.EndBalance, nil
}

func (r *BalanceRepository) RecomputeFromTransactions(ctx context.Context, tx *gorm.DB, accountID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Exec(`
		INSERT INTO balances (account_id, user_id, currency, balance)
		SELECT a.id,
		       a.user_id,
		       a.currency,
		       COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0)
		FROM   accounts a
		LEFT JOIN transactions t
		       ON t.account_id = a.id AND t.deleted_at IS NULL
		WHERE  a.id = ?::bigint
		GROUP BY a.id, a.user_id, a.currency
		ON CONFLICT (account_id) DO UPDATE
		SET balance  = EXCLUDED.balance,
		    user_id  = EXCLUDED.user_id,
		    currency = EXCLUDED.currency;
	`, accountID).Error
}
