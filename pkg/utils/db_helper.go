package utils

import (
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

type FieldMetadata struct {
	Column string
	Join   string
}

var FieldMap = map[string]map[string]FieldMetadata{
	"transactions": {
		"category": {Column: "categories.name", Join: "LEFT JOIN categories ON categories.id = transactions.category_id"},
		"account":  {Column: "accounts.name", Join: "LEFT JOIN accounts   ON accounts.id   = transactions.account_id"},
	},
}

func resolveMeta(source, field string) (FieldMetadata, bool) {
	m, ok := FieldMap[source]
	if !ok {
		return FieldMetadata{}, false
	}
	meta, ok2 := m[field]
	return meta, ok2
}

func ApplyFilters(query *gorm.DB, filters []Filter) *gorm.DB {

	for _, f := range filters {

		meta, ok := resolveMeta(f.Source, f.Field)
		column := f.Field
		if ok {
			column = meta.Column
		}

		switch f.Operator {
		case "equals", "=":
			query = query.Where(fmt.Sprintf("LOWER(%s) = ?", column), strings.ToLower(fmt.Sprint(f.Value)))
		case "not equals", "<>", "!=":
			query = query.Where(fmt.Sprintf("LOWER(%s) <> ?", column), strings.ToLower(fmt.Sprint(f.Value)))
		case "contains", "like":
			query = query.Where(fmt.Sprintf("LOWER(%s) LIKE ?", column), "%"+strings.ToLower(fmt.Sprint(f.Value))+"%")
		case "more than", ">":
			query = query.Where(fmt.Sprintf("%s > ?", column), f.Value)
		case "less than", "<":
			query = query.Where(fmt.Sprintf("%s < ?", column), f.Value)
		case ">=":
			query = query.Where(fmt.Sprintf("%s >= ?", column), f.Value)
		case "<=":
			query = query.Where(fmt.Sprintf("%s <= ?", column), f.Value)
		case "in":
			vals := reflect.ValueOf(f.Value)
			if vals.Kind() == reflect.Slice {
				lowered := []string{}
				for i := 0; i < vals.Len(); i++ {
					lowered = append(lowered, strings.ToLower(fmt.Sprint(vals.Index(i).Interface())))
				}
				query = query.Where(fmt.Sprintf("LOWER(%s) IN (?)", column), lowered)
			} else {
				query = query.Where(fmt.Sprintf("LOWER(%s) IN (?)", column), strings.ToLower(fmt.Sprint(f.Value)))
			}
		default:
			// Unknown operator
			fmt.Println("Unknown operator")
		}
	}
	return query
}

func GetRequiredJoins(filters []Filter) []string {
	needed := make(map[string]struct{})

	for _, f := range filters {
		if meta, ok := resolveMeta(f.Source, f.Field); ok && meta.Join != "" { // <— NEW
			needed[meta.Join] = struct{}{}
		}
	}

	var joins []string
	for join := range needed {
		joins = append(joins, join)
	}
	return joins
}

func ConstructOrderByClause(joins *[]string, source, sortField, sortOrder string) string {
	sortColumn := sortField

	if meta, ok := resolveMeta(source, sortField); ok {
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
