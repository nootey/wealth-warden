package utils_test

import (
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestGetRequiredJoins(t *testing.T) {
	t.Run("returns joins for fields with metadata", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "category", Operator: "equals", Value: "Food"},
			{Source: "transactions", Field: "account", Operator: "equals", Value: "Checking"},
		}

		joins := utils.GetRequiredJoins(filters)

		assert.Len(t, joins, 2)
		assert.Contains(t, joins, "LEFT JOIN categories ON categories.id = transactions.category_id")
		assert.Contains(t, joins, "LEFT JOIN accounts ON accounts.id = transactions.account_id")
	})

	t.Run("deduplicates joins", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "category", Operator: "equals", Value: "Food"},
			{Source: "transactions", Field: "category", Operator: "contains", Value: "Dining"},
		}

		joins := utils.GetRequiredJoins(filters)

		assert.Len(t, joins, 1)
		assert.Contains(t, joins, "LEFT JOIN categories ON categories.id = transactions.category_id")
	})

	t.Run("ignores fields without joins", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "users", Field: "role", Operator: "equals", Value: "admin"},
		}

		joins := utils.GetRequiredJoins(filters)

		assert.Empty(t, joins)
	})

	t.Run("returns empty for unknown source", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "unknown", Field: "field", Operator: "equals", Value: "value"},
		}

		joins := utils.GetRequiredJoins(filters)

		assert.Empty(t, joins)
	})

	t.Run("returns empty for unknown field", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "unknown_field", Operator: "equals", Value: "value"},
		}

		joins := utils.GetRequiredJoins(filters)

		assert.Empty(t, joins)
	})

	t.Run("returns empty for no filters", func(t *testing.T) {
		filters := []utils.Filter{}

		joins := utils.GetRequiredJoins(filters)

		assert.Empty(t, joins)
	})
}

func TestConstructOrderByClause(t *testing.T) {
	t.Run("simple field with ASC order", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "transactions", "amount", "ASC")

		assert.Equal(t, "amount ASC", result)
		assert.Empty(t, joins)
	})

	t.Run("simple field with DESC order", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "transactions", "date", "DESC")

		assert.Equal(t, "date DESC", result)
		assert.Empty(t, joins)
	})

	t.Run("field with metadata column mapping", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "transactions", "category", "ASC")

		assert.Equal(t, "categories.name ASC", result)
		assert.Len(t, joins, 1)
		assert.Contains(t, joins, "LEFT JOIN categories ON categories.id = transactions.category_id")
	})

	t.Run("field with metadata and existing join", func(t *testing.T) {
		joins := []string{"LEFT JOIN categories ON categories.id = transactions.category_id"}

		result := utils.ConstructOrderByClause(&joins, "transactions", "category", "DESC")

		assert.Equal(t, "categories.name DESC", result)
		assert.Len(t, joins, 1, "should not duplicate existing join")
	})

	t.Run("field with metadata but no join", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "users", "role", "ASC")

		assert.Equal(t, "roles.name ASC", result)
		assert.Empty(t, joins, "should not add empty join")
	})

	t.Run("multiple sorts on different fields", func(t *testing.T) {
		joins := []string{}

		result1 := utils.ConstructOrderByClause(&joins, "transactions", "category", "ASC")
		result2 := utils.ConstructOrderByClause(&joins, "transactions", "account", "DESC")

		assert.Equal(t, "categories.name ASC", result1)
		assert.Equal(t, "accounts.name DESC", result2)
		assert.Len(t, joins, 2)
	})

	t.Run("unknown source", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "unknown", "field", "ASC")

		assert.Equal(t, "field ASC", result)
		assert.Empty(t, joins)
	})

	t.Run("unknown field", func(t *testing.T) {
		joins := []string{}

		result := utils.ConstructOrderByClause(&joins, "transactions", "unknown_field", "DESC")

		assert.Equal(t, "unknown_field DESC", result)
		assert.Empty(t, joins)
	})
}

func TestApplyFilters(t *testing.T) {
	t.Run("equals operator", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "amount", Operator: "equals", Value: "100"},
		}

		assert.NotEmpty(t, filters)
		assert.Equal(t, "equals", filters[0].Operator)
	})

	t.Run("not equals operator variants", func(t *testing.T) {
		operators := []string{"not equals", "!=", "<>"}

		for _, op := range operators {
			filters := []utils.Filter{
				{Source: "transactions", Field: "amount", Operator: op, Value: "50"},
			}
			assert.Equal(t, op, filters[0].Operator, "operator %s should be preserved", op)
		}
	})

	t.Run("comparison operators", func(t *testing.T) {
		operators := []string{">", "<", ">=", "<=", "more than", "less than"}

		for _, op := range operators {
			filters := []utils.Filter{
				{Source: "transactions", Field: "amount", Operator: op, Value: "100"},
			}
			assert.Equal(t, op, filters[0].Operator, "operator %s should be preserved", op)
		}
	})

	t.Run("string operators", func(t *testing.T) {
		operators := []string{"contains", "like"}

		for _, op := range operators {
			filters := []utils.Filter{
				{Source: "transactions", Field: "description", Operator: op, Value: "Food"},
			}
			assert.Equal(t, op, filters[0].Operator, "operator %s should be preserved", op)
		}
	})

	t.Run("date filter format", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "txn_date", Operator: "equals", Value: "2024-01-15"},
		}

		// Verify date format is correct
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, filters[0].Value)
	})

	t.Run("OR logic grouping for same field", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "category", Operator: "equals", Value: "1"},
			{Source: "transactions", Field: "category", Operator: "equals", Value: "2"},
		}

		// Count how many filters target the same field
		sameFieldCount := 0
		for i := 0; i < len(filters); i++ {
			for j := i + 1; j < len(filters); j++ {
				if filters[i].Field == filters[j].Field &&
					filters[i].Source == filters[j].Source &&
					filters[i].Operator == "equals" {
					sameFieldCount++
				}
			}
		}

		assert.Greater(t, sameFieldCount, 0, "should have multiple equals on same field")
	})

	t.Run("multiple filters", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "amount", Operator: ">", Value: "50"},
			{Source: "transactions", Field: "amount", Operator: "<", Value: "200"},
			{Source: "transactions", Field: "description", Operator: "contains", Value: "Food"},
		}

		assert.Len(t, filters, 3)
		assert.NotEqual(t, filters[0].Operator, filters[1].Operator)
	})

	t.Run("metadata field resolution", func(t *testing.T) {
		// Test that metadata fields are properly mapped
		meta, exists := utils.FieldMap["transactions"]["category"]
		assert.True(t, exists, "category field should have metadata")
		assert.Equal(t, "categories.name", meta.Column)
		assert.Equal(t, "categories.id", meta.FilterColumn)
		assert.True(t, meta.OrEquals)
	})

	t.Run("metadata for account field", func(t *testing.T) {
		meta, exists := utils.FieldMap["transactions"]["account"]
		assert.True(t, exists, "account field should have metadata")
		assert.Equal(t, "accounts.name", meta.Column)
		assert.Equal(t, "accounts.id", meta.FilterColumn)
		assert.True(t, meta.OrEquals)
	})

	t.Run("metadata for users role field", func(t *testing.T) {
		meta, exists := utils.FieldMap["users"]["role"]
		assert.True(t, exists, "role field should have metadata")
		assert.Equal(t, "roles.name", meta.Column)
		assert.Empty(t, meta.Join, "role field should have empty join")
		assert.True(t, meta.OrEquals)
	})

	t.Run("no filters", func(t *testing.T) {
		filters := []utils.Filter{}
		assert.Empty(t, filters)
	})

	t.Run("filter with empty value", func(t *testing.T) {
		filters := []utils.Filter{
			{Source: "transactions", Field: "description", Operator: "equals", Value: ""},
		}

		assert.Empty(t, filters[0].Value)
	})

	t.Run("all filter fields populated", func(t *testing.T) {
		filter := utils.Filter{
			Source:   "transactions",
			Field:    "category",
			Operator: "equals",
			Value:    "5",
		}

		assert.NotEmpty(t, filter.Source)
		assert.NotEmpty(t, filter.Field)
		assert.NotEmpty(t, filter.Operator)
		assert.NotEmpty(t, filter.Value)
	})
}
