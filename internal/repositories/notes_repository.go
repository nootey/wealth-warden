package repositories

import (
	"context"
	"time"
	"wealth-warden/internal/models"

	"gorm.io/gorm"
)

type NotesRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	FindNotes(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int) ([]models.Note, error)
	CountNotes(ctx context.Context, tx *gorm.DB, userID int64) (int64, error)
	FindNoteByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Note, error)
	InsertNote(ctx context.Context, tx *gorm.DB, newRecord *models.Note) (int64, error)
	UpdateNote(ctx context.Context, tx *gorm.DB, record models.Note) (int64, error)
	ToggleResolveState(ctx context.Context, tx *gorm.DB, record models.Note) (int64, error)
	DeleteNote(ctx context.Context, tx *gorm.DB, id int64) error
}

type NotesRepository struct {
	db *gorm.DB
}

func NewNotesRepository(db *gorm.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

var _ NotesRepositoryInterface = (*NotesRepository)(nil)

func (r *NotesRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	return tx, tx.Error
}

func (r *NotesRepository) FindNotes(ctx context.Context, tx *gorm.DB, userID int64, offset, limit int) ([]models.Note, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var records []models.Note
	err := db.Model(&models.Note{}).
		Where("user_id = ?", userID).
		Order("resolved_at IS NULL DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *NotesRepository) CountNotes(ctx context.Context, tx *gorm.DB, userID int64) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var totalRecords int64
	if err := db.Model(&models.Note{}).
		Where("user_id = ?", userID).
		Count(&totalRecords).Error; err != nil {
		return 0, err
	}
	return totalRecords, nil
}

func (r *NotesRepository) FindNoteByID(ctx context.Context, tx *gorm.DB, ID, userID int64) (models.Note, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	var record models.Note
	q := db.Where("id = ? AND user_id = ?", ID, userID)

	q = q.First(&record)

	return record, q.Error
}

func (r *NotesRepository) InsertNote(ctx context.Context, tx *gorm.DB, newRecord *models.Note) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Create(&newRecord).Error; err != nil {
		return 0, err
	}
	return newRecord.ID, nil
}

func (r *NotesRepository) UpdateNote(ctx context.Context, tx *gorm.DB, record models.Note) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Model(models.Note{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"content":    record.Content,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *NotesRepository) ToggleResolveState(ctx context.Context, tx *gorm.DB, record models.Note) (int64, error) {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	updates := map[string]interface{}{
		"resolved_at": record.ResolvedAt,
		"updated_at":  time.Now().UTC(),
	}

	if err := db.Model(&models.Note{}).
		Where("id = ?", record.ID).
		Updates(updates).Error; err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (r *NotesRepository) DeleteNote(ctx context.Context, tx *gorm.DB, id int64) error {
	db := tx
	if db == nil {
		db = r.db
	}
	db = db.WithContext(ctx)

	if err := db.Where("id = ?", id).
		Delete(&models.Note{}).Error; err != nil {
		return err
	}
	return nil
}
