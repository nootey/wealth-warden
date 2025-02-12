package models

type PaginationResponse struct {
	CurrentPage  int         `json:"current_page"`
	RowsPerPage  int         `json:"rows_per_page"`
	From         int         `json:"from"`
	To           int         `json:"to"`
	TotalRecords int         `json:"total_records"`
	Data         interface{} `json:"data"`
}
