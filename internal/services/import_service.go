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

	checkingAcc, err := s.accService.FetchAccountByID(userID, checkID, false)
	if err != nil {
		return err
	}

	openedYear := checkingAcc.OpenedAt.Year()

	if openedYear >= payload.Year {
		return fmt.Errorf("account opened in %d cannot import data for year %d or earlier", openedYear, payload.Year)
	}

	// create the import as PENDING
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
		return err
	}

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		s.markImportFailed(importID)
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

	for _, txn := range payload.Txns {

		amount, err := decimal.NewFromString(txn.Amount)
		if err != nil {
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
						return fmt.Errorf("can't find category id %d: %w", *m.CategoryID, err)
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
				return fmt.Errorf("can't find default category: %w", err)
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
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

			err := s.accService.UpdateAccountCashBalance(tx, checkingAcc, t.TxnDate, t.TransactionType, t.Amount)
			if err != nil {
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

		}
	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(importID)
		return err
	}

	if err := s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"completed_at": time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	return nil
}
