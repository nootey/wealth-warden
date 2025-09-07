package workers

import (
	"context"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"time"
	"wealth-warden/internal/models"
)

func normalizeName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "_")
}

func SeedCategories(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
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
			Children:       []string{"Salary", "Food and transport", "Bonus", "Side hustle", "Annual allowance", "Refunds", "Other"},
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
			Name:           normalizeName(mainCat.Name),
			DisplayName:    mainCat.Name,
			Classification: mainCat.Classification,
			IsDefault:      true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := db.WithContext(ctx).Create(&mainCategory).Error; err != nil {
			logger.Error("failed to create main category", zap.String("name", mainCat.Name), zap.Error(err))
			return err
		}

		// Subcategories
		for _, childName := range mainCat.Children {
			subCategory := models.Category{
				UserID:         nil,
				Name:           normalizeName(childName),
				DisplayName:    childName,
				Classification: mainCat.Classification,
				ParentID:       &mainCategory.ID,
				IsDefault:      true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := db.WithContext(ctx).Create(&subCategory).Error; err != nil {
				logger.Error("failed to create subcategory", zap.String("name", childName), zap.Error(err))
				return err
			}
		}
	}
	return nil
}
