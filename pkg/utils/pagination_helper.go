package utils

import (
	"fmt"
	"net/url"
	"strconv"
)

type Paginator struct {
	CurrentPage  int `json:"current_page"`
	RowsPerPage  int `json:"rows_per_page"`
	TotalRecords int `json:"total_records"`
	From         int `json:"from"`
	To           int `json:"to"`
}

type Filter struct {
	Source   string
	Field    string
	Operator string
	Value    string
}

type PaginationParams struct {
	PageNumber  int
	RowsPerPage int
	SortField   string
	SortOrder   string
	Filters     []Filter
}

func GetPaginationParams(queryParams url.Values) PaginationParams {

	// Default values
	pageNumber := 1
	rowsPerPage := 10
	sortField := "created_at"
	sortOrder := "desc"
	var filters []Filter

	if pageParam := queryParams.Get("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil {
			pageNumber = parsedPage
		}
	}

	if rowsPerPageParam := queryParams.Get("rowsPerPage"); rowsPerPageParam != "" {
		if parsedRowsPerPage, err := strconv.Atoi(rowsPerPageParam); err == nil {
			rowsPerPage = parsedRowsPerPage
		}
	}
	if sortFieldParam := queryParams.Get("sort[field]"); sortFieldParam != "" {
		sortField = sortFieldParam
	}

	if sortOrderParam := queryParams.Get("sort[order]"); sortOrderParam != "" {
		if sortOrderParam == "asc" || sortOrderParam == "desc" {
			sortOrder = sortOrderParam
		} else if sortOrderParam == "1" {
			sortOrder = "asc"
		} else if sortOrderParam == "-1" {
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
		PageNumber:  pageNumber,
		RowsPerPage: rowsPerPage,
		SortField:   sortField,
		SortOrder:   sortOrder,
		Filters:     filters,
	}
}
