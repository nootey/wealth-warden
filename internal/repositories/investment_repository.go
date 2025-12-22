package repositories

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

type InvestmentRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	CountInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	CountInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	FindInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentHolding, error)
	FindAllInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64) ([]models.InvestmentHolding, error)
	FindInvestmentHoldingByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentHolding, error)
	FindInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTransaction, error)
	FindInvestmentTransactionByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentTransaction, error)
	InsertHolding(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentHolding) (int64, error)
}

type InvestmentRepository struct {
	db *gorm.DB
}

func NewInvestmentRepository(db *gorm.DB) *InvestmentRepository {
	return &InvestmentRepository{db: db}
}

var _ InvestmentRepositoryInterface = (*InvestmentRepository)(nil)

func (r *InvestmentRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *InvestmentRepository) CountInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.WithContext(ctx).Model(&models.InvestmentHolding{}).
		Where("user_id = ?", userID)

	if accountID != nil {
		q = q.Where("account_id = ?", *accountID)
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

func (r *InvestmentRepository) CountInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.WithContext(ctx).Model(&models.InvestmentTransaction{}).
		Where("user_id = ?", userID)

	if accountID != nil {
		q = q.Where("account_id = ?", *accountID)
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

func (r *InvestmentRepository) FindInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentHolding, error) {

	var records []models.InvestmentHolding
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.InvestmentHolding{}).
		Where("user_id = ?", userID).
		Preload("Account")

	if accountID != nil {
		q = q.Where("transactions.account_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "investment_holdings", sortField, sortOrder)

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

func (r *InvestmentRepository) FindAllInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64) ([]models.InvestmentHolding, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.InvestmentHolding
	query := db.Where("user_id = ?", userID)

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *InvestmentRepository) FindInvestmentHoldingByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentHolding, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.InvestmentHolding
	q := db.
		Preload("Account").
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *InvestmentRepository) FindInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTransaction, error) {

	var records []models.InvestmentTransaction
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.InvestmentTransaction{}).
		Where("user_id = ?", userID).
		Preload("Account")

	if accountID != nil {
		q = q.Where("transactions.account_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "investment_holdings", sortField, sortOrder)

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

func (r *InvestmentRepository) FindInvestmentTransactionByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentTransaction, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.InvestmentTransaction
	q := db.
		Preload("Holding").
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *InvestmentRepository) InsertHolding(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentHolding) (int64, error) {
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
