package utils

import (
	"fmt"
	"gorm.io/gorm"
)

func ApplyFilters(query *gorm.DB, filters []Filter) *gorm.DB {
	for _, filter := range filters {

		column := mapFilterToColumn(filter.Parameter)
		if column == "" {
			continue
		}

		switch filter.Operator {
		case "equals":
			query = query.Where(fmt.Sprintf("%s = ?", column), filter.Value)
		case "not equals":
			query = query.Where(fmt.Sprintf("%s <> ?", column), filter.Value)
		case "contains":
			query = query.Where(fmt.Sprintf("%s LIKE ?", column), "%"+filter.Value+"%")
		case "more than":
			query = query.Where(fmt.Sprintf("%s > ?", column), filter.Value)
		case "less than":
			query = query.Where(fmt.Sprintf("%s < ?", column), filter.Value)
		default:
			// Unknown operator — skip or log
		}
	}
	return query
}

func mapFilterToColumn(param string) string {
	switch param {
	case "inflow_category":
		return "inflow_categories.name"
	case "inflow_date":
		return "DATE(inflow_date)"
	default:
		return param
	}
}

func NeedsJoin(filters []Filter, param string) bool {
	for _, f := range filters {
		if f.Parameter == param {
			return true
		}
	}
	return false
}
