package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

type SavingsServiceInterface interface {
	FetchGoals(ctx context.Context, userID int64) ([]models.SavingGoalWithProgress, error)
	FetchGoalByID(ctx context.Context, userID, id int64) (*models.SavingGoalWithProgress, error)
	InsertGoal(ctx context.Context, userID int64, req *models.SavingGoalReq) (int64, error)
	UpdateGoal(ctx context.Context, userID, id int64, req *models.SavingGoalUpdateReq) (int64, error)
	DeleteGoal(ctx context.Context, userID, id int64) error

	FetchContributions(ctx context.Context, userID, goalID int64) ([]models.SavingContribution, error)
	FetchContributionsPaginated(ctx context.Context, userID, goalID int64, p utils.PaginationParams) ([]models.SavingContribution, *utils.Paginator, error)
	InsertContribution(ctx context.Context, userID, goalID int64, req *models.SavingContributionReq) (int64, error)
	DeleteContribution(ctx context.Context, userID, goalID, id int64) error

	AutoFundGoal(ctx context.Context, goal models.SavingGoal, month time.Time) (funded bool, skipReason string, err error)
	FetchActiveGoalsWithAllocation(ctx context.Context, dayOfMonth int) ([]models.SavingGoal, error)
}

type SavingsService struct {
	repo          repositories.SavingsRepositoryInterface
	accountRepo   repositories.AccountRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher queue.JobDispatcher
}

func NewSavingsService(
	repo *repositories.SavingsRepository,
	accountRepo *repositories.AccountRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher queue.JobDispatcher,
) *SavingsService {
	return &SavingsService{
		repo:          repo,
		accountRepo:   accountRepo,
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

	accType, err := s.accountRepo.FindAccountTypeByAccID(ctx, tx, req.AccountID, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("account not found: %w", err)
	}
	if accType.Type != "cash" {
		tx.Rollback()
		return 0, fmt.Errorf("goals can be linked only to cash accounts")
	}

	var targetDate *time.Time
	if req.TargetDate != nil && *req.TargetDate != "" {
		parsed, parseErr := time.Parse("2006-01-02", *req.TargetDate)
		if parseErr != nil {
			tx.Rollback()
			return 0, fmt.Errorf("invalid target_date: %w", parseErr)
		}
		targetDate = &parsed
	}

	record := models.SavingGoal{
		UserID:            userID,
		AccountID:         req.AccountID,
		Name:              req.Name,
		TargetAmount:      req.TargetAmount,
		TargetDate:        targetDate,
		Status:            models.SavingGoalStatusActive,
		Priority:          req.Priority,
		MonthlyAllocation: req.MonthlyAllocation,
		FundDayOfMonth:    req.FundDayOfMonth,
	}

	if req.InitialAmount != nil && req.InitialAmount.IsPositive() {
		record.CurrentAmount = *req.InitialAmount
	}

	id, err := s.repo.InsertGoal(ctx, tx, &record)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if req.InitialAmount != nil && req.InitialAmount.IsPositive() {
		now := time.Now().UTC()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		contrib := models.SavingContribution{
			UserID: userID,
			GoalID: id,
			Amount: *req.InitialAmount,
			Month:  monthStart,
			Source: models.SavingContributionSourceManual,
		}
		if _, err := s.repo.InsertContribution(ctx, tx, &contrib); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(id, 10), changes, "id")
	utils.CompareChanges("", record.Name, changes, "name")
	utils.CompareDecimalChange(nil, &record.TargetAmount, changes, "target_amount", 2)
	utils.CompareDecimalChange(nil, record.MonthlyAllocation, changes, "monthly_allocation", 2)
	utils.CompareDateChange(nil, record.TargetDate, changes, "target_date")
	if err := s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "saving_goal",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

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

	var targetDate *time.Time
	if req.TargetDate != nil && *req.TargetDate != "" {
		parsed, parseErr := time.Parse("2006-01-02", *req.TargetDate)
		if parseErr != nil {
			tx.Rollback()
			return 0, fmt.Errorf("invalid target_date: %w", parseErr)
		}
		targetDate = &parsed
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(existing.ID, 10), changes, "id")
	utils.CompareChanges(existing.Name, req.Name, changes, "name")
	utils.CompareDecimalChange(&existing.TargetAmount, &req.TargetAmount, changes, "target_amount", 2)
	utils.CompareDecimalChange(existing.MonthlyAllocation, req.MonthlyAllocation, changes, "monthly_allocation", 2)
	utils.CompareDateChange(existing.TargetDate, targetDate, changes, "target_date")
	utils.CompareChanges(string(existing.Status), string(req.Status), changes, "status")

	existing.Name = req.Name
	existing.TargetAmount = req.TargetAmount
	existing.TargetDate = targetDate
	existing.Status = req.Status
	existing.Priority = req.Priority
	existing.MonthlyAllocation = req.MonthlyAllocation
	existing.FundDayOfMonth = req.FundDayOfMonth

	goalID, err := s.repo.UpdateGoal(ctx, tx, existing)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "saving_goal",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return 0, err
		}
	}

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

	goal, err := s.repo.FindGoalByID(ctx, tx, id, userID)
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

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(goal.ID, 10), changes, "id")
	utils.CompareChanges(goal.Name, "", changes, "name")
	utils.CompareDecimalChange(&goal.TargetAmount, nil, changes, "target_amount", 2)
	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "delete",
			Category:    "saving_goal",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *SavingsService) FetchContributions(ctx context.Context, userID, goalID int64) ([]models.SavingContribution, error) {
	_, err := s.repo.FindGoalByID(ctx, nil, goalID, userID)
	if err != nil {
		return nil, fmt.Errorf("goal not found: %w", err)
	}

	return s.repo.FindContributions(ctx, nil, goalID)
}

func (s *SavingsService) FetchContributionsPaginated(ctx context.Context, userID, goalID int64, p utils.PaginationParams) ([]models.SavingContribution, *utils.Paginator, error) {
	_, err := s.repo.FindGoalByID(ctx, nil, goalID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("goal not found: %w", err)
	}

	total, err := s.repo.CountContributions(ctx, nil, goalID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage
	records, err := s.repo.FindContributionsPaginated(ctx, nil, goalID, offset, p.RowsPerPage)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if int(total) == 0 {
		from = 0
	} else if from > int(total) {
		from = int(total)
	}
	to := offset + len(records)
	if to > int(total) {
		to = int(total)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(total),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
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

	uncategorized, err := s.repo.GetUncategorizedBalance(ctx, tx, goal.AccountID, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to compute available balance: %w", err)
	}
	if req.Amount.GreaterThan(uncategorized) {
		tx.Rollback()
		return 0, fmt.Errorf("contribution of %s exceeds uncategorized balance of %s", req.Amount.StringFixed(2), uncategorized.StringFixed(2))
	}

	month, err := time.Parse("2006-01-02", req.Month)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("invalid month: %w", err)
	}

	record := models.SavingContribution{
		UserID: userID,
		GoalID: goalID,
		Amount: req.Amount,
		Month:  month,
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

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(id, 10), changes, "id")
	utils.CompareChanges("", goal.Name, changes, "goal")
	utils.CompareDecimalChange(nil, &req.Amount, changes, "amount", 2)
	utils.CompareChanges("", record.Month.Format("2006-01-02"), changes, "month")
	if err := s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "saving_contribution",
		Description: req.Note,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return 0, err
	}

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

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(contrib.ID, 10), changes, "id")
	utils.CompareChanges(goal.Name, "", changes, "goal")
	utils.CompareDecimalChange(&contrib.Amount, nil, changes, "amount", 2)
	utils.CompareChanges(contrib.Month.Format("2006-01-02"), "", changes, "month")
	if !changes.IsEmpty() {
		if err := s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "delete",
			Category:    "saving_contribution",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *SavingsService) FetchActiveGoalsWithAllocation(ctx context.Context, dayOfMonth int) ([]models.SavingGoal, error) {
	return s.repo.FindActiveGoalsWithAllocation(ctx, nil, dayOfMonth)
}

func (s *SavingsService) AutoFundGoal(ctx context.Context, goal models.SavingGoal, month time.Time) (bool, string, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return false, "", err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	alreadyFunded, err := s.repo.HasContributionForMonth(ctx, tx, goal.ID, monthStart)
	if err != nil {
		tx.Rollback()
		return false, "", err
	}
	if alreadyFunded {
		tx.Rollback()
		return false, "already_funded", nil
	}

	uncategorized, err := s.repo.GetUncategorizedBalance(ctx, tx, goal.AccountID, goal.UserID)
	if err != nil {
		tx.Rollback()
		return false, "", fmt.Errorf("failed to compute available balance: %w", err)
	}
	if uncategorized.LessThan(*goal.MonthlyAllocation) {
		tx.Rollback()
		return false, "insufficient_balance", nil
	}

	record := models.SavingContribution{
		UserID: goal.UserID,
		GoalID: goal.ID,
		Amount: *goal.MonthlyAllocation,
		Month:  monthStart,
		Source: models.SavingContributionSourceAuto,
	}
	cID, err := s.repo.InsertContribution(ctx, tx, &record)
	if err != nil {
		tx.Rollback()
		return false, "", err
	}

	goal.CurrentAmount = goal.CurrentAmount.Add(*goal.MonthlyAllocation)
	if err := s.repo.UpdateCurrentAmount(ctx, tx, goal.ID, goal); err != nil {
		tx.Rollback()
		return false, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return false, "", err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(cID, 10), changes, "id")
	utils.CompareChanges("", goal.Name, changes, "goal")
	utils.CompareDecimalChange(nil, goal.MonthlyAllocation, changes, "amount", 2)
	utils.CompareChanges("", monthStart.Format("2006-01-02"), changes, "month")
	_ = s.jobDispatcher.Dispatch(&queue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "saving_contribution",
		Payload:     changes,
	})

	return true, "", nil
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
