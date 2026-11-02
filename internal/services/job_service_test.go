//go:build integration

package services_test

import (
	"strconv"
	"testing"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/tests"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type JobServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationJobServiceSuite(t *testing.T) {
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

func (s *JobServiceTestSuite) newestFirst() utils.Keyset {
	ks, err := utils.NewKeyset(models.RiverJobSort, utils.PaginationParams{RowsPerPage: 20, SortField: "created_at", SortOrder: "desc"})
	s.Require().NoError(err)
	return ks
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
		Pagination: utils.PaginationParams{RowsPerPage: 10, SortField: "id", SortOrder: "asc"},
	})
	s.Require().NoError(err)
	s.Require().Len(all, 2)
	want := all[1]

	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
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
		Pagination: utils.PaginationParams{RowsPerPage: 10},
		States:     []string{"available", "bogus"},
	})
	s.Require().ErrorIs(err, services.ErrInvalidJobState)
}

func (s *JobServiceTestSuite) TestFetchJobsPaginates() {
	s.clearJobs()
	for range 5 {
		s.insertJob("notification", "default", "available")
	}

	page := utils.PaginationParams{RowsPerPage: 2, SortField: "id", SortOrder: "asc"}
	first, paginator, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{Pagination: page})
	s.Require().NoError(err)
	s.Require().Len(first, 2)
	s.Require().NotNil(paginator.NextCursor)

	page.Cursor = *paginator.NextCursor
	rows, paginator, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{Pagination: page})
	s.Require().NoError(err)
	s.Len(rows, 2)
	s.Equal(5, paginator.TotalRecords)
	s.Greater(rows[0].ID, first[1].ID, "the second page starts after the first")
	s.NotNil(paginator.NextCursor)
	s.NotNil(paginator.PrevCursor)
}

// A finished job's duration is fixed, so paging by it must neither skip nor repeat a job,
// including ties and never-attempted jobs whose duration is NULL.
func (s *JobServiceTestSuite) TestFetchJobsPagesByDuration() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")
	for _, d := range []string{"1.5 seconds", "3.125 seconds", "1 microsecond", "2 minutes", "1.5 seconds"} {
		s.Require().NoError(s.TC.DB.Exec(`INSERT INTO river_job (kind, queue, state, max_attempts, args, metadata, attempted_at, finalized_at)
			VALUES ('notification', 'default', 'completed'::river_job_state, 25, '{}'::jsonb, '{}'::jsonb,
			        now() - interval '1 hour', now() - interval '1 hour' + ?::interval)`, d).Error)
	}

	var want []int64
	s.Require().NoError(s.TC.DB.Raw(`SELECT id FROM river_job
		ORDER BY EXTRACT(EPOCH FROM (COALESCE(finalized_at, now()) - attempted_at)) DESC, id DESC`).Scan(&want).Error)

	page := utils.PaginationParams{RowsPerPage: 2, SortField: "duration", SortOrder: "desc"}
	var got []int64
	var pages [][]int64
	var prev *string
	for {
		rows, paginator, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{Pagination: page})
		s.Require().NoError(err)
		var ids []int64
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		got = append(got, ids...)
		pages = append(pages, ids)
		prev = paginator.PrevCursor
		if paginator.NextCursor == nil {
			break
		}
		page.Cursor = *paginator.NextCursor
	}
	s.Equal(want, got)

	s.Require().NotNil(prev)
	page.Cursor = *prev
	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{Pagination: page})
	s.Require().NoError(err)
	s.Require().Len(rows, 2)
	back := []int64{rows[0].ID, rows[1].ID}
	s.Equal(pages[len(pages)-2], back, "walking back returns the page before the last")
}

func (s *JobServiceTestSuite) TestFetchJobsSortsByNewlyWhitelistedFields() {
	s.clearJobs()
	s.insertJob("notification", "queue-b", "available")
	s.insertJob("notification", "queue-a", "available")
	s.insertJob("notification", "queue-c", "available")

	rows, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			RowsPerPage: 10,
			SortField:   "queue", SortOrder: "asc",
		},
	})
	s.Require().NoError(err)
	s.Require().Len(rows, 3)
	s.Equal([]string{"queue-a", "queue-b", "queue-c"},
		[]string{rows[0].Queue, rows[1].Queue, rows[2].Queue})

	// duration maps to a SQL expression, not a column; it must order without error.
	_, _, err = s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{
			RowsPerPage: 10,
			SortField:   "duration", SortOrder: "desc",
		},
	})
	s.Require().NoError(err)
}

func (s *JobServiceTestSuite) TestFetchJobsIgnoresUnknownSortField() {
	s.clearJobs()
	s.insertJob("notification", "default", "available")

	// A non-whitelisted sort field must not reach the query; it falls back to id.
	_, _, err := s.svc().FetchJobs(s.Ctx, services.JobQueryParams{
		Pagination: utils.PaginationParams{RowsPerPage: 10, SortField: "id; DROP TABLE river_job"},
	})
	s.Require().NoError(err)

	var count int64
	s.Require().NoError(s.TC.DB.Raw("SELECT count(*) FROM river_job").Scan(&count).Error)
	s.Equal(int64(1), count)
}

// Guards the allowlist gate itself, not any one kind.
func (s *JobServiceTestSuite) TestListUserJobsRejectsUnregisteredKind() {
	_, _, err := s.svc().ListUserJobs(s.Ctx, 1, "not_a_real_kind", utils.PaginationParams{})
	s.Require().ErrorIs(err, services.ErrJobKindNotAllowed)
}

func (s *JobServiceTestSuite) TestListUserJobsPagesOwnJobsOnly() {
	s.clearJobs()
	var own []int64
	for range 3 {
		own = append(own, s.insertUserJob(jobqueue.TypeMergeCategories, "completed", 1))
	}
	s.insertUserJob(jobqueue.TypeMergeCategories, "completed", 2)

	first, paginator, err := s.svc().ListUserJobs(s.Ctx, 1, jobqueue.TypeMergeCategories,
		utils.PaginationParams{RowsPerPage: 2})
	s.Require().NoError(err)
	s.Require().Len(first, 2)
	s.Equal(own[2], first[0].ID)
	s.Equal(3, paginator.TotalRecords)
	s.Require().NotNil(paginator.NextCursor)

	second, paginator, err := s.svc().ListUserJobs(s.Ctx, 1, jobqueue.TypeMergeCategories,
		utils.PaginationParams{RowsPerPage: 2, Cursor: *paginator.NextCursor})
	s.Require().NoError(err)
	s.Require().Len(second, 1)
	s.Equal(own[0], second[0].ID)
	s.Nil(paginator.NextCursor)
}

func (s *JobServiceTestSuite) TestFindUserJobsScopesToOwnerNewestFirst() {
	s.clearJobs()
	first := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "completed", 1)
	second := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 1)
	s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 2)

	rows, err := s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, s.newestFirst())
	s.Require().NoError(err)
	s.Require().Len(rows, 2)
	s.Equal(second, rows[0].ID)
	s.Equal(first, rows[1].ID)
}

func (s *JobServiceTestSuite) TestFindUserJobsExcludesOtherKinds() {
	s.clearJobs()
	s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 1)
	s.insertUserJob(jobqueue.TypeRecalculateAssetPnL, "running", 1)

	rows, err := s.repo().FindUserJobs(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, s.newestFirst())
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal(jobqueue.TypeGenerateCategoryReport, rows[0].Kind)
}

// Lookup by id is how RetryUserJob/CancelUserJob prove ownership before acting.
func (s *JobServiceTestSuite) TestFindUserJobRejectsOtherUsersJob() {
	s.clearJobs()
	id := s.insertUserJob(jobqueue.TypeGenerateCategoryReport, "running", 2)

	_, err := s.repo().FindUserJob(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 1, id)
	s.Require().ErrorIs(err, gorm.ErrRecordNotFound)

	row, err := s.repo().FindUserJob(s.Ctx, []string{jobqueue.TypeGenerateCategoryReport}, 2, id)
	s.Require().NoError(err)
	s.Equal(id, row.ID)
}
