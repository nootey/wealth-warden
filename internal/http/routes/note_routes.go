package routes

import (
	"wealth-warden/internal/http/handlers"
	"wealth-warden/pkg/authz"

	"github.com/gin-gonic/gin"
)

func NoteRoutes(apiGroup *gin.RouterGroup, handler *handlers.NotesHandler) {
	apiGroup.GET("", authz.RequireAllMW("view_data"), handler.GetNotesPaginated)
	apiGroup.GET("/:id", authz.RequireAllMW("view_data"), handler.GetNoteByID)
	apiGroup.PUT("", authz.RequireAllMW("manage_data"), handler.InsertNote)
	apiGroup.PUT(":id", authz.RequireAllMW("manage_data"), handler.UpdateNote)
	apiGroup.POST(":id/resolve", authz.RequireAllMW("manage_data"), handler.ToggleResolveState)
	apiGroup.DELETE(":id", authz.RequireAllMW("manage_data"), handler.DeleteNote)
}
