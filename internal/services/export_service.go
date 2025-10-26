package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"

	"go.uber.org/zap"
)

type ExportService struct {
	Config     *config.Config
	Ctx        *DefaultServiceContext
	Repo       *repositories.ExportRepository
	TxnRepo    *repositories.TransactionRepository
	accService *AccountService
}

func NewExportService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ExportRepository,
	txnRepo *repositories.TransactionRepository,
	accService *AccountService,
) *ExportService {
	return &ExportService{
		Ctx:        ctx,
		Config:     cfg,
		Repo:       repo,
		TxnRepo:    txnRepo,
		accService: accService,
	}
}

func (s *ExportService) FetchExports(userID int64) ([]models.Export, error) {
	return s.Repo.FindExports(nil, userID)
}

func (s *ExportService) FetchExportsByExportType(userID int64, exportType string) ([]models.Export, error) {
	return s.Repo.FindExportsByExportType(nil, userID, exportType)
}

func (s *ExportService) buildAccountExportJSON(accs []models.Account) ([]byte, error) {
	exports := make([]models.AccountExport, 0, len(accs))

	for _, a := range accs {
		e := models.AccountExport{
			Name:     a.Name,
			Currency: a.Currency,
			OpenedAt: a.OpenedAt,
		}
		e.AccountType.Type = a.AccountType.Type
		e.AccountType.SubType = a.AccountType.Subtype
		e.AccountType.Classification = a.AccountType.Classification

		e.Balance.StartBalance = a.Balance.StartBalance.String()
		e.Balance.CashInflows = a.Balance.CashInflows.String()
		e.Balance.CashOutflows = a.Balance.CashOutflows.String()
		e.Balance.NonCashInflows = a.Balance.NonCashInflows.String()
		e.Balance.NonCashOutflows = a.Balance.NonCashOutflows.String()
		e.Balance.NetMarketFlows = a.Balance.NetMarketFlows.String()
		e.Balance.Adjustments = a.Balance.Adjustments.String()

		exports = append(exports, e)
	}

	return json.MarshalIndent(exports, "", "  ")
}

func (s *ExportService) buildCategoryExportJSON(cats []models.Category) ([]byte, error) {
	exports := make([]models.CategoryExport, 0, len(cats))

	for _, c := range cats {
		e := models.CategoryExport{
			Name:           c.Name,
			DisplayName:    c.DisplayName,
			Classification: c.Classification,
			ParentID:       c.ParentID,
			IsDefault:      c.IsDefault,
		}
		exports = append(exports, e)
	}

	return json.MarshalIndent(exports, "", "  ")
}

func (s *ExportService) buildTxnAndTransfersExportJSON(
	txns []models.Transaction,
	transfers []models.Transfer,
) ([]byte, error) {

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

func (s *ExportService) CreateExport(userID int64) ([]byte, error) {

	l := s.Ctx.Logger.With(
		zap.String("op", "create_export"),
		zap.Int64("user_id", userID),
	)

	l.Info("Started a JSON export")

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	accs, err := s.accService.Repo.FindAllAccountsWithInitialBalance(tx, userID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	categories, err := s.TxnRepo.FindAllCategories(tx, &userID, false)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	txns, err := s.TxnRepo.FindAllTransactionsForUser(tx, userID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	transfers, err := s.TxnRepo.FindAllTransfersForUser(tx, userID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Build JSON payloads
	accJSON, err := s.buildAccountExportJSON(accs)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	catJSON, err := s.buildCategoryExportJSON(categories)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	txnsJSON, err := s.buildTxnAndTransfersExportJSON(txns, transfers)
	if err != nil {
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
			return nil, err
		}
		if _, err := f.Write(data); err != nil {
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
