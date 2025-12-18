package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"gorm.io/gorm"
)

type ExportServiceInterface interface {
	FetchExports(ctx context.Context, userID int64) ([]models.Export, error)
	FetchExportByID(ctx context.Context, tx *gorm.DB, id, userID int64) (*models.Export, error)
	FetchExportsByExportType(ctx context.Context, userID int64, exportType string) ([]models.Export, error)
	CreateExport(ctx context.Context, userID int64) (*models.Export, error)
	DownloadExport(ctx context.Context, id, userID int64) ([]byte, error)
	DeleteExport(ctx context.Context, userID, id int64) error
}

type ExportService struct {
	repo          repositories.ExportRepositoryInterface
	txnRepo       repositories.TransactionRepositoryInterface
	accRepo       repositories.AccountRepositoryInterface
	settingsRepo  repositories.SettingsRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
}

func NewExportService(
	repo *repositories.ExportRepository,
	txnRepo *repositories.TransactionRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
) *ExportService {
	return &ExportService{
		repo:          repo,
		txnRepo:       txnRepo,
		accRepo:       accRepo,
		settingsRepo:  settingsRepo,
		jobDispatcher: jobDispatcher,
		loggingRepo:   loggingRepo,
	}
}

var _ ExportServiceInterface = (*ExportService)(nil)

func (s *ExportService) FetchExports(ctx context.Context, userID int64) ([]models.Export, error) {
	return s.repo.FindExports(ctx, nil, userID)
}

func (s *ExportService) FetchExportByID(ctx context.Context, tx *gorm.DB, id, userID int64) (*models.Export, error) {
	return s.repo.FindExportByID(ctx, tx, id, userID)
}

func (s *ExportService) FetchExportsByExportType(ctx context.Context, userID int64, exportType string) ([]models.Export, error) {
	return s.repo.FindExportsByExportType(ctx, nil, userID, exportType)
}

func (s *ExportService) buildAccountExportJSON(accs []models.Account) ([]byte, error) {
	type bundle struct {
		GeneratedAt time.Time              `json:"generated_at"`
		Accounts    []models.AccountExport `json:"accounts"`
	}

	out := bundle{
		GeneratedAt: time.Now().UTC(),
		Accounts:    make([]models.AccountExport, 0, len(accs)),
	}

	for _, a := range accs {
		e := models.AccountExport{
			Name:     a.Name,
			Currency: a.Currency,
			OpenedAt: a.OpenedAt,
		}
		e.AccountType.Type = a.AccountType.Type
		e.AccountType.SubType = a.AccountType.Subtype
		e.AccountType.Classification = a.AccountType.Classification

		e.Balance = a.Balance.EndBalance

		out.Accounts = append(out.Accounts, e)
	}

	return json.MarshalIndent(out, "", "  ")
}

func (s *ExportService) buildCategoryExportJSON(cats []models.Category) ([]byte, error) {
	type bundle struct {
		GeneratedAt time.Time               `json:"generated_at"`
		Categories  []models.CategoryExport `json:"categories"`
	}

	out := bundle{
		GeneratedAt: time.Now().UTC(),
		Categories:  make([]models.CategoryExport, 0, len(cats)),
	}

	for _, c := range cats {
		e := models.CategoryExport{
			Name:           c.Name,
			DisplayName:    c.DisplayName,
			Classification: c.Classification,
			ParentID:       c.ParentID,
			IsDefault:      c.IsDefault,
		}
		out.Categories = append(out.Categories, e)
	}

	return json.MarshalIndent(out, "", "  ")
}

func (s *ExportService) buildTxnAndTransfersExportJSON(txns []models.Transaction, transfers []models.Transfer) ([]byte, error) {

	type bundle struct {
		GeneratedAt  time.Time        `json:"generated_at"`
		Transactions []models.JSONTxn `json:"transactions"`
		Transfers    []models.JSONTxn `json:"transfers"`
	}

	out := bundle{
		GeneratedAt: time.Now().UTC(),
	}

	out.Transactions = make([]models.JSONTxn, 0, len(txns))
	for _, t := range txns {
		if t.IsTransfer {
			continue
		}

		var cat string
		if t.Category.DisplayName != "" {
			cat = t.Category.DisplayName
		} else {
			cat = t.Category.Name
		}

		var desc string
		if t.Description != nil {
			desc = *t.Description
		}

		out.Transactions = append(out.Transactions, models.JSONTxn{
			TransactionType: t.TransactionType,
			Amount:          t.Amount.String(),
			Currency:        t.Currency,
			TxnDate:         t.TxnDate,
			Category:        cat,
			Description:     desc,
		})
	}

	out.Transfers = make([]models.JSONTxn, 0, len(transfers))
	for _, tr := range transfers {
		when := tr.TransactionOutflow.TxnDate
		if when.IsZero() {
			when = tr.TransactionInflow.TxnDate
		}

		destAccountName := tr.TransactionInflow.Account.Name

		var notes string
		if tr.Notes != nil {
			notes = *tr.Notes
		}

		out.Transfers = append(out.Transfers, models.JSONTxn{
			TransactionType: "investments",
			Amount:          tr.Amount.String(),
			Currency:        tr.Currency,
			TxnDate:         when,
			Category:        destAccountName,
			Description:     notes,
		})
	}

	return json.MarshalIndent(out, "", "  ")
}

func (s *ExportService) CreateExport(ctx context.Context, userID int64) (*models.Export, error) {

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil || loc == nil {
		loc = time.UTC
	}

	now := time.Now().UTC()
	localTime := now.In(loc)

	// Create pending export record
	export := &models.Export{
		Name:       fmt.Sprintf("Export %s", localTime.Format("2006-01-02 15:04:05")),
		UserID:     userID,
		ExportType: "custom",
		Status:     "pending",
		Currency:   models.DefaultCurrency,
		StartedAt:  &now,
	}

	if err := s.repo.InsertExport(ctx, nil, export); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	accs, err := s.accRepo.FindAllAccountsWithLatestBalance(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	categories, err := s.txnRepo.FindAllCategories(ctx, tx, &userID, false)
	if err != nil {
		tx.Rollback()
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	txns, err := s.txnRepo.FindAllTransactionsForUser(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	transfers, err := s.txnRepo.FindAllTransfersForUser(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	// Build JSON payloads
	accJSON, err := s.buildAccountExportJSON(accs)
	if err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	catJSON, err := s.buildCategoryExportJSON(categories)
	if err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	txnsJSON, err := s.buildTxnAndTransfersExportJSON(txns, transfers)
	if err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	// Create ZIP
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	files := map[string][]byte{
		"accounts.json":     accJSON,
		"categories.json":   catJSON,
		"transactions.json": txnsJSON,
	}

	for name, data := range files {
		f, err := zipWriter.Create(name)
		if err != nil {
			sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
			if sErr != nil {
				return nil, sErr
			}
			return nil, err
		}
		if _, err := f.Write(data); err != nil {
			sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
			if sErr != nil {
				return nil, sErr
			}
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	// Save to filesystem
	zipData := buf.Bytes()
	filePath, err := s.saveExportFile(userID, export.Name, zipData)
	if err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	completedAt := time.Now().UTC()
	fileSize := int64(len(zipData))

	updates := map[string]interface{}{
		"status":       "success",
		"file_path":    filePath,
		"file_size":    fileSize,
		"completed_at": completedAt,
	}

	if err := s.repo.UpdateExport(ctx, nil, export.ID, updates); err != nil {
		sErr := s.updateExportStatus(ctx, export.ID, "failed", err.Error())
		if sErr != nil {
			return nil, sErr
		}
		return nil, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", export.Name, changes, "export_name")
	utils.CompareChanges("", export.Currency, changes, "currency")
	utils.CompareChanges("", fmt.Sprintf("%d", fileSize), changes, "file_size")
	utils.CompareChanges("", fmt.Sprintf("%d", len(accs)), changes, "accounts_count")
	utils.CompareChanges("", fmt.Sprintf("%d", len(categories)), changes, "categories_count")
	utils.CompareChanges("", fmt.Sprintf("%d", len(txns)), changes, "transactions_count")
	utils.CompareChanges("", fmt.Sprintf("%d", len(transfers)), changes, "transfers_count")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "export",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return nil, err
	}

	return export, nil

}

func (s *ExportService) saveExportFile(userID int64, exportName string, data []byte) (string, error) {

	dir := filepath.Join("storage", "exports", fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	exportName = utils.NormalizeName(exportName)
	filename := fmt.Sprintf("%s.zip", exportName)
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *ExportService) updateExportStatus(ctx context.Context, exportID int64, status, errorMsg string) error {
	update := map[string]interface{}{
		"status": status,
	}

	if errorMsg != "" {
		update["error"] = errorMsg
	}

	if status == "failed" {
		now := time.Now().UTC()
		update["completed_at"] = now
	}

	return s.repo.UpdateExport(ctx, nil, exportID, update)
}

func (s *ExportService) DownloadExport(ctx context.Context, id, userID int64) ([]byte, error) {

	export, err := s.repo.FindExportByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	if export.Status != "success" {
		return nil, fmt.Errorf("export is not ready (status: %s)", export.Status)
	}

	if export.FilePath == nil {
		return nil, fmt.Errorf("export file path not found")
	}

	data, err := os.ReadFile(*export.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	return data, nil
}

func (s *ExportService) DeleteExport(ctx context.Context, userID, id int64) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Fetch export row
	ex, err := s.FetchExportByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete export row
	if err := s.repo.DeleteExport(ctx, tx, ex.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	exportName := utils.NormalizeName(ex.Name)

	// Delete import files
	finalPath := filepath.Join("storage", "exports", fmt.Sprintf("%d", userID), exportName+".zip")
	tmpPath := finalPath + ".tmp"
	for _, p := range []string{tmpPath, finalPath} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			tx.Rollback()
			return fmt.Errorf("failed to remove file %s: %w", p, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(ex.Name, "", changes, "export_name")

	if err := s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "export",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}
