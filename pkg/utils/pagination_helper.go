package utils

import (
	"net/url"
	"strconv"
)

type PaginationParams struct {
	PageNumber  int
	RowsPerPage int
	SortField   string
	SortOrder   string
}

func GetPaginationParams(queryParams url.Values) PaginationParams {

	// Default values
	pageNumber := 1
	rowsPerPage := 10
	sortField := "created_at"
	sortOrder := "desc"
	
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

	if sortFieldParam := queryParams.Get("sort_field"); sortFieldParam != "" {
		sortField = sortFieldParam
	}

	if sortOrderParam := queryParams.Get("sort_order"); sortOrderParam != "" {
		if sortOrderParam == "asc" || sortOrderParam == "desc" {
			sortOrder = sortOrderParam
		}
	}

	return PaginationParams{
		PageNumber:  pageNumber,
		RowsPerPage: rowsPerPage,
		SortField:   sortField,
		SortOrder:   sortOrder,
	}
}
