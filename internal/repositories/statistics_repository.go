package repositories

import (
	"gorm.io/gorm"
)

type StatisticsRepository struct {
	DB *gorm.DB
}

func NewStatisticsRepository(db *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{DB: db}
}

// rows for yearly totals
type yearlyTotalsRow struct {
	Year         int
	InflowText   string
	OutflowText  string
	NetText      string
	ActiveMonths int
}

// rows for per-category totals
type yearlyCategoryRow struct {
	Year        int
	CategoryID  int64
	DisplayName *string
	InflowText  string
	OutflowText string
	NetText     string
}

func (r *StatisticsRepository) FetchYearlyTotals(
	tx *gorm.DB, userID int64, accountID *int64, year int,
) (yearlyTotalsRow, error) {
	var row yearlyTotalsRow
	// Build SQL conditionally (simpler than juggling placeholders across drivers)
	if accountID != nil {
		sql := `
		  SELECT
		    $3::int AS year,
		    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
		    COALESCE(SUM(amount),0)::text                                                         AS net_text,
		    COALESCE(COUNT(DISTINCT date_trunc('month', txn_date)),0)                            AS active_months
		  FROM transactions
		  WHERE user_id = $1
		    AND account_id = $2
		    AND is_adjustment = false
		    AND txn_date >= make_date($3,1,1) AND txn_date < make_date($3+1,1,1)
		`
		if err := tx.Raw(sql, userID, *accountID, year).Scan(&row).Error; err != nil {
			return row, err
		}
	} else {
		sql := `
		  SELECT
		    $2::int AS year,
		    COALESCE(SUM(CASE WHEN transaction_type='income'  THEN amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN transaction_type='expense' THEN -amount ELSE 0 END),0)::text AS outflow_text,
		    COALESCE(SUM(amount),0)::text                                                         AS net_text,
		    COALESCE(COUNT(DISTINCT date_trunc('month', txn_date)),0)                            AS active_months
		  FROM transactions
		  WHERE user_id = $1
		    AND is_adjustment = false
		    AND txn_date >= make_date($2,1,1) AND txn_date < make_date($2+1,1,1)
		`
		if err := tx.Raw(sql, userID, year).Scan(&row).Error; err != nil {
			return row, err
		}
	}
	return row, nil
}

func (r *StatisticsRepository) FetchYearlyCategoryTotals(
	tx *gorm.DB, userID int64, accountID *int64, year int,
) ([]yearlyCategoryRow, error) {
	rows := []yearlyCategoryRow{}
	if accountID != nil {
		sql := `
		  SELECT
		    $3::int AS year,
		    t.category_id,
		    c.display_name,
		    COALESCE(SUM(CASE WHEN t.transaction_type='income'  THEN t.amount ELSE 0 END),0)::text  AS inflow_text,
		    COALESCE(SUM(CASE WHEN t.transaction_type='expense' THEN -t.amount ELSE 0 END),0)::text AS outflow_text,
		    COALESCE(SUM(t.amount),0)::text                                                         AS net_text
		  FROM transactions t
		  LEFT JOIN categories c ON c.id = t.category_id
		  WHERE t.user_id = $1
		    AND t.account_id = $2
		    AND t.is_adjustment = false
		    AND t.txn_date >= make_date($3,1,1) AND t.txn_date < make_date($3+1,1,1)
		  GROUP BY t.category_id, c.display_name
		  ORDER BY t.category_id NULLS LAST
		`
		if err := tx.Raw(sql, userID, *accountID, year).Scan(&rows).Error; err != nil {
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
            COALESCE(SUM(t.amount),0)::text                                                         AS net_text
		  FROM transactions t
		  LEFT JOIN categories c ON c.id = t.category_id
		  WHERE t.user_id = $1
		    AND t.is_adjustment = false
		    AND t.txn_date >= make_date($2,1,1) AND t.txn_date < make_date($2+1,1,1)
		  GROUP BY t.category_id, c.display_name
		  ORDER BY t.category_id NULLS LAST
		`
		if err := tx.Raw(sql, userID, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}
