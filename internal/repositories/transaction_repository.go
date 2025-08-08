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
	result := r.DB.Find(&records)
	return records, result.Error
}

func (r *TransactionRepository) FindTransactionByID(ID, userID uint) (models.Transaction, error) {
	var record models.Transaction
	result := r.DB.First(&record).Where("id = ? AND user_id = ?", ID, userID)
	return record, result.Error
}

func (r *TransactionRepository) FindCategoryByID(ID, userID uint) (models.Category, error) {
	var record models.Category
	result := r.DB.First(&record).Where("id = ? AND user_id = ?", ID, userID)
	return record, result.Error
}

func (r *TransactionRepository) InsertTransaction(tx *gorm.DB, newRecord models.Transaction) (uint, error) {
	if err := tx.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}
