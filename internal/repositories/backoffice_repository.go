package repositories

import (
	"context"

	"gorm.io/gorm"
)

type BackofficeRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
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
