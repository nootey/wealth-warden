package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	baseTxQuery(ctx context.Context, db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB
	baseTransferQuery(ctx context.Context, db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB
	FindTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool, accountID *int64) ([]models.Transaction, error)
	FindAllTransactionsForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Transaction, error)
	FindTransfers(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, includeDeleted bool) ([]models.Transfer, error)
	FindAllTransfersForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Transfer, error)
	GetMonthlyTransfersFromChecking(ctx context.Context, tx *gorm.DB, userID int64, checkingAccountIDs []int64, year, month int) ([]models.Transfer, error)
	CountTransactions(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, includeDeleted bool, accountID *int64) (int64, error)
	CountTransfers(ctx context.Context, tx *gorm.DB, userID int64, includeDeleted bool) (int64, error)
	scopeCategories(ctx context.Context, tx *gorm.DB, userID *int64, includeDeleted bool) *gorm.DB
	FindAllCategories(ctx context.Context, tx *gorm.DB, userID *int64, includeDeleted bool) ([]models.Category, error)
	FindAllCustomCategories(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Category, error)
	FindCategoryByID(ctx context.Context, tx *gorm.DB, ID int64, userID *int64, includeDeleted bool) (models.Category, error)
	FindCategoryByClassification(ctx context.Context, tx *gorm.DB, classification string, userID *int64) (models.Category, error)
	FindCategoryByName(ctx context.Context, tx *gorm.DB, name string, userID *int64) (models.Category, error)
	FindTransactionByID(ctx context.Context, tx *gorm.DB, ID, userID int64, includeDeleted bool) (models.Transaction, error)
	FindTransactionsByImportID(ctx context.Context, tx *gorm.DB, importID, userID int64) ([]models.Transaction, error)
	FindTransferByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Transfer, error)
	FindTransfersByImportID(ctx context.Context, tx *gorm.DB, importID, userID int64) ([]models.Transfer, error)
	CountActiveTransactionsForCategory(ctx context.Context, tx *gorm.DB, userID, categoryID int64) (int64, error)
	InsertTransaction(ctx context.Context, tx *gorm.DB, newRecord *models.Transaction) (int64, error)
	InsertTransfer(ctx context.Context, tx *gorm.DB, newRecord *models.Transfer) (int64, error)
	InsertCategory(ctx context.Context, tx *gorm.DB, newRecord *models.Category) (int64, error)
	UpdateTransaction(ctx context.Context, tx *gorm.DB, record models.Transaction) (int64, error)
	UpdateCategory(ctx context.Context, tx *gorm.DB, record models.Category) (int64, error)
	DeleteTransaction(ctx context.Context, tx *gorm.DB, id, userID int64) error
	DeleteTransfer(ctx context.Context, tx *gorm.DB, id, userID int64) error
	BulkDeleteTransactions(ctx context.Context, tx *gorm.DB, ids []int64, userID int64) error
	BulkDeleteTransfers(ctx context.Context, tx *gorm.DB, ids []int64, userID int64) error
	ArchiveCategory(ctx context.Context, tx *gorm.DB, id, userID int64) error
	DeleteCategory(ctx context.Context, tx *gorm.DB, id, userID int64) error
	RestoreTransaction(ctx context.Context, tx *gorm.DB, id, userID int64) error
	RestoreCategory(ctx context.Context, tx *gorm.DB, id int64, userID *int64) error
	RestoreCategoryName(ctx context.Context, tx *gorm.DB, id int64, userID *int64, name string) error
	FindTransactionTemplates(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int) ([]models.TransactionTemplate, error)
	CountTransactionTemplates(ctx context.Context, tx *gorm.DB, userID int64, onlyActive bool) (int64, error)
	FindTransactionTemplateByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.TransactionTemplate, error)
	InsertTransactionTemplate(ctx context.Context, tx *gorm.DB, newRecord *models.TransactionTemplate) (int64, error)
	UpdateTransactionTemplate(ctx context.Context, tx *gorm.DB, record models.TransactionTemplate, onlyActive bool) (int64, error)
	DeleteTransactionTemplate(ctx context.Context, tx *gorm.DB, id int64) error
	GetTransactionsForYear(ctx context.Context, tx *gorm.DB, userID int64, year int, accountID *int64) ([]models.Transaction, error)
	GetTransactionsByYearAndClass(ctx context.Context, tx *gorm.DB, userID int64, year int, class string, accountID *int64) ([]models.Transaction, error)
	GetAllTimeStatsByClass(ctx context.Context, tx *gorm.DB, userID int64, class string, accountID, categoryID *int64) (total decimal.Decimal, monthsWithData int, err error)
	PurgeImportedTransactions(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error)
	PurgeImportedTransfers(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error)
	PurgeImportedCategories(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error)
	FindAllCategoryGroups(ctx context.Context, tx *gorm.DB, userID int64) ([]models.CategoryGroup, error)
	FindAllCategoriesAndGroups(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Category, []models.CategoryGroup, error)
	FindCategoryGroupByID(ctx context.Context, tx *gorm.DB, ID int64, userID int64) (models.CategoryGroup, error)
	InsertCategoryGroup(ctx context.Context, tx *gorm.DB, newRecord *models.CategoryGroup) (int64, error)
	InsertCategoryGroupMember(ctx context.Context, tx *gorm.DB, groupID, categoryID int64) error
	UpdateCategoryGroup(ctx context.Context, tx *gorm.DB, record models.CategoryGroup) (int64, error)
	DeleteCategoryGroupMembers(ctx context.Context, tx *gorm.DB, groupingID int64) error
	DeleteCategoryGroup(ctx context.Context, tx *gorm.DB, id, userID int64) error
	IsCategoryInGroup(ctx context.Context, tx *gorm.DB, categoryID int64) (bool, error)
	GetYearlyAverageForCategory(ctx context.Context, tx *gorm.DB, userID int64, accountID int64, categoryID int64, year int) (float64, error)
	GetYearlyAverageForCategoryGroup(ctx context.Context, tx *gorm.DB, userID int64, accountID int64, groupID int64, year int) (float64, error)
}

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

var _ TransactionRepositoryInterface = (*TransactionRepository)(nil)

func (r *TransactionRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *TransactionRepository) baseTxQuery(ctx context.Context, db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB {
	q := db.WithContext(ctx).Model(&models.Transaction{}).
		Where("transactions.user_id = ?", userID)

	if !includeDeleted {
		q = q.Where("transactions.deleted_at IS NULL")
	}

	// exclude transactions that belong to an active transfer
	q = q.Where(`
		NOT EXISTS (
			SELECT 1 FROM transfers t
			WHERE (t.transaction_inflow_id = transactions.id OR t.transaction_outflow_id = transactions.id)
			  AND t.deleted_at IS NULL
		)
	`)

	return q
}

func (r *TransactionRepository) baseTransferQuery(ctx context.Context, db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB {
	q := db.WithContext(ctx).Model(&models.Transfer{}).
		Where("transfers.user_id = ?", userID)

	if !includeDeleted {
		q = q.Where("transfers.deleted_at IS NULL")

		// Also hide transfers whose legs are soft-deleted
		q = q.Where(`
			NOT EXISTS (
				SELECT 1 FROM transactions ti
				WHERE ti.id IN (transfers.transaction_inflow_id, transfers.transaction_outflow_id)
				  AND ti.deleted_at IS NOT NULL
			)
		`)
	}

	return q
}

func (r *TransactionRepository) FindTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool, accountID *int64) ([]models.Transaction, error) {

	var records []models.Transaction
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := r.baseTxQuery(ctx, db, userID, includeDeleted).
		Preload("Category").
		Preload("Account")

	if accountID != nil {
		q = q.Where("transactions.account_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "transactions", sortField, sortOrder)

	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *TransactionRepository) FindAllTransactionsForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Transaction, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Transaction

	q := r.baseTxQuery(ctx, db, userID, false).
		Preload("Category")

	err := q.
		Order("txn_date asc").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *TransactionRepository) FindTransfers(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, includeDeleted bool) ([]models.Transfer, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Transfer
	q := r.baseTransferQuery(ctx, db, userID, includeDeleted)

	if !includeDeleted {
		q = q.
			Preload("TransactionInflow", "deleted_at IS NULL").
			Preload("TransactionInflow.Account").
			Preload("TransactionOutflow", "deleted_at IS NULL").
			Preload("TransactionOutflow.Account")
	} else {
		q = q.
			Preload("TransactionInflow.Account").
			Preload("TransactionOutflow.Account")
	}

	err := q.
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *TransactionRepository) FindAllTransfersForUser(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Transfer, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Transfer

	q := r.baseTransferQuery(ctx, db, userID, false).
		Preload("TransactionInflow.Account.AccountType")

	err := q.
		Order("created_at desc").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *TransactionRepository) GetMonthlyTransfersFromChecking(ctx context.Context, tx *gorm.DB, userID int64, checkingAccountIDs []int64, year, month int) ([]models.Transfer, error) {
	var transfers []models.Transfer

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	q := r.baseTransferQuery(ctx, tx, userID, false).
		Preload("TransactionOutflow.Account.AccountType").
		Preload("TransactionInflow.Account.AccountType")

	q = q.Joins("JOIN transactions AS tx_out ON tx_out.id = transfers.transaction_outflow_id")

	err := q.
		Where("tx_out.account_id IN ?", checkingAccountIDs).
		Where("transfers.created_at >= ? AND transfers.created_at < ?", start, end).
		Find(&transfers).Error

	return transfers, err
}

func (r *TransactionRepository) CountTransactions(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, includeDeleted bool, accountID *int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := r.baseTxQuery(ctx, db, userID, includeDeleted)
	if accountID != nil {
		q = q.Where("transactions.account_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	for _, join := range joins {
		q = q.Joins(join)
	}

	q = utils.ApplyFilters(q, filters)

	err := q.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) CountTransfers(ctx context.Context, tx *gorm.DB, userID int64, includeDeleted bool) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := r.baseTransferQuery(ctx, db, userID, includeDeleted)

	if err := q.Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) scopeCategories(ctx context.Context, tx *gorm.DB, userID *int64, includeDeleted bool) *gorm.DB {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.Category{})

	if !includeDeleted {
		q = q.Where("deleted_at IS NULL")
	}

	if userID != nil {
		return q.Where("(user_id IS NULL OR user_id = ?)", *userID)
	}
	return q.Where("user_id IS NULL")
}

func (r *TransactionRepository) FindAllCategories(ctx context.Context, tx *gorm.DB, userID *int64, includeDeleted bool) ([]models.Category, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Category
	db = r.scopeCategories(ctx, db, userID, includeDeleted).
		Order("classification, name").
		Find(&records)
	return records, db.Error
}

func (r *TransactionRepository) FindAllCustomCategories(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Category, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Category
	q := db.Model(&models.Category{}).
		Where("user_id = ? AND is_default = false", userID).
		Order("classification, name").
		Find(&records)
	return records, q.Error
}

func (r *TransactionRepository) FindCategoryByID(ctx context.Context, tx *gorm.DB, ID int64, userID *int64, includeDeleted bool) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Category
	txn := r.scopeCategories(ctx, db, userID, includeDeleted).
		Where("id = ?", ID).
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindCategoryByClassification(ctx context.Context, tx *gorm.DB, classification string, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Category
	txn := r.scopeCategories(ctx, db, userID, false).
		Where("classification = ?", classification).
		Order("name").
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindCategoryByName(ctx context.Context, tx *gorm.DB, name string, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Category
	txn := r.scopeCategories(ctx, db, userID, false).
		Where("name = ?", name).
		Order("name").
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindTransactionByID(ctx context.Context, tx *gorm.DB, ID, userID int64, includeDeleted bool) (models.Transaction, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Transaction
	q := db.
		Preload("Category").
		Preload("Account").
		Where("id = ? AND user_id = ?", ID, userID)

	if !includeDeleted {
		q = q.Where("transactions.deleted_at IS NULL")
	}

	q = q.First(&record)

	return record, q.Error
}

func (r *TransactionRepository) FindTransactionsByImportID(ctx context.Context, tx *gorm.DB, importID, userID int64) ([]models.Transaction, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Transaction
	q := db.Model(&models.Transaction{}).
		Where("import_id = ? AND user_id = ?", importID, userID)

	q = q.Find(&records)

	return records, q.Error
}

func (r *TransactionRepository) FindTransferByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Transfer, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Transfer
	result := db.
		Where("id = ? AND user_id = ?", ID, userID).First(&record)
	return record, result.Error
}

func (r *TransactionRepository) FindTransfersByImportID(ctx context.Context, tx *gorm.DB, importID, userID int64) ([]models.Transfer, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Transfer
	q := db.Model(&models.Transfer{}).
		Where("import_id = ? AND user_id = ?", importID, userID)

	q = q.Find(&records)

	return records, q.Error
}

func (r *TransactionRepository) CountActiveTransactionsForCategory(ctx context.Context, tx *gorm.DB, userID, categoryID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var cnt int64
	err := db.Model(&models.Transaction{}).
		Where("user_id = ? AND category_id = ?", userID, categoryID).
		Count(&cnt).Error
	return cnt, err
}

func (r *TransactionRepository) InsertTransaction(ctx context.Context, tx *gorm.DB, newRecord *models.Transaction) (int64, error) {
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

func (r *TransactionRepository) InsertTransfer(ctx context.Context, tx *gorm.DB, newRecord *models.Transfer) (int64, error) {
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

func (r *TransactionRepository) InsertCategory(ctx context.Context, tx *gorm.DB, newRecord *models.Category) (int64, error) {
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

func (r *TransactionRepository) UpdateTransaction(ctx context.Context, tx *gorm.DB, record models.Transaction) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.Transaction{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"account_id":       record.AccountID,
			"category_id":      record.CategoryID,
			"transaction_type": record.TransactionType,
			"amount":           record.Amount,
			"currency":         record.Currency,
			"txn_date":         record.TxnDate,
			"description":      record.Description,
			"updated_at":       time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *TransactionRepository) UpdateCategory(ctx context.Context, tx *gorm.DB, record models.Category) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.Category{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"display_name":   record.DisplayName,
			"classification": record.Classification,
			"updated_at":     time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *TransactionRepository) DeleteTransaction(ctx context.Context, tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Transaction{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TransactionRepository) DeleteTransfer(ctx context.Context, tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Transfer{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TransactionRepository) BulkDeleteTransactions(ctx context.Context, tx *gorm.DB, ids []int64, userID int64) error {

	if len(ids) == 0 {
		return nil // nothing to delete
	}

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Transaction{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Updates(map[string]any{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *TransactionRepository) BulkDeleteTransfers(ctx context.Context, tx *gorm.DB, ids []int64, userID int64) error {

	if len(ids) == 0 {
		return nil
	}

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Transfer{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Updates(map[string]any{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *TransactionRepository) ArchiveCategory(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	now := time.Now().UTC()
	res := db.Model(&models.Category{}).
		Where("id = ? AND (user_id = ? OR user_id IS NULL) AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": now,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TransactionRepository) DeleteCategory(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var cat models.Category
	if err := db.
		Where("id = ? AND (user_id = ? OR user_id IS NULL)", id, userID).
		First(&cat).Error; err != nil {
		return err
	}

	if cat.IsDefault {
		return fmt.Errorf("default categories cannot be hard-deleted")
	}
	if cat.DeletedAt == nil {
		return fmt.Errorf("category must be soft-deleted before hard deletion")
	}

	if err := db.Where("id = ? AND (user_id = ? OR user_id IS NULL)", id, userID).
		Delete(&models.Category{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *TransactionRepository) RestoreTransaction(ctx context.Context, tx *gorm.DB, id, userID int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Model(&models.Transaction{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NOT NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": gorm.Expr("NULL"),
			"updated_at": time.Now().UTC(),
		})
	return res.Error
}

func (r *TransactionRepository) RestoreCategory(ctx context.Context, tx *gorm.DB, id int64, userID *int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var owner *gorm.DB
	if userID != nil {
		owner = db.Where("user_id = ?", *userID).
			Or("is_default = ? AND user_id IS NULL", true)
	} else {
		owner = db.Where("is_default = ? AND user_id IS NULL", true)
	}

	scope := db.Model(&models.Category{}).
		Unscoped().
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Where(owner)

	res := scope.Updates(map[string]any{
		"deleted_at": gorm.Expr("NULL"),
		"updated_at": time.Now().UTC(),
	})

	return res.Error
}

func (r *TransactionRepository) RestoreCategoryName(ctx context.Context, tx *gorm.DB, id int64, userID *int64, name string) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var owner *gorm.DB
	if userID != nil {
		owner = db.Where("user_id = ?", *userID).
			Or("is_default = ? AND user_id IS NULL", true)
	} else {
		owner = db.Where("is_default = ? AND user_id IS NULL", true)
	}

	s := strings.ReplaceAll(strings.ToLower(name), "_", " ")
	if s != "" {
		r0, size := utf8.DecodeRuneInString(s)
		if r0 != utf8.RuneError {
			s = string(unicode.ToUpper(r0)) + s[size:]
		}
	}

	scope := db.Model(&models.Category{}).
		Unscoped().
		Where("id = ?", id).
		Where(owner)

	res := scope.Updates(map[string]any{
		"display_name": s,
		"updated_at":   time.Now().UTC(),
	})

	return res.Error
}

func (r *TransactionRepository) FindTransactionTemplates(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int) ([]models.TransactionTemplate, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.TransactionTemplate
	err := db.Model(&models.TransactionTemplate{}).
		Where("user_id = ?", userID).
		Preload("Category").
		Preload("Account").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *TransactionRepository) CountTransactionTemplates(ctx context.Context, tx *gorm.DB, userID int64, onlyActive bool) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.Model(&models.TransactionTemplate{}).
		Where("user_id = ?", userID)

	if onlyActive {
		q = q.Where("is_active = ?", true)
	}

	if err := q.Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) FindTransactionTemplateByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.TransactionTemplate, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.TransactionTemplate
	q := db.
		Preload("Category").
		Preload("Account").
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *TransactionRepository) InsertTransactionTemplate(ctx context.Context, tx *gorm.DB, newRecord *models.TransactionTemplate) (int64, error) {

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

func (r *TransactionRepository) UpdateTransactionTemplate(ctx context.Context, tx *gorm.DB, record models.TransactionTemplate, onlyActive bool) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if onlyActive {
		updates := map[string]interface{}{}
		updates["is_active"] = record.IsActive
		updates["updated_at"] = time.Now().UTC()
		db.Model(&models.TransactionTemplate{}).Where("id = ?", record.ID).Updates(updates)
	} else {
		if err := db.Model(models.TransactionTemplate{}).
			Where("id = ?", record.ID).
			Updates(map[string]interface{}{
				"name":        record.Name,
				"amount":      record.Amount,
				"next_run_at": record.NextRunAt,
				"end_date":    record.EndDate,
				"max_runs":    record.MaxRuns,
				"updated_at":  time.Now().UTC(),
			}).Error; err != nil {
			return 0, err
		}
	}

	return record.ID, nil
}

func (r *TransactionRepository) DeleteTransactionTemplate(ctx context.Context, tx *gorm.DB, id int64) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Where("id = ?", id).
		Delete(&models.TransactionTemplate{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *TransactionRepository) GetTransactionsForYear(ctx context.Context, tx *gorm.DB, userID int64, year int, accountID *int64) ([]models.Transaction, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	query := db.Where("user_id = ? AND EXTRACT(YEAR FROM txn_date) = ?", userID, year)

	if accountID != nil {
		query = query.Where("account_id = ?", *accountID)
	}

	var txs []models.Transaction
	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *TransactionRepository) GetTransactionsByYearAndClass(ctx context.Context, tx *gorm.DB, userID int64, year int, class string, accountID *int64) ([]models.Transaction, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.Where("user_id = ? AND EXTRACT(YEAR FROM txn_date) = ? AND transaction_type = ?", userID, year, class)

	if accountID != nil {
		q = q.Where("account_id = ?", *accountID)
	}

	var txs []models.Transaction
	err := q.Find(&txs).Error
	return txs, err
}

func (r *TransactionRepository) GetAllTimeStatsByClass(ctx context.Context, tx *gorm.DB, userID int64, class string, accountID, categoryID *int64) (total decimal.Decimal, monthsWithData int, err error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.Model(&models.Transaction{}).
		Select("COALESCE(SUM(amount), 0) as total, COUNT(DISTINCT EXTRACT(YEAR FROM txn_date) || '-' || EXTRACT(MONTH FROM txn_date)) as months_with_data").
		Where("user_id = ? AND transaction_type = ?", userID, class)

	if accountID != nil {
		q = q.Where("account_id = ?", *accountID)
	}

	if categoryID != nil {
		q = q.Where("category_id = ?", *categoryID)
	}

	var result struct {
		Total          decimal.Decimal
		MonthsWithData int
	}

	err = q.Scan(&result).Error
	if err != nil {
		return decimal.Zero, 0, err
	}

	return result.Total, result.MonthsWithData, nil
}

func (r *TransactionRepository) PurgeImportedTransactions(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Exec(`SET LOCAL ww.hard_delete = 'on'`).Error; err != nil {
		return 0, err
	}
	res := db.Exec(`
        DELETE FROM transactions
        WHERE user_id = ? AND import_id = ?
    `, userID, importID)
	return res.RowsAffected, res.Error
}

func (r *TransactionRepository) PurgeImportedTransfers(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Exec(`SET LOCAL ww.hard_delete = 'on'`).Error; err != nil {
		return 0, err
	}
	res := db.Exec(`
        DELETE FROM transfers
        WHERE user_id = ? AND import_id = ?
    `, userID, importID)
	return res.RowsAffected, res.Error
}

func (r *TransactionRepository) PurgeImportedCategories(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	// reassign any transactions using these categories to (uncategorized)
	updateSQL := `
        UPDATE transactions
        SET category_id = (
            SELECT id FROM categories 
            WHERE name = '(uncategorized)' 
            AND classification = 'uncategorized'
            LIMIT 1
        )
        WHERE category_id IN (
            SELECT id FROM categories 
            WHERE user_id = ? AND import_id = ?
        )
    `

	if err := db.Exec(updateSQL, userID, importID).Error; err != nil {
		return 0, err
	}

	// Now delete the imported categories
	if err := db.Exec(`SET LOCAL ww.hard_delete = 'on'`).Error; err != nil {
		return 0, err
	}
	res := db.Exec(`
        DELETE FROM categories
        WHERE user_id = ? AND import_id = ?
    `, userID, importID)
	return res.RowsAffected, res.Error
}

func (r *TransactionRepository) FindAllCategoryGroups(ctx context.Context, tx *gorm.DB, userID int64) ([]models.CategoryGroup, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.CategoryGroup
	db.Model(&models.CategoryGroup{}).
		Preload("Categories").
		Where("user_id = ?", userID).
		Order("classification, name").
		Find(&records)
	return records, db.Error
}

func (r *TransactionRepository) FindAllCategoriesAndGroups(ctx context.Context, tx *gorm.DB, userID int64) ([]models.Category, []models.CategoryGroup, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var categories []models.Category
	if err := db.Model(&models.Category{}).
		Where("(user_id = ? OR user_id IS NULL) AND deleted_at IS NULL AND parent_id IS NOT NULL", userID).
		Order("classification, name").
		Find(&categories).Error; err != nil {
		return nil, nil, err
	}

	var groups []models.CategoryGroup
	if err := db.Model(&models.CategoryGroup{}).
		Preload("Categories").
		Where("user_id = ? OR user_id IS NULL", userID).
		Order("classification, name").
		Find(&groups).Error; err != nil {
		return nil, nil, err
	}

	return categories, groups, nil
}

func (r *TransactionRepository) FindCategoryGroupByID(ctx context.Context, tx *gorm.DB, ID int64, userID int64) (models.CategoryGroup, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.CategoryGroup
	db.Model(&models.CategoryGroup{}).
		Preload("Categories").
		Where("user_id = ?", userID).
		Order("classification, name").
		First(&record)
	return record, db.Error
}

func (r *TransactionRepository) InsertCategoryGroup(ctx context.Context, tx *gorm.DB, newRecord *models.CategoryGroup) (int64, error) {

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

func (r *TransactionRepository) InsertCategoryGroupMember(ctx context.Context, tx *gorm.DB, groupID, categoryID int64) error {
	query := `
        INSERT INTO category_group_members (group_id, category_id)
        VALUES (?, ?)
    `
	return tx.WithContext(ctx).Exec(query, groupID, categoryID).Error
}

func (r *TransactionRepository) UpdateCategoryGroup(ctx context.Context, tx *gorm.DB, record models.CategoryGroup) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.CategoryGroup{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":           record.Name,
			"classification": record.Classification,
			"description":    record.Description,
			"updated_at":     time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *TransactionRepository) DeleteCategoryGroupMembers(ctx context.Context, tx *gorm.DB, groupingID int64) error {
	query := `DELETE FROM category_group_members WHERE group_id = ?`
	return tx.WithContext(ctx).Exec(query, groupingID).Error
}

func (r *TransactionRepository) DeleteCategoryGroup(ctx context.Context, tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.CategoryGroup{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *TransactionRepository) IsCategoryInGroup(ctx context.Context, tx *gorm.DB, categoryID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM category_group_members WHERE category_id = ?)`
	err := tx.WithContext(ctx).Raw(query, categoryID).Scan(&exists).Error
	return exists, err
}

func (r *TransactionRepository) GetYearlyAverageForCategory(ctx context.Context, tx *gorm.DB, userID, accountID, categoryID int64, year int) (float64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var result struct {
		Total        float64
		Count        int64
		ActiveMonths int64
	}

	query := `
        SELECT 
            COALESCE(SUM(ABS(amount)), 0) as total,
            COUNT(*) as count,
            COUNT(DISTINCT EXTRACT(MONTH FROM txn_date)) as active_months
        FROM transactions
        WHERE user_id = ?
          AND account_id = ?
          AND category_id = ?
          AND deleted_at IS NULL
          AND EXTRACT(YEAR FROM txn_date) = ?
    `

	err := db.Raw(query, userID, accountID, categoryID, year).Scan(&result).Error
	if err != nil || result.ActiveMonths == 0 {
		return 0, err
	}

	monthlyAverage := result.Total / float64(result.ActiveMonths)
	return monthlyAverage, err
}

func (r *TransactionRepository) GetYearlyAverageForCategoryGroup(ctx context.Context, tx *gorm.DB, userID, accountID, groupID int64, year int) (float64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var result struct {
		Total        float64
		Count        int64
		ActiveMonths int64
	}

	query := `
        SELECT 
            COALESCE(SUM(ABS(t.amount)), 0) as total,
            COUNT(*) as count,
            COUNT(DISTINCT EXTRACT(MONTH FROM t.txn_date)) as active_months
        FROM transactions t
        INNER JOIN category_group_members cgm ON t.category_id = cgm.category_id
        WHERE t.user_id = ?
          AND t.account_id = ?
          AND cgm.group_id = ?
          AND t.deleted_at IS NULL
          AND EXTRACT(YEAR FROM t.txn_date) = ?
    `

	err := db.Raw(query, userID, accountID, groupID, year).Scan(&result).Error

	if err != nil || result.ActiveMonths == 0 {
		return 0, err
	}

	monthlyAverage := result.Total / float64(result.ActiveMonths)
	return monthlyAverage, err
}
