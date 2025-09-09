package repositories

import (
	"database/sql"
	"errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type AccountRepository struct {
	DB *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{DB: db}
}

func (r *AccountRepository) FindAccounts(userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeInactive bool) ([]models.Account, error) {

	var records []models.Account

	query := r.DB.
		Preload("AccountType").
		Preload("Balance").
		Where("user_id = ?", userID).
		Where("deleted_At is NULL")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "accounts", sortField, sortOrder)

	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) CountAccounts(userID int64, filters []utils.Filter, includeInactive bool) (int64, error) {
	var totalRecords int64

	query := r.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Where("deleted_At is NULL")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *AccountRepository) FindAllAccounts(tx *gorm.DB, userID int64, includeInactive bool) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.Account
	query := r.DB.Where("user_id = ?", userID).
		Where("deleted_at is NULL")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) FindAllAccountTypes(tx *gorm.DB, userID *int64) ([]models.AccountType, error) {

	db := tx
	if db == nil {
		db = r.DB
	}

	var records []models.AccountType
	result := r.DB.Find(&records)
	return records, result.Error
}

func (r *AccountRepository) FindAccountByID(tx *gorm.DB, ID, userID int64, withBalance bool) (models.Account, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Account

	query := db.Where("id = ? AND user_id = ?", ID, userID).
		Preload("AccountType")

	if withBalance {
		query = query.Preload("Balance")
	}

	result := query.First(&record)
	return record, result.Error
}

func (r *AccountRepository) FindAccountTypeByID(tx *gorm.DB, ID int64) (models.AccountType, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.AccountType
	result := db.Where("id = ?", ID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) FindBalanceForAccountID(tx *gorm.DB, accID int64) (models.Balance, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Balance
	result := db.Where("account_id = ?", accID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) InsertAccount(tx *gorm.DB, newRecord *models.Account) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) UpdateAccount(tx *gorm.DB, record *models.Account) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	updates := map[string]interface{}{}
	if record.Name != "" {
		updates["name"] = record.Name
	}
	if record.Currency != "" {
		updates["currency"] = record.Currency
	}
	if record.AccountTypeID != 0 {
		updates["account_type_id"] = record.AccountTypeID
	}
	updates["is_active"] = record.IsActive

	db.Model(&models.Account{}).Where("id = ?", record.ID).Updates(updates)

	return record.ID, nil
}

func (r *AccountRepository) InsertBalance(tx *gorm.DB, newRecord *models.Balance) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) UpdateBalance(tx *gorm.DB, record models.Balance) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.Balance{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"as_of":             record.AsOf,
			"start_balance":     record.StartBalance,
			"cash_inflows":      record.CashInflows,
			"cash_outflows":     record.CashOutflows,
			"non_cash_inflows":  record.NonCashInflows,
			"non_cash_outflows": record.NonCashOutflows,
			"net_market_flows":  record.NetMarketFlows,
			"adjustments":       record.Adjustments,
			"currency":          record.Currency,
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *AccountRepository) CloseAccount(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.Account{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"is_active":  false,
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *AccountRepository) GetAccountOpening(tx *gorm.DB, accountID int64) (time.Time, decimal.Decimal, error) {
	var asOf *time.Time
	var endBalStr *string

	// earliest balance
	err := tx.Raw(`
        SELECT as_of::date, end_balance::text
        FROM balances
        WHERE account_id = ?
        ORDER BY as_of ASC
        LIMIT 1
    `, accountID).Row().Scan(&asOf, &endBalStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && err != sql.ErrNoRows {
		return time.Time{}, decimal.Zero, err
	}

	// fallback opening date: earliest txn date or today
	var firstTxn *time.Time
	if err2 := tx.Raw(`
        SELECT MIN(txn_date)::date
        FROM transactions
        WHERE account_id = ? AND deleted_at IS NULL
    `, accountID).Row().Scan(&firstTxn); err2 != nil && err2 != sql.ErrNoRows {
		return time.Time{}, decimal.Zero, err2
	}

	today := time.Now().Truncate(24 * time.Hour)
	openingDate := today
	if asOf != nil && !asOf.IsZero() && asOf.Before(openingDate) {
		openingDate = *asOf
	}
	if firstTxn != nil && !firstTxn.IsZero() && firstTxn.Before(openingDate) {
		openingDate = *firstTxn
	}

	openingBal := decimal.Zero
	if endBalStr != nil {
		if v, e := decimal.NewFromString(*endBalStr); e == nil {
			openingBal = v
		}
	}

	return openingDate, openingBal, nil
}

// daily net deltas (income - expense) for [start..end] inclusive
func (r *AccountRepository) GetDailyTxnNet(tx *gorm.DB, accountID int64, start, end time.Time) (map[time.Time]decimal.Decimal, error) {
	type row struct {
		AsOf   time.Time
		Amount string
	}
	var rows []row
	err := tx.Raw(`
        SELECT DATE(txn_date) AS as_of,
               SUM(CASE WHEN transaction_type = 'expense' THEN -amount ELSE amount END)::text AS amount
        FROM transactions
        WHERE account_id = ?
          AND deleted_at IS NULL
          AND txn_date::date BETWEEN ? AND ?
        GROUP BY DATE(txn_date)
        ORDER BY DATE(txn_date)
    `, accountID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[time.Time]decimal.Decimal, len(rows))
	for _, r := range rows {
		v, _ := decimal.NewFromString(r.Amount)
		// normalize date to midnight
		out[r.AsOf.Truncate(24*time.Hour)] = v
	}
	return out, nil
}

// batch upsert snapshots for one account
func (r *AccountRepository) UpsertAccountSnapshots(tx *gorm.DB, rows []models.AccountDailySnapshot) error {
	if len(rows) == 0 {
		return nil
	}
	// GORM Upsert
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "account_id"}, {Name: "as_of"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"user_id":     gorm.Expr("EXCLUDED.user_id"),
			"currency":    gorm.Expr("EXCLUDED.currency"),
			"end_balance": gorm.Expr("EXCLUDED.end_balance"),
			"computed_at": gorm.Expr("NOW()"),
		}),
	}).Create(&rows).Error
}

// user-level helpers for default date range
func (r *AccountRepository) GetUserFirstBalanceDate(tx *gorm.DB, userID int64) (time.Time, error) {
	var d *time.Time
	err := tx.Raw(`
        SELECT MIN(b.as_of)::date
        FROM balances b
        JOIN accounts a ON a.id = b.account_id
        WHERE a.user_id = ? AND a.deleted_at IS NULL
    `, userID).Row().Scan(&d)
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}
	if d == nil {
		return time.Time{}, nil
	}
	return d.Truncate(24 * time.Hour), nil
}

func (r *AccountRepository) GetUserFirstTxnDate(tx *gorm.DB, userID int64) (time.Time, error) {
	var d *time.Time
	err := tx.Raw(`
        SELECT MIN(t.txn_date)::date
        FROM transactions t
        JOIN accounts a ON a.id = t.account_id
        WHERE a.user_id = ? AND t.deleted_at IS NULL AND a.deleted_at IS NULL
    `, userID).Row().Scan(&d)
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}
	if d == nil {
		return time.Time{}, nil
	}
	return d.Truncate(24 * time.Hour), nil
}
