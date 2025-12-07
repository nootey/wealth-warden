package workers

import (
	"context"
	"fmt"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

func SeedCategories(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	// Top-level categories
	mainCategories := []struct {
		Name           string
		Classification string
		Children       []string
	}{
		{
			Name:           "(Uncategorized)",
			Classification: "uncategorized",
			Children:       []string{},
		},
		{
			Name:           "(Adjustment)",
			Classification: "adjustment",
			Children:       []string{},
		},
		{
			Name:           "Income",
			Classification: "income",
			Children:       []string{"Salary", "Food and transport", "Bonus", "Side hustle", "Refunds", "Other"},
		},
		{
			Name:           "Expense",
			Classification: "expense",
			Children: []string{"Car - transportation", "Car - general", "Health", "Hygiene", "Entertainment",
				"Fees", "Food", "Rent", "Utilities", "Ecommerce", "Tech", "Clothes", "Gifts", "Other"},
		},
	}

	for _, mainCat := range mainCategories {
		mainCategory := models.Category{
			UserID:         nil,
			Name:           utils.NormalizeName(mainCat.Name),
			DisplayName:    mainCat.Name,
			Classification: mainCat.Classification,
			IsDefault:      true,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		if err := db.WithContext(ctx).Create(&mainCategory).Error; err != nil {
			return fmt.Errorf("failed to create main category %w", err)
		}

		// Subcategories
		for _, childName := range mainCat.Children {
			subCategory := models.Category{
				UserID:         nil,
				Name:           utils.NormalizeName(childName),
				DisplayName:    childName,
				Classification: mainCat.Classification,
				ParentID:       &mainCategory.ID,
				IsDefault:      true,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}
			if err := db.WithContext(ctx).Create(&subCategory).Error; err != nil {
				return fmt.Errorf("failed to create sub category %w", err)
			}
		}
	}
	return nil
}
