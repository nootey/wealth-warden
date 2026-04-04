package services

import (
	"context"
	"fmt"
	"math"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"

	"github.com/shopspring/decimal"
)

type SavingsServiceInterface interface {
	FetchGoals(ctx context.Context, userID int64) ([]models.SavingGoalWithProgress, error)
	FetchGoalByID(ctx context.Context, userID, id int64) (*models.SavingGoalWithProgress, error)
	InsertGoal(ctx context.Context, userID int64, req *models.SavingGoalReq) (int64, error)
	UpdateGoal(ctx context.Context, userID, id int64, req *models.SavingGoalUpdateReq) (int64, error)
	DeleteGoal(ctx context.Context, userID, id int64) error

	FetchContributions(ctx context.Context, userID, goalID int64) ([]models.SavingContribution, error)
	InsertContribution(ctx context.Context, userID, goalID int64, req *models.SavingContributionReq) (int64, error)
	DeleteContribution(ctx context.Context, userID, goalID, id int64) error
}

type SavingsService struct {
	repo          repositories.SavingsRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher queue.JobDispatcher
}

func NewSavingsService(
	repo *repositories.SavingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher queue.JobDispatcher,
) *SavingsService {
	return &SavingsService{
		repo:          repo,
		loggingRepo:   loggingRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ SavingsServiceInterface = (*SavingsService)(nil)

func (s *SavingsService) FetchGoals(ctx context.Context, userID int64) ([]models.SavingGoalWithProgress, error) {
	goals, err := s.repo.FindGoals(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.SavingGoalWithProgress, len(goals))
	for i, g := range goals {
		result[i] = computeProgress(g)
	}

	return result, nil
}

func (s *SavingsService) FetchGoalByID(ctx context.Context, userID, id int64) (*models.SavingGoalWithProgress, error) {
	goal, err := s.repo.FindGoalByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	wp := computeProgress(goal)
	return &wp, nil
}

func (s *SavingsService) InsertGoal(ctx context.Context, userID int64, req *models.SavingGoalReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	record := models.SavingGoal{
		UserID:            userID,
		AccountID:         req.AccountID,
		Name:              req.Name,
		TargetAmount:      req.TargetAmount,
		TargetDate:        req.TargetDate,
		Status:            models.SavingGoalStatusActive,
		Priority:          req.Priority,
		MonthlyAllocation: req.MonthlyAllocation,
	}

	id, err := s.repo.InsertGoal(ctx, tx, &record)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "saving_goal",
		Description: nil,
		Payload:     nil,
		Causer:      &userID,
	})

	return id, nil
}

func (s *SavingsService) UpdateGoal(ctx context.Context, userID, id int64, req *models.SavingGoalUpdateReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	existing, err := s.repo.FindGoalByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("goal not found: %w", err)
	}

	existing.Name = req.Name
	existing.TargetAmount = req.TargetAmount
	existing.TargetDate = req.TargetDate
	existing.Status = req.Status
	existing.Priority = req.Priority
	existing.MonthlyAllocation = req.MonthlyAllocation

	goalID, err := s.repo.UpdateGoal(ctx, tx, existing)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "update",
		Category:    "saving_goal",
		Description: nil,
		Payload:     nil,
		Causer:      &userID,
	})

	return goalID, nil
}

func (s *SavingsService) DeleteGoal(ctx context.Context, userID, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	_, err = s.repo.FindGoalByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("goal not found: %w", err)
	}

	if err := s.repo.DeleteGoal(ctx, tx, id); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "saving_goal",
		Description: nil,
		Payload:     nil,
		Causer:      &userID,
	})

	return nil
}

func (s *SavingsService) FetchContributions(ctx context.Context, userID, goalID int64) ([]models.SavingContribution, error) {
	_, err := s.repo.FindGoalByID(ctx, nil, goalID, userID)
	if err != nil {
		return nil, fmt.Errorf("goal not found: %w", err)
	}

	return s.repo.FindContributions(ctx, nil, goalID)
}

func (s *SavingsService) InsertContribution(ctx context.Context, userID, goalID int64, req *models.SavingContributionReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	goal, err := s.repo.FindGoalByID(ctx, tx, goalID, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("goal not found: %w", err)
	}

	record := models.SavingContribution{
		UserID: userID,
		GoalID: goalID,
		Amount: req.Amount,
		Month:  req.Month,
		Note:   req.Note,
		Source: models.SavingContributionSourceManual,
	}

	id, err := s.repo.InsertContribution(ctx, tx, &record)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	goal.CurrentAmount = goal.CurrentAmount.Add(req.Amount)
	if err := s.repo.UpdateCurrentAmount(ctx, tx, goalID, goal); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "saving_contribution",
		Description: nil,
		Payload:     nil,
		Causer:      &userID,
	})

	return id, nil
}

func (s *SavingsService) DeleteContribution(ctx context.Context, userID, goalID, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	goal, err := s.repo.FindGoalByID(ctx, tx, goalID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("goal not found: %w", err)
	}

	contrib, err := s.repo.FindContributionByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("contribution not found: %w", err)
	}

	if err := s.repo.DeleteContribution(ctx, tx, id); err != nil {
		tx.Rollback()
		return err
	}

	goal.CurrentAmount = goal.CurrentAmount.Sub(contrib.Amount)
	if goal.CurrentAmount.IsNegative() {
		goal.CurrentAmount = decimal.Zero
	}
	if err := s.repo.UpdateCurrentAmount(ctx, tx, goalID, goal); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "saving_contribution",
		Description: nil,
		Payload:     nil,
		Causer:      &userID,
	})

	return nil
}

func computeProgress(g models.SavingGoal) models.SavingGoalWithProgress {
	wp := models.SavingGoalWithProgress{SavingGoal: g}

	if g.TargetAmount.IsZero() {
		wp.TrackStatus = "no_target"
		return wp
	}

	wp.ProgressPercent = g.CurrentAmount.Div(g.TargetAmount).Mul(decimal.NewFromInt(100))

	if g.Status == models.SavingGoalStatusCompleted || g.CurrentAmount.GreaterThanOrEqual(g.TargetAmount) {
		wp.TrackStatus = "completed"
		return wp
	}

	if g.TargetDate == nil {
		wp.TrackStatus = "no_target"
		return wp
	}

	now := time.Now().UTC()
	totalDays := g.TargetDate.Sub(now).Hours() / 24
	if totalDays <= 0 {
		wp.TrackStatus = "late"
		return wp
	}

	// months remaining (ceil)
	monthsRemaining := int(math.Ceil(totalDays / 30.0))
	wp.MonthsRemaining = &monthsRemaining

	remaining := g.TargetAmount.Sub(g.CurrentAmount)
	if monthsRemaining > 0 {
		mn := remaining.Div(decimal.NewFromInt(int64(monthsRemaining)))
		wp.MonthlyNeeded = &mn
	}

	// expected progress based on time elapsed from creation
	createdDays := now.Sub(g.CreatedAt).Hours() / 24
	totalSpan := g.TargetDate.Sub(g.CreatedAt).Hours() / 24
	if totalSpan <= 0 {
		wp.TrackStatus = "no_target"
		return wp
	}

	expectedPercent := decimal.NewFromFloat(createdDays / totalSpan * 100)
	threshold := expectedPercent.Sub(decimal.NewFromInt(5))

	switch {
	case wp.ProgressPercent.GreaterThan(expectedPercent):
		wp.TrackStatus = "early"
	case wp.ProgressPercent.GreaterThanOrEqual(threshold):
		wp.TrackStatus = "on_track"
	default:
		wp.TrackStatus = "late"
	}

	return wp
}
