package repositories

import (
	"database/sql"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountRepository struct {
	DB *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{DB: db}
}

func (r *AccountRepository) FindAccounts(
	userID int64,
	offset, limit int,
	sortField, sortOrder string,
	filters []utils.Filter,
	includeInactive bool,
	classification *string,
) ([]models.Account, error) {

	var accounts []models.Account

	// Base query: accounts + preload AccountType
	q := r.DB.
		Model(&models.Account{}).
		Preload("AccountType").
		Where("user_id = ? AND closed_at IS NULL", userID)

	if classification != nil && *classification != "" {
		q = q.Joins("JOIN account_types at ON at.id = accounts.account_type_id").
			Where("at.classification = ?", *classification)
	}

	if !includeInactive {
		q = q.Where("is_active = TRUE")
	}

	// Apply filters
	joins := utils.GetRequiredJoins(filters)
	for _, j := range joins {
		q = q.Joins(j)
	}
	q = utils.ApplyFilters(q, filters)

	orderBy := utils.ConstructOrderByClause(&joins, "accounts", sortField, sortOrder)

	// fetch accounts
	if err := q.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&accounts).Error; err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return accounts, nil
	}

	// fetch latest balances for all accounts
	accountIDs := make([]int64, len(accounts))
	for i, acc := range accounts {
		accountIDs[i] = acc.ID
	}

	var latestBalances []models.Balance
	if err := r.DB.Raw(`
		SELECT DISTINCT ON (account_id) *
		FROM balances
		WHERE account_id IN ?
		ORDER BY account_id, as_of DESC
	`, accountIDs).Scan(&latestBalances).Error; err != nil {
		return nil, err
	}

	// map balances back into accounts
	balanceMap := make(map[int64]models.Balance, len(latestBalances))
	for _, b := range latestBalances {
		balanceMap[b.AccountID] = b
	}

	for i := range accounts {
		if b, ok := balanceMap[accounts[i].ID]; ok {
			accounts[i].Balance = b
		}
	}

	return accounts, nil
}

func (r *AccountRepository) CountAccounts(userID int64, filters []utils.Filter, includeInactive bool, classification *string) (int64, error) {
	var totalRecords int64

	query := r.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Where("closed_at is NULL")

	if classification != nil && *classification != "" {
		query = query.Joins("JOIN account_types at ON at.id = accounts.account_type_id").
			Where("at.classification = ?", *classification)
	}

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
		Where("closed_at is NULL")

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

func (r *AccountRepository) FindAccountByID(tx *gorm.DB, ID, userID int64, withBalance bool) (*models.Account, error) {
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
	return &record, result.Error
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
		Where("id = ? AND user_id = ? AND closed_at IS NULL", id, userID).
		Updates(map[string]any{
			"is_active":  false,
			"closed_at":  time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *AccountRepository) EnsureDailyBalanceRow(
	tx *gorm.DB, accountID int64, asOf time.Time, currency string,
) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	asOf = asOf.UTC().Truncate(24 * time.Hour)

	// Insert-if-missing with proper start_balance derived from previous end_balance
	return db.Exec(`
        WITH prev AS (
            SELECT end_balance
            FROM balances
            WHERE account_id = ? AND as_of < ?
            ORDER BY as_of DESC
            LIMIT 1
        )
        INSERT INTO balances (
            account_id, as_of, start_balance,
            cash_inflows, cash_outflows, non_cash_inflows, non_cash_outflows,
            net_market_flows, adjustments, currency, created_at, updated_at
        )
        VALUES (
            ?, ?, COALESCE((SELECT end_balance FROM prev), 0),
            0, 0, 0, 0,
            0, 0, ?, NOW(), NOW()
        )
        ON CONFLICT (account_id, as_of) DO NOTHING
    `, accountID, asOf, accountID, asOf, currency).Error
}

func (r *AccountRepository) AddToDailyBalance(
	tx *gorm.DB, accountID int64, asOf time.Time, field string, amt decimal.Decimal,
) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	asOf = asOf.UTC().Truncate(24 * time.Hour)

	// guard: only allow the expected columns
	switch field {
	case "cash_inflows", "cash_outflows", "non_cash_inflows", "non_cash_outflows", "net_market_flows", "adjustments":
	default:
		return fmt.Errorf("invalid balance field %q", field)
	}

	return db.Exec(fmt.Sprintf(`
        UPDATE balances
        SET %s = %s + ?, updated_at = NOW()
        WHERE account_id = ? AND as_of = ?
    `, field, field), amt, accountID, asOf).Error
}

func (r *AccountRepository) GetDailyBalances(
	tx *gorm.DB, accountID int64, from, to time.Time,
) (map[string]decimal.Decimal, error) {
	type row struct {
		AsOf  time.Time
		Value string
	}

	fromUTC := from.UTC().Truncate(24 * time.Hour)
	toUTC := to.UTC().Truncate(24 * time.Hour)

	var rows []row
	err := tx.Raw(`
        SELECT as_of, end_balance::text
        FROM balances
        WHERE account_id = ? AND as_of BETWEEN ? AND ?
        ORDER BY as_of
    `, accountID, fromUTC, toUTC).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[string]decimal.Decimal, len(rows))
	for _, r := range rows {
		v, _ := decimal.NewFromString(r.Value)
		k := r.AsOf.UTC().Format("2006-01-02")
		out[k] = v
	}
	return out, nil
}

func (r *AccountRepository) UpsertSnapshotsFromBalances(
	tx *gorm.DB,
	userID, accountID int64,
	currency string,
	from, to time.Time,
) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	from = from.UTC().Truncate(24 * time.Hour)
	to = to.UTC().Truncate(24 * time.Hour)

	// One-shot insert/update using generate_series and "last balance <= day"
	return db.Exec(`
		INSERT INTO account_daily_snapshots (
			user_id, account_id, as_of, end_balance, currency, computed_at
		)
		SELECT
			?::bigint        AS user_id,
			?::bigint        AS account_id,
			d.day            AS as_of,
			COALESCE(lb.end_balance, 0)::numeric(19,4) AS end_balance,
			?::char(3)       AS currency,
			NOW()            AS computed_at
		FROM generate_series(?::date, ?::date, '1 day') AS d(day)
		LEFT JOIN LATERAL (
			SELECT b.end_balance
			FROM balances b
			WHERE b.account_id = ? AND b.as_of::date <= d.day
			ORDER BY b.as_of DESC
			LIMIT 1
		) lb ON TRUE
		ON CONFLICT (account_id, as_of) DO UPDATE
		SET user_id     = EXCLUDED.user_id,
			currency    = EXCLUDED.currency,
			end_balance = EXCLUDED.end_balance,
			computed_at = NOW();
	`, userID, accountID, currency, from, to, accountID).Error
}

func (r *AccountRepository) GetUserFirstBalanceDate(tx *gorm.DB, userID int64) (time.Time, error) {
	var d *time.Time
	err := tx.Raw(`
        SELECT MIN(b.as_of)::date
        FROM balances b
        JOIN accounts a ON a.id = b.account_id
        WHERE a.user_id = ?
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
        WHERE a.user_id = ? AND t.deleted_at IS NULL
    `, userID).Row().Scan(&d)
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}
	if d == nil {
		return time.Time{}, nil
	}
	return d.Truncate(24 * time.Hour), nil
}

func (r *AccountRepository) GetAccountOpeningAsOf(
	tx *gorm.DB, accountID int64,
) (time.Time, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	// MIN(as_of) is the opening day; if no balance rows exist, return sql.ErrNoRows
	var open *time.Time
	if err := db.Raw(`
        SELECT MIN(as_of) FROM balances WHERE account_id = ?
    `, accountID).Scan(&open).Error; err != nil {
		return time.Time{}, err
	}
	if open == nil {
		return time.Time{}, sql.ErrNoRows
	}
	// normalize to UTC midnight
	t := open.UTC().Truncate(24 * time.Hour)
	return t, nil
}
