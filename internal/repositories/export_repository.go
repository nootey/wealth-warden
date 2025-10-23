package repositories

import "gorm.io/gorm"

type ExportRepository struct {
	DB *gorm.DB
}

func NewExportRepository(db *gorm.DB) *ExportRepository {
	return &ExportRepository{DB: db}
}
