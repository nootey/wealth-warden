package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	defaultRowsPerPage = 10
	maxRowsPerPage     = 200
)

var ErrInvalidCursor = apperr.New(apperr.Invalid, "invalid page cursor")

type Paginator struct {
	RowsPerPage  int     `json:"rows_per_page"`
	TotalRecords int     `json:"total_records"`
	NextCursor   *string `json:"next_cursor"`
	PrevCursor   *string `json:"prev_cursor"`
}

type Filter struct {
	Source   string
	Field    string
	Operator string
	Value    string
}

type PaginationParams struct {
	Cursor      string
	RowsPerPage int
	SortField   string
	SortOrder   string
	Filters     []Filter
}

var sortDirection = map[bool]string{false: " ASC", true: " DESC"}

type Keyset struct {
	Joins   []string
	field   string
	desc    bool
	before  bool
	columns []string
	descs   []bool
	values  []any
	limit   int
}

type pageCursor struct {
	Field  string `json:"f"`
	Desc   bool   `json:"d,omitempty"`
	Before bool   `json:"b,omitempty"`
	Values []any  `json:"v"`
}

func GetPaginationParams(queryParams url.Values) PaginationParams {
	// Default values
	rowsPerPage := defaultRowsPerPage
	sortField := "created_at"
	sortOrder := "desc"
	var filters []Filter

	if rowsPerPageParam := queryParams.Get("rowsPerPage"); rowsPerPageParam != "" {
		if parsedRowsPerPage, err := strconv.Atoi(rowsPerPageParam); err == nil {
			rowsPerPage = parsedRowsPerPage
		}
	}
	if sortFieldParam := queryParams.Get("sort[field]"); sortFieldParam != "" {
		sortField = sortFieldParam
	}

	if sortOrderParam := queryParams.Get("sort[order]"); sortOrderParam != "" {
		switch sortOrderParam {
		case "asc", "1":
			sortOrder = "asc"
		case "desc", "-1":
			sortOrder = "desc"
		}
	}

	for i := 0; ; i++ {
		operator := queryParams.Get(fmt.Sprintf("filters[%d][operator]", i))
		field := queryParams.Get(fmt.Sprintf("filters[%d][field]", i))
		value := queryParams.Get(fmt.Sprintf("filters[%d][value]", i))
		source := queryParams.Get(fmt.Sprintf("filters[%d][source]", i))

		if operator == "" && field == "" && value == "" && source == "" {
			break
		}

		if operator != "" && field != "" && value != "" {
			filters = append(filters, Filter{
				Source:   source,
				Field:    field,
				Operator: operator,
				Value:    value,
			})
		}
	}

	return PaginationParams{
		Cursor:      queryParams.Get("cursor"),
		RowsPerPage: rowsPerPage,
		SortField:   sortField,
		SortOrder:   sortOrder,
		Filters:     filters,
	}
}

func NewKeyset[T any](spec models.SortSpec[T], p PaginationParams) (Keyset, error) {
	field, keys := sortKeys(spec, p.SortField)

	ks := Keyset{
		field: field,
		desc:  !strings.EqualFold(p.SortOrder, "asc"),
		limit: p.RowsPerPage,
	}
	if ks.limit < 1 {
		ks.limit = defaultRowsPerPage
	}
	ks.limit = min(ks.limit, maxRowsPerPage)

	if p.Cursor != "" {
		c, err := decodeCursor(p.Cursor)
		if err != nil || c.Field != field || c.Desc != ks.desc || len(c.Values) != len(keys) || c.Values[len(keys)-1] == nil {
			return Keyset{}, ErrInvalidCursor
		}
		ks.before, ks.values = c.Before, c.Values
	}

	if meta, ok := resolveMeta(spec.Source, field); ok && meta.Join != "" {
		ks.Joins = []string{meta.Join}
	}

	for i, k := range keys {
		// Walking back a page reads the same order reversed, then flips the rows.
		desc := i >= len(spec.Fixed) && ks.desc
		ks.columns = append(ks.columns, k.Column)
		ks.descs = append(ks.descs, desc != ks.before)
	}

	return ks, nil
}

func ApplyKeyset(query *gorm.DB, ks Keyset) *gorm.DB {
	if ks.values != nil {
		where, args := ks.pastCursor()
		query = query.Where(where, args...)
	}

	order := make([]string, len(ks.columns))
	for i, col := range ks.columns {
		order[i] = col + sortDirection[ks.descs[i]]
	}

	// One row past the page tells Paginate whether another page follows.
	return query.Order(strings.Join(order, ", ")).Limit(ks.limit + 1)
}

// Only the column types know which cursor values are valid input, enums included, so a
// Postgres data error is how a tampered cursor shows.
func KeysetError(ks Keyset, err error) error {
	var pgErr *pgconn.PgError
	if ks.values != nil && errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "22") {
		return fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}
	return err
}

func Paginate[T any](records []T, ks Keyset, spec models.SortSpec[T], total int64) ([]T, *Paginator) {
	more := len(records) > ks.limit
	if more {
		records = records[:ks.limit]
	}

	hasNext, hasPrev := more, ks.values != nil
	if ks.before {
		slices.Reverse(records)
		hasNext, hasPrev = true, more
	}

	paginator := &Paginator{RowsPerPage: ks.limit, TotalRecords: int(total)}
	if len(records) == 0 {
		return records, paginator
	}

	_, keys := sortKeys(spec, ks.field)
	if hasNext {
		paginator.NextCursor = encodeCursor(ks, false, keys, records[len(records)-1])
	}
	if hasPrev {
		paginator.PrevCursor = encodeCursor(ks, true, keys, records[0])
	}

	return records, paginator
}

func sortKeys[T any](spec models.SortSpec[T], field string) (string, []models.SortKey[T]) {
	key, ok := spec.Fields[field]
	if !ok && field != "id" {
		field = spec.Default
		key, ok = spec.Fields[field]
	}

	keys := slices.Clone(spec.Fixed)
	if ok {
		keys = append(keys, key)
	}

	return field, append(keys, spec.ID)
}

// NULLs sort the Postgres default way, last ascending and first descending, so reversing
// every key reverses the rows.
func (ks Keyset) pastCursor() (string, []any) {
	last := len(ks.columns) - 1
	where, args := keyPast(ks.columns[last], ks.values[last], ks.descs[last], false)

	for i := last - 1; i >= 0; i-- {
		eq, eqArgs := keyEqual(ks.columns[i], ks.values[i])
		where, args = eq+" AND ("+where+")", slices.Concat(eqArgs, args)

		if past, pastArgs := keyPast(ks.columns[i], ks.values[i], ks.descs[i], true); past != "" {
			where, args = past+" OR ("+where+")", slices.Concat(pastArgs, args)
		}
	}

	// The OR chain alone hides the range from the planner; a plain bound lets it scan an index.
	if v := ks.values[0]; v != nil && ks.descs[0] && last > 0 {
		where, args = "("+ks.columns[0]+") <= ? AND ("+where+")", slices.Concat([]any{v}, args)
	}

	return "(" + where + ")", args
}

func keyPast(col string, v any, desc, nullable bool) (string, []any) {
	col = "(" + col + ")"
	switch {
	case v == nil && desc:
		return col + " IS NOT NULL", nil
	case v == nil:
		return "", nil
	case desc:
		return col + " < ?", []any{v}
	case nullable:
		return "(" + col + " > ? OR " + col + " IS NULL)", []any{v}
	default:
		return col + " > ?", []any{v}
	}
}

func keyEqual(col string, v any) (string, []any) {
	col = "(" + col + ")"
	if v == nil {
		return col + " IS NULL", nil
	}
	return col + " = ?", []any{v}
}

func encodeCursor[T any](ks Keyset, before bool, keys []models.SortKey[T], row T) *string {
	values := make([]any, len(keys))
	for i, k := range keys {
		values[i] = cursorValue(k.Value(row))
	}

	raw, _ := json.Marshal(pageCursor{Field: ks.field, Desc: ks.desc, Before: before, Values: values})
	cursor := base64.RawURLEncoding.EncodeToString(raw)
	return &cursor
}

func decodeCursor(s string) (pageCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return pageCursor{}, err
	}

	var c pageCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return pageCursor{}, err
	}
	for _, v := range c.Values {
		if _, ok := v.(string); !ok && v != nil {
			return pageCursor{}, ErrInvalidCursor
		}
	}

	return c, nil
}

// cursorValue renders a sort value as Postgres text input. pgx sends string parameters in
// text format, so the server parses each one as the type of the column it is compared to.
func cursorValue(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch x := rv.Interface().(type) {
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case decimal.Decimal:
		return x.String()
	}

	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	default:
		return fmt.Sprint(rv.Interface())
	}
}
