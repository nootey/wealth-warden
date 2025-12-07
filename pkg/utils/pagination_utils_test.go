package utils_test

import (
	"net/url"
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestGetPaginationParams(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		params := url.Values{}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 1, result.PageNumber)
		assert.Equal(t, 10, result.RowsPerPage)
		assert.Equal(t, "created_at", result.SortField)
		assert.Equal(t, "desc", result.SortOrder)
		assert.Empty(t, result.Filters)
	})

	t.Run("custom page and rows per page", func(t *testing.T) {
		params := url.Values{
			"page":        []string{"3"},
			"rowsPerPage": []string{"25"},
		}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 3, result.PageNumber)
		assert.Equal(t, 25, result.RowsPerPage)
	})

	t.Run("custom sort field and order", func(t *testing.T) {
		params := url.Values{
			"sort[field]": []string{"name"},
			"sort[order]": []string{"asc"},
		}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, "name", result.SortField)
		assert.Equal(t, "asc", result.SortOrder)
	})

	t.Run("sort order variations", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"asc", "asc"},
			{"1", "asc"},
			{"desc", "desc"},
			{"-1", "desc"},
		}

		for _, tc := range testCases {
			params := url.Values{"sort[order]": []string{tc.input}}
			result := utils.GetPaginationParams(params)
			assert.Equal(t, tc.expected, result.SortOrder)
		}
	})

	t.Run("single filter", func(t *testing.T) {
		params := url.Values{
			"filters[0][source]":   []string{"transactions"},
			"filters[0][field]":    []string{"status"},
			"filters[0][operator]": []string{"equals"},
			"filters[0][value]":    []string{"completed"},
		}

		result := utils.GetPaginationParams(params)

		assert.Len(t, result.Filters, 1)
		assert.Equal(t, "transactions", result.Filters[0].Source)
		assert.Equal(t, "status", result.Filters[0].Field)
		assert.Equal(t, "equals", result.Filters[0].Operator)
		assert.Equal(t, "completed", result.Filters[0].Value)
	})

	t.Run("multiple filters", func(t *testing.T) {
		params := url.Values{
			"filters[0][source]":   []string{"transactions"},
			"filters[0][field]":    []string{"status"},
			"filters[0][operator]": []string{"equals"},
			"filters[0][value]":    []string{"completed"},
			"filters[1][source]":   []string{"transactions"},
			"filters[1][field]":    []string{"amount"},
			"filters[1][operator]": []string{">"},
			"filters[1][value]":    []string{"100"},
		}

		result := utils.GetPaginationParams(params)

		assert.Len(t, result.Filters, 2)
		assert.Equal(t, "status", result.Filters[0].Field)
		assert.Equal(t, "amount", result.Filters[1].Field)
	})

	t.Run("ignores incomplete filters", func(t *testing.T) {
		params := url.Values{
			"filters[0][field]":    []string{"status"},
			"filters[0][operator]": []string{"equals"},
			// missing value
		}

		result := utils.GetPaginationParams(params)

		assert.Empty(t, result.Filters)
	})

	t.Run("invalid page number uses default", func(t *testing.T) {
		params := url.Values{"page": []string{"invalid"}}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 1, result.PageNumber)
	})

	t.Run("invalid rows per page uses default", func(t *testing.T) {
		params := url.Values{"rowsPerPage": []string{"invalid"}}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 10, result.RowsPerPage)
	})

	t.Run("all parameters combined", func(t *testing.T) {
		params := url.Values{
			"page":                 []string{"2"},
			"rowsPerPage":          []string{"50"},
			"sort[field]":          []string{"amount"},
			"sort[order]":          []string{"asc"},
			"filters[0][source]":   []string{"transactions"},
			"filters[0][field]":    []string{"category"},
			"filters[0][operator]": []string{"equals"},
			"filters[0][value]":    []string{"food"},
		}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 2, result.PageNumber)
		assert.Equal(t, 50, result.RowsPerPage)
		assert.Equal(t, "amount", result.SortField)
		assert.Equal(t, "asc", result.SortOrder)
		assert.Len(t, result.Filters, 1)
	})
}
