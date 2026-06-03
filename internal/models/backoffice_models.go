package models

type ZeroCostMigrationError struct {
	AssetID int64  `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}

type ZeroCostMigrationResult struct {
	TotalProcessed  int                      `json:"total_processed"`
	AssetsProcessed int                      `json:"assets_processed"`
	AssetsFailed    int                      `json:"assets_failed"`
	Errors          []ZeroCostMigrationError `json:"errors,omitempty"`
}
