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

func (r *TransactionRepository) FindTransactions(user *models.User, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Transaction, error) {

	var records []models.Transaction

	query := r.DB.
		Preload("Category").
		Preload("Account").
		Where("transactions.user_id = ?", user.ID)

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "transactions", sortField, sortOrder)

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

func (r *TransactionRepository) CountTransactions(user *models.User, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.DB.Model(&models.Transaction{}).
		Preload("Category").
		Preload("Account").
		Where("transactions.user_id = ?", user.ID)

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

func (r *TransactionRepository) InsertTransaction(tx *gorm.DB, newRecord models.Transaction) (int64, error) {
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

	res := db.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Transaction{})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
