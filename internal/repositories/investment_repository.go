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
	CountInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	CountInvestmentTrades(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error)
	FindInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentAsset, error)
	FindAllInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64) ([]models.InvestmentAsset, error)
	FindInvestmentAssetByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentAsset, error)
	FindAssetsByAccountID(ctx context.Context, tx *gorm.DB, accID, userID int64) ([]models.InvestmentAsset, error)
	FindInvestmentTrades(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTrade, error)
	FindInvestmentTradeByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentTrade, error)
	InsertAsset(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentAsset) (int64, error)
	InsertInvestmentTrade(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentTrade) (int64, error)
	UpdateAssetAfterTrade(ctx context.Context, tx *gorm.DB, assetID int64, quantity decimal.Decimal, pricePerUnit decimal.Decimal, currentPrice *decimal.Decimal, lastPriceUpdate *time.Time, TradeType models.TradeType, TradeValueAtBuy decimal.Decimal) error
	FindTotalInvestmentValue(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error)
	UpdateInvestmentAsset(ctx context.Context, tx *gorm.DB, record models.InvestmentAsset) (int64, error)
	UpdateInvestmentTrade(ctx context.Context, tx *gorm.DB, record models.InvestmentTrade) (int64, error)
	RecalculateAssetFromTrades(ctx context.Context, tx *gorm.DB, assetID, userID int64) error
	DeleteInvestmentTrade(ctx context.Context, tx *gorm.DB, id int64) error
	GetEarliestTradeDate(ctx context.Context, tx *gorm.DB, assetID, userID int64) (time.Time, error)
	FindSellTradesByAssetID(ctx context.Context, tx *gorm.DB, assetID, userID int64) ([]models.InvestmentTrade, error)
	DeleteAllTradesForAsset(ctx context.Context, tx *gorm.DB, assetID, userID int64) error
	DeleteInvestmentAsset(ctx context.Context, tx *gorm.DB, id int64) error
	GetInvestmentTotalsUpToDate(ctx context.Context, tx *gorm.DB, assetID int64, asOf time.Time) (decimal.Decimal, decimal.Decimal, error)
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

func (r *InvestmentRepository) CountInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.WithContext(ctx).Model(&models.InvestmentAsset{}).
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

func (r *InvestmentRepository) CountInvestmentTrades(ctx context.Context, tx *gorm.DB, userID int64, filters []utils.Filter, accountID *int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	q := db.WithContext(ctx).Model(&models.InvestmentTrade{}).
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

func (r *InvestmentRepository) FindInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentAsset, error) {

	var records []models.InvestmentAsset
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.InvestmentAsset{}).
		Where("investment_assets.user_id = ?", userID).
		Preload("Account")

	if accountID != nil {
		q = q.Where("investment_assets.account_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "investment_assets", sortField, sortOrder)

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

func (r *InvestmentRepository) FindAllInvestmentAssets(ctx context.Context, tx *gorm.DB, userID int64) ([]models.InvestmentAsset, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.InvestmentAsset
	query := db.Where("user_id = ?", userID)

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *InvestmentRepository) FindInvestmentAssetByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentAsset, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.InvestmentAsset
	q := db.
		Preload("Account.Balance", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(1)
		}).
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *InvestmentRepository) FindAssetsByAccountID(ctx context.Context, tx *gorm.DB, accID, userID int64) ([]models.InvestmentAsset, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.InvestmentAsset
	q := db.
		Where("user_id = ? AND account_id = ?", userID, accID)

	q = q.Find(&records)

	return records, q.Error
}

func (r *InvestmentRepository) FindInvestmentTrades(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int, sortField, sortOrder string, filters []utils.Filter, accountID *int64) ([]models.InvestmentTrade, error) {

	var records []models.InvestmentTrade
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.WithContext(ctx).Model(&models.InvestmentTrade{}).
		Where("investment_trades.user_id = ?", userID).
		Preload("Asset")

	if accountID != nil {
		q = q.Where("investment_trades.asset_id = ?", *accountID)
	}

	joins := utils.GetRequiredJoins(filters)
	orderBy := utils.ConstructOrderByClause(&joins, "investment_trades", sortField, sortOrder)

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

func (r *InvestmentRepository) FindInvestmentTradeByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.InvestmentTrade, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.InvestmentTrade
	q := db.
		Preload("Asset").
		Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *InvestmentRepository) InsertAsset(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentAsset) (int64, error) {
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

func (r *InvestmentRepository) InsertInvestmentTrade(ctx context.Context, tx *gorm.DB, newRecord *models.InvestmentTrade) (int64, error) {
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

func (r *InvestmentRepository) UpdateAssetAfterTrade(ctx context.Context, tx *gorm.DB, assetID int64, quantity decimal.Decimal, pricePerUnit decimal.Decimal, currentPrice *decimal.Decimal, lastPriceUpdate *time.Time, tradeType models.TradeType, tradeValueAtBuy decimal.Decimal) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var asset models.InvestmentAsset
	if err := db.First(&asset, assetID).Error; err != nil {
		return err
	}

	// Update quantity based on trade type
	var newQuantity decimal.Decimal
	var newAverageBuyPrice decimal.Decimal
	var newTotalValueAtBuy decimal.Decimal

	if tradeType == models.InvestmentBuy {
		newQuantity = asset.Quantity.Add(quantity)

		// Weighted average: (old_qty * old_avg + new_qty * new_price) / total_qty
		oldValue := asset.Quantity.Mul(asset.AverageBuyPrice)
		newValue := quantity.Mul(pricePerUnit)
		totalValue := oldValue.Add(newValue)
		newAverageBuyPrice = totalValue.Div(newQuantity)

		// Total value at buy
		newTotalValueAtBuy = asset.ValueAtBuy.Add(tradeValueAtBuy)
	} else {
		// Sell: decrease quantity and value at buy proportionally
		newQuantity = asset.Quantity.Sub(quantity)
		newAverageBuyPrice = asset.AverageBuyPrice

		// Reduce total value at buy proportionally
		soldProportion := quantity.Div(asset.Quantity)
		valueAtBuyReduction := asset.ValueAtBuy.Mul(soldProportion)
		newTotalValueAtBuy = asset.ValueAtBuy.Sub(valueAtBuyReduction)
	}

	// Calculate current value
	var newCurrentValue decimal.Decimal
	var newProfitLoss decimal.Decimal
	var newProfitLossPercent decimal.Decimal

	if currentPrice != nil && !currentPrice.IsZero() {
		newCurrentValue = newQuantity.Mul(*currentPrice)
		newProfitLoss = newCurrentValue.Sub(newTotalValueAtBuy)

		if !newTotalValueAtBuy.IsZero() {
			newProfitLossPercent = newProfitLoss.Div(newTotalValueAtBuy)
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

	return db.Model(&models.InvestmentAsset{}).
		Where("id = ?", assetID).
		Updates(updates).Error
}

func (r *InvestmentRepository) FindTotalInvestmentValue(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var assets []models.InvestmentAsset
	err := db.WithContext(ctx).
		Where("account_id = ? AND user_id = ? AND quantity > 0", accountID, userID).
		Find(&assets).Error

	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, a := range assets {
		total = total.Add(a.CurrentValue)
	}

	return total, nil
}

func (r *InvestmentRepository) UpdateInvestmentAsset(ctx context.Context, tx *gorm.DB, record models.InvestmentAsset) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.InvestmentAsset{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":       record.Name,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *InvestmentRepository) UpdateInvestmentTrade(ctx context.Context, tx *gorm.DB, record models.InvestmentTrade) (int64, error) {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if record.Description != nil {
		updates["description"] = *record.Description
	} else {
		updates["description"] = gorm.Expr("NULL")
	}

	if err := db.Model(models.InvestmentTrade{}).
		Where("id = ?", record.ID).
		Updates(updates).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *InvestmentRepository) RecalculateAssetFromTrades(ctx context.Context, tx *gorm.DB, assetID, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var asset models.InvestmentAsset
	if err := db.Preload("Account").First(&asset, assetID).Error; err != nil {
		return err
	}

	// Get all trades for this asset
	var trades []models.InvestmentTrade
	if err := db.Where("asset_id = ? AND user_id = ?", assetID, userID).
		Order("txn_date ASC, id ASC").
		Find(&trades).Error; err != nil {
		return err
	}

	// Start fresh
	totalQuantity := decimal.Zero
	totalValueAtBuy := decimal.Zero
	totalCostForAverage := decimal.Zero

	for _, txn := range trades {
		if txn.TradeType == models.InvestmentBuy {
			totalQuantity = totalQuantity.Add(txn.Quantity)
			totalValueAtBuy = totalValueAtBuy.Add(txn.ValueAtBuy)
			// For average: use quantity * price_per_unit
			totalCostForAverage = totalCostForAverage.Add(txn.Quantity.Mul(txn.PricePerUnit))
		} else {
			// Sell: reduce proportionally
			totalQuantity = totalQuantity.Sub(txn.Quantity)
			if totalQuantity.GreaterThan(decimal.Zero) {
				soldProportion := txn.Quantity.Div(totalQuantity.Add(txn.Quantity))
				totalValueAtBuy = totalValueAtBuy.Mul(decimal.NewFromInt(1).Sub(soldProportion))
				totalCostForAverage = totalCostForAverage.Mul(decimal.NewFromInt(1).Sub(soldProportion))
			} else {
				totalValueAtBuy = decimal.Zero
				totalCostForAverage = decimal.Zero
			}
		}
	}

	// Calculate average buy price
	var avgBuyPrice decimal.Decimal
	if totalQuantity.GreaterThan(decimal.Zero) {
		avgBuyPrice = totalCostForAverage.Div(totalQuantity)
	} else {
		avgBuyPrice = decimal.Zero
	}

	// Calculate current values
	var currentValue, profitLoss, profitLossPercent decimal.Decimal
	if asset.CurrentPrice != nil && !asset.CurrentPrice.IsZero() && totalQuantity.GreaterThan(decimal.Zero) {
		currentValue = totalQuantity.Mul(*asset.CurrentPrice)
		profitLoss = currentValue.Sub(totalValueAtBuy)
		if !totalValueAtBuy.IsZero() {
			profitLossPercent = profitLoss.Div(totalValueAtBuy)
		}
	}

	// Update asset
	return db.Model(&models.InvestmentAsset{}).
		Where("id = ?", assetID).
		Updates(map[string]interface{}{
			"quantity":            totalQuantity,
			"average_buy_price":   avgBuyPrice,
			"value_at_buy":        totalValueAtBuy,
			"current_value":       currentValue,
			"profit_loss":         profitLoss,
			"profit_loss_percent": profitLossPercent,
			"updated_at":          time.Now().UTC(),
		}).Error
}

func (r *InvestmentRepository) DeleteInvestmentTrade(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Delete(&models.InvestmentTrade{}, id).Error
}

func (r *InvestmentRepository) GetEarliestTradeDate(ctx context.Context, tx *gorm.DB, assetID, userID int64) (time.Time, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var txn models.InvestmentTrade
	err := db.Where("asset_id = ? AND user_id = ?", assetID, userID).
		Order("txn_date ASC").
		First(&txn).Error

	if err != nil {
		return time.Time{}, err
	}

	return txn.TxnDate, nil
}

func (r *InvestmentRepository) FindSellTradesByAssetID(ctx context.Context, tx *gorm.DB, assetID, userID int64) ([]models.InvestmentTrade, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var trades []models.InvestmentTrade
	err := db.Where("asset_id = ? AND user_id = ? AND trade_type = ?", assetID, userID, models.InvestmentSell).
		Order("txn_date ASC").
		Find(&trades).Error

	return trades, err
}

func (r *InvestmentRepository) DeleteAllTradesForAsset(ctx context.Context, tx *gorm.DB, assetID, userID int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("asset_id = ? AND user_id = ?", assetID, userID).
		Delete(&models.InvestmentTrade{}).Error
}

func (r *InvestmentRepository) DeleteInvestmentAsset(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Delete(&models.InvestmentAsset{}, id).Error
}

func (r *InvestmentRepository) GetInvestmentTotalsUpToDate(ctx context.Context, tx *gorm.DB, assetID int64, asOf time.Time) (decimal.Decimal, decimal.Decimal, error) {
	db := tx
	if db == nil {
		db = r.db
	}

	var result struct {
		Quantity decimal.Decimal
		Spent    decimal.Decimal
	}

	err := db.Raw(`
        SELECT 
            COALESCE(SUM(CASE WHEN trade_type = 'buy' THEN quantity ELSE -quantity END), 0) as quantity,
            COALESCE(SUM(CASE WHEN trade_type = 'buy' THEN value_at_buy ELSE 0 END), 0) as spent
        FROM investment_trades
        WHERE asset_id = ? AND txn_date <= ?
    `, assetID, asOf).Scan(&result).Error

	return result.Quantity, result.Spent, err
}
