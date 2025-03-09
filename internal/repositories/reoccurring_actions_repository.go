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

func (r *ReoccurringActionsRepository) CountReoccurringActionByCategory(user *models.User, categoryName string, categoryID uint, count *int64) error {
	return r.Db.Model(&models.RecurringAction{}).
		Where("category_id = ?", categoryID).
		Where("category_type = ?", categoryName).
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Count(count).Error
}

func (r *ReoccurringActionsRepository) FindDistinctYearsForRecords(user *models.User, table, field string) ([]int, error) {
	var years []int

	query := fmt.Sprintf("SELECT DISTINCT YEAR(%s) FROM %s WHERE organization_id = ? ORDER BY YEAR(%s) DESC", field, table, field)

	err := r.Db.Raw(query, *user.PrimaryOrganizationID).Scan(&years).Error
	if err != nil {
		return nil, err
	}

	return years, nil
}

func (r *ReoccurringActionsRepository) FindAllActionsForCategory(user *models.User, categoryName string) ([]models.RecurringAction, error) {
	var actions []models.RecurringAction
	result := r.Db.
		Where("organization_id = ?", *user.PrimaryOrganizationID).
		Where("category_type = ?", categoryName).Find(&actions)
	return actions, result.Error
}

func (r *ReoccurringActionsRepository) GetActionByID(user *models.User, recordID uint) (*models.RecurringAction, error) {
	var record models.RecurringAction
	err := r.Db.Where("id = ? AND organization_id = ?", recordID, *user.PrimaryOrganizationID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *ReoccurringActionsRepository) InsertReoccurringAction(tx *gorm.DB, user *models.User, reoccurringAction *models.RecurringAction) (uint, error) {
	reoccurringAction.OrganizationID = *user.PrimaryOrganizationID
	reoccurringAction.UserID = user.ID
	if err := tx.Create(&reoccurringAction).Error; err != nil {
		return 0, err
	}
	return reoccurringAction.ID, nil
}

func (r *ReoccurringActionsRepository) DropAction(tx *gorm.DB, user *models.User, recordID uint) error {
	return tx.Where("id = ? AND organization_id = ?", recordID, *user.PrimaryOrganizationID).Delete(&models.RecurringAction{}).Error
}
