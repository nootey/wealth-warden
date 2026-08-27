package repositories

import (
	"context"

	"gorm.io/gorm"
)

type JobRepositoryInterface interface {
	OldestPendingAgeSeconds(ctx context.Context) (int64, error)
}

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

var _ JobRepositoryInterface = (*JobRepository)(nil)

func (r *JobRepository) OldestPendingAgeSeconds(ctx context.Context) (int64, error) {
	var ageSeconds int64
	err := r.db.WithContext(ctx).
		Raw(`SELECT COALESCE(EXTRACT(EPOCH FROM now() - min(created_at)), 0)::bigint
			 FROM river_job WHERE state IN ('available', 'scheduled', 'retryable')`).
		Scan(&ageSeconds).Error
	return ageSeconds, err
}
