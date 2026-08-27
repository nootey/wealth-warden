package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/reports"
	"wealth-warden/internal/ws"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

var ErrNoCategoryData = errors.New("no transactions found for the selected categories")

type categoryReportSvc interface {
	MarkReportProcessing(ctx context.Context, reportID int64) error
	MarkReportCompleted(ctx context.Context, reportID int64, name, filePath string, fileSize int64, completedAt time.Time) error
	MarkReportFailed(ctx context.Context, reportID int64, reason string) error
	FetchCategoryReportData(ctx context.Context, userID int64, params models.CategoryReportParams) ([]models.CategoryReportDataRow, error)
	FindReportAccountScope(ctx context.Context, userID, accountID int64) (*models.ReportAccountScope, error)
}

type GenerateCategoryReportWorker struct {
	river.WorkerDefaults[jobqueue.GenerateCategoryReportArgs]
	logger       *zap.Logger
	analyticsSvc categoryReportSvc
	broadcaster  ws.Broadcaster
}

func NewGenerateCategoryReportWorker(
	logger *zap.Logger,
	analyticsSvc categoryReportSvc,
	broadcaster ws.Broadcaster,
) *GenerateCategoryReportWorker {
	return &GenerateCategoryReportWorker{
		logger:       logger,
		analyticsSvc: analyticsSvc,
		broadcaster:  broadcaster,
	}
}

func (w *GenerateCategoryReportWorker) Work(ctx context.Context, job *river.Job[jobqueue.GenerateCategoryReportArgs]) error {
	return (&categoryReportRun{
		logger:       w.logger,
		analyticsSvc: w.analyticsSvc,
		broadcaster:  w.broadcaster,
		ReportID:     job.Args.ReportID,
		UserID:       job.Args.UserID,
		Params:       job.Args.Params,
	}).run(ctx)
}

type categoryReportRun struct {
	logger       *zap.Logger
	analyticsSvc categoryReportSvc
	broadcaster  ws.Broadcaster
	ReportID     int64
	UserID       int64
	Params       models.CategoryReportParams
}

func (j *categoryReportRun) run(ctx context.Context) error {
	if err := j.analyticsSvc.MarkReportProcessing(ctx, j.ReportID); err != nil {
		return err
	}

	scopeLabel, err := j.accountScopeLabel(ctx)
	if err != nil {
		return j.fail(ctx, err)
	}

	rows, err := j.analyticsSvc.FetchCategoryReportData(ctx, j.UserID, j.Params)
	if err != nil {
		return j.fail(ctx, err)
	}

	if len(rows) == 0 {
		return j.fail(ctx, ErrNoCategoryData)
	}

	data, err := reports.BuildCategoryXLSX(rows, j.Params, scopeLabel)
	if err != nil {
		return j.fail(ctx, err)
	}

	filePath, err := j.saveFile(data)
	if err != nil {
		return j.fail(ctx, err)
	}

	now := time.Now().UTC()
	if err := j.analyticsSvc.MarkReportCompleted(ctx, j.ReportID, j.reportName(now), filePath, int64(len(data)), now); err != nil {
		return err
	}

	j.broadcaster.Send(j.UserID, ws.Event{Type: ws.TypeReportCompleted, Payload: ws.ReportPayload{ReportID: j.ReportID}})
	return nil
}

// A failed report is a finished run, not a retryable one. Only an unrecorded failure is worth retrying.
func (j *categoryReportRun) fail(ctx context.Context, cause error) error {
	j.logger.Error("category report generation failed", zap.Int64("reportID", j.ReportID), zap.Error(cause))

	if err := j.analyticsSvc.MarkReportFailed(ctx, j.ReportID, cause.Error()); err != nil {
		return err
	}

	j.broadcaster.Send(j.UserID, ws.Event{Type: ws.TypeReportFailed, Payload: ws.ReportPayload{ReportID: j.ReportID}})
	return nil
}

func (j *categoryReportRun) accountScopeLabel(ctx context.Context) (string, error) {
	if j.Params.AccountID == nil {
		return "All accounts", nil
	}

	scope, err := j.analyticsSvc.FindReportAccountScope(ctx, j.UserID, *j.Params.AccountID)
	if err != nil {
		return "", err
	}
	if j.Params.AccountTypeOnly {
		return fmt.Sprintf("All %s accounts", humanizeSubtype(scope.Subtype)), nil
	}
	return scope.Name, nil
}

func humanizeSubtype(subtype string) string {
	words := strings.Split(subtype, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (j *categoryReportRun) reportName(generatedAt time.Time) string {
	categoryPart := pluralize(len(j.Params.InflowCategoryIDs)+len(j.Params.OutflowCategoryIDs), "category", "categories")

	yearPart := "All Time"
	if !j.Params.AllTime {
		yearPart = pluralize(len(j.Params.Years), "year", "years")
	}

	name := fmt.Sprintf("Category Report - %s - %s - %s", categoryPart, yearPart, generatedAt.Format("2006-01-02 15:04"))
	if j.Params.Description != "" {
		name += fmt.Sprintf(" (%s)", j.Params.Description)
	}
	return name
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func (j *categoryReportRun) saveFile(data []byte) (string, error) {
	dir := filepath.Join("storage", "reports", fmt.Sprintf("%d", j.UserID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, fmt.Sprintf("%d.xlsx", j.ReportID))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	return filePath, nil
}
