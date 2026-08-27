package models

type ZeroCostMigrationResult struct {
	TotalProcessed  int `json:"total_processed"`
	AssetsProcessed int `json:"assets_processed"`
	AssetsFailed    int `json:"assets_failed"`
}
