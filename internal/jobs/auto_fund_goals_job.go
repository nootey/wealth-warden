package jobs

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"

	"go.uber.org/zap"
)

type AutoFundGoalsJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	notifDispatcher   jobqueue.NotificationDispatcher
	concurrentWorkers int
}

func NewAutoFundGoalsJob(logger *zap.Logger, container *bootstrap.ServiceContainer, notifDispatcher jobqueue.NotificationDispatcher, concurrentWorkers int) *AutoFundGoalsJob {
	return &AutoFundGoalsJob{
		logger:            logger,
		container:         container,
		notifDispatcher:   notifDispatcher,
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
		goalName   string
		userID     int64
		accountID  int64
		funded     bool
		skipReason string
		err        error
	}

	groupSlice := make([][]models.SavingGoal, 0, len(accountGroups))
	for _, group := range accountGroups {
		groupSlice = append(groupSlice, group)
	}

	results := make(chan result, len(goals))

	// Failures are classified below, so the pool itself collects nothing.
	runPool(ctx, groupSlice, j.concurrentWorkers, "account groups", func(ctx context.Context, group []models.SavingGoal) error {
		// Process goals in this account sequentially so each successive goal
		// sees the updated uncategorized balance from prior inserts.
		for _, goal := range group {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			funded, skipReason, err := j.container.SavingsService.AutoFundGoal(ctx, goal, month)
			results <- result{
				goalName:   goal.Name,
				userID:     goal.UserID,
				accountID:  goal.AccountID,
				funded:     funded,
				skipReason: skipReason,
				err:        err,
			}
			if skipReason == "insufficient_balance" {
				break
			}
		}
		return nil
	})
	close(results)

	type userSummary struct {
		funded              []string
		insufficientBalance []string
		failed              []string
	}

	funded, insufficient, alreadyFunded, processed := 0, 0, 0, 0
	failures := newPoolResult(len(goals), "goals")
	userResults := make(map[int64]*userSummary)

	for r := range results {
		processed++
		s, ok := userResults[r.userID]
		if !ok {
			s = &userSummary{}
			userResults[r.userID] = s
		}
		failures.record(r.err)
		switch {
		case r.err != nil:
			s.failed = append(s.failed, r.goalName)
		case r.funded:
			s.funded = append(s.funded, r.goalName)
			funded++
		case r.skipReason == "insufficient_balance":
			s.insufficientBalance = append(s.insufficientBalance, r.goalName)
			insufficient++
		default:
			alreadyFunded++
		}
	}

	failures.logFailures(j.logger, "Failed to auto-fund goal")

	j.logger.Info("Savings goals auto-fund completed",
		zap.Int("funded", funded),
		zap.Int("insufficient_balance", insufficient),
		zap.Int("already_funded", alreadyFunded),
		zap.Int("failed", failures.failed),
		zap.Int("goals_seen", processed),
		zap.Int("goals_total", len(goals)))

	if j.notifDispatcher != nil {
		for userID, s := range userResults {
			if len(s.failed) > 0 {
				title := fmt.Sprintf("%d goal(s) failed to fund", len(s.failed))
				_ = j.notifDispatcher.Dispatch(ctx, userID, title, strings.Join(s.failed, ",\n"), models.NotificationTypeError)
			}
			if len(s.insufficientBalance) > 0 {
				title := fmt.Sprintf("%d goal(s) skipped - insufficient balance", len(s.insufficientBalance))
				_ = j.notifDispatcher.Dispatch(ctx, userID, title, strings.Join(s.insufficientBalance, ",\n"), models.NotificationTypeWarning)
			}
			if len(s.funded) > 0 {
				title := fmt.Sprintf("%d goal(s) funded", len(s.funded))
				_ = j.notifDispatcher.Dispatch(ctx, userID, title, strings.Join(s.funded, ",\n"), models.NotificationTypeSuccess)
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("auto-fund stopped early: %w", err)
	}
	return nil
}
