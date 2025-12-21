package repositories

import (
	"context"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type InvestmentRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
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
