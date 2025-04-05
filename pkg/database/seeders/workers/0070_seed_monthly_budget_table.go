package workers

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services/shared"
)

func fetchSumsForDynamicCategory(budgetInterface shared.BudgetInterface, categoryName string, mapping *models.DynamicCategoryMapping, year, month int, user *models.User) (float64, error) {
	var sum float64
	var err error

	switch categoryName {
	case "inflow":
		{
			var inflowSums float64
			var dynamicSums float64

			if mapping.RelatedCategoryName == "inflow" {
				inflowSums, err = budgetInterface.FetchTotalValuesForInflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			} else if mapping.RelatedCategoryName == "dynamic" {
				dynamicSums, err = budgetInterface.FetchTotalValuesForDynamicCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			}

			sum = inflowSums + dynamicSums
		}
	case "outflow":
		{
			if mapping.RelatedCategoryName == "dynamic" {
				sum = 0
			} else {
				sum, err = budgetInterface.FetchTotalValuesForOutflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			}
		}
	default:
		return 0, nil
	}

	return sum, nil
}

func SeedMonthlyBudget(ctx context.Context, db *gorm.DB) error {

	user, err := GetUser(db)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	var trueSalaryCategory models.DynamicCategory
	if err := db.Preload("Mappings").
		Where("organization_id = ? AND name = ?", user.PrimaryOrganizationID, "Effective salary").
		First(&trueSalaryCategory).Error; err != nil {
		return fmt.Errorf("failed to retrieve dynamic category 'True Salary': %w", err)
	}
	if trueSalaryCategory.ID == 0 {
		return fmt.Errorf("dynamic category 'True Salary' not found")
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	budgetRepo := repositories.NewBudgetRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)

	budgetInterface := shared.BudgetInterface{BudgetRepo: budgetRepo, InflowRepo: inflowRepo, OutflowRepo: outflowRepo}

	sums := make(map[string]float64)

	for _, mapping := range trueSalaryCategory.Mappings {
		inflowSums, err := fetchSumsForDynamicCategory(budgetInterface, "inflow", &mapping, year, month, user)
		if err != nil {
			return err
		}

		// Only sum outflows if it's the first time processing this category
		var totalOutflow float64
		if mapping.RelatedCategoryName == "outflow" {
			outflowSums, err := fetchSumsForDynamicCategory(budgetInterface, "outflow", &mapping, year, month, user)
			if err != nil {
				return err
			}
			totalOutflow += outflowSums
		}

		sums["inflow"] += inflowSums
		sums["outflow"] += totalOutflow
	}

	sums["total_inflows"], err = inflowRepo.SumInflowsByMonth(user, year, month)
	if err != nil {
		return err
	}
	sums["total_outflows"], err = outflowRepo.SumOutflowsByMonth(user, year, month)
	if err != nil {
		return err
	}

	monthlyBudget := models.MonthlyBudget{
		OrganizationID:    *user.PrimaryOrganizationID,
		UserID:            user.ID,
		DynamicCategoryID: trueSalaryCategory.ID,
		Month:             month,
		Year:              year,
		TotalInflow:       sums["total_inflows"],
		TotalOutflow:      sums["total_outflows"],
		BudgetInflow:      sums["inflow"],
		BudgetOutflow:     sums["outflow"],
		EffectiveBudget:   sums["inflow"] - sums["outflow"],
		BudgetSnapshot:    sums["inflow"] - sums["outflow"],
		SnapshotThreshold: 500,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := db.Create(&monthlyBudget).Error; err != nil {
		return fmt.Errorf("failed to create monthly budget: %w", err)
	}

	userRepo := repositories.NewUserRepository(db)

	err = userRepo.UpdateUserSecret(db, user, "budget_initialized", true)
	if err != nil {
		return err
	}

	return nil
}

func UnseedMonthlyBudget(ctx context.Context, db *gorm.DB) error {
	var orgID uint
	if err := db.Raw(`SELECT id FROM organizations WHERE name = ?`, "Super Admin").Scan(&orgID).Error; err != nil {
		return fmt.Errorf("failed to retrieve organization id for 'Super Admin': %w", err)
	}
	if orgID == 0 {
		return fmt.Errorf("organization 'Super Admin' not found")
	}

	now := time.Now()
	year, month, _ := now.Date()
	intMonth := int(month)

	var trueSalaryCategory models.DynamicCategory
	if err := db.Where("organization_id = ? AND name = ?", orgID, "True Salary").First(&trueSalaryCategory).Error; err != nil {
		return fmt.Errorf("failed to find dynamic category 'True Salary': %w", err)
	}

	if err := db.Where("organization_id = ? AND dynamic_category_id = ? AND year = ? AND month = ?",
		orgID, trueSalaryCategory.ID, year, intMonth).
		Delete(&models.MonthlyBudget{}).Error; err != nil {
		return fmt.Errorf("failed to delete monthly budget: %w", err)
	}

	return nil
}
