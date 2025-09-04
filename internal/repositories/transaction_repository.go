package repositories

import (
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
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

func (r *TransactionRepository) FindTransactions(user *models.User, offset, limit int, sortField, sortOrder string, filters []utils.Filter, includeDeleted bool) ([]models.Transaction, error) {

	var records []models.Transaction

	q := r.baseTxQuery(r.DB, user.ID, includeDeleted).
		Preload("Category").
		Preload("Account")

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

func (r *TransactionRepository) FindTransfers(user *models.User, offset, limit int, includeDeleted bool) ([]models.Transfer, error) {

	var records []models.Transfer

	q := r.baseTransferQuery(r.DB, user.ID, includeDeleted)

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

func (r *TransactionRepository) CountTransactions(user *models.User, filters []utils.Filter, includeDeleted bool) (int64, error) {
	var totalRecords int64

	q := r.baseTxQuery(r.DB, user.ID, includeDeleted)

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

func (r *TransactionRepository) CountTransfers(user *models.User, includeDeleted bool) (int64, error) {
	var totalRecords int64

	q := r.baseTransferQuery(r.DB, user.ID, includeDeleted)

	if err := q.Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *TransactionRepository) FindAllCategories(user *models.User) ([]models.Category, error) {
	var records []models.Category
	var userID *int64
	if user != nil {
		userID = &user.ID
	}

	tx := r.DB.
		Model(&models.Category{}).
		Scopes(r.scopeVisibleCategories(userID)).
		Order("classification, name").
		Find(&records)

	return records, tx.Error
}

func (r *TransactionRepository) FindCategoryByID(tx *gorm.DB, ID int64, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Category
	txn := db.Model(&models.Category{}).
		Scopes(r.scopeVisibleCategories(userID)).
		Where("categories.id = ?", ID).
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) FindCategoryByClassification(tx *gorm.DB, classification string, userID *int64) (models.Category, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Category
	txn := db.Model(&models.Category{}).
		Scopes(r.scopeVisibleCategories(userID)).
		Where("categories.classification = ?", classification).
		First(&record)
	return record, txn.Error
}

func (r *TransactionRepository) scopeVisibleCategories(userID *int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if userID == nil {
			// No user: only default categories
			return db.Where("categories.user_id IS NULL")
		}
		uid := *userID
		return db.Where(`
			(categories.user_id = ?) OR
			(
				categories.user_id IS NULL
				AND NOT EXISTS (
					SELECT 1
					FROM hidden_categories hc
					WHERE hc.category_id = categories.id
					  AND hc.user_id = ?
				)
			)
		`, uid, uid)
	}
}

func (r *TransactionRepository) FindTransactionByID(tx *gorm.DB, ID, userID int64) (models.Transaction, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	var record models.Transaction
	result := db.
		Preload("Category").
		Preload("Account").
		Where("id = ? AND user_id = ?", ID, userID).First(&record)
	return record, result.Error
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
