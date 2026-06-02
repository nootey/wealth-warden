package repositories

import (
	"context"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type BackofficeRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	GetZeroCostBuyTrades(ctx context.Context) ([]models.InvestmentTrade, error)
}
type BackofficeRepository struct {
	db *gorm.DB
}

func NewBackofficeRepository(db *gorm.DB) *BackofficeRepository {
	return &BackofficeRepository{db: db}
}

var _ BackofficeRepositoryInterface = (*BackofficeRepository)(nil)

func (r *BackofficeRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *BackofficeRepository) GetZeroCostBuyTrades(ctx context.Context) ([]models.InvestmentTrade, error) {
	var trades []models.InvestmentTrade
	err := r.db.WithContext(ctx).
		Preload("Asset").
		Where("price_per_unit = 0 AND trade_type = ?", models.InvestmentBuy).
		Order("asset_id, txn_date").
		Find(&trades).Error
	return trades, err
}
