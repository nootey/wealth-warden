package repositories

import (
	"gorm.io/gorm"
)

type StatisticsRepository struct {
	DB *gorm.DB
}

func NewStatisticsRepository(db *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{DB: db}
}
