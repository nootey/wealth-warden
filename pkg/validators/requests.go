package validators

import (
	"time"
)

type CreateInflowRequest struct {
	InflowCategoryID uint      `json:"inflow_category_id" validate:"required,numeric"`
	Amount           float64   `json:"amount" validate:"required,numeric,min=0,max=1000000000"`
	InflowDate       time.Time `json:"inflow_date" validate:"required"`
}

type CreateInflowCategoryRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

type CreateReoccurringActionRequest struct {
	StartDate     time.Time  `json:"start_date" validate:"required"`
	EndDate       *time.Time `json:"end_date"`
	Category      string     `json:"category_type" validate:"required"`
	IntervalUnit  string     `json:"interval_unit" validate:"required"`
	IntervalValue int        `json:"interval_value" validate:"required"`
}

type ReoccurringInflowRequest struct {
	Inflow    CreateInflowRequest            `json:"Inflow" validate:"required"`
	RecInflow CreateReoccurringActionRequest `json:"RecInflow" validate:"required"`
}
