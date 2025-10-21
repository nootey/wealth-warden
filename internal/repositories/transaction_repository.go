package repositories

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

type TransactionRepository struct {
	DB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

func (r *TransactionRepository) baseTxQuery(db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB {
	q := db.Model(&models.Transaction{}).
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

func (r *TransactionRepository) baseTransferQuery(db *gorm.DB, userID int64, includeDeleted bool) *gorm.DB {
	q := db.Model(&models.Transfer{}).
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

func (r *TransactionRepository) FindTransactions(userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool, accountID *int64) ([]models.Transaction, error) {

	var records []models.Transaction

	q := r.baseTxQuery(r.DB, userID, includeDeleted).
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

func (r *TransactionRepository) FindTransfers(userID int64, offset, limit int, includeDeleted bool) ([]models.Transfer, error) {

	var records []models.Transfer

	q := r.baseTransferQuery(r.DB, userID, includeDeleted)

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

func (r *TransactionRepository) GetMonthlyTransfersFromChecking(
	tx *gorm.DB,
	userID int64,
	checkingAccountIDs []int64,
	year int,
	month int,
) ([]models.Transfer, error) {
	var transfers []models.Transfer

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	q := r.baseTransferQuery(tx, userID, false).
		Preload("TransactionOutflow.Account.AccountType").
		Preload("TransactionInflow.Account.AccountType")

	q = q.Joins("JOIN transactions AS tx_out ON tx_out.id = transfers.transaction_outflow_id")

	err := q.
		Where("tx_out.account_id IN ?", checkingAccountIDs).
		Where("transfers.created_at >= ? AND transfers.created_at < ?", start, end).
		Find(&transfers).Error

	return transfers, err
}

func (r *TransactionRepository) CountTransactions(userID int64, filters []utils.Filter, includeDeleted bool, accountID *int64) (int64, error) {
	var totalRecords int64

	q := r.baseTxQuery(r.DB, userID, includeDeleted)
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

func (r *TransactionRepository) CountTransfers(userID int64, includeDeleted bool) (int64, error) {
	var totalRecords int64

	q := r.baseTransferQuery(r.DB, userID, includeDeleted)

	if err := q.Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) scopeCategories(db *gorm.DB, userID *int64, includeDeleted bool) *gorm.DB {
	q := db.Model(&models.Category{})

	if !includeDeleted {
		q = q.Where("deleted_at IS NULL")
	}

	if userID != nil {
		return q.Where("(user_id IS NULL OR user_id = ?)", *userID)
	}
	return q.Where("user_id IS NULL")
}

func (r *TransactionRepository) FindAllCategories(userID *int64, includeDeleted bool) ([]models.Category, error) {
	var records []models.Category
	tx := r.scopeCategories(r.DB, userID, includeDeleted).
		Order("classification, name").
		Find(&records)
	return records, tx.Error
}

func (r *TransactionRepository) FindCategoryByID(tx *gorm.DB, ID int64, userID *int64, includeDeleted bool) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Category
	txn := r.scopeCategories(db, userID, includeDeleted).
		Where("id = ?", ID).
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindCategoryByClassification(tx *gorm.DB, classification string, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Category
	txn := r.scopeCategories(db, userID, false).
		Where("classification = ?", classification).
		Order("name").
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindCategoryByName(tx *gorm.DB, name string, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Category
	txn := r.scopeCategories(db, userID, false).
		Where("name = ?", name).
		Order("name").
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindTransactionByID(tx *gorm.DB, ID, userID int64, includeDeleted bool) (models.Transaction, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

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

func (r *TransactionRepository) FindTransactionIDsByImportID(tx *gorm.DB, importID, userID int64) ([]int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var ids []int64
	q := db.Model(&models.Transaction{}).
		Where("import_id = ? AND user_id = ?", importID, userID)

	q = q.Pluck("id", &ids)

	return ids, q.Error
}

func (r *TransactionRepository) FindTransferByID(tx *gorm.DB, ID, userID int64) (models.Transfer, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Transfer
	result := db.
		Where("id = ? AND user_id = ?", ID, userID).First(&record)
	return record, result.Error
}

func (r *TransactionRepository) FindTransferIDsByImportID(tx *gorm.DB, importID, userID int64) ([]int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var ids []int64
	q := db.Model(&models.Transfer{}).
		Where("import_id = ? AND user_id = ?", importID, userID)

	q = q.Pluck("id", &ids)

	return ids, q.Error
}

func (r *TransactionRepository) CountActiveTransactionsForCategory(tx *gorm.DB, userID, categoryID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var cnt int64
	err := db.Model(&models.Transaction{}).
		Where("user_id = ? AND category_id = ?", userID, categoryID).
		Count(&cnt).Error
	return cnt, err
}

func (r *TransactionRepository) InsertTransaction(tx *gorm.DB, newRecord *models.Transaction) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *TransactionRepository) InsertTransfer(tx *gorm.DB, newRecord *models.Transfer) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *TransactionRepository) InsertCategory(tx *gorm.DB, newRecord *models.Category) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *TransactionRepository) UpdateTransaction(tx *gorm.DB, record models.Transaction) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

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
			"updated_at":       time.Now(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *TransactionRepository) UpdateCategory(tx *gorm.DB, record models.Category) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Model(models.Category{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"display_name":   record.DisplayName,
			"classification": record.Classification,
			"updated_at":     time.Now(),
		}).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *TransactionRepository) DeleteTransaction(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.Transaction{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TransactionRepository) DeleteTransfer(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.Transfer{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *TransactionRepository) BulkDeleteTransactions(tx *gorm.DB, ids []int64, userID int64) error {
	if len(ids) == 0 {
		return nil // nothing to delete
	}

	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.Transaction{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *TransactionRepository) BulkDeleteTransfers(tx *gorm.DB, ids []int64, userID int64) error {
	if len(ids) == 0 {
		return nil
	}

	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Model(&models.Transfer{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Updates(map[string]any{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (r *TransactionRepository) ArchiveCategory(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	now := time.Now()
	res := db.Model(&models.Category{}).
		Where("id = ? AND (user_id = ? OR user_id IS NULL) AND deleted_at IS NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": now,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Treat as idempotent success if it exists but is already soft-deleted
		// Optional: verify existence if you want stricter feedback
	}
	return nil
}

func (r *TransactionRepository) DeleteCategory(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

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

func (r *TransactionRepository) RestoreTransaction(tx *gorm.DB, id, userID int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}
	res := db.Model(&models.Transaction{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NOT NULL", id, userID).
		Updates(map[string]any{
			"deleted_at": gorm.Expr("NULL"),
			"updated_at": time.Now(),
		})
	return res.Error
}

func (r *TransactionRepository) RestoreCategory(tx *gorm.DB, id int64, userID *int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

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
		"updated_at": time.Now(),
	})

	return res.Error
}

func (r *TransactionRepository) RestoreCategoryName(tx *gorm.DB, id int64, userID *int64, name string) error {
	db := tx
	if db == nil {
		db = r.DB
	}

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
		"updated_at":   time.Now(),
	})

	return res.Error
}

func (r *TransactionRepository) FindTransactionTemplates(userID int64, offset, limit int) ([]models.TransactionTemplate, error) {

	var records []models.TransactionTemplate
	err := r.DB.Model(&models.TransactionTemplate{}).
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

func (r *TransactionRepository) CountTransactionTemplates(userID int64, onlyActive bool) (int64, error) {
	var totalRecords int64

	q := r.DB.Model(&models.TransactionTemplate{}).
		Where("user_id = ?", userID)

	if onlyActive {
		q = q.Where("is_active = ?", true)
	}

	if err := q.Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) FindTransactionTemplateByID(tx *gorm.DB, ID, userID int64) (models.TransactionTemplate, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.TransactionTemplate
	q := db.
		Preload("Category").
		Preload("Account").
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *TransactionRepository) InsertTransactionTemplate(tx *gorm.DB, newRecord *models.TransactionTemplate) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *TransactionRepository) UpdateTransactionTemplate(tx *gorm.DB, record models.TransactionTemplate, onlyActive bool) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if onlyActive {
		updates := map[string]interface{}{}
		updates["is_active"] = record.IsActive
		updates["updated_at"] = time.Now()
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
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return 0, err
		}
	}

	return record.ID, nil
}

func (r *TransactionRepository) DeleteTransactionTemplate(tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Where("id = ?", id).
		Delete(&models.TransactionTemplate{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *TransactionRepository) GetTransactionsForYear(userID int64, year int, accountID *int64) ([]models.Transaction, error) {
	query := r.DB.Where("user_id = ? AND EXTRACT(YEAR FROM txn_date) = ?", userID, year)

	if accountID != nil {
		query = query.Where("account_id = ?", *accountID)
	}

	var txs []models.Transaction
	if err := query.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *TransactionRepository) GetTransactionsByYearAndClass(
	userID int64, year int, class string, accountID *int64,
) ([]models.Transaction, error) {
	q := r.DB.
		Where("user_id = ? AND EXTRACT(YEAR FROM txn_date) = ? AND transaction_type = ?", userID, year, class)

	if accountID != nil {
		q = q.Where("account_id = ?", *accountID)
	}

	var txs []models.Transaction
	err := q.Find(&txs).Error
	return txs, err
}

func (r *TransactionRepository) PurgeImportedTransactions(
	tx *gorm.DB, importID, userID int64,
) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Exec(`
		DELETE FROM transactions
		WHERE user_id = ? AND import_id = ?
	`, userID, importID)
	return res.RowsAffected, res.Error
}

func (r *TransactionRepository) PurgeImportedTransfers(
	tx *gorm.DB, importID, userID int64,
) (int64, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	res := db.Exec(`
		DELETE FROM transfers
		WHERE user_id = ? AND import_id = ?
	`, userID, importID)
	return res.RowsAffected, res.Error
}
