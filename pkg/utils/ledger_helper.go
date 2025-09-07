package utils

import (
	"database/sql"
	"fmt"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
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

func ValidateAccount(acc *models.Account, role string) error {
	if acc.DeletedAt != nil {
		return fmt.Errorf("%s account (ID=%d) is closed and cannot be used", role, acc.ID)
	}
	if !acc.IsActive {
		return fmt.Errorf("%s account (ID=%d) is inactive and cannot be used", role, acc.ID)
	}
	return nil
}
