package models

import "time"

type Note struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64      `gorm:"not null;index:idx_notes_user_id" json:"user_id"`
	Content    string     `gorm:"type:text;not null" json:"content"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime;not null;index:idx_notes_created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime;not null" json:"updated_at"`
}

var NoteSort = SortSpec[Note]{
	// Unresolved notes list first.
	Fixed: []SortKey[Note]{
		{Column: "notes.resolved_at IS NOT NULL", Value: func(n Note) any { return n.ResolvedAt != nil }},
	},
	Fields: map[string]SortKey[Note]{
		"created_at": {Column: "notes.created_at", Value: func(n Note) any { return n.CreatedAt }},
	},
	ID:      SortKey[Note]{Column: "notes.id", Value: func(n Note) any { return n.ID }},
	Default: "created_at",
}

type NoteReq struct {
	Content string `json:"content" validate:"required,min=1,max=5000"`
}
