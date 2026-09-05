package services_test

import (
	"strconv"
	"testing"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type JobServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestJobServiceSuite(t *testing.T) {
	suite.Run(t, new(JobServiceTestSuite))
}

func (s *JobServiceTestSuite) svc() *services.JobService {
	return services.NewJobService(
		zap.NewNop(),
		repositories.NewJobRepository(s.TC.DB),
		jobqueue.NoopJobManager{},
		&tests.NoOpDispatcher{},
		ws.NoopBroadcaster{},
	)
}

func (s *JobServiceTestSuite) repo() repositories.JobRepositoryInterface {
	return repositories.NewJobRepository(s.TC.DB)
}

func (s *JobServiceTestSuite) clearJobs() {
	s.Require().NoError(s.TC.DB.Exec("DELETE FROM river_job").Error)
}

func (s *JobServiceTestSuite) insertJob(kind, queue, state string) {
	finalized := state == "completed" || state == "cancelled" || state == "discarded"
	sql := `INSERT INTO river_job (kind, queue, state, max_attempts, args, metadata, finalized_at)
	        VALUES (?, ?, ?::river_job_state, 25, '{}'::jsonb, '{}'::jsonb, CASE WHEN ? THEN now() ELSE NULL END)`
	s.Require().NoError(s.TC.DB.Exec(sql, kind, queue, state, finalized).Error)
}

func (s *JobServiceTestSuite) insertUserJob(kind, state string, userID int64) int64 {
	finalized := state == "completed" || state == "cancelled" || state == "discarded"
	sql := `INSERT INTO river_job (kind, queue, state, max_attempts, args, metadata, finalized_at)
	        VALUES (?, 'default', ?::river_job_state, 25, jsonb_build_object('UserID', ?::bigint), '{}'::jsonb, CASE WHEN ? THEN now() ELSE NULL END)
	        RETURNING id`
	var id int64
	s.Require().NoError(s.TC.DB.Raw(sql, kind, state, userID, finalized).Scan(&id).Error)
	return id
}

func (s *JobServiceTestSuite) TestFetchJobCountsZeroFillsEveryState() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")
	s.insertJob("notification", "default", "available")
	s.insertJob("asset_price_sync", "scheduler", "completed")

	counts, err := s.svc().FetchJobCounts(s.Ctx)
	s.Require().NoError(err)

	s.Equal(int64(2), counts["available"])
	s.Equal(int64(1), counts["completed"])
	s.Equal(int64(0), counts["running"])
	s.Equal(int64(0), counts["discarded"])
	s.Len(counts, 8) // every known state present
}

func (s *JobServiceTestSuite) TestFetchJobsFiltersByStateAndKind() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")
	s.insertJob("notification", "default", "completed")
	s.insertJob("asset_price_sync", "scheduler", "available")

	rows, paginator, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			PageNumber:  1,
			RowsPerPage: 10,
			Filters: []utils.Filter{
				{Source: "jobs", Field: "kind", Operator: "=", Value: "notification"},
			},
		},
		States: []string{"available"},
	})
	s.Require().NoError(err)
	s.Len(rows, 1)
	s.Equal(1, paginator.TotalRecords)
	s.Equal("notification", rows[0].Kind)
	s.Equal("available", rows[0].State)
}

// Two "=" filters on the same field must OR together (queue/kind opt into
// OrEquals in the registry), not exclude every row.
func (s *JobServiceTestSuite) TestFetchJobsFiltersByMultipleKinds() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")
	s.insertJob("asset_price_sync", "scheduler", "available")
	s.insertJob("balance_backfill", "scheduler", "available")

	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			PageNumber:  1,
			RowsPerPage: 10,
			Filters: []utils.Filter{
				{Source: "jobs", Field: "kind", Operator: "=", Value: "notification"},
				{Source: "jobs", Field: "kind", Operator: "=", Value: "balance_backfill"},
			},
		},
	})
	s.Require().NoError(err)
	s.Len(rows, 2)
	s.ElementsMatch(
		[]string{"notification", "balance_backfill"},
		[]string{rows[0].Kind, rows[1].Kind},
	)
}

func (s *JobServiceTestSuite) TestFetchJobsFiltersByID() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")
	s.insertJob("asset_price_sync", "scheduler", "available")

	all, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{PageNumber: 1, RowsPerPage: 10, SortField: "id", SortOrder: "asc"},
	})
	s.Require().NoError(err)
	s.Require().Len(all, 2)
	want := all[1]

	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			PageNumber:  1,
			RowsPerPage: 10,
			Filters: []utils.Filter{
				{Source: "jobs", Field: "id", Operator: "=", Value: strconv.FormatInt(want.ID, 10)},
			},
		},
	})
	s.Require().NoError(err)
	s.Len(rows, 1)
	s.Equal(want.ID, rows[0].ID)
}

func (s *JobServiceTestSuite) TestFetchJobsRejectsUnknownState() {
	_, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{PageNumber: 1, RowsPerPage: 10},
		States:     []string{"available", "bogus"},
	})
	s.Require().ErrorIs(err, services.ErrInvalidJobState)
}

func (s *JobServiceTestSuite) TestFetchJobsPaginates() {
	s.clearJobs()
	for i := 0; i < 5; i++ {
		s.insertJob("notification", "default", "available")
	}

	rows, paginator, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{PageNumber: 2, RowsPerPage: 2, SortField: "id", SortOrder: "asc"},
	})
	s.Require().NoError(err)
	s.Len(rows, 2)
	s.Equal(5, paginator.TotalRecords)
	s.Equal(3, paginator.From)
	s.Equal(4, paginator.To)
}

func (s *JobServiceTestSuite) TestFetchJobsSortsByNewlyWhitelistedFields() {
	s.clearJobs()
	s.insertJob("notification", "queue-b", "available")
	s.insertJob("notification", "queue-a", "available")
	s.insertJob("notification", "queue-c", "available")

	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			PageNumber: 1, RowsPerPage: 10,
			SortField: "queue", SortOrder: "asc",
		},
	})
	s.Require().NoError(err)
	s.Require().Len(rows, 3)
	s.Equal([]string{"queue-a", "queue-b", "queue-c"},
		[]string{rows[0].Queue, rows[1].Queue, rows[2].Queue})

	// duration maps to a SQL expression, not a column; it must order without error.
	_, _, err = s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			PageNumber: 1, RowsPerPage: 10,
			SortField: "duration", SortOrder: "desc",
		},
	})
	s.Require().NoError(err)
}

func (s *JobServiceTestSuite) TestFetchJobsIgnoresUnknownSortField() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")

	// A non-whitelisted sort field must not reach the query; it falls back to id.
	_, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{PageNumber: 1, RowsPerPage: 10, SortField: "id; DROP TABLE river_job"},
	})
	s.Require().NoError(err)

	var count int64
	s.Require().NoError(s.TC.DB.Raw("SELECT count(*) FROM river_job").Scan(&count).Error)
	s.Equal(int64(1), count)
}

// No kind is registered in the self-service allowlist yet, so every kind is
// rejected. This guards the gate itself, not any one kind.
func (s *JobServiceTestSuite) TestListUserJobsRejectsUnregisteredKind() {
	_, err := s.svc().ListUserJobs(s.Ctx, 1, "not_a_real_kind")
	s.Require().ErrorIs(err, services.ErrJobKindNotAllowed)
}

func (s *JobServiceTestSuite) TestFindUserJobsScopesToOwnerNewestFirst() {
	s.clearJobs()
	first := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "completed", 1)
	second := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 1)
	s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 2)

	rows, err := s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, nil, 20)
	s.Require().NoError(err)
	s.Require().Len(rows, 2)
	s.Equal(second, rows[0].ID)
	s.Equal(first, rows[1].ID)
}

func (s *JobServiceTestSuite) TestFindUserJobsExcludesOtherKinds() {
	s.clearJobs()
	s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 1)
	s.insertUserJob(jobqueue.TypeRecalculateAssetPnL, "running", 1)

	rows, err := s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, nil, 20)
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal(jobqueue.TypeGenerateCategoryReport, rows[0].Kind)
}

// Lookup by id is how RetryUserJob/CancelUserJob prove ownership before acting.
func (s *JobServiceTestSuite) TestFindUserJobsByIDRejectsOtherUsersJob() {
	s.clearJobs()
	id := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 2)

	rows, err := s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, &id, 1)
	s.Require().NoError(err)
	s.Empty(rows)

	rows, err = s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 2, &id, 1)
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal(id, rows[0].ID)
}
