package repositories

import (
	"gorm.io/gorm"
)

type ImportRepository struct {
	DB *gorm.DB
}

func NewImportRepository(db *gorm.DB) *ImportRepository {
	return &ImportRepository{DB: db}
}
