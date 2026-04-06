package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SavingsRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindGoals(ctx context.Context, tx *gorm.DB, userID int64) ([]models.SavingGoal, error)
	FindGoalByID(ctx context.Context, tx *gorm.DB, id, userID int64) (models.SavingGoal, error)
	InsertGoal(ctx context.Context, tx *gorm.DB, record *models.SavingGoal) (int64, error)
	UpdateGoal(ctx context.Context, tx *gorm.DB, record models.SavingGoal) (int64, error)
	DeleteGoal(ctx context.Context, tx *gorm.DB, id int64) error
	UpdateCurrentAmount(ctx context.Context, tx *gorm.DB, goalID int64, amount models.SavingGoal) error

	GetUncategorizedBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error)
	CountContributions(ctx context.Context, tx *gorm.DB, goalID int64) (int64, error)
	FindContributions(ctx context.Context, tx *gorm.DB, goalID int64) ([]models.SavingContribution, error)
	FindContributionsPaginated(ctx context.Context, tx *gorm.DB, goalID int64, offset, limit int) ([]models.SavingContribution, error)
	FindContributionByID(ctx context.Context, tx *gorm.DB, id, userID int64) (models.SavingContribution, error)
	InsertContribution(ctx context.Context, tx *gorm.DB, record *models.SavingContribution) (int64, error)
	DeleteContribution(ctx context.Context, tx *gorm.DB, id int64) error
}

type SavingsRepository struct {
	db *gorm.DB
}

func NewSavingsRepository(db *gorm.DB) *SavingsRepository {
	return &SavingsRepository{db: db}
}

var _ SavingsRepositoryInterface = (*SavingsRepository)(nil)

func (r *SavingsRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *SavingsRepository) FindGoals(ctx context.Context, tx *gorm.DB, userID int64) ([]models.SavingGoal, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.SavingGoal
	err := db.Model(&models.SavingGoal{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *SavingsRepository) FindGoalByID(ctx context.Context, tx *gorm.DB, id, userID int64) (models.SavingGoal, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.SavingGoal
	err := db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	return record, err
}

func (r *SavingsRepository) InsertGoal(ctx context.Context, tx *gorm.DB, record *models.SavingGoal) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *SavingsRepository) UpdateGoal(ctx context.Context, tx *gorm.DB, record models.SavingGoal) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(&models.SavingGoal{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"name":               record.Name,
			"target_amount":      record.TargetAmount,
			"target_date":        record.TargetDate,
			"status":             record.Status,
			"priority":           record.Priority,
			"monthly_allocation": record.MonthlyAllocation,
			"updated_at":         time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *SavingsRepository) DeleteGoal(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("id = ?", id).Delete(&models.SavingGoal{}).Error
}

func (r *SavingsRepository) UpdateCurrentAmount(ctx context.Context, tx *gorm.DB, goalID int64, record models.SavingGoal) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Model(&models.SavingGoal{}).
		Where("id = ?", goalID).
		Updates(map[string]interface{}{
			"current_amount": record.CurrentAmount,
			"updated_at":     time.Now().UTC(),
		}).Error
}

func (r *SavingsRepository) GetUncategorizedBalance(ctx context.Context, tx *gorm.DB, accountID, userID int64) (decimal.Decimal, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var endBalance decimal.Decimal
	err := db.Raw(
		`SELECT COALESCE(end_balance, 0) FROM balances WHERE account_id = ? ORDER BY as_of DESC LIMIT 1`,
		accountID,
	).Scan(&endBalance).Error
	if err != nil {
		return decimal.Zero, err
	}

	var allocated decimal.Decimal
	err = db.Raw(
		`SELECT COALESCE(SUM(current_amount), 0) FROM saving_goals WHERE account_id = ? AND user_id = ?`,
		accountID, userID,
	).Scan(&allocated).Error
	if err != nil {
		return decimal.Zero, err
	}

	uncategorized := endBalance.Sub(allocated)
	if uncategorized.IsNegative() {
		return decimal.Zero, nil
	}
	return uncategorized, nil
}

func (r *SavingsRepository) CountContributions(ctx context.Context, tx *gorm.DB, goalID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var count int64
	err := db.Model(&models.SavingContribution{}).Where("goal_id = ?", goalID).Count(&count).Error
	return count, err
}

func (r *SavingsRepository) FindContributionsPaginated(ctx context.Context, tx *gorm.DB, goalID int64, offset, limit int) ([]models.SavingContribution, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.SavingContribution
	err := db.Model(&models.SavingContribution{}).
		Where("goal_id = ?", goalID).
		Order("month DESC, created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error
	return records, err
}

func (r *SavingsRepository) FindContributions(ctx context.Context, tx *gorm.DB, goalID int64) ([]models.SavingContribution, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.SavingContribution
	err := db.Model(&models.SavingContribution{}).
		Where("goal_id = ?", goalID).
		Order("month DESC, created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *SavingsRepository) FindContributionByID(ctx context.Context, tx *gorm.DB, id, userID int64) (models.SavingContribution, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.SavingContribution
	err := db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	return record, err
}

func (r *SavingsRepository) InsertContribution(ctx context.Context, tx *gorm.DB, record *models.SavingContribution) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (r *SavingsRepository) DeleteContribution(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	return db.Where("id = ?", id).Delete(&models.SavingContribution{}).Error
}
