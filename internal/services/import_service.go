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
	"gorm.io/gorm"
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

func (s *ImportService) FetchImportByID(tx *gorm.DB, id, userID int64, importType string) (*models.Import, error) {
	return s.Repo.FindImportByID(tx, id, userID, importType)
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

func (s *ImportService) TransferInvestmentsFromImport(userID, importID, checkingAccID int64, mappings []models.InvestmentMapping) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "TransferInvestmentsFromImport"),
		zap.Int64("user_id", userID),
		zap.Int64("account_id", checkingAccID),
	)
	l.Info("started investment transfer from custom import")

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

	checkingAcc, err := s.accService.Repo.FindAccountByID(tx, checkingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	imp, err := s.FetchImportByID(tx, importID, userID, "custom")
	if err != nil {
		return err
	}

	filePath := filepath.Join("storage", imp.Name+".json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var payload models.CustomImportPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}

	sort.SliceStable(payload.Txns, func(i, j int) bool {
		return payload.Txns[i].TxnDate.Before(payload.Txns[j].TxnDate)
	})

	catToAccID := make(map[string]int64, len(mappings))
	distinctAccIDs := make(map[int64]struct{})
	for _, m := range mappings {
		var id int64
		switch {
		case m.AccountID == 0:
			continue
		case m.AccountID != 0:
			id = m.AccountID
		}
		if id == 0 {
			continue
		}
		catToAccID[m.Name] = id
		distinctAccIDs[id] = struct{}{}
	}

	accCache := make(map[int64]*models.Account, len(distinctAccIDs))
	for id := range distinctAccIDs {
		acc, err := s.accService.Repo.FindAccountByID(tx, id, userID, true)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("destination account %d not found: %w", id, err)
		}
		accCache[id] = acc
	}

	// Track earliest affected date per account (for one-time frontfill)
	earliest := map[int64]time.Time{}
	markEarliest := func(accID int64, d time.Time) {
		d = d.UTC().Truncate(24 * time.Hour)
		if cur, ok := earliest[accID]; !ok || d.Before(cur) {
			earliest[accID] = d
		}
	}

	l.Info("transferring investments from custom import")
	for _, txn := range payload.Txns {
		if txn.TransactionType != "investments" {
			continue
		}

		// find mapped destination by category
		toAccID, ok := catToAccID[txn.Category]
		if !ok {
			l.Debug("skipping investment txn without mapping",
				zap.String("category", txn.Category),
				zap.Time("date", txn.TxnDate),
			)
			continue
		}

		toAccount, ok := accCache[toAccID]
		if !ok {
			_ = tx.Rollback()
			return fmt.Errorf("account %d not cached (internal error)", toAccID)
		}

		amt, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("invalid amount '%s': %w", txn.Amount, err)
		}

		if amt.IsNegative() {
			amt = amt.Abs()
		}

		desc := txn.Description
		expense := models.Transaction{
			UserID:          userID,
			AccountID:       checkingAcc.ID,
			TransactionType: "expense",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txn.TxnDate,
			Description:     &desc,
			IsTransfer:      true,
		}
		if _, err := s.TxnRepo.InsertTransaction(tx, &expense); err != nil {
			_ = tx.Rollback()
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccount.ID,
			TransactionType: "income",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txn.TxnDate,
			Description:     &desc,
			IsTransfer:      true,
		}
		if _, err := s.TxnRepo.InsertTransaction(tx, &income); err != nil {
			_ = tx.Rollback()
			return err
		}

		transfer := models.Transfer{
			UserID:               userID,
			TransactionInflowID:  income.ID,
			TransactionOutflowID: expense.ID,
			Amount:               amt,
			Currency:             models.DefaultCurrency,
			Status:               "success",
			CreatedAt:            txn.TxnDate,
		}
		if _, err := s.TxnRepo.InsertTransfer(tx, &transfer); err != nil {
			_ = tx.Rollback()
			return err
		}

		err = s.accService.UpdateBalancesForTransfer(tx, checkingAcc, toAccount, txn.TxnDate, amt)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Track earliest affected day for both accounts, ensure frontfill cascades forward
		markEarliest(checkingAcc.ID, txn.TxnDate)
		markEarliest(toAccount.ID, txn.TxnDate)
	}

	// One-time frontfill of snapshots for all affected accounts
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for accID, from := range earliest {
		// get *Account
		acc, ok := accCache[accID]
		if !ok || acc == nil {
			if accID == checkingAcc.ID {
				acc = checkingAcc
			} else {
				a, err := s.accService.Repo.FindAccountByID(tx, accID, userID, true)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				acc = a
			}
		}

		// start from min(opening_as_of, from)
		if open, err := s.accService.Repo.GetAccountOpeningAsOf(tx, accID); err == nil && !open.IsZero() && open.Before(from) {
			from = open
		}
		from = from.UTC().Truncate(24 * time.Hour)

		if err := s.accService.Repo.EnsureDailyBalanceRow(tx, checkingAcc.ID, time.Now().UTC(), models.DefaultCurrency); err != nil {
			tx.Rollback()
			return err
		}

		// re-thread start_balance forward
		if err := s.accService.FrontfillBalancesForAccount(tx, userID, accID, acc.Currency, from); err != nil {
			_ = tx.Rollback()
			return err
		}
		// refresh snapshots from same point
		if err := s.accService.backfillAccountRange(tx, acc, from, today); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	l.Info("import completed successfully",
		zap.Int64("import_id", importID),
		zap.Duration("elapsed", time.Since(start)),
		zap.String("status", "success"),
	)

	return nil
}
