package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/riverqueue/river/rivertype"
)

type jobIDsBody struct {
	IDs []int64 `json:"ids"`
}
type JobAdminHandler struct {
	service services.JobAdminServiceInterface
}

func NewJobAdminHandler(service services.JobAdminServiceInterface) *JobAdminHandler {
	return &JobAdminHandler{service: service}
}

func (h *JobAdminHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("", authz.RequireAllMW("access_backoffice"), h.ListJobs)
	ap.GET("/counts", authz.RequireAllMW("access_backoffice"), h.JobCounts)
	ap.GET("/queues", authz.RequireAllMW("access_backoffice"), h.ListQueues)
	ap.GET("/periodic", authz.RequireAllMW("access_backoffice"), h.ListPeriodic)
	ap.GET("/:id", authz.RequireAllMW("access_backoffice"), h.GetJob)

	ap.POST("/retry", authz.RequireAllMW("manage_jobs"), h.RetryJobs)
	ap.POST("/cancel", authz.RequireAllMW("manage_jobs"), h.CancelJobs)
	ap.POST("/delete", authz.RequireAllMW("manage_jobs"), h.DeleteJobs)
	ap.POST("/queues/:name/pause", authz.RequireAllMW("manage_jobs"), h.PauseQueue)
	ap.POST("/queues/:name/resume", authz.RequireAllMW("manage_jobs"), h.ResumeQueue)
}

func (h *JobAdminHandler) ListJobs(c *gin.Context) {
	qp := c.Request.URL.Query()
	ctx := c.Request.Context()

	params := services.JobQueryParams{
		Pagination: utils.GetPaginationParams(qp),
		States:     utils.ParseStates(qp["state"]),
	}

	rows, paginator, err := h.service.FetchJobs(ctx, params)
	if err != nil {
		if errors.Is(err, services.ErrInvalidJobState) {
			utils.ErrorMessage(c, "Invalid filter", err.Error(), http.StatusBadRequest, err)
			return
		}
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          rows,
	})
}

func (h *JobAdminHandler) JobCounts(c *gin.Context) {
	counts, err := h.service.FetchJobCounts(c.Request.Context())
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": counts})
}

func (h *JobAdminHandler) GetJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorMessage(c, "Invalid id", "id must be a valid integer", http.StatusBadRequest, err)
		return
	}

	job, err := h.service.FetchJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, rivertype.ErrNotFound) {
			utils.ErrorMessage(c, "Not found", "job not found", http.StatusNotFound, err)
			return
		}
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job})
}

func (h *JobAdminHandler) ListQueues(c *gin.Context) {
	queues, err := h.service.FetchQueues(c.Request.Context())
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": queues})
}

func (h *JobAdminHandler) ListPeriodic(c *gin.Context) {
	jobs, err := h.service.FetchPeriodicJobs(c.Request.Context())
	if err != nil {
		utils.ErrorMessage(c, "Fetch error", err.Error(), http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *JobAdminHandler) RetryJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs queued for retry", h.service.RetryJobs)
}

func (h *JobAdminHandler) CancelJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs cancelled", h.service.CancelJobs)
}

func (h *JobAdminHandler) DeleteJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs deleted", h.service.DeleteJobs)
}

func (h *JobAdminHandler) runJobAction(c *gin.Context, okMsg string, action func(context.Context, []int64) error) {
	var body jobIDsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorMessage(c, "Invalid request", "body must be {\"ids\": [...]}", http.StatusBadRequest, err)
		return
	}
	if len(body.IDs) == 0 {
		err := errors.New("no job ids provided")
		utils.ErrorMessage(c, "Invalid request", err.Error(), http.StatusBadRequest, err)
		return
	}

	if err := action(c.Request.Context(), body.IDs); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, rivertype.ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, rivertype.ErrJobRunning) {
			status = http.StatusConflict
		}
		utils.ErrorMessage(c, "Action failed", err.Error(), status, err)
		return
	}

	utils.SuccessMessage(c, okMsg, "Success", http.StatusOK)
}

func (h *JobAdminHandler) PauseQueue(c *gin.Context) {
	if err := h.service.PauseQueue(c.Request.Context(), c.Param("name")); err != nil {
		utils.ErrorMessage(c, "Action failed", err.Error(), http.StatusInternalServerError, err)
		return
	}
	utils.SuccessMessage(c, "Queue paused", "Success", http.StatusOK)
}

func (h *JobAdminHandler) ResumeQueue(c *gin.Context) {
	if err := h.service.ResumeQueue(c.Request.Context(), c.Param("name")); err != nil {
		utils.ErrorMessage(c, "Action failed", err.Error(), http.StatusInternalServerError, err)
		return
	}
	utils.SuccessMessage(c, "Queue resumed", "Success", http.StatusOK)
}
