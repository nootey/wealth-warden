package repositories

import (
	"context"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type RulesRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindRules(ctx context.Context, tx *gorm.DB, userID int64, activeOnly bool) ([]models.Rule, error)
	FindRuleByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Rule, error)
	InsertRule(ctx context.Context, tx *gorm.DB, newRecord *models.Rule) (int64, error)
	UpdateRule(ctx context.Context, tx *gorm.DB, record models.Rule) (int64, error)
	DeleteRule(ctx context.Context, tx *gorm.DB, id int64) error
	PurgeImportedRules(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error)
}

type RulesRepository struct {
	db *gorm.DB
}

func NewRulesRepository(db *gorm.DB) *RulesRepository {
	return &RulesRepository{db: db}
}

var _ RulesRepositoryInterface = (*RulesRepository)(nil)

func (r *RulesRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *RulesRepository) FindRules(ctx context.Context, tx *gorm.DB, userID int64, activeOnly bool) ([]models.Rule, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	q := db.Model(&models.Rule{}).
		Preload("Conditions", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Preload("Actions", func(db *gorm.DB) *gorm.DB { return db.Order("position, id") }).
		Where("user_id = ?", userID)
	if activeOnly {
		q = q.Where("is_active = TRUE")
	}

	var records []models.Rule
	if err := q.Order("id").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *RulesRepository) FindRuleByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Rule, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Rule
	q := db.
		Preload("Conditions", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Preload("Actions", func(db *gorm.DB) *gorm.DB { return db.Order("position, id") }).
		Where("id = ? AND user_id = ?", ID, userID).
		First(&record)

	return record, q.Error
}

func (r *RulesRepository) InsertRule(ctx context.Context, tx *gorm.DB, newRecord *models.Rule) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	conditions := newRecord.Conditions
	newRecord.Conditions = nil
	if err := db.Create(newRecord).Error; err != nil {
		return 0, err
	}
	if err := insertConditions(db, newRecord.ID, nil, conditions); err != nil {
		return 0, err
	}
	newRecord.Conditions = conditions
	return newRecord.ID, nil
}

func insertConditions(db *gorm.DB, ruleID int64, parentID *int64, conditions []models.RuleCondition) error {
	for i := range conditions {
		c := &conditions[i]
		c.ID = 0
		c.RuleID = ruleID
		c.ParentID = parentID
		if err := db.Create(c).Error; err != nil {
			return err
		}
		if err := insertConditions(db, ruleID, &c.ID, c.Children); err != nil {
			return err
		}
	}
	return nil
}

func (r *RulesRepository) UpdateRule(ctx context.Context, tx *gorm.DB, record models.Rule) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(&models.Rule{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":       record.Name,
			"is_active":  record.IsActive,
			"match_type": record.MatchType,
		}).Error; err != nil {
		return 0, err
	}

	if err := db.Where("rule_id = ?", record.ID).Delete(&models.RuleCondition{}).Error; err != nil {
		return 0, err
	}
	if err := db.Where("rule_id = ?", record.ID).Delete(&models.RuleAction{}).Error; err != nil {
		return 0, err
	}

	if err := insertConditions(db, record.ID, nil, record.Conditions); err != nil {
		return 0, err
	}
	for i := range record.Actions {
		record.Actions[i].ID = 0
		record.Actions[i].RuleID = record.ID
	}
	if len(record.Actions) > 0 {
		if err := db.Create(&record.Actions).Error; err != nil {
			return 0, err
		}
	}

	return record.ID, nil
}

func (r *RulesRepository) DeleteRule(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("id = ?", id).Delete(&models.Rule{}).Error
}

func (r *RulesRepository) PurgeImportedRules(ctx context.Context, tx *gorm.DB, importID, userID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	res := db.Where("user_id = ? AND import_id = ?", userID, importID).Delete(&models.Rule{})
	return res.RowsAffected, res.Error
}
