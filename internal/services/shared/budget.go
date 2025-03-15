package shared

import (
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

type BudgetInterface struct {
	InflowRepo  *repositories.InflowRepository
	OutflowRepo *repositories.OutflowRepository
}

func NewBudgetInterface(inflowRepo *repositories.InflowRepository, outflowRepo *repositories.OutflowRepository) *BudgetInterface {
	return &BudgetInterface{
		InflowRepo:  inflowRepo,
		OutflowRepo: outflowRepo,
	}
}

func (b *BudgetInterface) UpdateTotalInflow(user *models.User, year, month int, amount float64) error {
	// Implement logic for updating inflow in the budget
	return nil
}

func (b *BudgetInterface) UpdateTotalOutflow(user *models.User, year, month int, amount float64) error {
	// Implement logic for updating outflow in the budget
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
