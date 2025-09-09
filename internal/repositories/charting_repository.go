package repositories

import (
	"fmt"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
)

type ChartingRepository struct {
	DB *gorm.DB
}

func NewChartingRepository(db *gorm.DB) *ChartingRepository {
	return &ChartingRepository{DB: db}
}

func (r *ChartingRepository) FetchNetWorthSeries(tx *gorm.DB, userID int64, currency string, from, to time.Time, gran string) ([]models.ChartPoint, error) {
	type row struct {
		Date  time.Time
		Value string
	}
	rows := []row{}

	switch gran {
	case "day":
		err := tx.Raw(`
            SELECT as_of AS date,
                   end_balance::text AS value
            FROM v_user_daily_networth_snapshots
            WHERE user_id = ? AND currency = ?
              AND as_of BETWEEN ? AND ?
            ORDER BY as_of
        `, userID, currency, from, to).Scan(&rows).Error
		if err != nil {
			return nil, err
		}

	case "week":
		err := tx.Raw(`
            WITH s AS (
              SELECT as_of, end_balance
              FROM v_user_daily_networth_snapshots
              WHERE user_id = ? AND currency = ? AND as_of BETWEEN ? AND ?
            ),
            b AS (
              SELECT date_trunc('week', as_of)::date AS bucket, as_of, end_balance
              FROM s
            )
            SELECT DISTINCT ON (bucket)
                   bucket AS date,
                   end_balance::text AS value
            FROM b
            ORDER BY bucket, as_of DESC
        `, userID, currency, from, to).Scan(&rows).Error
		if err != nil {
			return nil, err
		}

	case "month":
		err := tx.Raw(`
            WITH s AS (
              SELECT as_of, end_balance
              FROM v_user_daily_networth_snapshots
              WHERE user_id = ? AND currency = ? AND as_of BETWEEN ? AND ?
            ),
            b AS (
              SELECT date_trunc('month', as_of)::date AS bucket, as_of, end_balance
              FROM s
            )
            SELECT DISTINCT ON (bucket)
                   bucket AS date,
                   end_balance::text AS value
            FROM b
            ORDER BY bucket, as_of DESC
        `, userID, currency, from, to).Scan(&rows).Error
		if err != nil {
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

func (r *ChartingRepository) FetchLatestNetWorth(tx *gorm.DB, userID int64, currency string) (time.Time, string, error) {
	var date time.Time
	var value string
	err := tx.Raw(`
        SELECT as_of, end_balance::text
        FROM v_user_daily_networth_snapshots
        WHERE user_id = ? AND currency = ?
        ORDER BY as_of DESC
        LIMIT 1
    `, userID, currency).Row().Scan(&date, &value)
	if err != nil {
		return time.Time{}, "", err
	}
	return date, value, nil
}
