package models

type PaginationResponse struct {
	RowsPerPage  int         `json:"rows_per_page"`
	TotalRecords int         `json:"total_records"`
	NextCursor   *string     `json:"next_cursor"`
	PrevCursor   *string     `json:"prev_cursor"`
	Data         interface{} `json:"data"`
}

// Value must return what the query sorts on for the row, read from the loaded record, or
// nil where the column is NULL.
type SortKey[T any] struct {
	Column string
	Value  func(T) any
}

// Fixed keys always lead the order and sort ascending; ID breaks ties and must be unique.
// Joins for sort fields come from the filter field map under Source.
type SortSpec[T any] struct {
	Source  string
	Fixed   []SortKey[T]
	Fields  map[string]SortKey[T]
	ID      SortKey[T]
	Default string
}

type MergeReq struct {
	SourceID      int64 `json:"source_id" validate:"required"`
	DestinationID int64 `json:"destination_id" validate:"required"`
}
