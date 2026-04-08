package jobscheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"

	"go.uber.org/zap"
)

type AutoFundGoalsJob struct {
	logger            *zap.Logger
	container         *bootstrap.ServiceContainer
	notifDispatcher   queue.NotificationDispatcher
	concurrentWorkers int
}

func NewAutoFundGoalsJob(logger *zap.Logger, container *bootstrap.ServiceContainer, notifDispatcher queue.NotificationDispatcher, concurrentWorkers int) *AutoFundGoalsJob {
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
		goalID     int64
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
			}
		}()
	}

	for _, group := range groupSlice {
		jobs <- group
	}
	close(jobs)

	wg.Wait()
	close(results)

	type userSummary struct {
		funded              []string
		insufficientBalance []string
		failed              []string
	}

	funded, skipped, failed := 0, 0, 0
	userResults := make(map[int64]*userSummary)

	for r := range results {
		s, ok := userResults[r.userID]
		if !ok {
			s = &userSummary{}
			userResults[r.userID] = s
		}
		switch {
		case r.err != nil:
			j.logger.Error("Failed to auto-fund goal",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName),
				zap.Int64("accountID", r.accountID),
				zap.Error(r.err))
			s.failed = append(s.failed, r.goalName)
			failed++
		case r.funded:
			j.logger.Info("Auto-funded goal",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName))
			s.funded = append(s.funded, r.goalName)
			funded++
		case r.skipReason == "insufficient_balance":
			j.logger.Warn("Skipped goal - insufficient uncategorized balance",
				zap.Int64("goalID", r.goalID),
				zap.String("goalName", r.goalName),
				zap.Int64("accountID", r.accountID))
			s.insufficientBalance = append(s.insufficientBalance, r.goalName)
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

	if j.notifDispatcher != nil {
		for userID, s := range userResults {
			if len(s.failed) > 0 {
				title := fmt.Sprintf("%d goal(s) failed to fund", len(s.failed))
				_ = j.notifDispatcher.Dispatch(userID, title, strings.Join(s.failed, ",\n"), models.NotificationTypeError)
			}
			if len(s.insufficientBalance) > 0 {
				title := fmt.Sprintf("%d goal(s) skipped - insufficient balance", len(s.insufficientBalance))
				_ = j.notifDispatcher.Dispatch(userID, title, strings.Join(s.insufficientBalance, ",\n"), models.NotificationTypeWarning)
			}
			if len(s.funded) > 0 {
				title := fmt.Sprintf("%d goal(s) funded", len(s.funded))
				_ = j.notifDispatcher.Dispatch(userID, title, strings.Join(s.funded, ",\n"), models.NotificationTypeSuccess)
			}
		}
	}

	return nil
}
