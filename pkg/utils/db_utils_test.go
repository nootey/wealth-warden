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
