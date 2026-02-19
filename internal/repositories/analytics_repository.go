package repositories

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AnalyticsRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FetchNetWorthSeries(ctx context.Context, tx *gorm.DB, userID int64, currency string, from, to time.Time, gran string, accountID *int64) ([]models.ChartPoint, error)
	FetchLatestNetWorth(ctx context.Context, tx *gorm.DB, userID int64, currency string, accountID *int64) (time.Time, string, error)
	FetchDailyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, date time.Time) (*models.MonthlyTotalsRow, error)
	FetchDailyTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, date time.Time) (*models.MonthlyTotalsRow, error)
	FetchYearlyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) (models.YearlyTotalsRow, error)
	FetchYearlyCategoryTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) ([]models.YearlyCategoryRow, error)
	FetchMonthlyCategoryTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int, month int) ([]models.YearlyCategoryRow, error)
	FetchMonthlyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) ([]models.MonthlyTotalsRow, error)
	FetchMonthlyTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, year int) ([]models.MonthlyTotalsRow, error)
	FetchMonthlyCategoryTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, year, month int) ([]models.YearlyCategoryRow, error)
	GetAvailableStatsYears(ctx context.Context, tx *gorm.DB, accID *int64, userID int64) ([]int64, error)
}
type AnalyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

var _ AnalyticsRepositoryInterface = (*AnalyticsRepository)(nil)

func (r *AnalyticsRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *AnalyticsRepository) sourceView(accountID *int64) string {
	if accountID != nil {
		return "v_user_account_daily_snapshots"
	}
	return "v_user_daily_networth_snapshots"
}

func (r *AnalyticsRepository) FetchNetWorthSeries(ctx context.Context, tx *gorm.DB, userID int64, currency string, from, to time.Time, gran string, accountID *int64) ([]models.ChartPoint, error) {

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

func (r *AnalyticsRepository) FetchLatestNetWorth(ctx context.Context, tx *gorm.DB, userID int64, currency string, accountID *int64) (time.Time, string, error) {

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

func (r *AnalyticsRepository) FetchYearlyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) (models.YearlyTotalsRow, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var row models.YearlyTotalsRow
	if accountID != nil {
		sql := `
		  SELECT
		    $3::int AS year,
		    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
		    		    COALESCE(SUM(
						  CASE
							WHEN transaction_type='income'  THEN amount
							WHEN transaction_type='expense' THEN -amount
							ELSE 0
						  END
						),0)::text AS net_text,
		    COALESCE(COUNT(DISTINCT date_trunc('month', txn_date)),0)                            AS active_months
		  FROM transactions
		  WHERE user_id = $1
		    AND account_id = $2
		    AND is_adjustment = false
		    AND is_transfer = false
		    AND txn_date >= make_date($3,1,1) AND txn_date < make_date($3+1,1,1)
		    AND deleted_at IS NULL
		`
		if err := db.Raw(sql, userID, *accountID, year).Scan(&row).Error; err != nil {
			return row, err
		}
	} else {
		sql := `
		  SELECT
		    $2::int AS year,
		    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
		    		    COALESCE(SUM(
						  CASE
							WHEN transaction_type='income'  THEN amount
							WHEN transaction_type='expense' THEN -amount
							ELSE 0
						  END
						),0)::text AS net_text,
		    COALESCE(COUNT(DISTINCT date_trunc('month', txn_date)),0)                            AS active_months
		  FROM transactions
		  WHERE user_id = $1
		    AND is_adjustment = false
		    AND is_transfer = false
		    AND txn_date >= make_date($2,1,1) AND txn_date < make_date($2+1,1,1)
		    AND deleted_at IS NULL
		`
		if err := db.Raw(sql, userID, year).Scan(&row).Error; err != nil {
			return row, err
		}
	}
	return row, nil
}

func (r *AnalyticsRepository) FetchYearlyCategoryTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) ([]models.YearlyCategoryRow, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var rows []models.YearlyCategoryRow
	if accountID != nil {
		sql := `
		  SELECT
		    $3::int AS year,
		    t.category_id,
		    c.display_name,
		    COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
			COALESCE(SUM(
			  CASE
				WHEN t.transaction_type='income'  THEN t.amount
				WHEN t.transaction_type='expense' THEN -t.amount
				ELSE 0
			  END
			),0)::text AS net_text
		  FROM transactions t
		  LEFT JOIN categories c ON c.id = t.category_id
		  WHERE t.user_id = $1
		    AND t.account_id = $2
		    AND t.is_adjustment = false
		    AND t.is_transfer = false
		    AND t.txn_date >= make_date($3,1,1) AND t.txn_date < make_date($3+1,1,1)
		    AND t.deleted_at IS NULL
		  GROUP BY t.category_id, c.display_name
		  ORDER BY t.category_id NULLS LAST
		`
		if err := db.Raw(sql, userID, *accountID, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	} else {
		sql := `
		  SELECT
		    $2::int AS year,
		    t.category_id,
		    c.display_name,
		    COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
            COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
			COALESCE(SUM(
			  CASE
				WHEN t.transaction_type='income'  THEN t.amount
				WHEN t.transaction_type='expense' THEN -t.amount
				ELSE 0
			  END
			),0)::text AS net_text
		  FROM transactions t
		  LEFT JOIN categories c ON c.id = t.category_id
		  WHERE t.user_id = $1
		    AND t.is_adjustment = false
		    AND t.is_transfer = false
		    AND t.txn_date >= make_date($2,1,1) AND t.txn_date < make_date($2+1,1,1)
		    AND t.deleted_at IS NULL
		  GROUP BY t.category_id, c.display_name
		  ORDER BY t.category_id NULLS LAST
		`
		if err := db.Raw(sql, userID, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *AnalyticsRepository) FetchMonthlyCategoryTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year, month int) ([]models.YearlyCategoryRow, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var rows []models.YearlyCategoryRow

	if accountID != nil {
		sql := `
         SELECT
           $3::int AS year,
           t.category_id,
           c.display_name,
           COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
           COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
          COALESCE(SUM(
            CASE
             WHEN t.transaction_type='income'  THEN t.amount
             WHEN t.transaction_type='expense' THEN -t.amount
             ELSE 0
            END
          ),0)::text AS net_text
         FROM transactions t
         LEFT JOIN categories c ON c.id = t.category_id
         WHERE t.user_id = $1
           AND t.account_id = $2
           AND t.is_adjustment = false
           AND t.is_transfer = false
           AND t.txn_date >= make_date($3, $4, 1) 
           AND t.txn_date < make_date($3, $4, 1) + interval '1 month'
           AND t.deleted_at IS NULL
         GROUP BY t.category_id, c.display_name
         ORDER BY t.category_id NULLS LAST
       `
		if err := db.Raw(sql, userID, *accountID, year, month).Scan(&rows).Error; err != nil {
			return nil, err
		}
	} else {
		sql := `
         SELECT
           $2::int AS year,
           t.category_id,
           c.display_name,
           COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
           COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
          COALESCE(SUM(
            CASE
             WHEN t.transaction_type='income'  THEN t.amount
             WHEN t.transaction_type='expense' THEN -t.amount
             ELSE 0
            END
          ),0)::text AS net_text
         FROM transactions t
         LEFT JOIN categories c ON c.id = t.category_id
         WHERE t.user_id = $1
           AND t.is_adjustment = false
           AND t.is_transfer = false
           AND t.txn_date >= make_date($2, $3, 1) 
           AND t.txn_date < make_date($2, $3, 1) + interval '1 month'
           AND t.deleted_at IS NULL
         GROUP BY t.category_id, c.display_name
         ORDER BY t.category_id NULLS LAST
       `
		if err := db.Raw(sql, userID, year, month).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *AnalyticsRepository) FetchMonthlyCategoryTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, year, month int) ([]models.YearlyCategoryRow, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var rows []models.YearlyCategoryRow

	if len(accountIDs) == 0 {
		return rows, nil
	}

	sql := `
      SELECT
        ?::int AS year,
        t.category_id,
        c.display_name,
        COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
        COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
        COALESCE(SUM(
          CASE
           WHEN t.transaction_type='income'  THEN t.amount
           WHEN t.transaction_type='expense' THEN -t.amount
           ELSE 0
          END
        ),0)::text AS net_text
      FROM transactions t
      LEFT JOIN categories c ON c.id = t.category_id
      WHERE t.user_id = ?
        AND t.account_id IN ?
        AND t.is_adjustment = false
        AND t.is_transfer = false
        AND t.txn_date >= make_date(?, ?, 1) 
        AND t.txn_date < make_date(?, ?, 1) + interval '1 month'
        AND t.deleted_at IS NULL
      GROUP BY t.category_id, c.display_name
      ORDER BY t.category_id NULLS LAST
    `

	if err := db.Raw(sql, year, userID, accountIDs, year, month, year, month).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AnalyticsRepository) FetchMonthlyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, year int) ([]models.MonthlyTotalsRow, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var rows []models.MonthlyTotalsRow

	base := `
	  SELECT
	    EXTRACT(MONTH FROM txn_date)::int AS month,
	    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
	    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
	    COALESCE(SUM(
	      CASE
	        WHEN transaction_type='income'  THEN amount
	        WHEN transaction_type='expense' THEN -amount
	        ELSE 0
	      END
	    ),0)::text AS net_text
	  FROM transactions
	  WHERE user_id = ? %s
	    AND is_adjustment = false
	    AND is_transfer = false
	    AND txn_date >= make_date(?,1,1) AND txn_date < make_date(?+1,1,1)
	    AND deleted_at IS NULL
	  GROUP BY month
	  ORDER BY month;
	`

	if accountID != nil {
		sql := fmt.Sprintf(base, "AND account_id = ?")
		if err := db.Raw(sql, userID, *accountID, year, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	} else {
		sql := fmt.Sprintf(base, "")
		if err := db.Raw(sql, userID, year, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *AnalyticsRepository) FetchMonthlyTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, year int) ([]models.MonthlyTotalsRow, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var rows []models.MonthlyTotalsRow

	if len(accountIDs) == 0 {
		return rows, nil
	}

	query := `
	  SELECT
	    EXTRACT(MONTH FROM txn_date)::int AS month,
	    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
	    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
	    COALESCE(SUM(
	      CASE
	        WHEN transaction_type='income'  THEN amount
	        WHEN transaction_type='expense' THEN -amount
	        ELSE 0
	      END
	    ),0)::text AS net_text
	  FROM transactions
	  WHERE user_id = ?
	    AND is_adjustment = false
	    AND is_transfer = false
	    AND txn_date >= make_date(?,1,1)
	    AND txn_date < make_date(?+1,1,1)
	    AND account_id IN ?
	  	AND deleted_at IS NULL
	  GROUP BY month
	  ORDER BY month;
	`

	err := db.Raw(query, userID, year, year, accountIDs).Scan(&rows).Error
	return rows, err
}

func (r *AnalyticsRepository) GetAvailableStatsYears(ctx context.Context, tx *gorm.DB, accID *int64, userID int64) ([]int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var (
		query string
		args  []any
	)

	if accID == nil {
		query = `
			SELECT DISTINCT EXTRACT(YEAR FROM txn_date)::int AS year
			FROM transactions
			WHERE user_id = ?
			AND deleted_at IS NULL
			ORDER BY year;
		`
		args = []any{userID}
	} else {
		query = `
			SELECT DISTINCT EXTRACT(YEAR FROM txn_date)::int AS year
			FROM transactions
			WHERE user_id = ?
			  AND account_id = ?
			  AND deleted_at IS NULL
			ORDER BY year;
		`
		args = []any{userID, *accID}
	}

	var years []int64
	if err := db.Raw(query, args...).Scan(&years).Error; err != nil {
		return nil, fmt.Errorf("querying available stats years: %w", err)
	}
	return years, nil
}

func (r *AnalyticsRepository) FetchDailyTotals(ctx context.Context, tx *gorm.DB, userID int64, accountID *int64, date time.Time) (*models.MonthlyTotalsRow, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var row models.MonthlyTotalsRow

	base := `
        SELECT
            COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
            COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
            COALESCE(SUM(
                CASE
                    WHEN transaction_type='income'  THEN amount
                    WHEN transaction_type='expense' THEN -amount
                    ELSE 0
                END
            ),0)::text AS net_text
        FROM transactions
        WHERE user_id = ? %s
            AND is_adjustment = false
            AND is_transfer = false
            AND txn_date = ?
        	AND deleted_at IS NULL
    `

	if accountID != nil {
		sql := fmt.Sprintf(base, "AND account_id = ?")
		if err := db.Raw(sql, userID, *accountID, date).Scan(&row).Error; err != nil {
			return nil, err
		}
	} else {
		sql := fmt.Sprintf(base, "")
		if err := db.Raw(sql, userID, date).Scan(&row).Error; err != nil {
			return nil, err
		}
	}
	return &row, nil
}

func (r *AnalyticsRepository) FetchDailyTotalsCheckingOnly(ctx context.Context, tx *gorm.DB, userID int64, accountIDs []int64, date time.Time) (*models.MonthlyTotalsRow, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var row models.MonthlyTotalsRow

	if len(accountIDs) == 0 {
		return &row, nil
	}

	query := `
        SELECT
            COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
            COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
            COALESCE(SUM(
                CASE
                    WHEN transaction_type='income'  THEN amount
                    WHEN transaction_type='expense' THEN -amount
                    ELSE 0
                END
            ),0)::text AS net_text
        FROM transactions
        WHERE user_id = ?
            AND is_adjustment = false
            AND is_transfer = false
            AND txn_date = ?
            AND account_id IN ?
        	AND deleted_at IS NULL
    `

	err := db.Raw(query, userID, date, accountIDs).Scan(&row).Error
	return &row, err
}
