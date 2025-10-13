package services

import (
	"fmt"
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

func (s *ImportService) markImportFailed(importID int64) {
	_ = s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "failed",
		"completed_at": nil,
	})
}

func (s *ImportService) FetchImportsByImportType(userID int64, importType string) ([]models.Import, error) {
	return s.Repo.FindImportsByImportType(nil, userID, importType)
}

func (s *ImportService) ImportFromJSON(userID, checkID, investID int64, payload models.CustomImportPayload) error {

	checkingAcc, err := s.accService.FetchAccountByID(userID, checkID, false)
	if err != nil {
		return err
	}
	investmentAcc, err := s.accService.FetchAccountByID(userID, investID, false)
	if err != nil {
		return err
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

	// Track the earliest affected date per account
	earliest := map[int64]time.Time{}

	setEarliest := func(accID int64, t time.Time) {
		d := t.UTC().Truncate(24 * time.Hour)
		if e, ok := earliest[accID]; !ok || d.Before(e) {
			earliest[accID] = d
		}
	}

	for _, txn := range payload.Txns {

		amount, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount %q: %w", txn.Amount, err)
		}

		if txn.TransactionType == "income" || txn.TransactionType == "expense" {

			t := models.Transaction{
				UserID:          userID,
				AccountID:       checkingAcc.ID,
				TransactionType: txn.TransactionType,
				Amount:          amount,
				Currency:        models.DefaultCurrency,
				TxnDate:         txn.TxnDate,
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

			// this account’s snapshots must be recomputed forward from this date
			setEarliest(checkingAcc.ID, t.TxnDate)

		} else if txn.Category == "investments" {

			expense := models.Transaction{
				UserID:          userID,
				AccountID:       checkingAcc.ID,
				TransactionType: "expense",
				Amount:          amount,
				Currency:        models.DefaultCurrency,
				TxnDate:         txn.TxnDate,
				Description:     &txn.Description,
				IsTransfer:      true,
			}

			if _, err := s.TxnRepo.InsertTransaction(tx, &expense); err != nil {
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

			income := models.Transaction{
				UserID:          userID,
				AccountID:       investmentAcc.ID,
				TransactionType: "income",
				Amount:          amount,
				Currency:        models.DefaultCurrency,
				TxnDate:         txn.TxnDate,
				Description:     &txn.Description,
				IsTransfer:      true,
			}

			if _, err := s.TxnRepo.InsertTransaction(tx, &income); err != nil {
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

			transfer := models.Transfer{
				UserID:               userID,
				TransactionInflowID:  income.ID,
				TransactionOutflowID: expense.ID,
				Amount:               amount,
				Currency:             models.DefaultCurrency,
				Status:               "success",
			}

			if _, err := s.TxnRepo.InsertTransfer(tx, &transfer); err != nil {
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

			if err := s.accService.UpdateBalancesForTransfer(tx, checkingAcc, investmentAcc, txn.TxnDate, amount); err != nil {
				tx.Rollback()
				s.markImportFailed(importID)
				return err
			}

			// mark both accounts to be (re)materialized forward
			setEarliest(checkingAcc.ID, txn.TxnDate)
			setEarliest(investmentAcc.ID, txn.TxnDate)
		}

	}

	// Minimal “frontfill” -> recompute snapshots from earliest affected date through today
	for accID, from := range earliest {
		// (a) fix balances forward
		if err := s.accService.FrontfillBalancesFrom(tx, accID, from); err != nil {
			tx.Rollback()
			s.markImportFailed(importID)
			return err
		}
		var cc string
		switch accID {
		case checkingAcc.ID:
			cc = checkingAcc.Currency
		case investmentAcc.ID:
			cc = investmentAcc.Currency
		default:
			cc = models.DefaultCurrency
		}
		if err := s.accService.materializeTodaySnapshot(tx, userID, accID, cc, from); err != nil {
			tx.Rollback()
			s.markImportFailed(importID)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(importID)
		return err
	}

	_ = s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"completed_at": time.Now().UTC(),
	})

	return nil
}
