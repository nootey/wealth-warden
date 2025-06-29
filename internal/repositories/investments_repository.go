package repositories

import (
	"gorm.io/gorm"
)

type InvestmentsRepository struct {
	Db *gorm.DB
}

func NewInvestmentsRepository(db *gorm.DB) *InvestmentsRepository {
	return &InvestmentsRepository{Db: db}
}
