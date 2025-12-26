package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

type LoggingRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	InsertActivityLog(ctx context.Context, tx *gorm.DB, event string, category string, description *string, payload *utils.Changes, causer *int64) error
	CountLogs(ctx context.Context, filters []utils.Filter) (int64, error)
	FindLogs(ctx context.Context, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.ActivityLog, error)
	FindActivityLogFilterData(ctx context.Context, activityIndex string) (map[string]interface{}, error)
	FindActivityLogByID(ctx context.Context, tx *gorm.DB, ID int64) (models.ActivityLog, error)
	FindAuditTrailByRecordID(ctx context.Context, recordID, category string, events []string) ([]models.ActivityLog, error)
	DeleteActivityLog(ctx context.Context, tx *gorm.DB, id int64) error
}

type LoggingRepository struct {
	db *gorm.DB
}

func NewLoggingRepository(db *gorm.DB) *LoggingRepository {
	return &LoggingRepository{db: db}
}

var _ LoggingRepositoryInterface = (*LoggingRepository)(nil)

type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *LoggingRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *LoggingRepository) CountLogs(ctx context.Context, filters []utils.Filter) (int64, error) {
	var totalRecords int64

	query := r.db.WithContext(ctx).Model(&models.ActivityLog{})

	query = utils.ApplyFilters(query, filters)

	err := query.Count(&totalRecords).Error
	if err != nil {
		return 0, err
	}

	return totalRecords, nil
}

func (r *LoggingRepository) FindLogs(ctx context.Context, offset, limit int, sortField, sortOrder string, filters []utils.Filter) ([]models.ActivityLog, error) {

	var records []models.ActivityLog
	query := r.db.WithContext(ctx).Table("activity_logs").Select("*")

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

func (r *LoggingRepository) FindActivityLogFilterData(ctx context.Context, activityIndex string) (map[string]interface{}, error) {
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

	db := r.db.Table(tableName).WithContext(ctx)

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
						"id":   u.ID,
						"name": u.DisplayName,
					})
				}
			}
		}
		response["causers"] = causers
	}

	return response, nil
}

func (r *LoggingRepository) FindActivityLogByID(ctx context.Context, tx *gorm.DB, ID int64) (models.ActivityLog, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.ActivityLog
	result := db.Where("id = ?", ID).First(&record)
	return record, result.Error
}

func (r *LoggingRepository) FindAuditTrailByRecordID(ctx context.Context, recordID, category string, events []string) ([]models.ActivityLog, error) {
	var allLogs []models.ActivityLog
	err := r.db.WithContext(ctx).
		Where("event IN ? AND category = ?", events, category).
		Order("created_at DESC").
		Find(&allLogs).Error
	if err != nil {
		return nil, err
	}

	// Filter by ID in metadata
	var records []models.ActivityLog
	for _, log := range allLogs {
		if log.Metadata == nil {
			continue
		}

		var metadata map[string]map[string]interface{}
		if err := json.Unmarshal(log.Metadata, &metadata); err != nil {
			continue
		}

		newData, ok := metadata["new"]
		if !ok {
			continue
		}

		idVal, ok := newData["id"]
		if !ok {
			continue
		}

		idStr := fmt.Sprintf("%v", idVal)
		if idStr == recordID {
			records = append(records, log)
		}
	}

	return records, nil
}

func (r *LoggingRepository) InsertActivityLog(
	ctx context.Context,
	tx *gorm.DB,
	event string,
	category string,
	description *string,
	payload *utils.Changes,
	causerID *int64,
) error {

	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

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

	if causerID != nil {
		doc.CauserID = causerID
	}

	return db.Table("activity_logs").Create(&doc).Error
}

func (r *LoggingRepository) DeleteActivityLog(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.
		Where("id = ?", id).
		Delete(&models.ActivityLog{})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
