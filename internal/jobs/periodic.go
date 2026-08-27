package jobs

import (
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/pkg/config"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// River's default set also holds `completed`, which would block tomorrow's run
// while yesterday's row is still in the table.
var periodicUniqueStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRetryable,
	rivertype.JobStateRunning,
	rivertype.JobStateScheduled,
}

type periodicSpec struct {
	schedule   river.PeriodicSchedule
	args       river.JobArgs
	runOnStart bool
}

func PeriodicJobs(cfg config.SchedulerConfig) []*river.PeriodicJob {
	specs := periodicSpecs(cfg)
	jobs := make([]*river.PeriodicJob, 0, len(specs))

	for _, spec := range specs {
		constructor := func() (river.JobArgs, *river.InsertOpts) {
			return spec.args, &river.InsertOpts{
				Queue:      jobqueue.QueueScheduler,
				UniqueOpts: river.UniqueOpts{ByState: periodicUniqueStates},
			}
		}
		jobs = append(jobs, river.NewPeriodicJob(spec.schedule, constructor, &river.PeriodicJobOpts{
			ID:         spec.args.Kind(),
			RunOnStart: spec.runOnStart,
		}))
	}

	return jobs
}

func periodicSpecs(cfg config.SchedulerConfig) []periodicSpec {
	immediate := make(map[string]bool, len(cfg.ImmediateJobs))
	for _, job := range cfg.ImmediateJobs {
		immediate[job] = true
	}

	return []periodicSpec{
		{dailyAt{0, 0}, jobqueue.AssetPriceHistoryBackfillArgs{}, immediate["asset_history_backfill"]},
		{dailyAt{0, 6}, jobqueue.BalanceBackfillArgs{}, immediate["balance_backfill"]},
		{dailyAt{0, 10}, jobqueue.RecurringTransactionsArgs{}, immediate["recurring_transactions"]},
		{river.PeriodicInterval(8 * time.Hour), jobqueue.AssetPriceSyncArgs{}, immediate["asset_price_sync"]},
	}
}
