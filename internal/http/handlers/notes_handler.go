package handlers

import (
	"wealth-warden/internal/services"
	"wealth-warden/pkg/validators"
)

type NotesHandler struct {
	service services.NotesServiceInterface
	v       validators.Validator
}

func NewNotesHandler(
	service services.NotesServiceInterface,
	v validators.Validator,
) *NotesHandler {
	return &NotesHandler{
		service: service,
		v:       v,
	}
}
