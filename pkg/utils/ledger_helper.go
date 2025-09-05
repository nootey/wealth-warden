package utils

import (
	"database/sql"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
)

func GetEndBalanceAsOf(tx *gorm.DB, accountID int64, t time.Time) (decimal.Decimal, error) {
	var endBalance sql.NullString

	err := tx.
		Raw(`
			SELECT end_balance::text
			FROM balances
			WHERE account_id = ? AND as_of = ?
		`, accountID, t.UTC().Format("2006-01-02")).
		Scan(&endBalance).Error
	if err != nil {
		return decimal.Zero, err
	}

	if !endBalance.Valid {
		// no row found → treat as 0
		return decimal.Zero, nil
	}

	d, err := decimal.NewFromString(endBalance.String)
	if err != nil {
		return decimal.Zero, err
	}

	return d, nil
}
