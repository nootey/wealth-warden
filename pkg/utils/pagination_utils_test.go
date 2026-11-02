package utils_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGetPaginationParams(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		params := url.Values{}

		result := utils.GetPaginationParams(params)

		assert.Empty(t, result.Cursor)
		assert.Equal(t, 10, result.RowsPerPage)
		assert.Equal(t, "created_at", result.SortField)
		assert.Equal(t, "desc", result.SortOrder)
		assert.Empty(t, result.Filters)
	})

	t.Run("custom cursor and rows per page", func(t *testing.T) {
		params := url.Values{
			"cursor":      []string{"abc"},
			"rowsPerPage": []string{"25"},
		}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, "abc", result.Cursor)
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

	t.Run("invalid rows per page uses default", func(t *testing.T) {
		params := url.Values{"rowsPerPage": []string{"invalid"}}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, 10, result.RowsPerPage)
	})

	t.Run("all parameters combined", func(t *testing.T) {
		params := url.Values{
			"cursor":               []string{"abc"},
			"rowsPerPage":          []string{"50"},
			"sort[field]":          []string{"amount"},
			"sort[order]":          []string{"asc"},
			"filters[0][source]":   []string{"transactions"},
			"filters[0][field]":    []string{"category"},
			"filters[0][operator]": []string{"equals"},
			"filters[0][value]":    []string{"food"},
		}

		result := utils.GetPaginationParams(params)

		assert.Equal(t, "abc", result.Cursor)
		assert.Equal(t, 50, result.RowsPerPage)
		assert.Equal(t, "amount", result.SortField)
		assert.Equal(t, "asc", result.SortOrder)
		assert.Len(t, result.Filters, 1)
	})
}

type pageRow struct {
	ID   int64
	Name *string
	At   time.Time
}

var pageRowSort = models.SortSpec[pageRow]{
	Fields: map[string]models.SortKey[pageRow]{
		"at":   {Column: "page_rows.at", Value: func(r pageRow) any { return r.At }},
		"name": {Column: "owners.name", Value: func(r pageRow) any { return r.Name }},
	},
	ID:      models.SortKey[pageRow]{Column: "page_rows.id", Value: func(r pageRow) any { return r.ID }},
	Default: "at",
}

func keysetSQL(t *testing.T, ks utils.Keyset) (string, []any) {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	require.NoError(t, err)

	stmt := utils.ApplyKeyset(db.Table("page_rows"), ks).Find(&[]map[string]any{}).Statement
	return stmt.SQL.String(), stmt.Vars
}

func encodeTestCursor(t *testing.T, c map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestKeysetPagination(t *testing.T) {
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	firstPage := utils.PaginationParams{RowsPerPage: 2, SortField: "at", SortOrder: "desc"}

	t.Run("first page orders by the sort field then id, with no bound", func(t *testing.T) {
		ks, err := utils.NewKeyset(pageRowSort, firstPage)
		require.NoError(t, err)

		sql, vars := keysetSQL(t, ks)

		assert.NotContains(t, sql, "WHERE")
		assert.Contains(t, sql, "ORDER BY page_rows.at DESC, page_rows.id DESC LIMIT $1")
		assert.Equal(t, []any{3}, vars, "one row past the page tells whether another page follows")
	})

	t.Run("unknown sort field falls back to the default", func(t *testing.T) {
		ks, err := utils.NewKeyset(pageRowSort, utils.PaginationParams{SortField: "at; DROP TABLE users", SortOrder: "desc"})
		require.NoError(t, err)

		sql, _ := keysetSQL(t, ks)

		assert.Contains(t, sql, "ORDER BY page_rows.at DESC, page_rows.id DESC")
		assert.NotContains(t, sql, "DROP")
	})

	t.Run("id sorts by the tiebreaker alone", func(t *testing.T) {
		ks, err := utils.NewKeyset(pageRowSort, utils.PaginationParams{SortField: "id", SortOrder: "asc"})
		require.NoError(t, err)

		sql, _ := keysetSQL(t, ks)

		assert.Contains(t, sql, "ORDER BY page_rows.id ASC LIMIT")
		assert.Empty(t, ks.Joins)
	})

	t.Run("joined sort field takes its join from the field map", func(t *testing.T) {
		ks, err := utils.NewKeyset(models.TransactionSort, utils.PaginationParams{SortField: "category"})
		require.NoError(t, err)

		assert.Equal(t, []string{utils.FieldMap["transactions"]["category"].Join}, ks.Joins)
	})

	t.Run("rows per page is clamped", func(t *testing.T) {
		for rows, want := range map[int]int{0: 10, -5: 10, 25: 25, 1000: 200} {
			ks, err := utils.NewKeyset(pageRowSort, utils.PaginationParams{RowsPerPage: rows})
			require.NoError(t, err)

			_, paginator := utils.Paginate([]pageRow{}, ks, pageRowSort, 0)
			assert.Equal(t, want, paginator.RowsPerPage, "rows %d", rows)
		}
	})

	t.Run("walks forward and back across pages", func(t *testing.T) {
		ks, err := utils.NewKeyset(pageRowSort, firstPage)
		require.NoError(t, err)

		page, paginator := utils.Paginate([]pageRow{{ID: 5, At: at}, {ID: 4, At: at}, {ID: 3, At: at.Add(-time.Hour)}}, ks, pageRowSort, 3)
		assert.Equal(t, []int64{5, 4}, []int64{page[0].ID, page[1].ID})
		assert.Equal(t, 3, paginator.TotalRecords)
		assert.Nil(t, paginator.PrevCursor)
		require.NotNil(t, paginator.NextCursor)

		next := firstPage
		next.Cursor = *paginator.NextCursor
		ks, err = utils.NewKeyset(pageRowSort, next)
		require.NoError(t, err)

		sql, vars := keysetSQL(t, ks)
		assert.Contains(t, sql, "WHERE ((page_rows.at) <= $1 AND ((page_rows.at) < $2 OR ((page_rows.at) = $3 AND ((page_rows.id) < $4))))")
		assert.Contains(t, sql, "ORDER BY page_rows.at DESC, page_rows.id DESC")
		assert.Equal(t, []any{"2026-03-01T12:00:00Z", "2026-03-01T12:00:00Z", "2026-03-01T12:00:00Z", "4"}, vars[:4])

		page, paginator = utils.Paginate([]pageRow{{ID: 3, At: at.Add(-time.Hour)}}, ks, pageRowSort, 3)
		assert.Len(t, page, 1)
		assert.Nil(t, paginator.NextCursor)
		require.NotNil(t, paginator.PrevCursor)

		prev := firstPage
		prev.Cursor = *paginator.PrevCursor
		ks, err = utils.NewKeyset(pageRowSort, prev)
		require.NoError(t, err)

		sql, vars = keysetSQL(t, ks)
		assert.Contains(t, sql, "WHERE (((page_rows.at) > $1 OR (page_rows.at) IS NULL) OR ((page_rows.at) = $2 AND ((page_rows.id) > $3)))")
		assert.Contains(t, sql, "ORDER BY page_rows.at ASC, page_rows.id ASC")
		assert.Equal(t, []any{"2026-03-01T11:00:00Z", "2026-03-01T11:00:00Z", "3"}, vars[:3])

		// Rows come back nearest first when reading backwards.
		page, paginator = utils.Paginate([]pageRow{{ID: 4, At: at}, {ID: 5, At: at}}, ks, pageRowSort, 3)
		assert.Equal(t, []int64{5, 4}, []int64{page[0].ID, page[1].ID})
		assert.Nil(t, paginator.PrevCursor)
		assert.NotNil(t, paginator.NextCursor)
	})

	t.Run("a NULL cursor value pages through the NULL group", func(t *testing.T) {
		name := "a"
		rows := []pageRow{{ID: 1, Name: &name}, {ID: 2}, {ID: 3}}

		for order, where := range map[string]string{
			"asc":  "WHERE ((owners.name) IS NULL AND ((page_rows.id) > $1))",
			"desc": "WHERE ((owners.name) IS NOT NULL OR ((owners.name) IS NULL AND ((page_rows.id) < $1)))",
		} {
			p := utils.PaginationParams{RowsPerPage: 2, SortField: "name", SortOrder: order}
			ks, err := utils.NewKeyset(pageRowSort, p)
			require.NoError(t, err)

			_, paginator := utils.Paginate(rows, ks, pageRowSort, 3)
			require.NotNil(t, paginator.NextCursor)

			p.Cursor = *paginator.NextCursor
			ks, err = utils.NewKeyset(pageRowSort, p)
			require.NoError(t, err)

			sql, vars := keysetSQL(t, ks)
			assert.Contains(t, sql, where, order)
			assert.Equal(t, "2", vars[0], order)
		}
	})

	t.Run("rejects cursors that do not match the request", func(t *testing.T) {
		ks, err := utils.NewKeyset(pageRowSort, firstPage)
		require.NoError(t, err)
		_, paginator := utils.Paginate([]pageRow{{ID: 2, At: at}, {ID: 1, At: at}, {ID: 0, At: at}}, ks, pageRowSort, 3)
		require.NotNil(t, paginator.NextCursor)
		valid := *paginator.NextCursor

		cases := map[string]utils.PaginationParams{
			"other sort field": {SortField: "name", SortOrder: "desc", Cursor: valid},
			"other sort order": {SortField: "at", SortOrder: "asc", Cursor: valid},
			"not base64":       {SortField: "at", SortOrder: "desc", Cursor: "not a cursor!"},
			"not json":         {SortField: "at", SortOrder: "desc", Cursor: base64.RawURLEncoding.EncodeToString([]byte("{"))},
			"non-text value":   {SortField: "at", SortOrder: "desc", Cursor: encodeTestCursor(t, map[string]any{"f": "at", "d": true, "v": []any{1, 2}})},
			"missing id":       {SortField: "at", SortOrder: "desc", Cursor: encodeTestCursor(t, map[string]any{"f": "at", "d": true, "v": []any{"x", nil}})},
			"too few values":   {SortField: "at", SortOrder: "desc", Cursor: encodeTestCursor(t, map[string]any{"f": "at", "d": true, "v": []any{"1"}})},
		}
		for name, p := range cases {
			_, err := utils.NewKeyset(pageRowSort, p)
			assert.ErrorIs(t, err, utils.ErrInvalidCursor, name)
		}
	})
}

func TestKeysetError(t *testing.T) {
	badInput := &pgconn.PgError{Code: "22P02"}
	resumed, err := utils.NewKeyset(pageRowSort, utils.PaginationParams{
		SortField: "at",
		SortOrder: "desc",
		Cursor:    encodeTestCursor(t, map[string]any{"f": "at", "d": true, "v": []any{"not a date", "1"}}),
	})
	require.NoError(t, err)
	firstPage, err := utils.NewKeyset(pageRowSort, utils.PaginationParams{})
	require.NoError(t, err)

	t.Run("bad input past a cursor is an invalid cursor", func(t *testing.T) {
		err := utils.KeysetError(resumed, badInput)

		assert.ErrorIs(t, err, utils.ErrInvalidCursor)
		assert.ErrorIs(t, err, badInput)
		status, message := apperr.Resolve(err)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid page cursor", message)
	})

	t.Run("other errors pass through", func(t *testing.T) {
		other := &pgconn.PgError{Code: "40001"}

		assert.Same(t, other, utils.KeysetError(resumed, other))
		assert.Same(t, badInput, utils.KeysetError(firstPage, badInput))
	})
}
