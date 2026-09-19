package jobs

import (
	"context"
	"errors"
	"fmt"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

type importRunner interface {
	RunImportTransactions(ctx context.Context, userID, importID, checkID int64, source string) error
	RunImportAccounts(ctx context.Context, userID, importID int64, useBalances bool) error
	RunImportCategories(ctx context.Context, userID, importID int64) error
	RunImportRules(ctx context.Context, userID, importID int64) error
	RunTransferInvestments(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error
	RunTransferSavings(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error
	RunTransferRepayments(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error
	RunTransferTrades(ctx context.Context, userID, importID int64, mappings []models.TransferMapping) error
	DiscardStagedImport(ctx context.Context, userID, importID int64) error
}

// Subtypes that stage their own payload file; transfers reuse the parent import's file.
var stagedFileSubTypes = map[string]bool{
	"transactions": true,
	"accounts":     true,
	"categories":   true,
	"rules":        true,
	"trades":       true,
}

type ImportWorker struct {
	river.WorkerDefaults[jobqueue.ImportArgs]
	logger  *zap.Logger
	imports importRunner
}

func NewImportWorker(logger *zap.Logger, imports importRunner) *ImportWorker {
	return &ImportWorker{logger: logger, imports: imports}
}

func (w *ImportWorker) Work(ctx context.Context, job *river.Job[jobqueue.ImportArgs]) error {
	a := job.Args

	var err error
	switch a.SubType {
	case "transactions":
		err = w.imports.RunImportTransactions(ctx, a.UserID, a.ImportID, a.CheckAccID, a.Source)
	case "accounts":
		err = w.imports.RunImportAccounts(ctx, a.UserID, a.ImportID, a.UseBalances)
	case "categories":
		err = w.imports.RunImportCategories(ctx, a.UserID, a.ImportID)
	case "rules":
		err = w.imports.RunImportRules(ctx, a.UserID, a.ImportID)
	case "investments":
		err = w.imports.RunTransferInvestments(ctx, a.UserID, a.ImportID, a.CheckAccID, a.Mappings)
	case "savings":
		err = w.imports.RunTransferSavings(ctx, a.UserID, a.ImportID, a.CheckAccID, a.Mappings)
	case "repayments":
		err = w.imports.RunTransferRepayments(ctx, a.UserID, a.ImportID, a.CheckAccID, a.Mappings)
	case "trades":
		err = w.imports.RunTransferTrades(ctx, a.UserID, a.ImportID, a.Mappings)
	default:
		return river.JobCancel(fmt.Errorf("unknown import subtype %q for import %d", a.SubType, a.ImportID))
	}

	if err != nil {
		w.logger.Error("import run failed",
			zap.String("sub_type", a.SubType),
			zap.Int64("import_id", a.ImportID),
			zap.Error(err))

		// A bad payload never recovers, so drop its staged file and stop retrying.
		if nonRetryable(err) {
			if stagedFileSubTypes[a.SubType] {
				_ = w.imports.DiscardStagedImport(ctx, a.UserID, a.ImportID)
			}
			return river.JobCancel(err)
		}
		return err
	}
	return nil
}

func nonRetryable(err error) bool {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Kind != apperr.Internal
}
