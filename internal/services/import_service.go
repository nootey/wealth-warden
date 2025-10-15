package services

import (
	"errors"
	"fmt"
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

func (s *ImportService) ValidateCustomImport(payload *models.CustomImportPayload) ([]string, error) {
	if payload.Year == 0 {
		return nil, errors.New("missing or invalid 'year' field")
	}

	if payload.GeneratedAt.IsZero() {
		return nil, errors.New("missing or invalid 'generated_at' field")
	}

	if len(payload.Txns) == 0 {
		return nil, errors.New("no transactions found")
	}

	for _, t := range payload.Txns {
		if t.TransactionType == "" {
			return nil, errors.New("missing transaction_type")
		}

		tt := strings.ToLower(t.TransactionType)
		if tt != "income" && tt != "expense" && tt != "investments" && tt != "savings" {
			return nil, errors.New("invalid transaction_type")
		}

		if t.Amount == "" {
			return nil, errors.New("missing amount")
		}

		if t.TxnDate.IsZero() {
			return nil, errors.New("missing or invalid txn_date")
		}
	}

	unique := make(map[string]bool)

	for _, t := range payload.Txns {
		tt := strings.ToLower(strings.TrimSpace(t.TransactionType))
		if tt != "income" && tt != "expense" {
			continue
		}

		cat := strings.TrimSpace(t.Category)
		if cat == "" {
			continue
		}

		unique[cat] = true
	}

	categories := make([]string, 0, len(unique))
	for cat := range unique {
		categories = append(categories, cat)
	}

	return categories, nil
}

func (s *ImportService) markImportFailed(importID int64) {
	_ = s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "failed",
		"completed_at": nil,
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

	checkingAcc, err := s.accService.FetchAccountByID(userID, checkID, false)
	if err != nil {
		return err
	}

	openedYear := checkingAcc.OpenedAt.Year()

	if openedYear >= payload.Year {
		return fmt.Errorf("account opened in %d cannot import data for year %d or earlier", openedYear, payload.Year)
	}

	// create the import as PENDING
	l.Info("creating import row", zap.String("status", "pending"))
	started := time.Now().UTC()
	importName := fmt.Sprintf("Custom import created on %s", started.Format("2006-01-02 15:04:05"))
	importID, err := s.Repo.InsertImport(nil, models.Import{
		Name:       importName,
		UserID:     userID,
		AccountID:  checkingAcc.ID,
		ImportType: "custom",
		Status:     "pending",
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
			s.markImportFailed(importID)
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
			s.markImportFailed(importID)
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
						s.markImportFailed(importID)
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
				s.markImportFailed(importID)
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
				s.markImportFailed(importID)
				return err
			}

			err := s.accService.UpdateAccountCashBalance(tx, checkingAcc, t.TxnDate, t.TransactionType, t.Amount)
			if err != nil {
				l.Error("failed to update account balances", zap.Error(err))
				tx.Rollback()
				s.markImportFailed(importID)
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
		s.markImportFailed(importID)
		l.Error("failed to frontfill balances", zap.Error(err))
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if err := s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"completed_at": time.Now().UTC(),
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
