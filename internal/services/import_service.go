package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"

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

func (s *ImportService) ValidateCustomImport(payload *models.TxnImportPayload, step string) ([]string, int, error) {
	if payload.GeneratedAt.IsZero() {
		return nil, 0, errors.New("missing or invalid 'generated_at' field")
	}

	step = strings.ToLower(strings.TrimSpace(step))
	if step == "" {
		step = "cash"
	}

	var set []models.JSONTxn
	switch step {
	case "investment", "investments":
		set = payload.Transfers
	default: // "cash"
		set = payload.Txns
	}

	if len(set) == 0 {
		return nil, 0, nil
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

	for _, t := range set {
		if strings.TrimSpace(t.TransactionType) == "" {
			return nil, 0, errors.New("missing transaction_type")
		}
		tt := strings.ToLower(strings.TrimSpace(t.TransactionType))
		if !allowed[tt] {
			return nil, 0, errors.New("invalid transaction_type for selected step")
		}
		if strings.TrimSpace(t.Amount) == "" {
			return nil, 0, errors.New("missing amount")
		}
		if t.TxnDate.IsZero() {
			return nil, 0, errors.New("missing or invalid txn_date")
		}
	}

	unique := make(map[string]bool)
	for _, t := range set {
		if cat := strings.TrimSpace(t.Category); cat != "" {
			unique[cat] = true
		}
	}

	categories := make([]string, 0, len(unique))
	for cat := range unique {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	return categories, len(set), nil
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

func (s *ImportService) ImportTransactions(userID, checkID int64, payload models.TxnImportPayload) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "import_transactions"),
		zap.Int64("user_id", userID),
		zap.Int64("account_id", checkID),
	)
	l.Info("started transactions JSON import")

	l.Info("validating requirements")
	checkingAcc, err := s.accService.FetchAccountByID(userID, checkID, false)
	if err != nil {
		return err
	}

	openedYear := checkingAcc.OpenedAt.Year()

	var first time.Time
	for _, t := range payload.Txns {
		if !t.TxnDate.IsZero() {
			first = t.TxnDate
			break
		}
	}
	if first.IsZero() {
		for _, t := range payload.Transfers {
			if !t.TxnDate.IsZero() {
				first = t.TxnDate
				break
			}
		}
	}
	if first.IsZero() {
		return fmt.Errorf("cannot infer import year: no valid txn_date in transactions or transfers")
	}
	importYear := first.Year()

	if openedYear >= importYear {
		return fmt.Errorf("account opened in %d cannot import data for year %d or earlier", openedYear, importYear)
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_transactions_year_%d_generated_%s", importYear, todayStr)

	dir := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID))
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
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "transactions",
		Status:    "pending",
		Step:      "cash",
		Currency:  models.DefaultCurrency,
		StartedAt: &started,
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

	settings, err := s.accService.Ctx.SettingsRepo.FetchUserSettings(nil, userID)
	if err != nil {
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

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

		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

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
				TxnDate:         txDay,
				Description:     &txn.Category,
				ImportID:        &importID,
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

	frontfillFrom := utils.LocalMidnightUTC(payload.Txns[0].TxnDate, loc)
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
		s.markImportFailed(importID, err)
		return err
	}

	l.Info("saved transactions import JSON file", zap.String("path", finalPath))

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

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "transactions", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", checkingAcc.Name, changes, "checking_account")
	utils.CompareChanges("", models.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Txns)), changes, "transactions_count")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) ImportAccounts(userID int64, payload models.AccImportPayload, useBalances bool) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "import_accounts"),
		zap.Int64("user_id", userID),
		zap.Boolp("use_balances", &useBalances),
	)
	l.Info("started accounts JSON import")

	l.Info("validating requirements")
	accCount, err := s.accService.Repo.CountAccounts(userID, nil, false, nil)
	if err != nil {
		return err
	}

	maxAcc, err := s.Ctx.SettingsRepo.FetchMaxAccountsForUser(nil)
	if err != nil {
		return err
	}

	if accCount >= maxAcc {
		return fmt.Errorf("you can only have %d active accounts", maxAcc)
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_accounts_generated_%s", todayStr)

	dir := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID))
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

	// Reserve the name with an exclusive temp file
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
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "accounts",
		Status:    "pending",
		Step:      "accounts",
		Currency:  models.DefaultCurrency,
		StartedAt: &started,
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
			s.markImportFailed(importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	settings, err := s.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(importID, err)
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	l.Info("processing accounts")
	for _, acc := range payload.Accounts {

		openedAt := acc.OpenedAt
		if openedAt.IsZero() {
			openedAt = time.Now()
		}
		openedDay := utils.LocalMidnightUTC(openedAt, loc)

		accType, err := s.accService.Repo.FindAccountTypeByType(tx, acc.AccountType.Type, acc.AccountType.SubType)
		if err != nil {
			s.markImportFailed(importID, err)
			tx.Rollback()
			return fmt.Errorf("can't find account_type from schema %w", err)
		}

		account := &models.Account{
			Name:          acc.Name,
			Currency:      models.DefaultCurrency,
			AccountTypeID: accType.ID,
			UserID:        userID,
			ImportID:      &importID,
			OpenedAt:      openedDay,
		}

		accountID, err := s.accService.Repo.InsertAccount(tx, account)
		if err != nil {
			s.markImportFailed(importID, err)
			tx.Rollback()
			return err
		}

		var amount decimal.Decimal
		if useBalances {
			amount = acc.Balance.Round(4)

			if accType.Classification == "liability" {
				amount = amount.Neg()
			}
		} else {
			amount = decimal.Zero
		}

		asOf := openedDay

		balance := &models.Balance{
			AccountID:    accountID,
			Currency:     models.DefaultCurrency,
			StartBalance: amount,
			AsOf:         asOf,
		}

		_, err = s.accService.Repo.InsertBalance(tx, balance)
		if err != nil {
			s.markImportFailed(importID, err)
			tx.Rollback()
			return err
		}

		l.Info("seeding snapshots",
			zap.Int64("import_id", importID),
			zap.String("currency", models.DefaultCurrency),
		)

		// seed snapshots from opened day to today
		if err := s.accService.Repo.UpsertSnapshotsFromBalances(
			tx,
			userID,
			accountID,
			models.DefaultCurrency,
			asOf,
			time.Now().UTC().Truncate(24*time.Hour),
		); err != nil {
			s.markImportFailed(importID, err)
			tx.Rollback()
			return err
		}

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
		s.markImportFailed(importID, err)
		return err
	}

	l.Info("saved accounts import JSON file", zap.String("path", finalPath))

	if err := s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "end",
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

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "accounts", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", models.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Accounts)), changes, "accounts_count")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) ImportCategories(userID int64, payload models.CategoryImportPayload) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "import_categories"),
		zap.Int64("user_id", userID),
	)
	l.Info("started category JSON import")

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_categories_generated_%s", todayStr)

	dir := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID))
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

	// Reserve the name with an exclusive temp file
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
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "categories",
		Status:    "pending",
		Step:      "categories",
		Currency:  models.DefaultCurrency,
		StartedAt: &started,
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
			s.markImportFailed(importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	settings, err := s.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(importID, err)
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	l.Info("processing categories")
	for _, cat := range payload.Categories {

		if cat.IsDefault == true {

			exCat, err := s.TxnRepo.FindCategoryByName(tx, cat.Name, nil)
			if err != nil {
				s.markImportFailed(importID, err)
				tx.Rollback()
				return err
			}

			upCat := models.Category{
				ID:             exCat.ID,
				DisplayName:    cat.DisplayName,
				Classification: exCat.Classification,
			}

			_, err = s.TxnRepo.UpdateCategory(tx, upCat)
			if err != nil {
				s.markImportFailed(importID, err)
				tx.Rollback()
				return fmt.Errorf("can't find category from schema %w", err)
			}
			continue
		}

		category := &models.Category{
			UserID:         &userID,
			Name:           cat.Name,
			DisplayName:    cat.DisplayName,
			Classification: cat.Classification,
			ParentID:       nil,
			IsDefault:      false,
			ImportID:       &importID,
		}

		_, err := s.TxnRepo.InsertCategory(tx, category)
		if err != nil {
			s.markImportFailed(importID, err)
			tx.Rollback()
			return err
		}

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
		s.markImportFailed(importID, err)
		return err
	}

	l.Info("saved categories import JSON file", zap.String("path", finalPath))

	if err := s.Repo.UpdateImport(nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "end",
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

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "categories", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", strconv.Itoa(len(payload.Categories)), changes, "categories_count")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "create",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) TransferInvestmentsFromImport(userID int64, payload models.InvestmentTransferPayload) error {

	start := time.Now().UTC()
	l := s.Ctx.Logger.With(
		zap.String("op", "transfer_investments"),
		zap.Int64("user_id", userID),
		zap.Int64("account_id", payload.CheckingAccID),
	)
	l.Info("started investment transfer from custom transactions import")

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accService.Repo.FindAccountByID(tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	imp, err := s.FetchImportByID(tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.InvestmentsTransferred {
		return errors.New("investments have already been transferred for this import")
	}

	filePath := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), imp.Name+".json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var txnPayload models.TxnImportPayload
	if err := json.Unmarshal(b, &txnPayload); err != nil {
		return err
	}

	settings, err := s.accService.Ctx.SettingsRepo.FetchUserSettings(tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(payload.ImportID, err)
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	sort.SliceStable(txnPayload.Transfers, func(i, j int) bool {
		return txnPayload.Transfers[i].TxnDate.Before(txnPayload.Transfers[j].TxnDate)
	})

	catToAccID := make(map[string]int64, len(payload.InvestmentMappings))
	distinctAccIDs := make(map[int64]struct{})
	for _, m := range payload.InvestmentMappings {
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
			s.markImportFailed(payload.ImportID, err)
			return fmt.Errorf("destination account %d not found: %w", id, err)
		}
		accCache[id] = acc
	}

	// track earliest touched date per account
	earliest := make(map[int64]time.Time)
	touch := func(accID int64, d time.Time) {
		if t, ok := earliest[accID]; !ok || d.Before(t) {
			earliest[accID] = d
		}
	}

	l.Info("transferring investments from custom transactions import")
	for _, txn := range txnPayload.Transfers {
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

		// normalize date
		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

		desc := txn.Description
		expense := models.Transaction{
			UserID:          userID,
			AccountID:       checkingAcc.ID,
			TransactionType: "expense",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.TxnRepo.InsertTransaction(tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccount.ID,
			TransactionType: "income",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.TxnRepo.InsertTransaction(tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}

		transfer := models.Transfer{
			UserID:               userID,
			TransactionInflowID:  income.ID,
			TransactionOutflowID: expense.ID,
			Amount:               amt,
			Currency:             models.DefaultCurrency,
			Status:               "success",
			CreatedAt:            txDay,
			ImportID:             &imp.ID,
		}
		if _, err := s.TxnRepo.InsertTransfer(tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}

		err = s.accService.UpdateBalancesForTransfer(tx, checkingAcc, toAccount, txDay, amt)
		if err != nil {
			tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}

		// record earliest touched date
		touch(checkingAcc.ID, txDay)
		touch(toAccount.ID, txDay)
	}

	// frontfill balances
	frontfillFrom := utils.LocalMidnightUTC(txnPayload.Txns[0].TxnDate, loc)
	if err := s.accService.FrontfillBalancesForAccount(
		tx,
		userID,
		checkingAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(payload.ImportID, err)
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for accID, from := range earliest {
		if err := s.accService.FrontfillBalancesForAccount(tx, userID, accID, models.DefaultCurrency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}
		if err := s.accService.Repo.UpsertSnapshotsFromBalances(tx, userID, accID, models.DefaultCurrency, from, today); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(payload.ImportID, err)
			return err
		}
	}

	if err := s.Repo.UpdateImport(tx, payload.ImportID, map[string]interface{}{
		"status":                  "success",
		"step":                    "end",
		"investments_transferred": true,
		"error":                   "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", payload.ImportID, err)
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	l.Info("investment transfer completed successfully",
		zap.Int64("import_id", payload.ImportID),
		zap.Duration("elapsed", time.Since(start)),
		zap.String("status", "success"),
	)

	changes := utils.InitChanges()
	utils.CompareChanges("", imp.Name, changes, "import_name")
	utils.CompareChanges("", checkingAcc.Name, changes, "source_account")
	utils.CompareChanges("", strconv.Itoa(len(payload.InvestmentMappings)), changes, "investment_mappings_count")

	// collect destination account names for readability
	var destNames []string
	for _, acc := range accCache {
		destNames = append(destNames, acc.Name)
	}
	utils.CompareChanges("", strings.Join(destNames, ", "), changes, "destination_accounts")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "transfer_investments",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) DeleteImport(userID, id int64) error {

	l := s.Ctx.Logger.With(
		zap.String("op", "delete_import"),
		zap.Int64("user_id", userID),
		zap.Int64("import_id", id),
	)
	l.Info("deleting import")

	imp, err := s.FetchImportByID(nil, id, userID, "custom")
	if err != nil {
		return err
	}

	switch imp.SubType {
	case "transactions":
		err = s.DeleteTxnImport(userID, imp)
		if err != nil {
			return err
		}
	case "accounts":
		err = s.DeleteAccImport(userID, imp)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	changes := utils.InitChanges()
	utils.CompareChanges(imp.Name, "", changes, "import_name")
	utils.CompareChanges(imp.Type, "", changes, "type")
	utils.CompareChanges(imp.SubType, "", changes, "sub_type")

	if err := s.Ctx.JobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.Ctx.LoggingService.Repo,
		Logger:      s.Ctx.Logger,
		Event:       "delete",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	l.Info("import deleted successfully")
	return nil
}

func (s *ImportService) DeleteTxnImport(userID int64, imp *models.Import) error {
	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Collect all transactions and transfers to reverse their balance effects
	txns, err := s.TxnRepo.FindTransactionsByImportID(tx, imp.ID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find transactions: %w", err)
	}

	trs, err := s.TxnRepo.FindTransfersByImportID(tx, imp.ID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find transfers: %w", err)
	}

	// Track which accounts need balance recomputation
	type accountTouch struct {
		acc   *models.Account
		minAs time.Time
	}
	touched := make(map[int64]*accountTouch)
	touch := func(acc *models.Account, asOf time.Time) {
		at, ok := touched[acc.ID]
		if !ok {
			at = &accountTouch{acc: acc, minAs: asOf}
			touched[acc.ID] = at
		}
		if asOf.Before(at.minAs) {
			at.minAs = asOf
		}
	}

	skipTxn := make(map[int64]struct{})

	// Reverse transfers BEFORE purging
	for _, tr := range trs {
		inflow, err := s.TxnRepo.FindTransactionByID(tx, tr.TransactionInflowID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find inflow transaction %w", err)
		}
		outflow, err := s.TxnRepo.FindTransactionByID(tx, tr.TransactionOutflowID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find outflow transaction %w", err)
		}

		skipTxn[inflow.ID] = struct{}{}
		skipTxn[outflow.ID] = struct{}{}

		fromAcc, err := s.accService.Repo.FindAccountByID(tx, outflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find source account %w", err)
		}
		toAcc, err := s.accService.Repo.FindAccountByID(tx, inflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find destination account %w", err)
		}

		touch(fromAcc, outflow.TxnDate)
		touch(toAcc, outflow.TxnDate)

		// Reverse the balance changes
		if err := s.accService.UpdateDailyCashNoSnapshot(tx, fromAcc, outflow.TxnDate, "expense", outflow.Amount.Neg()); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.accService.UpdateDailyCashNoSnapshot(tx, toAcc, outflow.TxnDate, "income", outflow.Amount.Neg()); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Reverse regular transactions BEFORE purging
	for _, t := range txns {
		if _, ok := skipTxn[t.ID]; ok {
			continue
		}
		if t.IsTransfer {
			continue
		}

		acc, err := s.accService.Repo.FindAccountByID(tx, t.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find account %w", err)
		}

		touch(acc, t.TxnDate)

		// Reverse cash
		amt := t.Amount.Neg()
		kind := "income"
		if strings.ToLower(t.TransactionType) == "expense" {
			kind = "expense"
		}
		if err := s.accService.UpdateDailyCashNoSnapshot(tx, acc, t.TxnDate, kind, amt); err != nil {
			tx.Rollback()
			return err
		}
	}

	// hard delete the data
	if _, err := s.TxnRepo.PurgeImportedTransfers(tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := s.TxnRepo.PurgeImportedTransactions(tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Recompute balances and snapshots for all touched accounts
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, at := range touched {
		if at == nil || at.acc == nil || at.minAs.IsZero() {
			continue
		}
		if err := s.accService.FrontfillBalancesForAccount(tx, at.acc.UserID, at.acc.ID, at.acc.Currency, at.minAs); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.accService.Repo.UpsertSnapshotsFromBalances(tx, at.acc.UserID, at.acc.ID, at.acc.Currency, at.minAs, today); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Delete import row
	if err := s.Repo.DeleteImport(tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete import files
	finalPath := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), imp.Name+".json")
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

	return nil
}

func (s *ImportService) DeleteAccImport(userID int64, imp *models.Import) error {

	// Check if any transactions exist for accounts linked to this import
	var txnCount int64
	if err := s.Repo.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND account_id IN (?)", userID,
			s.Repo.DB.Model(&models.Account{}).
				Select("id").
				Where("user_id = ? AND import_id = ?", userID, imp.ID),
		).
		Count(&txnCount).Error; err != nil {
		return fmt.Errorf("failed to check transactions: %w", err)
	}

	if txnCount > 0 {
		return errors.New("account import cannot be deleted, transactions linked to same import")
	}

	tx := s.Repo.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// hard delete the data
	if err := s.accService.Repo.PurgeImportedAccounts(tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete import row
	if err := s.Repo.DeleteImport(tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete import files
	finalPath := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), imp.Name+".json")
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

	return nil
}
