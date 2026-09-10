package handlers

import (
	"context"
	"net/http"
	"strconv"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/utils"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	service services.JobServiceInterface
}

func NewJobHandler(service services.JobServiceInterface) *JobHandler {
	return &JobHandler{service: service}
}

func (h *JobHandler) Routes(ap *gin.RouterGroup) {
	ap.GET("/admin", authz.RequireAllMW("access_backoffice"), h.ListJobs)
	ap.GET("/admin/counts", authz.RequireAllMW("access_backoffice"), h.JobCounts)
	ap.GET("/admin/queues", authz.RequireAllMW("access_backoffice"), h.ListQueues)
	ap.GET("/admin/periodic", authz.RequireAllMW("access_backoffice"), h.ListPeriodic)
	ap.GET("/admin/:id", authz.RequireAllMW("access_backoffice"), h.GetJob)

	ap.POST("/admin/retry", authz.RequireAllMW("manage_jobs"), h.RetryJobs)
	ap.POST("/admin/cancel", authz.RequireAllMW("manage_jobs"), h.CancelJobs)
	ap.POST("/admin/delete", authz.RequireAllMW("manage_jobs"), h.DeleteJobs)
	ap.POST("/admin/queues/:name/pause", authz.RequireAllMW("manage_jobs"), h.PauseQueue)
	ap.POST("/admin/queues/:name/resume", authz.RequireAllMW("manage_jobs"), h.ResumeQueue)

	ap.GET("/user", authz.RequireAllMW("manage_data"), h.ListUserJobs)
	ap.POST("/user/:id/retry", authz.RequireAllMW("manage_data"), h.RetryUserJob)
	ap.POST("/user/:id/cancel", authz.RequireAllMW("manage_data"), h.CancelUserJob)
}

func (h *JobHandler) ListJobs(c *gin.Context) {
	qp := c.Request.URL.Query()
	ctx := c.Request.Context()

	params := services.JobQueryParams{
		Pagination: utils.GetPaginationParams(qp),
		States:     utils.ParseStates(qp["state"]),
	}

	rows, paginator, err := h.service.FetchJobs(ctx, params)
	if err != nil {
		_ = c.Error(err)
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

func (h *JobHandler) JobCounts(c *gin.Context) {
	counts, err := h.service.FetchJobCounts(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": counts})
}

func (h *JobHandler) GetJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	job, err := h.service.FetchJob(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": job})
}

func (h *JobHandler) ListQueues(c *gin.Context) {
	queues, err := h.service.FetchQueues(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": queues})
}

func (h *JobHandler) ListPeriodic(c *gin.Context) {
	jobs, err := h.service.FetchPeriodicJobs(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *JobHandler) RetryJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs queued for retry", h.service.RetryJobs)
}

func (h *JobHandler) CancelJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs cancelled", h.service.CancelJobs)
}

func (h *JobHandler) DeleteJobs(c *gin.Context) {
	h.runJobAction(c, "Jobs deleted", h.service.DeleteJobs)
}

func (h *JobHandler) runJobAction(c *gin.Context, okMsg string, action func(context.Context, []int64) error) {
	var body models.JobIDsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "body must be {\"ids\": [...]}", err))
		return
	}

	if err := action(c.Request.Context(), body.IDs); err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, okMsg, "Success", http.StatusOK)
}

func (h *JobHandler) PauseQueue(c *gin.Context) {
	if err := h.service.PauseQueue(c.Request.Context(), c.Param("name")); err != nil {
		_ = c.Error(err)
		return
	}
	utils.SuccessMessage(c, "Queue paused", "Success", http.StatusOK)
}

func (h *JobHandler) ResumeQueue(c *gin.Context) {
	if err := h.service.ResumeQueue(c.Request.Context(), c.Param("name")); err != nil {
		_ = c.Error(err)
		return
	}
	utils.SuccessMessage(c, "Queue resumed", "Success", http.StatusOK)
}

func (h *JobHandler) ListUserJobs(c *gin.Context) {
	userID := c.GetInt64("user_id")
	kind := c.Query("kind")

	params := utils.GetPaginationParams(c.Request.URL.Query())

	jobs, paginator, err := h.service.ListUserJobs(c.Request.Context(), userID, kind, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_page":  paginator.CurrentPage,
		"rows_per_page": paginator.RowsPerPage,
		"from":          paginator.From,
		"to":            paginator.To,
		"total_records": paginator.TotalRecords,
		"data":          jobs,
	})
}

func (h *JobHandler) RetryUserJob(c *gin.Context) {
	userID := c.GetInt64("user_id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	err = h.service.RetryUserJob(c.Request.Context(), userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Job queued for retry", "Success", http.StatusOK)
}

func (h *JobHandler) CancelUserJob(c *gin.Context) {
	userID := c.GetInt64("user_id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(apperr.Wrap(apperr.Invalid, "id must be a valid integer", err))
		return
	}

	err = h.service.CancelUserJob(c.Request.Context(), userID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SuccessMessage(c, "Job cancelled", "Success", http.StatusOK)
}
