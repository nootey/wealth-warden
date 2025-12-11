package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindAccounts(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeInactive bool, classification *string) ([]models.Account, error)
	CountAccounts(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, includeInactive bool, classification *string) (int64, error)
	FindAllAccounts(ctx context.Context, tx *gorm.DB, userID int64, includeInactive bool) ([]models.Account, error)
	FindAllAccountTypes(ctx context.Context, tx *gorm.DB, userID *int64) ([]models.AccountType, error)
	FindAccountsBySubtype(ctx context.Context, tx *gorm.DB, userID int64, subtype string, activeOnly bool) ([]models.Account, error)
	FetchAccountsByType(ctx context.Context, tx *gorm.DB, userID int64, t string, activeOnly bool) ([]models.Account, error)
	FindAccountsByImportID(ctx context.Context, tx *gorm.DB, ID, userID int64) ([]models.Account, error)
	FindAccountByID(ctx context.Context, tx *gorm.DB, ID, userID int64, withBalance bool) (*models.Account, error)
	FindAccountByName(ctx context.Context, tx *gorm.DB, userID int64, name string) (*models.Account, error)
	FindAccountTypeByAccID(ctx context.Context, tx *gorm.DB, accID, userID int64) (*models.AccountType, error)
	FindAllAccountsWithLatestBalance(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error)
	FindAccountByIDWithInitialBalance(ctx context.Context, tx *gorm.DB, ID, userID int64) (*models.Account, error)
	FindAccountTypeByID(ctx context.Context, tx *gorm.DB, ID int64) (models.AccountType, error)
	FindAccountTypeByType(ctx context.Context, tx *gorm.DB, atype, sub_type string) (models.AccountType, error)
	FindBalanceForAccountID(ctx context.Context, tx *gorm.DB, accID int64) (models.Balance, error)
	InsertAccount(ctx context.Context, tx *gorm.DB, newRecord *models.Account) (int64, error)
	UpdateAccount(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error)
	UpdateAccountProjection(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error)
	FindEarliestTransactionDate(ctx context.Context, tx *gorm.DB, accountID int64) (*time.Time, error)
	InsertBalance(ctx context.Context, tx *gorm.DB, newRecord *models.Balance) (int64, error)
	UpdateBalance(ctx context.Context, tx *gorm.DB, record models.Balance) (int64, error)
	CloseAccount(ctx context.Context, tx *gorm.DB, id, userID int64) error
	PurgeImportedAccounts(ctx context.Context, tx *gorm.DB, importID, userID int64) error
	EnsureDailyBalanceRow(ctx context.Context, tx *gorm.DB, accountID int64, asOf time.Time, currency string) error
	AddToDailyBalance(ctx context.Context, tx *gorm.DB, accountID int64, asOf time.Time, field string, amt decimal.Decimal) error
	UpsertSnapshotsFromBalances(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from, to time.Time) error
	GetUserFirstBalanceDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error)
	GetUserFirstTxnDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error)
	GetAccountOpeningAsOf(ctx context.Context, tx *gorm.DB, accountID int64) (time.Time, error)
	FrontfillBalances(ctx context.Context, tx *gorm.DB, accountID int64, currency string, from time.Time) error
	DeleteAccountSnapshots(ctx context.Context, tx *gorm.DB, accountID int64) error
	FindLatestBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (*models.Balance, error)
}

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

var _ AccountRepositoryInterface = (*AccountRepository)(nil)

func (r *AccountRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *AccountRepository) FindAccounts(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeInactive bool, classification *string) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var accounts []models.Account
	q := db.Model(&models.Account{}).
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
	if err := r.db.WithContext(ctx).Raw(`
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

func (r *AccountRepository) CountAccounts(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, includeInactive bool, classification *string) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	query := db.Model(&models.Account{}).
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

func (r *AccountRepository) FindAllAccounts(ctx context.Context, tx *gorm.DB, userID int64, includeInactive bool) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account
	query := db.Where("user_id = ?", userID).
		Where("closed_at is NULL")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) FindAllAccountTypes(ctx context.Context, tx *gorm.DB, userID *int64) ([]models.AccountType, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.AccountType
	result := db.Find(&records)
	return records, result.Error
}

func (r *AccountRepository) FindAccountsBySubtype(ctx context.Context, tx *gorm.DB, userID int64, subtype string, activeOnly bool) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account

	query := db.
		Model(&models.Account{}).
		Joins(`JOIN account_types AS at ON at.id = accounts.account_type_id`).
		Where(`at.sub_type = ? AND accounts.user_id = ?`, subtype, userID)

	if activeOnly {
		query = query.Where(`accounts.is_active = ?`, true)
	}

	err := query.
		Select("accounts.*").
		Preload("AccountType").
		Find(&records).
		Error

	return records, err
}

func (r *AccountRepository) FetchAccountsByType(ctx context.Context, tx *gorm.DB, userID int64, t string, activeOnly bool) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account

	query := db.
		Model(&models.Account{}).
		Joins(`JOIN account_types AS at ON at.id = accounts.account_type_id`).
		Where(`at.type = ? AND accounts.user_id = ?`, t, userID)

	if activeOnly {
		query = query.Where(`accounts.is_active = ?`, true)
	}

	err := query.
		Select("accounts.*").
		Preload("AccountType").
		Find(&records).
		Error

	return records, err
}

func (r *AccountRepository) FindAccountsByImportID(ctx context.Context, tx *gorm.DB, ID, userID int64) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account

	query := db.
		Model(&models.Account{}).
		Where(`import_id = ? AND user_id = ?`, ID, userID)

	err := query.Find(&records).Error

	return records, err
}

func (r *AccountRepository) FindAccountByID(ctx context.Context, tx *gorm.DB, ID, userID int64, withBalance bool) (*models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Account
	query := db.Where("id = ? AND user_id = ? AND closed_at IS NULL AND is_active = true", ID, userID).
		Preload("AccountType")

	if withBalance {
		query = query.Preload("Balance", func(db *gorm.DB) *gorm.DB {
			return db.Order("as_of desc").Limit(1)
		})
	}

	result := query.First(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Check if account exists but is closed / not active
		var closedAccount models.Account
		err := db.Where("id = ? AND user_id = ?", ID, userID).First(&closedAccount).Error
		if err == nil && closedAccount.ClosedAt != nil {
			return nil, fmt.Errorf("account is closed")
		}
		if err == nil && !closedAccount.IsActive {
			return nil, fmt.Errorf("account is not active")
		}
	}

	return &record, result.Error
}

func (r *AccountRepository) FindAccountByName(ctx context.Context, tx *gorm.DB, userID int64, name string) (*models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Account
	query := db.Where("name = ? AND user_id = ? AND closed_at IS NULL AND is_active = true", name, userID).
		Preload("AccountType").
		Preload("Balance", func(db *gorm.DB) *gorm.DB {
			return db.Order("as_of desc").Limit(1)
		})

	result := query.First(&record)
	return &record, result.Error
}

func (r *AccountRepository) FindAccountTypeByAccID(ctx context.Context, tx *gorm.DB, accID, userID int64) (*models.AccountType, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var accountType models.AccountType
	err := db.Table("account_types").
		Joins("JOIN accounts ON accounts.account_type_id = account_types.id").
		Where("accounts.id = ? AND accounts.user_id = ?", accID, userID).
		First(&accountType).Error

	return &accountType, err
}

func (r *AccountRepository) FindAllAccountsWithLatestBalance(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var accounts []models.Account

	if err := db.
		Where("user_id = ? AND is_active = ?", userID, true).
		Preload("AccountType").
		Find(&accounts).Error; err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return accounts, nil
	}

	ids := make([]int64, 0, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.ID)
	}

	var bals []models.Balance
	if err := db.
		Where("account_id IN ?", ids).
		Order("account_id DESC, as_of DESC").
		Find(&bals).Error; err != nil {
		return nil, err
	}

	earliest := make(map[int64]models.Balance, len(ids))
	for _, b := range bals {
		if _, ok := earliest[b.AccountID]; !ok {
			earliest[b.AccountID] = b
		}
	}

	for i := range accounts {
		if b, ok := earliest[accounts[i].ID]; ok {
			accounts[i].Balance = b
		}
	}

	return accounts, nil
}

func (r *AccountRepository) FindAccountByIDWithInitialBalance(ctx context.Context, tx *gorm.DB, ID, userID int64) (*models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Account

	query := db.Where("id = ? AND user_id = ?", ID, userID).
		Preload("AccountType").
		Preload("Balance", func(db *gorm.DB) *gorm.DB {
			return db.Order("as_of asc").Limit(1)
		})

	result := query.First(&record)
	return &record, result.Error
}

func (r *AccountRepository) FindAccountTypeByID(ctx context.Context, tx *gorm.DB, ID int64) (models.AccountType, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.AccountType
	result := db.Where("id = ?", ID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) FindAccountTypeByType(ctx context.Context, tx *gorm.DB, atype, sub_type string) (models.AccountType, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.AccountType
	result := db.Where("type = ? AND sub_type =?", atype, sub_type).First(&record)
	return record, result.Error
}

func (r *AccountRepository) FindBalanceForAccountID(ctx context.Context, tx *gorm.DB, accID int64) (models.Balance, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Balance
	result := db.Where("account_id = ?", accID).First(&record)
	return record, result.Error
}

func (r *AccountRepository) InsertAccount(ctx context.Context, tx *gorm.DB, newRecord *models.Account) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) UpdateAccount(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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
	if !record.OpenedAt.IsZero() {
		updates["opened_at"] = record.OpenedAt
	}
	updates["is_active"] = record.IsActive
	updates["updated_at"] = time.Now().UTC()

	db.Model(&models.Account{}).Where("id = ?", record.ID).Updates(updates)

	return record.ID, nil
}

func (r *AccountRepository) UpdateAccountProjection(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	updates := map[string]interface{}{}

	if record.BalanceProjection != "" {
		updates["balance_projection"] = record.BalanceProjection
	}

	if !record.ExpectedBalance.IsZero() || record.ExpectedBalance.Equal(decimal.NewFromInt(0)) {
		updates["expected_balance"] = record.ExpectedBalance
	}
	updates["updated_at"] = time.Now().UTC()

	result := db.Model(&models.Account{}).Where("id = ?", record.ID).Updates(updates)
	if result.Error != nil {
		return 0, result.Error
	}

	return record.ID, nil
}

func (r *AccountRepository) FindEarliestTransactionDate(ctx context.Context, tx *gorm.DB, accountID int64) (*time.Time, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var result struct {
		TxnDate time.Time
	}

	err := db.Model(&models.Transaction{}).
		Where("account_id = ?", accountID).
		Order("txn_date ASC").
		Limit(1).
		Select("txn_date").
		First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No transactions found - return nil
			return nil, nil
		}
		return nil, err
	}

	return &result.TxnDate, nil
}

func (r *AccountRepository) InsertBalance(ctx context.Context, tx *gorm.DB, newRecord *models.Balance) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *AccountRepository) UpdateBalance(ctx context.Context, tx *gorm.DB, record models.Balance) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *AccountRepository) CloseAccount(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Account{}).
		Where("id = ? AND user_id = ? AND closed_at IS NULL", id, userID).
		Updates(map[string]any{
			"is_active":  false,
			"closed_at":  time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *AccountRepository) PurgeImportedAccounts(ctx context.Context, tx *gorm.DB, importID, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Exec("SET LOCAL ww.hard_delete = 'on'").Error; err != nil {
		return err
	}

	// delete balances for these accounts
	res := db.Exec(`
        DELETE FROM balances
        WHERE account_id IN (
            SELECT id FROM accounts 
            WHERE user_id = ? AND import_id = ?
        )
    `, userID, importID)
	if res.Error != nil {
		return fmt.Errorf("failed to delete balances: %w", res.Error)
	}

	// delete snapshots for these accounts
	res = db.Exec(`
        DELETE FROM account_daily_snapshots
        WHERE user_id = ? AND account_id IN (
            SELECT id FROM accounts 
            WHERE user_id = ? AND import_id = ?
        )
    `, userID, userID, importID)
	if res.Error != nil {
		return fmt.Errorf("failed to delete snapshots: %w", res.Error)
	}

	// delete the accounts themselves
	res = db.Exec(`
        DELETE FROM accounts
        WHERE user_id = ? AND import_id = ?
    `, userID, importID)

	return res.Error
}

func (r *AccountRepository) EnsureDailyBalanceRow(ctx context.Context, tx *gorm.DB, accountID int64, asOf time.Time, currency string) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	asOf = asOf.UTC().Truncate(24 * time.Hour)
	return db.Exec(`
        WITH prev AS (
            SELECT end_balance
            FROM balances
            WHERE account_id = ? AND as_of < ?
            ORDER BY as_of DESC
            LIMIT 1
        ),
        nxt AS (
            -- earliest future row (used when there is no previous row)
            SELECT start_balance
            FROM balances
            WHERE account_id = ? AND as_of > ?
            ORDER BY as_of ASC
            LIMIT 1
        )
        INSERT INTO balances (
            account_id, as_of, start_balance,
            cash_inflows, cash_outflows, non_cash_inflows, non_cash_outflows,
            net_market_flows, adjustments, currency, created_at, updated_at
        )
        VALUES (
            ?, ?, COALESCE((SELECT end_balance FROM prev),
                           (SELECT start_balance FROM nxt), 0),
            0, 0, 0, 0,
            0, 0, ?, NOW(), NOW()
        )
        ON CONFLICT (account_id, as_of) DO NOTHING
    `, accountID, asOf, accountID, asOf, accountID, asOf, currency).Error
}

func (r *AccountRepository) AddToDailyBalance(ctx context.Context, tx *gorm.DB, accountID int64, asOf time.Time, field string, amt decimal.Decimal) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *AccountRepository) UpsertSnapshotsFromBalances(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from, to time.Time) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *AccountRepository) GetUserFirstBalanceDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var d *time.Time
	err := db.Raw(`
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

func (r *AccountRepository) GetUserFirstTxnDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var d *time.Time
	err := db.Raw(`
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

func (r *AccountRepository) GetAccountOpeningAsOf(ctx context.Context, tx *gorm.DB, accountID int64) (time.Time, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

func (r *AccountRepository) FrontfillBalances(ctx context.Context, tx *gorm.DB, accountID int64, currency string, from time.Time) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	from = from.UTC().Truncate(24 * time.Hour)

	return db.Exec(`
		WITH params AS (
		  SELECT ?::bigint AS account_id, ?::date AS from_date
		),
		base AS (
		  SELECT COALESCE(
			-- prefer exact 'from' day end_balance if a row exists
			(SELECT b.end_balance
			 FROM balances b, params p
			 WHERE b.account_id = p.account_id
			   AND b.as_of = p.from_date
			 LIMIT 1),
			-- otherwise, last end_balance before 'from'
			(SELECT b.end_balance
			 FROM balances b, params p
			 WHERE b.account_id = p.account_id
			   AND b.as_of < p.from_date
			 ORDER BY b.as_of DESC
			 LIMIT 1),
			0
		  )::numeric(19,4) AS base_end
		),
		series AS (
		  SELECT
			b.account_id,
			b.as_of,
			( b.cash_inflows
			- b.cash_outflows
			+ b.non_cash_inflows
			- b.non_cash_outflows
			+ b.net_market_flows
			+ b.adjustments )::numeric(19,4) AS delta
		  FROM balances b, params p
		  WHERE b.account_id = p.account_id
			AND b.as_of > p.from_date
		  ORDER BY b.as_of
		),
		chain AS (
		  SELECT
			s.account_id,
			s.as_of,
			( SELECT base_end FROM base )
			+ ( SUM(s.delta) OVER (ORDER BY s.as_of
								   ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)
				- s.delta ) AS new_start
		  FROM series s
		)
		UPDATE balances b
		SET start_balance = c.new_start,
			updated_at    = NOW()
		FROM chain c
		WHERE b.account_id = c.account_id
		  AND b.as_of      = c.as_of;
	`, accountID, from).Error
}

func (r *AccountRepository) DeleteAccountSnapshots(ctx context.Context, tx *gorm.DB, accountID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("account_id = ?", accountID).
		Delete(&models.AccountDailySnapshot{}).Error
}

func (r *AccountRepository) FindLatestBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (*models.Balance, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var balance models.Balance
	err := db.Where("account_id = ?", accountID).
		Order("as_of DESC").
		Limit(1).
		First(&balance).Error

	if err != nil {
		return nil, err
	}

	return &balance, nil
}
