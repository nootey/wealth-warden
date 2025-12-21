package repositories

import (
	"context"

	"gorm.io/gorm"
)

type InvestmentRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
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
