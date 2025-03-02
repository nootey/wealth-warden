package repositories

import (
	"fmt"
	"gorm.io/gorm"
	"wealth-warden/internal/models"
)

type ReoccurringActionsRepository struct {
	Db *gorm.DB
}

func NewReoccurringActionsRepository(db *gorm.DB) *ReoccurringActionsRepository {
	return &ReoccurringActionsRepository{Db: db}
}

func (r *ReoccurringActionsRepository) CountReoccurringActionByCategory(userID uint, categoryName string, categoryID uint, count *int64) error {
	return r.Db.Model(&models.RecurringAction{}).
		Where("category_id = ?", categoryID).
		Where("category_type = ?", categoryName).
		Where("user_id = ?", userID).
		Count(count).Error
}

func (r *ReoccurringActionsRepository) FindDistinctYearsForRecords(userID uint, table, field string) ([]int, error) {
	var years []int

	query := fmt.Sprintf("SELECT DISTINCT YEAR(%s) FROM %s WHERE user_id = ? ORDER BY YEAR(%s) DESC", field, table, field)

	err := r.Db.Raw(query, userID).Scan(&years).Error
	if err != nil {
		return nil, err
	}

	return years, nil
}

func (r *ReoccurringActionsRepository) FindAllActionsForCategory(userID uint, categoryName string) ([]models.RecurringAction, error) {
	var actions []models.RecurringAction
	result := r.Db.Where("user_id = ?", userID).Where("category_type = ?", categoryName).Find(&actions)
	return actions, result.Error
}

func (r *ReoccurringActionsRepository) GetActionByID(userID, recordID uint) (*models.RecurringAction, error) {
	var record models.RecurringAction
	err := r.Db.Where("id = ? AND user_id = ?", recordID, userID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *ReoccurringActionsRepository) InsertReoccurringAction(tx *gorm.DB, userID uint, reoccurringAction *models.RecurringAction) (uint, error) {
	reoccurringAction.UserID = userID
	if err := tx.Create(&reoccurringAction).Error; err != nil {
		return 0, err
	}
	return reoccurringAction.ID, nil
}

func (r *ReoccurringActionsRepository) DropAction(tx *gorm.DB, userID uint, recordID uint) error {
	return tx.Where("id = ? AND user_id = ?", recordID, userID).Delete(&models.RecurringAction{}).Error
}
