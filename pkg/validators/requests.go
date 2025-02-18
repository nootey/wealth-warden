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
