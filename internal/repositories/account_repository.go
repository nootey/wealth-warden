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
	FindAllAccounts(ctx context.Context, tx *gorm.DB, userID int64, includeInactive bool, includeAccountTypes bool) ([]models.Account, error)
	FindAllAccountTypes(ctx context.Context, tx *gorm.DB, userID *int64) ([]models.AccountType, error)
	FindAccountsBySubtype(ctx context.Context, tx *gorm.DB, userID int64, subtype string, activeOnly bool) ([]models.Account, error)
	FetchAccountsByType(ctx context.Context, tx *gorm.DB, userID int64, t string, activeOnly bool) ([]models.Account, error)
	FindAccountsByImportID(ctx context.Context, tx *gorm.DB, ID, userID int64) ([]models.Account, error)
	FindAccountByID(ctx context.Context, tx *gorm.DB, ID, userID int64, withBalance bool) (*models.Account, error)
	FindAccountByName(ctx context.Context, tx *gorm.DB, userID int64, name string) (*models.Account, error)
	FindAccountTypeByAccID(ctx context.Context, tx *gorm.DB, accID, userID int64) (*models.AccountType, error)
	FindAllAccountsWithLatestBalance(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error)
	FindAccountByIDWithOpening(ctx context.Context, tx *gorm.DB, ID, userID int64) (*models.Account, decimal.Decimal, error)
	FindAccountTypeByID(ctx context.Context, tx *gorm.DB, ID int64) (models.AccountType, error)
	FindAccountTypeByType(ctx context.Context, tx *gorm.DB, atype, sub_type string) (models.AccountType, error)
	InsertAccount(ctx context.Context, tx *gorm.DB, newRecord *models.Account) (int64, error)
	UpdateAccount(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error)
	UpdateAccountProjection(ctx context.Context, tx *gorm.DB, record *models.Account) (int64, error)
	FindEarliestTransactionDate(ctx context.Context, tx *gorm.DB, accountID int64) (*time.Time, error)
	CloseAccount(ctx context.Context, tx *gorm.DB, id, userID int64) error
	PurgeImportedAccounts(ctx context.Context, tx *gorm.DB, importID, userID int64) error
	PurgeAccount(ctx context.Context, tx *gorm.DB, accountID, userID int64) error
	FindAccountForPurge(ctx context.Context, tx *gorm.DB, accountID int64) (*models.Account, error)
	FindAllAccountsForRebuild(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error)
	FindAccountsForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.AccountLookup, error)
	GetUserFirstBalanceDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error)
	GetUserFirstTxnDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error)
	GetAccountOpeningAsOf(ctx context.Context, tx *gorm.DB, accountID int64) (time.Time, error)
	FindAccountsWithDefaults(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error)
	FindAccountTypesWithoutDefaults(ctx context.Context, tx *gorm.DB, userID int64) ([]models.AccountType, error)
	UpdateDefaultAccount(ctx context.Context, tx *gorm.DB, account models.Account, setAsDefault bool) error
	HasDefaultForAccountType(ctx context.Context, tx *gorm.DB, userID, accountTypeID int64) (bool, error)
}

type AccountRepository struct {
	db       *gorm.DB
	balances *BalanceRepository
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db, balances: NewBalanceRepository(db)}
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

	balanceMap, err := r.balances.findAccountBalancesByIDs(ctx, db, accountIDs)
	if err != nil {
		return nil, err
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

func (r *AccountRepository) FindAllAccounts(ctx context.Context, tx *gorm.DB, userID int64, includeInactive bool, includeAccountTypes bool) ([]models.Account, error) {
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

	if includeAccountTypes {
		query = query.Preload("AccountType")
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

	// Deliberately unfiltered by closed_at/is_active: the caller decides what
	// a closed or inactive account means for it via utils.ValidateAccount.
	result := db.Where("id = ? AND user_id = ?", ID, userID).
		Preload("AccountType").
		First(&record)

	if result.Error == nil && withBalance {
		bal, err := r.balances.findAccountBalanceByID(ctx, db, record.ID)
		if err != nil {
			return &record, err
		}
		record.Balance = bal
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
		Preload("AccountType")

	if err := query.First(&record).Error; err != nil {
		return &record, err
	}

	bal, err := r.balances.findAccountBalanceByID(ctx, db, record.ID)
	if err != nil {
		return &record, err
	}
	record.Balance = bal

	return &record, nil
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

	balanceMap, err := r.balances.findAccountBalancesByIDs(ctx, db, ids)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		if b, ok := balanceMap[accounts[i].ID]; ok {
			accounts[i].Balance = b
		}
	}

	return accounts, nil
}

func (r *AccountRepository) FindAccountByIDWithOpening(ctx context.Context, tx *gorm.DB, ID, userID int64) (*models.Account, decimal.Decimal, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Account

	result := db.Where("id = ? AND user_id = ?", ID, userID).
		Preload("AccountType").
		First(&record)
	if result.Error != nil {
		return &record, decimal.Zero, result.Error
	}

	var opening decimal.Decimal
	if err := db.Raw(`
		SELECT COALESCE(SUM(CASE WHEN direction = 'expense' THEN -amount ELSE amount END), 0)
		FROM transactions
		WHERE account_id = ? AND transaction_type = 'opening' AND deleted_at IS NULL
	`, record.ID).Scan(&opening).Error; err != nil {
		return &record, decimal.Zero, err
	}

	return &record, opening, nil
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
	updates["credit_limit"] = record.CreditLimit
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
        DELETE FROM balance_snapshots
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

func (r *AccountRepository) FindAccountForPurge(ctx context.Context, tx *gorm.DB, accountID int64) (*models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Account
	if err := db.Where("id = ?", accountID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AccountRepository) FindAccountsForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.AccountLookup, error) {

	db := tx
	if db == nil {
		db = r.db
	}

	var accounts []models.AccountLookup
	err := db.WithContext(ctx).
		Model(&models.Account{}).
		Select("id, name, currency, closed_at").
		Where("user_id = ?", userID).
		Order("name ASC").
		Find(&accounts).Error
	return accounts, err
}

func (r *AccountRepository) FindAllAccountsForRebuild(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account
	if err := db.Where("user_id = ?", userID).Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// Hard deletes one account and everything hanging off it. The soft-delete triggers
// on accounts, transactions and transfers all stand down for ww.hard_delete.
func (r *AccountRepository) PurgeAccount(ctx context.Context, tx *gorm.DB, accountID, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Exec("SET LOCAL ww.hard_delete = 'on'").Error; err != nil {
		return err
	}

	// Both legs of every transfer that touches the account. A surviving far leg
	// would point at a transfer that no longer exists.
	var legIDs []int64
	if err := db.Raw(`
        SELECT DISTINCT leg.txn_id
        FROM transfers tf
        CROSS JOIN LATERAL (VALUES (tf.transaction_inflow_id), (tf.transaction_outflow_id)) AS leg(txn_id)
        WHERE EXISTS (
            SELECT 1 FROM transactions t
            WHERE t.id IN (tf.transaction_inflow_id, tf.transaction_outflow_id)
              AND t.account_id = ?
        )
    `, accountID).Scan(&legIDs).Error; err != nil {
		return fmt.Errorf("failed to collect transfer legs: %w", err)
	}

	if len(legIDs) > 0 {
		// The far accounts survive the purge, so their balance has to lose the leg
		// that is about to go. Collected before the delete, recomputed after it.
		var farAccountIDs []int64
		if err := db.Raw(`
            SELECT DISTINCT account_id FROM transactions
            WHERE id IN ? AND account_id <> ?
        `, legIDs, accountID).Scan(&farAccountIDs).Error; err != nil {
			return fmt.Errorf("failed to collect far leg accounts: %w", err)
		}

		if err := db.Exec(`
            DELETE FROM transfers
            WHERE transaction_inflow_id IN ? OR transaction_outflow_id IN ?
        `, legIDs, legIDs).Error; err != nil {
			return fmt.Errorf("failed to delete transfers: %w", err)
		}

		if err := db.Exec(`DELETE FROM transactions WHERE id IN ?`, legIDs).Error; err != nil {
			return fmt.Errorf("failed to delete transfer legs: %w", err)
		}

		for _, farID := range farAccountIDs {
			if err := r.balances.RecomputeFromTransactions(ctx, db, farID); err != nil {
				return fmt.Errorf("failed to recompute account %d: %w", farID, err)
			}
		}
	}

	if err := db.Exec(`DELETE FROM transactions WHERE account_id = ?`, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete transactions: %w", err)
	}

	// A transfer template pointing at the account has no destination left.
	if err := db.Exec(`
        DELETE FROM transaction_templates WHERE account_id = ? OR to_account_id = ?
    `, accountID, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete transaction templates: %w", err)
	}

	// saving_contributions cascade off the goal.
	if err := db.Exec(`DELETE FROM saving_goals WHERE account_id = ?`, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete saving goals: %w", err)
	}

	// investment_income cascades off the asset, trades do not.
	if err := db.Exec(`
        DELETE FROM investment_trades
        WHERE asset_id IN (SELECT id FROM investment_assets WHERE account_id = ?)
    `, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete investment trades: %w", err)
	}

	if err := db.Exec(`DELETE FROM investment_assets WHERE account_id = ?`, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete investment assets: %w", err)
	}

	if err := db.Exec(`DELETE FROM balance_snapshots WHERE account_id = ?`, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete snapshots: %w", err)
	}

	if err := db.Exec(`DELETE FROM balances WHERE account_id = ?`, accountID).Error; err != nil {
		return fmt.Errorf("failed to delete balances: %w", err)
	}

	res := db.Exec(`DELETE FROM accounts WHERE id = ? AND user_id = ?`, accountID, userID)
	if res.Error != nil {
		return fmt.Errorf("failed to delete account: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("account %d was not deleted", accountID)
	}

	return nil
}

func (r *AccountRepository) GetUserFirstBalanceDate(ctx context.Context, tx *gorm.DB, userID int64) (time.Time, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var d *time.Time
	err := db.Raw(`
        SELECT MIN(a.opened_at)::date
        FROM accounts a
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

	// Row().Scan, not gorm's Scan: only the former reads a NULL into a *time.Time.
	var open *time.Time
	if err := db.Raw(`
        SELECT opened_at FROM accounts WHERE id = ?
    `, accountID).Row().Scan(&open); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, sql.ErrNoRows
		}
		return time.Time{}, err
	}
	if open == nil {
		return time.Time{}, sql.ErrNoRows
	}
	// normalize to UTC midnight
	t := open.UTC().Truncate(24 * time.Hour)
	return t, nil
}

func (r *AccountRepository) FindAccountsWithDefaults(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Account, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Account
	if err := db.Preload("AccountType").
		Where("user_id = ?", userID).
		Where("closed_at IS NULL").
		Where("is_active = ?", true).
		Where("is_default = ?", true).
		Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) FindAccountTypesWithoutDefaults(ctx context.Context, tx *gorm.DB, userID int64) ([]models.AccountType, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.AccountType

	// account_type_ids that do have a default account for this user
	subQuery := db.Table("accounts").
		Select("DISTINCT account_type_id").
		Where("user_id = ?", userID).
		Where("is_default = ?", true).
		Where("closed_at IS NULL").
		Where("is_active = ?", true)

	// Get all account types that are NOT in the subquery
	if err := db.Select("DISTINCT ON (sub_type) *").
		Where("id NOT IN (?)", subQuery).
		Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *AccountRepository) UpdateDefaultAccount(ctx context.Context, tx *gorm.DB, account models.Account, setAsDefault bool) error {
	db := tx.WithContext(ctx)

	// Always unset all defaults for this account type first (whether setting or unsetting)
	result := db.Model(&models.Account{}).
		Where("user_id = ? AND account_type_id = ?", account.UserID, account.AccountTypeID).
		Update("is_default", false)
	if result.Error != nil {
		return result.Error
	}

	// Then set this specific account if requested
	if setAsDefault {
		result = db.Model(&models.Account{}).
			Where("id = ?", account.ID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func (r *AccountRepository) HasDefaultForAccountType(ctx context.Context, tx *gorm.DB, userID, accountTypeID int64) (bool, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var count int64
	err := db.Model(&models.Account{}).
		Where("user_id = ? AND account_type_id = ? AND is_default = ? AND closed_at IS NULL AND is_active = ?",
			userID, accountTypeID, true, true).
		Count(&count).Error

	return count > 0, err
}
