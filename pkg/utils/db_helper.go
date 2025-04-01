package utils

import (
	"fmt"
	"gorm.io/gorm"
)

type FieldMetadata struct {
	Column string
	Join   string
}

var FieldMap = map[string]FieldMetadata{
	"inflow_category":  {Column: "inflow_categories.name", Join: "LEFT JOIN inflow_categories ON inflow_categories.id = inflows.inflow_category_id"},
	"inflow_date":      {Column: "DATE(inflows.inflow_date)"},
	"outflow_category": {Column: "outflow_categories.name", Join: "LEFT JOIN outflow_categories ON outflow_categories.id = outflows.outflow_category_id"},
	"outflow_date":     {Column: "DATE(inflows.outflow_date)"},
	"transaction_date": {Column: "DATE(savings_transactions.transaction_date)"},
	"savings_category": {Column: "savings_categories.name", Join: "LEFT JOIN savings_categories ON savings_categories.id = savings_transactions.savings_category_id"},
}

func ApplyFilters(query *gorm.DB, filters []Filter) *gorm.DB {
	for _, f := range filters {

		meta, ok := FieldMap[f.Parameter]
		column := f.Parameter
		if ok {
			column = meta.Column
		}

		switch f.Operator {
		case "equals":
			query = query.Where(fmt.Sprintf("%s = ?", column), f.Value)
		case "not equals":
			query = query.Where(fmt.Sprintf("%s <> ?", column), f.Value)
		case "contains":
			query = query.Where(fmt.Sprintf("%s LIKE ?", column), "%"+f.Value+"%")
		case "more than":
			query = query.Where(fmt.Sprintf("%s > ?", column), f.Value)
		case "less than":
			query = query.Where(fmt.Sprintf("%s < ?", column), f.Value)
		default:
			// Unknown operator — skip or log
		}
	}
	return query
}

func GetRequiredJoins(filters []Filter) []string {
	needed := make(map[string]struct{})

	checkField := func(field string) {
		if meta, ok := FieldMap[field]; ok && meta.Join != "" {
			needed[meta.Join] = struct{}{}
		}
	}

	for _, f := range filters {
		checkField(f.Parameter)
	}

	var joins []string
	for join := range needed {
		joins = append(joins, join)
	}
	return joins
}

func ConstructOrderByClause(joins *[]string, sortField, sortOrder string) string {
	sortColumn := sortField

	if meta, ok := FieldMap[sortField]; ok {
		sortColumn = meta.Column

		if meta.Join != "" {
			// Deduplicate join
			alreadyJoined := false
			for _, j := range *joins {
				if j == meta.Join {
					alreadyJoined = true
					break
				}
			}
			if !alreadyJoined {
				*joins = append(*joins, meta.Join)
			}
		}
	}

	return sortColumn + " " + sortOrder
}
