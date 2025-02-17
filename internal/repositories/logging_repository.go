package repositories

import (
	"encoding/json"
	"gorm.io/gorm"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"
)

type LoggingRepository struct {
	db *gorm.DB
}

func NewLoggingRepository(db *gorm.DB) *LoggingRepository {
	return &LoggingRepository{db: db}
}

func (r *LoggingRepository) CountLogs(tableName string, filters map[string]interface{}) (int64, error) {
	var totalRecords int64
	query := r.db.Table(tableName).Where(filters)
	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *LoggingRepository) FindLogs(tableName string, offset, limit int, sortField, sortOrder string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	var logs []map[string]interface{}
	orderBy := sortField + " " + sortOrder

	query := r.db.Table(tableName).Where(filters).Order(orderBy).Limit(limit).Offset(offset)
	err := query.Find(&logs).Error
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *LoggingRepository) InsertActivityLog(event string, category string, description *string, payload *utils.Changes, causer *models.User) error {

	doc := models.ActivityLog{
		Event:       event,
		Category:    category,
		Description: description,
	}

	if payload != nil && (len(payload.New) != 0 || len(payload.Old) != 0) {
		metadata, err := json.Marshal(map[string]interface{}{
			"new": payload.New,
			"old": payload.Old,
		})
		if err != nil {
			return err
		}
		doc.Metadata = metadata
	}

	if causer != nil {
		doc.CauserID = &causer.ID
	}

	if err := r.db.Table("activity_logs").Create(&doc).Error; err != nil {
		return err
	}

	return nil
}
