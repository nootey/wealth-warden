package jobscheduler

import (
	"context"
	"sort"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"

	"go.uber.org/zap"
)

type AutoFundGoalsJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	concurrentWorkers int
}

func NewAutoFundGoalsJob(logger *zap.Logger, container *bootstrap.ServiceContainer, concurrentWorkers int) *AutoFundGoalsJob {
	return &AutoFundGoalsJob{
		logger:            logger,
		container:         container,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AutoFundGoalsJob) Run(ctx context.Context) error {
	now := time.Now().UTC()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	goals, err := j.container.SavingsService.FetchActiveGoalsWithAllocation(ctx, now.Day())
	if err != nil {
		return err
	}

	if len(goals) == 0 {
		j.logger.Info("No active goals with allocation to process")
		return nil
	}

	j.logger.Info("Processing savings goals auto-fund", zap.Int("count", len(goals)), zap.String("month", month.Format("2006-01")))

	// Group goals by account so balance reads are consistent within an account
	accountGroups := make(map[int64][]models.SavingGoal)
	for _, g := range goals {
		accountGroups[g.AccountID] = append(accountGroups[g.AccountID], g)
	}

	// Sort goals within each account by priority: higher value first, 0-priority last
	for accountID := range accountGroups {
		sort.Slice(accountGroups[accountID], func(i, j int) bool {
			pi := accountGroups[accountID][i].Priority
			pj := accountGroups[accountID][j].Priority
			if pi == 0 && pj != 0 {
				return false
			}
			if pj == 0 && pi != 0 {
				return true
			}
			return pi > pj
		})
	}

	type result struct {
		goalID     int64
		goalName   string
		accountID  int64
		funded     bool
		skipReason string
		err        error
	}

	groupSlice := make([][]models.SavingGoal, 0, len(accountGroups))
	for _, group := range accountGroups {
		groupSlice = append(groupSlice, group)
	}

	jobs := make(chan []models.SavingGoal, len(groupSlice))
	results := make(chan result, len(goals))

	var wg sync.WaitGroup
	for i := 0; i < j.concurrentWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for group := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Process goals in this account sequentially so each successive
				// goal sees the updated uncategorized balance from prior inserts.
				for _, goal := range group {
					select {
					case <-ctx.Done():
						return
					default:
					}
					funded, skipReason, err := j.container.SavingsService.AutoFundGoal(ctx, goal, month)
					results <- result{
						goalID:     goal.ID,
						goalName:   goal.Name,
						accountID:  goal.AccountID,
						funded:     funded,
						skipReason: skipReason,
						err:        err,
					}
					if skipReason == "insufficient_balance" {
						break
					}
				}
			}
		}()
	}

	for _, group := range groupSlice {
		jobs <- group
	}
	close(jobs)

	wg.Wait()
	close(results)

	funded, skipped, failed := 0, 0, 0
	for r := range results {
		switch {
		case r.err != nil:
			j.logger.Error("Failed to auto-fund goal",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName),
				zap.Int64("accountID", r.accountID),
				zap.Error(r.err))
			failed++
		case r.funded:
			j.logger.Info("Auto-funded goal",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName))
			funded++
		case r.skipReason == "insufficient_balance":
			j.logger.Warn("Skipped goal - insufficient uncategorized balance",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName),
				zap.Int64("accountID", r.accountID))
			skipped++
		default:
			j.logger.Debug("Skipped goal - already funded this month",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName))
			skipped++
		}
	}

	j.logger.Info("Savings goals auto-fund completed",
		zap.Int("funded", funded),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed))

	return nil
}
