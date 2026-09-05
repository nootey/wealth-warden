package models

type PeriodicJobDisplay struct {
	ID       string
	Kind     string
	Schedule string
	Queue    string
}

type JobIDsBody struct {
	IDs []int64 `json:"ids"`
}
