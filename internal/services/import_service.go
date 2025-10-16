package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type ImportService struct {
	Config     *config.Config
	Ctx        *DefaultServiceContext
	Repo       *repositories.ImportRepository
	TxnRepo    *repositories.TransactionRepository
	accService *AccountService
}

func NewImportService(
	cfg *config.Config,
	ctx *DefaultServiceContext,
	repo *repositories.ImportRepository,
	txnRepo *repositories.TransactionRepository,
	accService *AccountService,
) *ImportService {
	return &ImportService{
		Ctx:        ctx,
		Config:     cfg,
		Repo:       repo,
		TxnRepo:    txnRepo,
		accService: accService,
	}
}

func (s *ImportService) ValidateCustomImport(payload *models.CustomImportPayload, step string) ([]string, int, error) {

	if payload.Year == 0 {
		return nil, 0, errors.New("missing or invalid 'year' field")
	}
	if payload.GeneratedAt.IsZero() {
		return nil, 0, errors.New("missing or invalid 'generated_at' field")
	}
	if len(payload.Txns) == 0 {
		return nil, 0, errors.New("no transactions found")
	}

	for _, t := range payload.Txns {
		if t.TransactionType == "" {
			return nil, 0, errors.New("missing transaction_type")
		}
		tt := strings.ToLower(strings.TrimSpace(t.TransactionType))
		if tt != "income" && tt != "expense" && tt != "investments" && tt != "savings" {
			return nil, 0, errors.New("invalid transaction_type")
		}
		if t.Amount == "" {
			return nil, 0, errors.New("missing amount")
		}
		if t.TxnDate.IsZero() {
			return nil, 0, errors.New("missing or invalid txn_date")
		}
	}

	step = strings.ToLower(strings.TrimSpace(step))
	if step == "" {
		step = "cash"
	}

	allowed := map[string]bool{}
	switch step {
	case "cash":
		allowed["income"] = true
		allowed["expense"] = true
	case "investment", "investments":
		allowed["investments"] = true
	default:
		allowed["income"] = true
		allowed["expense"] = true
		step = "cash"
	}

	unique := make(map[string]bool)
	filteredCount := 0

	for _, t := range payload.Txns {
		tt := strings.ToLower(strings.TrimSpace(t.TransactionType))
		if !allowed[tt] {
			continue
		}
		filteredCount++

		cat := strings.TrimSpace(t.Category)
		if cat != "" {
			unique[cat] = true
		}
	}

	categories := make([]string, 0, len(unique))
	for cat := range unique {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	return categories, filteredCount, nil
}

func (s *ImportService) markImportFailed(importID int64, cause error) {

	msg := ""
	if cause != nil {
		msg = cause.Error()
	}

	_ = s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "failed",
		"completed_at": nil,
		"error":        msg,
	})
}

func (s *ImportService) FetchImportsByImportType(userID int64, importType string) ([]models.Import, error) {
	return s.Repo.FindImportsByImportType(nil, userID, importType)
}

func (s *ImportService) ImportFromJSON(userID, checkID int64, payload models.CustomImportPayload) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "ImportFromJSON"),
		zap.Int64("user_id", userID),
		zap.Int64("account_id", checkID),
		zap.Int("import_year", payload.Year),
	)
	l.Info("started custom JSON import")

	l.Info("validating requirements")
	checkingAcc, err := s.accService.FetchAccountByID(userID, checkID, false)
	if err != nil {
		return err
	}

	openedYear := checkingAcc.OpenedAt.Year()

	if openedYear >= payload.Year {
		return fmt.Errorf("account opened in %d cannot import data for year %d or earlier", openedYear, payload.Year)
	}

	importName := fmt.Sprintf("custom_year_%d_txns_%d", payload.Year, len(payload.Txns))
	dir := "storage"
	finalPath := filepath.Join(dir, importName+".json")
	tmpPath := filepath.Join(dir, importName+".json.tmp")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Hard duplicate check
	if _, err := os.Stat(finalPath); err == nil {
		return errors.New("import file already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Reserve the name with an exclusive temp file (prevents races)
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("import file already exists")
		}
		return err
	}
	reserved := true
	defer func() {
		if reserved {
			_ = os.Remove(tmpPath)
		}
	}()

	// create the import as PENDING
	l.Info("creating import row", zap.String("status", "pending"))
	started := time.Now().UTC()

	importID, err := s.Repo.InsertImport(nil, models.Import{
		Name:       importName,
		UserID:     userID,
		AccountID:  checkingAcc.ID,
		ImportType: "custom",
		Status:     "pending",
		Step:       "cash",
		Currency:   models.DefaultCurrency,
		StartedAt:  &started,
	})
	if err != nil {
		l.Error("failed to create import row", zap.Error(err))
		return err
	}

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(importID, nil)
			panic(p)
		}
	}()

	sort.SliceStable(payload.Txns, func(i, j int) bool {
		return payload.Txns[i].TxnDate.Before(payload.Txns[j].TxnDate)
	})

	l.Info("inserting transactions and updating balances")
	for _, txn := range payload.Txns {

		amount, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			tx.Rollback()
			s.markImportFailed(importID, err)
			return fmt.Errorf("invalid amount %q: %w", txn.Amount, err)
		}

		txnDate := txn.TxnDate.UTC().Truncate(24 * time.Hour)

		var category models.Category
		var found bool

		for _, m := range payload.CategoryMappings {
			if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(txn.Category)) {
				if m.CategoryID != nil {
					category, err = s.TxnRepo.FindCategoryByID(tx, *m.CategoryID, &userID, false)
					if err != nil {
						tx.Rollback()
						s.markImportFailed(importID, err)
						l.Error("can't find category id", zap.Error(err))
						return fmt.Errorf(" %d: %w", *m.CategoryID, err)
					}
					found = true
				}
				break
			}
		}

		// Fallback if no mapping or category_id is nil
		if !found {
			category, err = s.TxnRepo.FindCategoryByClassification(tx, "uncategorized", &userID)
			if err != nil {
				tx.Rollback()
				s.markImportFailed(importID, err)
				l.Error("can't find default category", zap.Error(err))
				return fmt.Errorf(": %w", err)
			}
		}

		if txn.TransactionType == "income" || txn.TransactionType == "expense" {

			t := models.Transaction{
				UserID:          userID,
				AccountID:       checkingAcc.ID,
				CategoryID:      &category.ID,
				TransactionType: txn.TransactionType,
				Amount:          amount,
				Currency:        models.DefaultCurrency,
				TxnDate:         txnDate,
				Description:     &txn.Description,
			}

			if _, err := s.TxnRepo.InsertTransaction(tx, &t); err != nil {
				l.Error("failed to insert transaction", zap.Error(err))
				tx.Rollback()
				s.markImportFailed(importID, err)
				return err
			}

			err := s.accService.UpdateAccountCashBalance(tx, checkingAcc, t.TxnDate, t.TransactionType, t.Amount)
			if err != nil {
				l.Error("failed to update account balances", zap.Error(err))
				tx.Rollback()
				s.markImportFailed(importID, err)
				return err
			}

		}
	}

	frontfillFrom := payload.Txns[0].TxnDate.UTC().Truncate(24 * time.Hour)
	l.Info("frontfilling balances",
		zap.Int64("import_id", importID),
		zap.Time("from", frontfillFrom),
		zap.String("currency", models.DefaultCurrency),
	)

	if err := s.accService.FrontfillBalancesForAccount(
		tx,
		userID,
		checkingAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(importID, err)
		l.Error("failed to frontfill balances", zap.Error(err))
		return err
	}

	// Write payload to the reserved temp file
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tx.Rollback()
		s.markImportFailed(importID, err)
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(importID, err)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(importID, err)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(importID, err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Promote the temp file to final
	if err := os.Rename(tmpPath, finalPath); err != nil {
		// DB already committed - mark the import failed
		s.markImportFailed(importID, err)
		return err
	}

	l.Info("saved import JSON file", zap.String("path", finalPath))

	if err := s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "investments",
		"completed_at": time.Now().UTC(),
		"error":        "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	l.Info("import completed successfully",
		zap.Int64("import_id", importID),
		zap.Duration("elapsed", time.Since(start)),
		zap.String("status", "success"),
	)

	return nil
}
