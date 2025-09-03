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

type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *LoggingRepository) CountLogs(filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.db.Model(&models.ActivityLog{})

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}

	return totalRecords, nil
}

func (r *LoggingRepository) FindLogs(offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.ActivityLog, error) {

	var records []models.ActivityLog
	query := r.db.Table("activity_logs").Select("*")

	joins := utils.GetRequiredJoins(filters)
	orderBy := sortField + " " + sortOrder

	for _, join := range joins {
		query = query.Joins(join)
	}

	query = utils.ApplyFilters(query, filters)

	err := query.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
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

	// events
	var eventVals []string
	if err := db.Distinct("event").Pluck("event", &eventVals).Error; err == nil {
		events := make([]Option, 0, len(eventVals))
		for _, v := range eventVals {
			if v == "" {
				continue
			}
			events = append(events, Option{ID: v, Name: v})
		}
		response["events"] = events
	}

	if activityIndex == "activity" {
		var categoryVals []string
		if err := db.Distinct("category").Pluck("category", &categoryVals).Error; err == nil {
			categories := make([]Option, 0, len(categoryVals))
			for _, v := range categoryVals {
				if v == "" {
					continue
				}
				categories = append(categories, Option{ID: v, Name: v})
			}
			response["categories"] = categories
		}
	}

	if activityIndex == "access" {
		var stateVals []string
		if err := db.Distinct("status").Pluck("status", &stateVals).Error; err == nil {
			states := make([]Option, 0, len(stateVals))
			for _, v := range stateVals {
				if v == "" {
					continue
				}
				states = append(states, Option{ID: v, Name: v})
			}
			response["states"] = states
		}
	}

	// causers
	var rawCauserIDs []sql.NullInt64
	if err := db.Distinct("causer_id").Pluck("causer_id", &rawCauserIDs).Error; err == nil {
		var causerIDs []int64
		for _, id := range rawCauserIDs {
			if id.Valid {
				causerIDs = append(causerIDs, id.Int64)
			}
		}

		var causers []map[string]interface{}
		if len(causerIDs) > 0 {
			var users []models.User
			if err := r.db.Where("id IN ? AND deleted_at IS NULL", causerIDs).Find(&users).Error; err == nil {
				for _, u := range users {
					causers = append(causers, map[string]interface{}{
						"id":       u.ID,
						"username": u.Username,
						// add a generic "name" for consistency with optionLabel: 'name'
						"name": u.Username,
					})
				}
			}
		}
		response["causers"] = causers
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
