package services

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type ReoccurringActionService struct {
	ActionRepo     *repositories.ReoccurringActionsRepository
	AuthService    *AuthService
	LoggingService *LoggingService
}

func NewReoccurringActionService(repo *repositories.ReoccurringActionsRepository, authService *AuthService, loggingService *LoggingService) *ReoccurringActionService {
	return &ReoccurringActionService{
		ActionRepo:     repo,
		AuthService:    authService,
		LoggingService: loggingService,
	}
}

func (s *ReoccurringActionService) FetchAllActionsForCategory(c *gin.Context, categoryName string) ([]models.RecurringAction, error) {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}
	return s.ActionRepo.FindAllActionsForCategory(user, categoryName)
}

func (s *ReoccurringActionService) DeleteReoccurringAction(c *gin.Context, id uint, categoryName string) error {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return err
	}
	changes := utils.InitChanges()

	tx := s.ActionRepo.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	record, err := s.ActionRepo.GetActionByID(user, id)
	if err != nil {
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

	err = s.ActionRepo.DropAction(tx, user, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.LoggingService.LoggingRepo.InsertActivityLog(tx, "delete", "reoccurring_action", nil, changes, user)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *ReoccurringActionService) FetchAvailableYearsForRecords(c *gin.Context, table, dateField string) ([]int, error) {

	user, err := s.AuthService.GetCurrentUser(c, false)
	if err != nil {
		return nil, err
	}

	years, err := s.ActionRepo.FindDistinctYearsForRecords(user, table, dateField)
	if err != nil {
		return nil, err
	}
	
	if len(years) == 0 {
		year := time.Now().Year()
		years = append(years, year)
	}

	return years, nil

}
