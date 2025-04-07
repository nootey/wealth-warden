package shared

import (
	"errors"
	"gorm.io/gorm"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type BudgetInterface struct {
	BudgetRepo  *repositories.BudgetRepository
	InflowRepo  *repositories.InflowRepository
	OutflowRepo *repositories.OutflowRepository
}

func NewBudgetInterface(budgetRepo *repositories.BudgetRepository, inflowRepo *repositories.InflowRepository, outflowRepo *repositories.OutflowRepository) *BudgetInterface {
	return &BudgetInterface{
		BudgetRepo:  budgetRepo,
		InflowRepo:  inflowRepo,
		OutflowRepo: outflowRepo,
	}
}

func (b *BudgetInterface) FindBudgetByDynamicCategoryID(user *models.User, categoryID uint, year, month int) (*models.MonthlyBudget, error) {
	var record models.MonthlyBudget

	query := b.BudgetRepo.Db.
		Preload("Allocations", "method = ?", "percentage").
		Where("dynamic_category_id = ? AND organization_id = ? AND year = ? AND month = ?",
			categoryID, *user.PrimaryOrganizationID, year, month)

	result := query.Find(&record)

	if result.Error != nil {
		return nil, result.Error
	}

	return &record, nil
}

func (b *BudgetInterface) findBudgetForUser(user *models.User, year, month int) (*models.MonthlyBudget, error) {
	var record models.MonthlyBudget

	query := b.BudgetRepo.Db.
		Where("organization_id = ? AND year = ? AND month = ?",
			*user.PrimaryOrganizationID, year, month)

	result := query.Find(&record)

	if result.Error != nil {
		return nil, result.Error
	}

	return &record, nil
}

func (b *BudgetInterface) findDynamicCategoryByRelatedCategoryID(user *models.User, categoryType string, categoryID uint) (*models.DynamicCategory, error) {
	visited := make(map[uint]bool)

	var dynamicCategory models.DynamicCategory
	err := b.InflowRepo.Db.
		Joins("JOIN dynamic_category_mappings ON dynamic_category_mappings.dynamic_category_id = dynamic_categories.id").
		Where("organization_id = ? AND dynamic_category_mappings.related_id = ? AND dynamic_category_mappings.related_type = ?", *user.PrimaryOrganizationID, categoryID, categoryType).
		First(&dynamicCategory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Traverse higher-level mappings until none is found.
	for {
		if visited[dynamicCategory.ID] {
			return nil, errors.New("circular dynamic category mapping detected")
		}
		visited[dynamicCategory.ID] = true

		var higherMapping models.DynamicCategoryMapping
		err = b.InflowRepo.Db.
			Where("related_id = ? AND related_type = ?", dynamicCategory.ID, "dynamic").
			First(&higherMapping).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// No higher mapping found: return the current (highest) dynamic category.
				return &dynamicCategory, nil
			}
			return nil, err
		}

		// Load the higher-level dynamic category using the mapping found.
		nextCategory, err := b.InflowRepo.FindDynamicCategoryByID(user, higherMapping.DynamicCategoryID)
		if err != nil {
			return nil, err
		}
		if nextCategory == nil {
			// If the next category is nil, return the current one.
			return &dynamicCategory, nil
		}

		// Set dynamicCategory to the higher-level category and continue loop.
		dynamicCategory = *nextCategory
	}
}

func (b *BudgetInterface) updateBudget(tx *gorm.DB, user *models.User, dynamicCategoryID uint, category string, amount float64, date time.Time) error {

	currentYear, currentMonth := time.Now().Year(), int(time.Now().Month())
	recordYear, recordMonth := date.Year(), int(date.Month())
	if recordYear != currentYear || recordMonth != currentMonth {
		return nil
	}

	var budget *models.MonthlyBudget
	var err error

	if dynamicCategoryID == 0 {
		budget, err = b.findBudgetForUser(user, currentYear, currentMonth)
		if err != nil {
			return err
		}

		if budget == nil {
			return nil
		}

		switch category {
		case "inflow":
			budget.TotalInflow += amount
		case "outflow":
			budget.TotalOutflow += amount
		default:
			return errors.New("invalid category type, must be 'inflow' or 'outflow'")
		}

	} else {
		budget, err = b.FindBudgetByDynamicCategoryID(user, dynamicCategoryID, currentYear, currentMonth)
		if err != nil {
			return err
		}

		if budget == nil {
			return nil
		}

		switch category {
		case "inflow":
			budget.BudgetInflow += amount
			budget.TotalInflow += amount
		case "outflow":
			budget.BudgetOutflow += amount
			budget.TotalOutflow += amount
		default:
			return errors.New("invalid category type, must be 'inflow' or 'outflow'")
		}

		budget.EffectiveBudget = budget.BudgetInflow - budget.BudgetOutflow
		if budget.EffectiveBudget < 0 {
			return errors.New("effective budget can not be negative. Can not insert/delete this record")
		}

		if (budget.BudgetSnapshot == 0 && budget.EffectiveBudget > 0) ||
			budget.EffectiveBudget < budget.SnapshotThreshold ||
			budget.EffectiveBudget < budget.BudgetSnapshot {
			budget.BudgetSnapshot = budget.EffectiveBudget
		}

		err = b.BudgetRepo.RecalculatePercentileAllocations(tx, budget)
		if err != nil {
			return err
		}
	}

	err = b.BudgetRepo.UpdateMonthlyBudget(tx, user, budget)
	if err != nil {
		return err
	}

	return nil
}

func (b *BudgetInterface) UpdateTotalInflow(tx *gorm.DB, user *models.User, inflow *models.Inflow, operation string, amountDifference float64) error {

	category := "inflow"
	dynamicCategory, err := b.findDynamicCategoryByRelatedCategoryID(user, category, inflow.InflowCategoryID)
	if err != nil {
		return err
	}

	var amount float64
	switch operation {
	case "update":
		amount = amountDifference
	case "delete":
		amount = inflow.Amount * -1
	default:
		amount = inflow.Amount
	}

	var categoryID uint

	if dynamicCategory == nil {
		categoryID = 0
	} else {
		categoryID = dynamicCategory.ID
	}

	err = b.updateBudget(tx, user, categoryID, category, amount, inflow.InflowDate)
	if err != nil {
		return err
	}

	return nil
}

func (b *BudgetInterface) UpdateTotalOutflow(tx *gorm.DB, user *models.User, outflow *models.Outflow, operation string, amountDifference float64) error {

	category := "outflow"
	dynamicCategory, err := b.findDynamicCategoryByRelatedCategoryID(user, category, outflow.OutflowCategoryID)
	if err != nil {
		return err
	}

	var amount float64
	switch operation {
	case "update":
		amount = amountDifference
	case "delete":
		amount = outflow.Amount * -1
	default:
		amount = outflow.Amount
	}

	var categoryID uint

	if dynamicCategory == nil {
		categoryID = 0
	} else {
		categoryID = dynamicCategory.ID
	}

	err = b.updateBudget(tx, user, categoryID, category, amount, outflow.OutflowDate)
	if err != nil {
		return err
	}

	return nil
}

func (b *BudgetInterface) FetchTotalValuesForInflowCategory(user *models.User, categoryID uint, year, month int) (float64, error) {
	total, err := b.InflowRepo.SumInflowsByCategory(user, categoryID, year, month)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (b *BudgetInterface) FetchTotalValuesForOutflowCategory(user *models.User, categoryID uint, year, month int) (float64, error) {
	total, err := b.OutflowRepo.SumOutflowsByCategory(user, categoryID, year, month)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (b *BudgetInterface) FetchTotalValuesForDynamicCategory(user *models.User, dynamicCategoryID uint, year, month int) (float64, error) {
	var total float64

	// Fetch all mappings for this dynamic category
	mappings, err := b.InflowRepo.FetchMappingsForDynamicCategory(dynamicCategoryID)
	if err != nil {
		return 0, err
	}

	for _, mapping := range mappings {
		var sum float64
		switch mapping.RelatedCategoryName {
		case "inflow":
			{
				sum, err = b.FetchTotalValuesForInflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
			}
		case "outflow":
			{
				outflowSum, err := b.FetchTotalValuesForOutflowCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
				// Subtract outflows
				sum -= outflowSum
			}
		case "dynamic":
			{
				// Recursive call for nested dynamic category
				dynamicSum, err := b.FetchTotalValuesForDynamicCategory(user, mapping.RelatedCategoryID, year, month)
				if err != nil {
					return 0, err
				}
				sum += dynamicSum
			}
		}

		total += sum
	}

	return total, nil
}
