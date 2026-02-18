package repositories

import (
	"context"

	"gorm.io/gorm"
)

type AnalyticsRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
}
type AnalyticsRepository struct {
	db *gorm.DB
}

func (r AnalyticsRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

var _ AnalyticsRepositoryInterface = (*AnalyticsRepository)(nil)
