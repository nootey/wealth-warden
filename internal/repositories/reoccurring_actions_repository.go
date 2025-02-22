package repositories

import (
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type ReoccurringActionsRepository struct {
	db *gorm.DB
}

func NewReoccurringActionsRepository(db *gorm.DB) *ReoccurringActionsRepository {
	return &ReoccurringActionsRepository{db: db}
}


func (r *ReoccurringActionsRepository) FindAllActionsForCategory(userID uint, categoryName string) ([]models.RecurringAction, error) {
	var actions []models.RecurringAction
	result := r.db.Where("user_id = ?", userID).Where("category_type = ?", categoryName).Find(&actions)
	return actions, result.Error
}

func (r *ReoccurringActionsRepository) InsertReoccurringAction(tx *gorm.DB, userID uint, reoccurringAction *models.RecurringAction) (uint, error) {
	reoccurringAction.UserID = userID
	if err := tx.Create(&reoccurringAction).Error; err != nil {
		return 0, err
	}
	return reoccurringAction.ID, nil
}
