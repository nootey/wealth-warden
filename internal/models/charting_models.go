package models

import (
	"github.com/shopspring/decimal"
	"time"
)

type ChartPoint struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}
