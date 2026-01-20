package repositories

import (
	"context"

	"gorm.io/gorm"
)

type NotesRepositoryInterface interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
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
