package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

type ReoccurringActionService struct {
	Config *config.Config
	Ctx    *DefaultServiceContext
	Repo   *repositories.ReoccurringActionsRepository
}

func NewReoccurringActionService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ReoccurringActionsRepository,
) *ReoccurringActionService {
	return &ReoccurringActionService{
		Ctx:    ctx,
		Config: cfg,
		Repo:   repo,
	}
}

func (s *ReoccurringActionService) FetchAllActionsForCategory(c *gin.Context, categoryName string) ([]models.RecurringAction, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.Repo.FindAllActionsForCategory(user, categoryName)
}

func (s *ReoccurringActionService) DeleteReoccurringAction(c *gin.Context, id uint, categoryName string) error {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.Repo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	preventDelete := map[string]bool{
		"savings_categories": true,
	}

	record, err := s.Repo.GetActionByID(user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if preventDelete[categoryName] {
		err = fmt.Errorf("reoccurring action cannot be deleted for category: %s", categoryName)
		tx.Rollback()
		return err
	}

	if record.CategoryType != categoryName {
		err = errors.New(fmt.Sprintf("category names do not match: %s != %s", categoryName, record.CategoryType))
	}

	amountString := strconv.FormatFloat(record.Amount, 'f', 2, 64)
	intervalValue := strconv.FormatInt(int64(record.IntervalValue), 10)
	startDateStr := record.StartDate.UTC().Format(time.RFC3339)

	utils.CompareChanges(record.CategoryType, "", changes, "category")
	utils.CompareChanges(amountString, "", changes, "amount")
	utils.CompareChanges(record.IntervalUnit, "", changes, "interval_unit")
	utils.CompareChanges(intervalValue, "", changes, "interval_value")
	utils.CompareChanges(startDateStr, "", changes, "start_date")

	var endDateStr *string
	if record.EndDate != nil {
		formatted := record.EndDate.UTC().Format(time.RFC3339)
		endDateStr = &formatted
	} else {
		endDateStr = nil // Ensure it remains nil instead of an empty string
	}

	if endDateStr != nil {
		utils.CompareChanges("", *endDateStr, changes, "end_date")
	}

	err = s.Repo.DropAction(tx, user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	go func(changes *utils.Changes, user *models.User) {
		err := s.Ctx.LoggingService.LoggingRepo.InsertActivityLog(nil, "delete", "reoccurring_action", nil, changes, user)
		if err != nil {
			s.Ctx.Logger.Error("failed to insert activity log: %v", zap.Error(err))
		}
	}(changes, user)

	return nil
}

func (s *ReoccurringActionService) FetchAvailableYearsForRecords(c *gin.Context, table, dateField string) ([]int, error) {

	user, err := s.Ctx.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	years, err := s.Repo.FindDistinctYearsForRecords(user, table, dateField)
	if err != nil {
		return nil, err
	}

	if len(years) == 0 {
		year := time.Now().Year()
		years = append(years, year)
	}

	return years, nil

}
