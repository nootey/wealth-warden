package utils

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type FieldMetadata struct {
	Column       string
	Join         string
	FilterColumn string
	OrEquals     bool
}

var FieldMap = map[string]map[string]FieldMetadata{
	"transactions": {
		"category": {
			Column:       "categories.name",
			FilterColumn: "categories.id",
			Join:         "LEFT JOIN categories ON categories.id = transactions.category_id",
			OrEquals:     true,
		},
		"account": {
			Column:       "accounts.name",
			FilterColumn: "accounts.id",
			Join:         "LEFT JOIN accounts ON accounts.id = transactions.account_id",
			OrEquals:     true,
		},
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
	// Group for fields that opt-in to OR behavior
	type key struct{ source, field string }
	eq := map[key][]any{}

	// collect OR-able "="
	for _, f := range filters {
		if f.Operator == "=" || f.Operator == "equals" {
			if meta, ok := resolveMeta(f.Source, f.Field); ok && meta.OrEquals {
				eq[key{f.Source, f.Field}] = append(eq[key{f.Source, f.Field}], f.Value)
			}
		}
	}

	// apply everything else
	for _, f := range filters {
		meta, ok := resolveMeta(f.Source, f.Field)
		column := f.Field
		if ok {
			column = meta.Column
		}

		switch f.Operator {
		case "equals", "=":
			if ok && meta.OrEquals {
				continue
			}
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
			col := column
			if ok && meta.FilterColumn != "" {
				col = meta.FilterColumn
			}
			query = query.Where(fmt.Sprintf("%s IN ?", col), f.Value)

		default:
			fmt.Println("Unknown operator")
		}
	}

	// apply grouped "=" as one IN per field
	for k, vals := range eq {
		meta, ok := resolveMeta(k.source, k.field)
		col := k.field
		if ok && meta.FilterColumn != "" {
			col = meta.FilterColumn
		} else if ok {
			col = meta.Column
		}
		query = query.Where(fmt.Sprintf("%s IN ?", col), vals)
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
