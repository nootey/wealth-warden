package workers

import (
	"context"
	"errors"
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
		mainCategory, err := ensureDefaultCategory(ctx, db, mainCat.Name, mainCat.Classification, nil)
		if err != nil {
			return fmt.Errorf("failed to create main category %w", err)
		}

		// Subcategories
		for _, childName := range mainCat.Children {
			if _, err := ensureDefaultCategory(ctx, db, childName, mainCat.Classification, &mainCategory.ID); err != nil {
				return fmt.Errorf("failed to create sub category %w", err)
			}
		}
	}
	return nil
}

// Get-or-create, so the seeder can run again on a populated database.
func ensureDefaultCategory(ctx context.Context, db *gorm.DB, displayName, classification string, parentID *int64) (models.Category, error) {
	name := utils.NormalizeName(displayName)

	var existing models.Category
	err := db.WithContext(ctx).
		Where("name = ? AND classification = ?", name, classification).
		First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Category{}, err
	}

	category := models.Category{
		UserID:         nil,
		Name:           name,
		DisplayName:    displayName,
		Classification: classification,
		ParentID:       parentID,
		IsDefault:      true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := db.WithContext(ctx).Create(&category).Error; err != nil {
		return models.Category{}, err
	}
	return category, nil
}
