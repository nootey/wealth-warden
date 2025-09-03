package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func (r *LoggingRepository) filterLogs(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	for key, value := range filters {
		switch key {
		case "date_range":
			if v, ok := value.([]string); ok && len(v) == 2 {
				query = query.Where("created_at BETWEEN ? AND ?", v[0], v[1])
			}
		default:
			switch v := value.(type) {
			case []string:
				if len(v) > 0 {
					query = query.Where(key+" IN ?", v)
				}
			case []int:
				if len(v) > 0 {
					query = query.Where(key+" IN ?", v)
				}
			default:
				query = query.Where(key+" = ?", value)
			}
		}
	}

	return query
}

func (r *LoggingRepository) CountLogs(tableName string, filters map[string]interface{}) (int64, error) {
	var totalRecords int64

	query := r.db.Table(tableName).Select("*")
	query = r.filterLogs(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *LoggingRepository) FindLogs(tableName string, offset, limit int, sortField, sortOrder string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	var logs []map[string]interface{}
	orderBy := sortField + " " + sortOrder

	query := r.db.Table(tableName).Select("*")
	query = r.filterLogs(query, filters)

	err := query.Order(orderBy).Limit(limit).Offset(offset).Find(&logs).Error
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *LoggingRepository) FindActivityLogFilterData(activityIndex string) (map[string]interface{}, error) {
	response := make(map[string]interface{})

	var tableName string
	switch activityIndex {
	case "activity":
		tableName = "activity_logs"
	case "access":
		tableName = "access_logs"
	default:
		return nil, fmt.Errorf("invalid activity index")
	}

	db := r.db.Table(tableName)

	var events []string
	if err := db.Distinct("event").Pluck("event", &events).Error; err == nil {
		response["events"] = events
	}

	if activityIndex == "activity" {
		var categories []string
		if err := db.Distinct("category").Pluck("category", &categories).Error; err == nil {
			response["categories"] = categories
		}
	}

	if activityIndex == "access" {
		var states []string
		if err := db.Distinct("status").Pluck("status", &states).Error; err == nil {
			response["states"] = states
		}
	}

	var rawCauserIDs []sql.NullInt64
	if err := db.Distinct("causer_id").Pluck("causer_id", &rawCauserIDs).Error; err == nil {
		var causerIDs []int64
		for _, id := range rawCauserIDs {
			if id.Valid {
				causerIDs = append(causerIDs, int64(id.Int64))
			}
		}

		var causers []map[string]interface{}
		if len(causerIDs) > 0 {
			var users []models.User
			err := r.db.Where("id IN ? AND deleted_at IS NULL", causerIDs).Find(&users).Error
			if err == nil {
				for _, u := range users {
					causers = append(causers, map[string]interface{}{
						"id":       u.ID,
						"username": u.Username,
					})
				}
			}
		}
		response["causers"] = causers
	} else {
		fmt.Println(err)
	}

	return response, nil
}

func (r *LoggingRepository) InsertActivityLog(
	tx *gorm.DB,
	event string,
	category string,
	description *string,
	payload *utils.Changes,
	causer *models.User,
) error {

	db := tx
	if db == nil {
		db = r.db
	}

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

	return db.Table("activity_logs").Create(&doc).Error
}
