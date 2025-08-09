package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type TransactionRepository struct {
	DB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

func (r *TransactionRepository) FindTransactions(user *models.User, year, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.Transaction, error) {

	var records []models.Transaction

	query := r.DB.
		Preload("Category").
		Preload("Account").
		Where("transactions.user_id = ?", user.ID)

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, sortField, sortOrder)

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

func (r *TransactionRepository) CountTransactions(user *models.User, year int, filters []utils.Filter) (int64, error) {
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
	var userID *uint
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

func (r *TransactionRepository) FindCategoryByID(tx *gorm.DB, ID uint, userID *uint) (models.Category, error) {
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

func (r *TransactionRepository) scopeVisibleCategories(userID *uint) func(*gorm.DB) *gorm.DB {
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

func (r *TransactionRepository) FindTransactionByID(ID, userID uint) (models.Transaction, error) {
	var record models.Transaction
	result := r.DB.Where("id = ? AND user_id = ?", ID, userID).First(&record)
	return record, result.Error
}

func (r *TransactionRepository) InsertTransaction(tx *gorm.DB, newRecord models.Transaction) (uint, error) {
	db := tx
	if db == nil {
		db = r.DB
	}

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}
