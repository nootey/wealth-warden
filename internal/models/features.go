package models

import "time"

type Feature struct {
	ID          uint
	Name        string
	Description string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
