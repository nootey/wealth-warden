package repositories

import (
	"context"
	"strconv"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

const jobListColumns = "id, kind, queue, state, attempt, max_attempts, priority, args, metadata, created_at, scheduled_at, attempted_at, finalized_at"

type JobListFilter struct {
	States  []string
	Filters []utils.Filter
}

type JobRepositoryInterface interface {
	OldestPendingAgeSeconds(ctx context.Context) (int64, error)
	CountJobs(ctx context.Context, filter JobListFilter) (int64, error)
	ListJobs(ctx context.Context, filter JobListFilter, offset, limit int, sortExpr, sortOrder string) ([]models.RiverJobRow, error)
	CountJobsByState(ctx context.Context) ([]models.RiverJobStateCount, error)
	CountJobsByQueueState(ctx context.Context) ([]models.RiverQueueStateCount, error)
	LastRunByKind(ctx context.Context, kinds []string) ([]models.RiverKindLastRun, error)
	FindUserJobs(ctx context.Context, kinds []string, userID int64, id *int64, limit int) ([]models.RiverJobRow, error)
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

func (r *JobRepository) applyJobFilter(query *gorm.DB, f JobListFilter) *gorm.DB {
	if len(f.States) > 0 {
		query = query.Where("state IN ?", f.States)
	}
	return utils.ApplyFilters(query, f.Filters)
}

func (r *JobRepository) CountJobs(ctx context.Context, f JobListFilter) (int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Table("river_job")
	err := r.applyJobFilter(query, f).Count(&total).Error
	return total, err
}

func (r *JobRepository) ListJobs(ctx context.Context, f JobListFilter, offset, limit int, sortExpr, sortOrder string) ([]models.RiverJobRow, error) {
	var rows []models.RiverJobRow
	query := r.db.WithContext(ctx).Table("river_job").Select(jobListColumns)
	query = r.applyJobFilter(query, f)

	err := query.
		Order(sortExpr + " " + sortOrder).
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *JobRepository) CountJobsByState(ctx context.Context) ([]models.RiverJobStateCount, error) {
	var out []models.RiverJobStateCount
	err := r.db.WithContext(ctx).
		Table("river_job").
		Select("state, count(*) AS count").
		Group("state").
		Scan(&out).Error
	return out, err
}

func (r *JobRepository) CountJobsByQueueState(ctx context.Context) ([]models.RiverQueueStateCount, error) {
	var out []models.RiverQueueStateCount
	err := r.db.WithContext(ctx).
		Table("river_job").
		Select("queue, state, count(*) AS count").
		Group("queue, state").
		Scan(&out).Error
	return out, err
}

func (r *JobRepository) LastRunByKind(ctx context.Context, kinds []string) ([]models.RiverKindLastRun, error) {
	var out []models.RiverKindLastRun
	if len(kinds) == 0 {
		return out, nil
	}
	err := r.db.WithContext(ctx).
		Table("river_job").
		Select("kind, max(created_at) AS last_run_at").
		Where("kind IN ?", kinds).
		Group("kind").
		Scan(&out).Error
	return out, err
}

func (r *JobRepository) FindUserJobs(ctx context.Context, kinds []string, userID int64, id *int64, limit int) ([]models.RiverJobRow, error) {
	var rows []models.RiverJobRow
	query := r.db.WithContext(ctx).
		Table("river_job").
		Select(jobListColumns).
		Where("kind IN ? AND args->>'UserID' = ?", kinds, strconv.FormatInt(userID, 10))
	if id != nil {
		query = query.Where("id = ?", *id)
	}
	err := query.
		Order("id DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
