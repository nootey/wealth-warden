package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sort"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services/shared"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type SavingsService struct {
	Config            *config.Config
	Ctx               *DefaultServiceContext
	SavingsRepo       *repositories.SavingsRepository
	RecActionsService *ReoccurringActionService
	BudgetInterface   *shared.BudgetInterface
}

func NewSavingsService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.SavingsRepository,
	budgetInterface *shared.BudgetInterface,
	recActionsService *ReoccurringActionService,
) *SavingsService {
	return &SavingsService{
		Ctx:               ctx,
		Config:            cfg,
		SavingsRepo:       repo,
		RecActionsService: recActionsService,
		BudgetInterface:   budgetInterface,
	}
}

func (s *SavingsService) FetchSavingsPaginated(c *gin.Context) ([]models.SavingsTransaction, *utils.Paginator, error) {
	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, nil, err
	}

	queryParams := c.Request.URL.Query()
	paginationParams := utils.GetPaginationParams(queryParams)
	yearParam := queryParams.Get("year")
	currentYear := time.Now().Year()

	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 {
		year = currentYear
	}

	totalRecords, err := s.SavingsRepo.CountSavingsTransactions(user, year, paginationParams.Filters)
	if err != nil {
		return nil, nil, err
	}

	offset := (paginationParams.PageNumber - 1) * paginationParams.RowsPerPage
	records, err := s.SavingsRepo.FindSavingsTransactions(user, year, offset, paginationParams.RowsPerPage, paginationParams.SortField, paginationParams.SortOrder, paginationParams.Filters)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  paginationParams.PageNumber,
		RowsPerPage:  paginationParams.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *SavingsService) FetchAllSavingsGroupedByMonth(c *gin.Context, yearParam string) ([]models.SavingsSummary, error) {
	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	currentYear := time.Now().Year()
	year, err := strconv.Atoi(yearParam)
	if err != nil || year > currentYear || year < 2000 {
		year = currentYear
	}

	var summaries []models.SavingsSummary

	total, err := s.SavingsRepo.FindTotalForGroupedSavings(user, year)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, total...)

	categorized, err := s.SavingsRepo.FetchGroupedSavingsByCategoryAndMonth(user, year)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, categorized...)

	sort.SliceStable(summaries, func(i, j int) bool {
		a, b := summaries[i], summaries[j]

		typeRank := map[string]int{
			"fixed":    0,
			"variable": 1,
		}

		// Compare category types (fixed before variable)
		if typeRank[a.CategoryType] != typeRank[b.CategoryType] {
			return typeRank[a.CategoryType] < typeRank[b.CategoryType]
		}

		// Inside "fixed" category, ensure "Total" (category_id == 0) comes first
		if a.CategoryType == "fixed" && b.CategoryType == "fixed" {
			if a.CategoryID == 0 && b.CategoryID != 0 {
				return true
			}
			if b.CategoryID == 0 && a.CategoryID != 0 {
				return false
			}
		}

		if a.CategoryID != b.CategoryID {
			return a.CategoryID < b.CategoryID
		}

		return a.Month < b.Month
	})

	return summaries, nil
}

func (s *SavingsService) FetchAllSavingsCategories(c *gin.Context) ([]models.SavingsCategory, error) {
	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.SavingsRepo.FindAllSavingCategories(user)
}

func (s *SavingsService) CreateSavingsAllocation(c *gin.Context, newRecord *models.SavingsTransaction) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.AllocatedAmount, 'f', 2, 64)
	dateStr := newRecord.TransactionDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", newRecord.SavingsCategory.Name, changes, "category")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", dateStr, changes, "allocation_date")

	newRecord.AdjustedAmount = newRecord.AllocatedAmount

	err = s.SavingsRepo.InsertSavingsAllocation(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.SavingsRepo.UpdateCategoryGoalProgress(tx, user, newRecord.SavingsCategoryID, newRecord.AllocatedAmount, 1)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "create", "savings_allocation", nil, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}

func (s *SavingsService) CreateSavingsDeduction(c *gin.Context, newRecord *models.SavingsTransaction) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	amountString := strconv.FormatFloat(newRecord.AllocatedAmount, 'f', 2, 64)
	dateStr := newRecord.TransactionDate.UTC().Format(time.RFC3339)

	utils.CompareChanges("", newRecord.SavingsCategory.Name, changes, "category")
	utils.CompareChanges("", amountString, changes, "amount")
	utils.CompareChanges("", dateStr, changes, "deduction_date")
	utils.CompareChanges("", utils.SafeString(newRecord.Description), changes, "reason")

	allocationTotal, err := s.SavingsRepo.FindTotalForTransactionForCategory(user, "allocation", newRecord.SavingsCategoryID, newRecord.TransactionDate.Year())
	if err != nil {
		tx.Rollback()
		return err
	}

	deductionTotal, err := s.SavingsRepo.FindTotalForTransactionForCategory(user, "deduction", newRecord.SavingsCategoryID, newRecord.TransactionDate.Year())
	if err != nil {
		tx.Rollback()
		return err
	}

	availableTotal := allocationTotal - deductionTotal

	if availableTotal-newRecord.AllocatedAmount <= 0 {
		tx.Rollback()
		return errors.New(fmt.Sprintf("deduction amount is greater than the total of allocations, max value: %.2f", availableTotal))
	}

	newRecord.AdjustedAmount = newRecord.AllocatedAmount

	err = s.SavingsRepo.InsertSavingsDeduction(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.SavingsRepo.UpdateCategoryGoalProgress(tx, user, newRecord.SavingsCategoryID, newRecord.AdjustedAmount, -1)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "create", "savings_deduction", nil, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}

func (s *SavingsService) CreateSavingsCategory(c *gin.Context, newRecord *models.SavingsCategory, newReoccurringRecord *models.RecurringAction) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	utils.CompareChanges("", newRecord.Name, changes, "category")
	utils.CompareChanges("", newRecord.SavingsType, changes, "type")
	utils.CompareChanges("", newRecord.AccountType, changes, "account_type")

	if newRecord.AccountType == "interest" && newRecord.InterestRate != nil {
		interestString := strconv.FormatFloat(*newRecord.InterestRate, 'f', 2, 64)
		utils.CompareChanges("", interestString, changes, "interest")
	}

	if newRecord.GoalTarget != nil {
		goalTargetString := strconv.FormatFloat(*newRecord.GoalTarget, 'f', 2, 64)
		utils.CompareChanges("", goalTargetString, changes, "goal_target")
	}

	ID, err := s.SavingsRepo.InsertSavingsCategory(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	if newReoccurringRecord != nil {

		newReoccurringRecord.CategoryID = ID

		startDateStr := newReoccurringRecord.StartDate.UTC().Format(time.RFC3339)
		var endDateStr *string
		if newReoccurringRecord.EndDate != nil {
			formatted := newReoccurringRecord.EndDate.UTC().Format(time.RFC3339)
			endDateStr = &formatted
		} else {
			endDateStr = nil
		}

		if endDateStr != nil {
			utils.CompareChanges("", *endDateStr, changes, "end_date")
		}

		amountString := strconv.FormatFloat(newReoccurringRecord.Amount, 'f', 2, 64)
		utils.CompareChanges("", amountString, changes, "allocated_amount")
		utils.CompareChanges("", startDateStr, changes, "start_date")
		utils.CompareChanges("", newReoccurringRecord.CategoryType, changes, "category")
		utils.CompareChanges("", newReoccurringRecord.IntervalUnit, changes, "interval_unit")
		utils.CompareChanges("", strconv.Itoa(newReoccurringRecord.IntervalValue), changes, "interval_value")

		_, err = s.RecActionsService.Repo.InsertReoccurringAction(tx, user, newReoccurringRecord)
		if err != nil {
			tx.Rollback()
			return err
		}

		err = s.BudgetInterface.UpdateAllocation(tx, user, newReoccurringRecord, "savings", "create", 0)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "create", "savings_category", nil, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}

func (s *SavingsService) UpdateSavingsCategory(c *gin.Context, newRecord *models.SavingsCategory) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	existingRecord, err := s.SavingsRepo.GetSavingsCategoryByID(user, newRecord.ID)
	if err != nil {
		return err
	}

	utils.CompareChanges(existingRecord.Name, newRecord.Name, changes, "category")
	utils.CompareChanges(existingRecord.SavingsType, newRecord.SavingsType, changes, "type")
	utils.CompareChanges(existingRecord.AccountType, newRecord.AccountType, changes, "account_type")

	if newRecord.AccountType == "interest" && newRecord.InterestRate != nil {
		existingInterestString := strconv.FormatFloat(*existingRecord.InterestRate, 'f', 2, 64)
		interestString := strconv.FormatFloat(*newRecord.InterestRate, 'f', 2, 64)
		utils.CompareChanges(existingInterestString, interestString, changes, "interest")
	}

	if newRecord.GoalTarget != nil {
		existingGoalTargetString := strconv.FormatFloat(*existingRecord.GoalTarget, 'f', 2, 64)
		goalTargetString := strconv.FormatFloat(*newRecord.GoalTarget, 'f', 2, 64)
		utils.CompareChanges(existingGoalTargetString, goalTargetString, changes, "goal_target")
	}

	err = s.SavingsRepo.UpdateSavingsCategory(tx, user, newRecord)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	description := fmt.Sprintf("Savings category with ID: %d has been updated", newRecord.ID)
	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "update", "savings_category", &description, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}

func (s *SavingsService) DeleteSavingsCategory(c *gin.Context, id uint) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.SavingsRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	record, err := s.SavingsRepo.GetSavingsCategoryByID(user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	utils.CompareChanges(record.SavingsType, "", changes, "savings_type")
	utils.CompareChanges(record.AccountType, "", changes, "account_type")

	recRecord, err := s.RecActionsService.Repo.GetActionByRelatedCategory(tx, user, record.ID, "savings_categories")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			recRecord = nil
		} else {
			tx.Rollback()
			return err
		}
	}

	if recRecord != nil {

		err = s.RecActionsService.DeleteReoccurringAction(c, recRecord.ID, "savings_categories")
		if err != nil {
			tx.Rollback()
			return err
		}

		err = s.BudgetInterface.UpdateAllocation(tx, user, recRecord, "savings", "delete", 0)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = s.SavingsRepo.DropSavingsCategory(tx, user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "delete", "savings_category", nil, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}
