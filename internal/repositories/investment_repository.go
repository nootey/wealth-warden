package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InvestmentRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	CountInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	CountInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	FindInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentHolding, error)
	FindAllInvestmentHoldings(ctx context.Context, tx *gorm.DB, userID int64) ([]models.InvestmentHolding, error)
	FindInvestmentHoldingByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentHolding, error)
	FindHoldingsByAccountID(ctx context.Context, tx *gorm.DB, accID, userID int64) ([]models.InvestmentHolding, error)
	FindInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTransaction, error)
	FindInvestmentTransactionByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentTransaction, error)
	InsertHolding(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentHolding) (int64, error)
	InsertInvestmentTransaction(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentTransaction) (int64, error)
	UpdateHoldingAfterTransaction(ctx context.Context, tx *gorm.DB, holdingID int64, quantity decimal.Decimal, pricePerUnit decimal.Decimal, currentPrice *decimal.Decimal, lastPriceUpdate *time.Time, transactionType models.TransactionType, transactionValueAtBuy decimal.Decimal) error
	FindTotalInvestmentValue(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error)
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
		Where("investment_holdings.user_id = ?", userID).
		Preload("Account")

	if accountID != nil {
		q = q.Where("investment_holdings.account_id = ?", *accountID)
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
		Preload("Account.Balance", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(1)
		}).
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *InvestmentRepository) FindHoldingsByAccountID(ctx context.Context, tx *gorm.DB, accID, userID int64) ([]models.InvestmentHolding, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.InvestmentHolding
	q := db.
		Where("user_id = ? AND account_id = ?", userID, accID)

	q = q.Find(&records)

	return records, q.Error
}

func (r *InvestmentRepository) FindInvestmentTransactions(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTransaction, error) {

	var records []models.InvestmentTransaction
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.InvestmentTransaction{}).
		Where("investment_transactions.user_id = ?", userID).
		Preload("Holding")

	if accountID != nil {
		q = q.Where("investment_transactions.holding_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "investment_transactions", sortField, sortOrder)

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

func (r *InvestmentRepository) InsertInvestmentTransaction(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentTransaction) (int64, error) {
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

func (r *InvestmentRepository) UpdateHoldingAfterTransaction(ctx context.Context, tx *gorm.DB, holdingID int64, quantity decimal.Decimal, pricePerUnit decimal.Decimal, currentPrice *decimal.Decimal, lastPriceUpdate *time.Time, transactionType models.TransactionType, transactionValueAtBuy decimal.Decimal) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var holding models.InvestmentHolding
	if err := db.First(&holding, holdingID).Error; err != nil {
		return err
	}

	// Update quantity based on transaction type
	var newQuantity decimal.Decimal
	var newAverageBuyPrice decimal.Decimal
	var newTotalValueAtBuy decimal.Decimal

	if transactionType == models.InvestmentBuy {
		newQuantity = holding.Quantity.Add(quantity)

		// Weighted average: (old_qty * old_avg + new_qty * new_price) / total_qty
		oldValue := holding.Quantity.Mul(holding.AverageBuyPrice)
		newValue := quantity.Mul(pricePerUnit)
		totalValue := oldValue.Add(newValue)
		newAverageBuyPrice = totalValue.Div(newQuantity)

		// Total value at buy
		newTotalValueAtBuy = holding.ValueAtBuy.Add(transactionValueAtBuy)
	} else {
		// Sell: decrease quantity and value at buy proportionally
		newQuantity = holding.Quantity.Sub(quantity)
		newAverageBuyPrice = holding.AverageBuyPrice

		// Reduce total value at buy proportionally
		soldProportion := quantity.Div(holding.Quantity)
		valueAtBuyReduction := holding.ValueAtBuy.Mul(soldProportion)
		newTotalValueAtBuy = holding.ValueAtBuy.Sub(valueAtBuyReduction)
	}

	// Calculate current value
	var newCurrentValue decimal.Decimal
	var newProfitLoss decimal.Decimal
	var newProfitLossPercent decimal.Decimal

	if currentPrice != nil && !currentPrice.IsZero() {
		newCurrentValue = newQuantity.Mul(*currentPrice)
		newProfitLoss = newCurrentValue.Sub(newTotalValueAtBuy)

		if !newTotalValueAtBuy.IsZero() {
			newProfitLossPercent = newProfitLoss.Div(newTotalValueAtBuy).Mul(decimal.NewFromInt(100))
		}
	} else {
		newCurrentValue = decimal.Zero
		newProfitLoss = decimal.Zero
		newProfitLossPercent = decimal.Zero
	}

	updates := map[string]interface{}{
		"quantity":            newQuantity,
		"average_buy_price":   newAverageBuyPrice,
		"value_at_buy":        newTotalValueAtBuy,
		"current_value":       newCurrentValue,
		"profit_loss":         newProfitLoss,
		"profit_loss_percent": newProfitLossPercent,
	}

	if currentPrice != nil {
		updates["current_price"] = currentPrice
	}
	if lastPriceUpdate != nil {
		updates["last_price_update"] = lastPriceUpdate
	}

	return db.Model(&models.InvestmentHolding{}).
		Where("id = ?", holdingID).
		Updates(updates).Error
}

func (r *InvestmentRepository) FindTotalInvestmentValue(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var holdings []models.InvestmentHolding
	err := db.WithContext(ctx).
		Where("account_id = ? AND user_id = ? AND quantity > 0", accountID, userID).
		Find(&holdings).Error

	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, h := range holdings {
		total = total.Add(h.CurrentValue)
	}

	return total, nil
}
