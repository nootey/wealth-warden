package repositories

import (
	"fmt"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type StatisticsRepository struct {
	DB *gorm.DB
}

func NewStatisticsRepository(db *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{DB: db}
}

func (r *StatisticsRepository) FetchYearlyTotals(
	tx *gorm.DB, userID int64, accountID *int64, year int,
) (models.YearlyTotalsRow, error) {
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
		`
		if err := tx.Raw(sql, userID, year).Scan(&row).Error; err != nil {
			return row, err
		}
	}
	return row, nil
}

func (r *StatisticsRepository) FetchYearlyCategoryTotals(
	tx *gorm.DB, userID int64, accountID *int64, year int,
) ([]models.YearlyCategoryRow, error) {
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
		  GROUP BY t.category_id, c.display_name
		  ORDER BY t.category_id NULLS LAST
		`
		if err := tx.Raw(sql, userID, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *StatisticsRepository) FetchMonthlyTotals(
	tx *gorm.DB, userID int64, accountID *int64, year int,
) ([]models.MonthlyTotalsRow, error) {
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
	  GROUP BY month
	  ORDER BY month;
	`

	if accountID != nil {
		sql := fmt.Sprintf(base, "AND account_id = ?")
		if err := tx.Raw(sql, userID, *accountID, year, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	} else {
		sql := fmt.Sprintf(base, "")
		if err := tx.Raw(sql, userID, year, year).Scan(&rows).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *StatisticsRepository) FetchMonthlyTotalsCheckingOnly(
	tx *gorm.DB, userID int64, accountIDs []int64, year int,
) ([]models.MonthlyTotalsRow, error) {
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
	  GROUP BY month
	  ORDER BY month;
	`

	err := tx.Raw(query, userID, year, year, accountIDs).Scan(&rows).Error
	return rows, err
}

func (r *StatisticsRepository) GetAvailableStatsYears(accID *int64, userID int64) ([]int64, error) {
	var (
		query string
		args  []any
	)

	if accID == nil {
		query = `
			SELECT DISTINCT EXTRACT(YEAR FROM txn_date)::int AS year
			FROM transactions
			WHERE user_id = ?
			ORDER BY year;
		`
		args = []any{userID}
	} else {
		query = `
			SELECT DISTINCT EXTRACT(YEAR FROM txn_date)::int AS year
			FROM transactions
			WHERE user_id = ?
			  AND account_id = ?
			ORDER BY year;
		`
		args = []any{userID, *accID}
	}

	var years []int64
	if err := r.DB.Raw(query, args...).Scan(&years).Error; err != nil {
		return nil, fmt.Errorf("querying available stats years: %w", err)
	}
	return years, nil
}
