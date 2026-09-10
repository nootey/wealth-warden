package handlers

import (
	"net/http"
	"strconv"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
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

func (h *NotesHandler) Routes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), h.GetNotesPaginated)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), h.GetNoteByID)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), h.InsertNote)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_data"), h.UpdateNote)
	apiGroup.POST(":id/resolve", authz.RequireAllMW("manage_data"), h.ToggleResolveState)
	apiGroup.DELETE(":id", authz.RequireAllMW("manage_data"), h.DeleteNote)
}

func (h *NotesHandler) GetNotesPaginated(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	qp := c.Request.URL.Query()
	p := utils.GetPaginationParams(qp)

	records, paginator, err := h.service.FetchNotesPaginated(ctx, userID, p)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          records,
	}

	c.JSON(http.StatusOK, response)
}

func (h *NotesHandler) GetNoteByID(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	record, err := h.service.FetchNoteByID(ctx, userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *NotesHandler) InsertNote(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	var record *models.NoteReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err := h.service.InsertNote(ctx, userID, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Note created", "Success", http.StatusOK)
}

func (h *NotesHandler) UpdateNote(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	var record *models.NoteReq

	if err := c.ShouldBindJSON(&record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "Invalid JSON", err))
		return
	}

	if err := h.v.ValidateStruct(record); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Validation, err.Error(), err))
		return
	}

	_, err = h.service.UpdateNote(ctx, userID, id, record)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Note updated", "Success", http.StatusOK)
}

func (h *NotesHandler) ToggleResolveState(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	if err := h.service.ToggleResolveState(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Note resolve state toggled", "Success", http.StatusOK)
}

func (h *NotesHandler) DeleteNote(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetInt64("user_id")

	idStr := c.Param("id")

	if idStr == "" {
		_ = c.Error(apperr.New(apperr.Invalid, "invalid id provided"))
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	if err := h.service.DeleteNote(ctx, userID, id); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Note deleted", "Success", http.StatusOK)
}
