package models

type PaginationResponse struct {
	CurrentPage  int         `json:"current_page"`
	RowsPerPage  int         `json:"rows_per_page"`
	From         int         `json:"from"`
	To           int         `json:"to"`
	TotalRecords int         `json:"total_records"`
	Data         interface{} `json:"data"`
}

type MergeReq struct {
	SourceID      int64 `json:"source_id" validate:"required"`
	DestinationID int64 `json:"destination_id" validate:"required"`
}
