package services

import (
	"context"
	"errors"
	"fmt"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/riverqueue/river/rivertype"
	"go.uber.org/zap"
)

type periodicJobDisplay struct {
	ID       string
	Kind     string
	Schedule string
	Queue    string
}

var (
	periodicJobDisplays = []periodicJobDisplay{
		{jobqueue.TypeAssetHistoryBackfill, jobqueue.TypeAssetHistoryBackfill, "Daily at 00:00 UTC", jobqueue.QueueScheduler},
		{jobqueue.TypeBalanceBackfill, jobqueue.TypeBalanceBackfill, "Daily at 00:10 UTC", jobqueue.QueueScheduler},
		{jobqueue.TypeRecurringTransactions, jobqueue.TypeRecurringTransactions, "Daily at 00:20 UTC", jobqueue.QueueScheduler},
		{jobqueue.TypeAssetPriceSync, jobqueue.TypeAssetPriceSync, "Every 8 hours", jobqueue.QueueScheduler},
	}
	ErrInvalidJobState = errors.New("invalid job state")
)

type JobQueryParams struct {
	Pagination utils.PaginationParams
	States     []string
}

type JobAdminServiceInterface interface {
	FetchJobs(ctx context.Context, p JobQueryParams) ([]models.RiverJobRow, *utils.Paginator, error)
	FetchJobCounts(ctx context.Context) (map[string]int64, error)
	FetchJob(ctx context.Context, id int64) (*models.RiverJobDetail, error)
	RetryJobs(ctx context.Context, ids []int64) error
	CancelJobs(ctx context.Context, ids []int64) error
	DeleteJobs(ctx context.Context, ids []int64) error
	FetchQueues(ctx context.Context) ([]models.RiverQueueRow, error)
	PauseQueue(ctx context.Context, name string) error
	ResumeQueue(ctx context.Context, name string) error
	FetchPeriodicJobs(ctx context.Context) ([]models.RiverPeriodicJob, error)
}

type JobAdminService struct {
	logger *zap.Logger
	repo   repositories.JobRepositoryInterface
	jobs   jobqueue.JobManager
}

func NewJobAdminService(
	logger *zap.Logger,
	repo repositories.JobRepositoryInterface,
	jobs jobqueue.JobManager,
) *JobAdminService {
	return &JobAdminService{logger: logger, repo: repo, jobs: jobs}
}

var _ JobAdminServiceInterface = (*JobAdminService)(nil)

func (s *JobAdminService) FetchJobs(ctx context.Context, p JobQueryParams) ([]models.RiverJobRow, *utils.Paginator, error) {
	for _, st := range p.States {
		if !utils.ValidJobState(st) {
			return nil, nil, fmt.Errorf("%w: %q", ErrInvalidJobState, st)
		}
	}

	sortExpr := models.RiverJobSortFields["id"]
	if expr, ok := models.RiverJobSortFields[p.Pagination.SortField]; ok {
		sortExpr = expr
	}
	sortOrder := "desc"
	if p.Pagination.SortOrder == "asc" {
		sortOrder = "asc"
	}

	rowsPerPage := p.Pagination.RowsPerPage
	if rowsPerPage <= 0 {
		rowsPerPage = 10
	}
	if rowsPerPage > 200 {
		rowsPerPage = 200
	}
	pageNumber := p.Pagination.PageNumber
	if pageNumber < 1 {
		pageNumber = 1
	}

	filter := repositories.JobListFilter{
		States:  p.States,
		Filters: p.Pagination.Filters,
	}

	total, err := s.repo.CountJobs(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	offset := (pageNumber - 1) * rowsPerPage
	rows, err := s.repo.ListJobs(ctx, filter, offset, rowsPerPage, sortExpr, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(total) {
		from = int(total)
	}
	to := offset + len(rows)
	if to > int(total) {
		to = int(total)
	}

	paginator := &utils.Paginator{
		CurrentPage:  pageNumber,
		RowsPerPage:  rowsPerPage,
		TotalRecords: int(total),
		From:         from,
		To:           to,
	}
	return rows, paginator, nil
}

func (s *JobAdminService) FetchJobCounts(ctx context.Context) (map[string]int64, error) {
	buckets, err := s.repo.CountJobsByState(ctx)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(models.RiverJobStates))
	for _, st := range models.RiverJobStates {
		counts[st] = 0
	}
	for _, b := range buckets {
		counts[b.State] = b.Count
	}
	return counts, nil
}

func (s *JobAdminService) FetchJob(ctx context.Context, id int64) (*models.RiverJobDetail, error) {
	row, err := s.jobs.JobGet(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, rivertype.ErrNotFound
	}

	detail := &models.RiverJobDetail{
		ID:          row.ID,
		Kind:        row.Kind,
		Queue:       row.Queue,
		State:       string(row.State),
		Attempt:     row.Attempt,
		MaxAttempts: row.MaxAttempts,
		Priority:    row.Priority,
		AttemptedBy: row.AttemptedBy,
		Tags:        row.Tags,
		Args:        row.EncodedArgs,
		Metadata:    row.Metadata,
		CreatedAt:   row.CreatedAt,
		ScheduledAt: row.ScheduledAt,
		AttemptedAt: row.AttemptedAt,
		FinalizedAt: row.FinalizedAt,
	}
	for _, e := range row.Errors {
		detail.Errors = append(detail.Errors, models.RiverAttemptError{
			At:      e.At,
			Attempt: e.Attempt,
			Error:   e.Error,
			Trace:   e.Trace,
		})
	}
	return detail, nil
}

func (s *JobAdminService) RetryJobs(ctx context.Context, ids []int64) error {
	return s.applyToJobs(ctx, "retry", ids, func(id int64) error {
		_, err := s.jobs.JobRetry(ctx, id)
		return err
	})
}

func (s *JobAdminService) CancelJobs(ctx context.Context, ids []int64) error {
	return s.applyToJobs(ctx, "cancel", ids, func(id int64) error {
		_, err := s.jobs.JobCancel(ctx, id)
		return err
	})
}

func (s *JobAdminService) DeleteJobs(ctx context.Context, ids []int64) error {
	return s.applyToJobs(ctx, "delete", ids, func(id int64) error {
		_, err := s.jobs.JobDelete(ctx, id)
		return err
	})
}

func (s *JobAdminService) applyToJobs(ctx context.Context, action string, ids []int64, fn func(int64) error) error {
	if len(ids) == 0 {
		return errors.New("no job ids provided")
	}

	var firstErr error
	for _, id := range ids {
		if err := fn(id); err != nil {
			s.logger.Warn("job action failed",
				zap.String("action", action), zap.Int64("job_id", id), zap.Error(err))
			if firstErr == nil {
				firstErr = fmt.Errorf("%s job %d: %w", action, id, err)
			}
		}
	}
	return firstErr
}

func (s *JobAdminService) FetchQueues(ctx context.Context) ([]models.RiverQueueRow, error) {
	queues, err := s.jobs.QueueList(ctx)
	if err != nil {
		return nil, err
	}

	buckets, err := s.repo.CountJobsByQueueState(ctx)
	if err != nil {
		return nil, err
	}

	countsByQueue := make(map[string]map[string]int64)
	for _, b := range buckets {
		if countsByQueue[b.Queue] == nil {
			countsByQueue[b.Queue] = make(map[string]int64)
		}
		countsByQueue[b.Queue][b.State] = b.Count
	}

	rows := make([]models.RiverQueueRow, 0, len(queues))
	seen := make(map[string]struct{}, len(queues))
	for _, q := range queues {
		seen[q.Name] = struct{}{}
		rows = append(rows, models.RiverQueueRow{
			Name:     q.Name,
			PausedAt: q.PausedAt,
			Counts:   countsByQueue[q.Name],
		})
	}
	// Queues with jobs but no active client record (e.g. never worked yet).
	for name, counts := range countsByQueue {
		if _, ok := seen[name]; ok {
			continue
		}
		rows = append(rows, models.RiverQueueRow{Name: name, Counts: counts})
	}
	return rows, nil
}

func (s *JobAdminService) PauseQueue(ctx context.Context, name string) error {
	return s.jobs.QueuePause(ctx, name)
}

func (s *JobAdminService) ResumeQueue(ctx context.Context, name string) error {
	return s.jobs.QueueResume(ctx, name)
}

func (s *JobAdminService) FetchPeriodicJobs(ctx context.Context) ([]models.RiverPeriodicJob, error) {
	kinds := make([]string, 0, len(periodicJobDisplays))
	for _, d := range periodicJobDisplays {
		kinds = append(kinds, d.Kind)
	}

	lastRuns, err := s.repo.LastRunByKind(ctx, kinds)
	if err != nil {
		return nil, err
	}
	lastRunByKind := make(map[string]models.RiverKindLastRun, len(lastRuns))
	for _, lr := range lastRuns {
		lastRunByKind[lr.Kind] = lr
	}

	out := make([]models.RiverPeriodicJob, 0, len(periodicJobDisplays))
	for _, d := range periodicJobDisplays {
		row := models.RiverPeriodicJob{
			ID:       d.ID,
			Kind:     d.Kind,
			Schedule: d.Schedule,
			Queue:    d.Queue,
		}
		if lr, ok := lastRunByKind[d.Kind]; ok {
			t := lr.LastRunAt
			row.LastRunAt = &t
		}
		out = append(out, row)
	}
	return out, nil
}
