package jobs

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type goalFundResult struct {
	goalName   string
	userID     int64
	funded     bool
	skipReason string
	err        error
}

type userGoalSummary struct {
	funded              []string
	insufficientBalance []string
	failed              []string
}

type AutoFundGoalsJob struct {
	logger            *zap.Logger
	savingsSvc        services.SavingsServiceInterface
	notifDispatcher   jobqueue.NotificationDispatcher
	concurrentWorkers int
}

func NewAutoFundGoalsJob(logger *zap.Logger, savingsSvc services.SavingsServiceInterface, notifDispatcher jobqueue.NotificationDispatcher, concurrentWorkers int) *AutoFundGoalsJob {
	if concurrentWorkers < 1 {
		concurrentWorkers = defaultWorkers
	}
	return &AutoFundGoalsJob{
		logger:            logger,
		savingsSvc:        savingsSvc,
		notifDispatcher:   notifDispatcher,
		concurrentWorkers: concurrentWorkers,
	}
}

func (j *AutoFundGoalsJob) Run(ctx context.Context) error {
	now := time.Now().UTC()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	goals, err := j.savingsSvc.FetchActiveGoalsWithAllocation(ctx, now.Day())
	if err != nil {
		return err
	}

	if len(goals) == 0 {
		j.logger.Info("No active goals with allocation to process")
		return nil
	}

	j.logger.Info("Processing savings goals auto-fund", zap.Int("count", len(goals)), zap.String("month", month.Format("2006-01")))

	groups := groupGoalsByAccount(goals)
	results := make(chan goalFundResult, len(goals))

	var cancelled atomic.Int64

	g := new(errgroup.Group)
	g.SetLimit(j.concurrentWorkers)

	for _, group := range groups {
		g.Go(func() error {
			cancelled.Add(int64(j.fundAccountGroup(ctx, group, month, results)))
			return nil
		})
	}
	_ = g.Wait()
	close(results)

	funded, insufficient, alreadyFunded, failed, processed := 0, 0, 0, 0, 0
	userResults := make(map[int64]*userGoalSummary)
	var firstErr error

	for r := range results {
		processed++
		s, ok := userResults[r.userID]
		if !ok {
			s = &userGoalSummary{}
			userResults[r.userID] = s
		}
		switch {
		case r.err != nil:
			s.failed = append(s.failed, r.goalName)
			failed++
			if firstErr == nil {
				firstErr = fmt.Errorf("goal %q: %w", r.goalName, r.err)
			}
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

	j.logger.Info("Savings goals auto-fund completed",
		zap.Int("funded", funded),
		zap.Int("insufficient_balance", insufficient),
		zap.Int("already_funded", alreadyFunded),
		zap.Int("failed", failed),
		zap.Int("cancelled", int(cancelled.Load())),
		zap.Int("goals_seen", processed),
		zap.Int("goals_total", len(goals)))

	if firstErr != nil {
		j.logger.Error("Savings goals auto-fund had failures",
			zap.Int("failed", failed),
			zap.Error(firstErr))
	}

	j.notifyUsers(ctx, userResults)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("auto-fund stopped early: %w", err)
	}
	return nil
}

func groupGoalsByAccount(goals []models.SavingGoal) [][]models.SavingGoal {
	byAccount := make(map[int64][]models.SavingGoal)
	for _, g := range goals {
		byAccount[g.AccountID] = append(byAccount[g.AccountID], g)
	}

	groups := make([][]models.SavingGoal, 0, len(byAccount))
	for _, group := range byAccount {
		sort.Slice(group, func(i, j int) bool {
			pi, pj := group[i].Priority, group[j].Priority
			if pi == 0 && pj != 0 {
				return false
			}
			if pj == 0 && pi != 0 {
				return true
			}
			return pi > pj
		})
		groups = append(groups, group)
	}
	return groups
}

// Returns how many goals the group dropped because the run was cancelled. They
// produce no result, so without this the summary counts would not add up.
func (j *AutoFundGoalsJob) fundAccountGroup(ctx context.Context, group []models.SavingGoal, month time.Time, results chan<- goalFundResult) int {
	for i, goal := range group {
		if ctx.Err() != nil {
			return len(group) - i
		}

		funded, skipReason, err := j.savingsSvc.AutoFundGoal(ctx, goal, month)
		results <- goalFundResult{
			goalName:   goal.Name,
			userID:     goal.UserID,
			funded:     funded,
			skipReason: skipReason,
			err:        err,
		}

		if skipReason != "insufficient_balance" {
			continue
		}

		// The account is spent. Every lower-priority goal in the group would hit
		// the same wall, so report them rather than dropping them silently.
		for _, rest := range group[i+1:] {
			results <- goalFundResult{
				goalName:   rest.Name,
				userID:     rest.UserID,
				skipReason: "insufficient_balance",
			}
		}
		return 0
	}
	return 0
}

func (j *AutoFundGoalsJob) notifyUsers(ctx context.Context, userResults map[int64]*userGoalSummary) {
	if j.notifDispatcher == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)

	for userID, s := range userResults {
		if len(s.failed) > 0 {
			title := fmt.Sprintf("%d goal(s) failed to fund", len(s.failed))
			j.dispatch(ctx, userID, title, strings.Join(s.failed, ", "), models.NotificationTypeError)
		}
		if len(s.insufficientBalance) > 0 {
			title := fmt.Sprintf("%d goal(s) skipped - insufficient balance", len(s.insufficientBalance))
			j.dispatch(ctx, userID, title, strings.Join(s.insufficientBalance, ", "), models.NotificationTypeWarning)
		}
		if len(s.funded) > 0 {
			title := fmt.Sprintf("%d goal(s) funded", len(s.funded))
			j.dispatch(ctx, userID, title, strings.Join(s.funded, ", "), models.NotificationTypeSuccess)
		}
	}
}

func (j *AutoFundGoalsJob) dispatch(ctx context.Context, userID int64, title, message string, notifType models.NotificationType) {
	if err := j.notifDispatcher.Dispatch(ctx, userID, title, message, notifType); err != nil {
		j.logger.Error("Failed to dispatch notification",
			zap.Int64("user_id", userID),
			zap.String("title", title),
			zap.Error(err))
	}
}
