package repositories

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ChartingRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FetchNetWorthSeries(ctx context.Context, tx *gorm.DB, userID int64, currency string, from, to time.Time, gran string, accountID *int64) ([]models.ChartPoint, error)
	FetchLatestNetWorth(ctx context.Context, tx *gorm.DB, userID int64, currency string, accountID *int64) (time.Time, string, error)
}
type ChartingRepository struct {
	db *gorm.DB
}

func NewChartingRepository(db *gorm.DB) *ChartingRepository {
	return &ChartingRepository{db: db}
}

var _ ChartingRepositoryInterface = (*ChartingRepository)(nil)

func (r *ChartingRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *ChartingRepository) sourceView(accountID *int64) string {
	if accountID != nil {
		return "v_user_account_daily_snapshots"
	}
	return "v_user_daily_networth_snapshots"
}

func (r *ChartingRepository) FetchNetWorthSeries(ctx context.Context, tx *gorm.DB, userID int64, currency string, from, to time.Time, gran string, accountID *int64) ([]models.ChartPoint, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	type row struct {
		Date  time.Time
		Value string
	}
	rows := []row{}

	src := r.sourceView(accountID)

	switch gran {
	case "day":
		sql := `
		  SELECT as_of AS date, end_balance::text AS value
		  FROM ` + src + `
		  WHERE user_id = ? AND currency = ? AND as_of BETWEEN ? AND ?
		`
		args := []any{userID, currency, from, to}
		if accountID != nil {
			sql += " AND account_id = ?"
			args = append(args, *accountID)
		}
		sql += " ORDER BY as_of"
		if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}

	case "week":
		sql := `
		  WITH s AS (
		    SELECT as_of, end_balance
		    FROM ` + src + `
		    WHERE user_id = ? AND currency = ? AND as_of BETWEEN ? AND ?
		`
		args := []any{userID, currency, from, to}
		if accountID != nil {
			sql += " AND account_id = ?"
			args = append(args, *accountID)
		}
		sql += `
		  ),
		  b AS (
		    SELECT date_trunc('week', as_of)::date AS bucket, as_of, end_balance
		    FROM s
		  )
		  SELECT DISTINCT ON (bucket) 
			as_of::date      AS date, 
			end_balance::text AS value
		  FROM b
		  ORDER BY bucket, as_of DESC
		`
		if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}

	case "month":
		sql := `
		  WITH s AS (
		    SELECT as_of, end_balance
		    FROM ` + src + `
		    WHERE user_id = ? AND currency = ? AND as_of BETWEEN ? AND ?
		`
		args := []any{userID, currency, from, to}
		if accountID != nil {
			sql += " AND account_id = ?"
			args = append(args, *accountID)
		}
		sql += `
		  ),
		  b AS (
		    SELECT date_trunc('month', as_of)::date AS bucket, as_of, end_balance
		    FROM s
		  )
		  SELECT DISTINCT ON (bucket)
			  as_of::date       AS date,
			  end_balance::text AS value
		  FROM b
		  ORDER BY bucket, as_of DESC
		`
		if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown granularity %q", gran)
	}

	out := make([]models.ChartPoint, 0, len(rows))
	for _, r := range rows {
		v, _ := decimal.NewFromString(r.Value)
		out = append(out, models.ChartPoint{Date: r.Date, Value: v})
	}

	return out, nil
}

func (r *ChartingRepository) FetchLatestNetWorth(ctx context.Context, tx *gorm.DB, userID int64, currency string, accountID *int64) (time.Time, string, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)
	src := r.sourceView(accountID)

	sql := `
	  SELECT as_of, end_balance::text
	  FROM ` + src + `
	  WHERE user_id = ? AND currency = ?
	`
	args := []any{userID, currency}
	if accountID != nil {
		sql += " AND account_id = ?"
		args = append(args, *accountID)
	}
	sql += ` ORDER BY as_of DESC LIMIT 1`

	var date time.Time
	var value string
	if err := db.Raw(sql, args...).Row().Scan(&date, &value); err != nil {
		return time.Time{}, "", err
	}
	return date, value, nil
}
