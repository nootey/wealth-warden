package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/utils"
	"wealth-warden/pkg/validators"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	Service *services.ImportService
	v       *validators.GoValidator
}

func NewImportHandler(
	service *services.ImportService,
	v *validators.GoValidator,
) *ImportHandler {
	return &ImportHandler{
		Service: service,
		v:       v,
	}
}

func (h *ImportHandler) GetImportsByImportType(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	importType := c.Param("import_type")

	records, err := h.Service.FetchImportsByImportType(userID, importType)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *ImportHandler) ValidateCustomImport(c *gin.Context) {
	var payload models.CustomImportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorMessage(c, "Invalid Request", "Invalid JSON format", http.StatusBadRequest, err)
		return
	}

	categories, apiErr := h.Service.ValidateCustomImport(&payload)
	if apiErr != nil {
		utils.ErrorMessage(c, "Error occurred", apiErr.Error(), http.StatusInternalServerError, nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"year":       payload.Year,
		"count":      len(payload.Txns),
		"sample":     payload.Txns[0],
		"categories": categories, // simple map[string][]string
	})
}

func (h *ImportHandler) ImportFromJSON(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorMessage(c, "Unauthorized", err.Error(), http.StatusUnauthorized, err)
		return
	}

	checkAccIDStr := c.Query("check_acc_id")
	if checkAccIDStr == "" {
		utils.ErrorMessage(c, "param error", "missing account ids", http.StatusBadRequest, nil)
		return
	}

	checkAccID, err := strconv.ParseInt(checkAccIDStr, 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Error occurred", "check acc id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	var payload models.CustomImportPayload

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	ct := c.GetHeader("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			utils.ErrorMessage(c, "Invalid upload", "file is required", http.StatusBadRequest, err)
			return
		}

		f, err := fileHeader.Open()
		if err != nil {
			utils.ErrorMessage(c, "Invalid upload", "cannot open uploaded file", http.StatusBadRequest, err)
			return
		}
		defer f.Close()

		dec := json.NewDecoder(f)
		if err := dec.Decode(&payload); err != nil {
			utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
			return
		}
	} else {
		if err := c.ShouldBindJSON(&payload); err != nil {
			utils.ErrorMessage(c, "Invalid JSON", err.Error(), http.StatusBadRequest, err)
			return
		}
	}

	if err := h.Service.ImportFromJSON(userID, checkAccID, payload); err != nil {
		utils.ErrorMessage(c, "Create error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	utils.SuccessMessage(c, "JSON import successful", "Success", http.StatusOK)
}
