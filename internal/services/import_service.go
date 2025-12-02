package services

import (
	"context"
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
	"gorm.io/gorm"
)

type ImportServiceInterface interface {
	ValidateCustomImport(ctx context.Context, payload *models.TxnImportPayload, step string) ([]string, int, error)
	FetchImportsByImportType(ctx context.Context, userID int64, importType string) ([]models.Import, error)
	FetchImportByID(ctx context.Context, id, userID int64, importType string) (*models.Import, error)
	ImportTransactions(ctx context.Context, userID, checkID int64, payload models.TxnImportPayload) error
	ImportAccounts(ctx context.Context, userID int64, payload models.AccImportPayload, useBalances bool) error
	ImportCategories(ctx context.Context, userID int64, payload models.CategoryImportPayload) error
	TransferInvestmentsFromImport(ctx context.Context, userID int64, payload models.InvestmentTransferPayload) error
	TransferSavingsFromImport(ctx context.Context, userID int64, payload models.SavingTransferPayload) error
	TransferRepaymentsFromImport(ctx context.Context, userID int64, payload models.RepaymentTransferPayload) error
	DeleteImport(ctx context.Context, userID, id int64) error
	DeleteTxnImport(ctx context.Context, userID int64, imp *models.Import) error
	DeleteAccImport(ctx context.Context, userID int64, imp *models.Import) error
	DeleteCatImport(ctx context.Context, userID int64, imp *models.Import) error
}

type ImportService struct {
	cfg           *config.Config
	repo          repositories.ImportRepositoryInterface
	txnRepo       repositories.TransactionRepositoryInterface
	accRepo       repositories.AccountRepositoryInterface
	settingsRepo  repositories.SettingsRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobs.JobDispatcher
}

func NewImportService(
	cfg *config.Config,
	repo *repositories.ImportRepository,
	txnRepo *repositories.TransactionRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobs.JobDispatcher,
) *ImportService {
	return &ImportService{
		cfg:           cfg,
		repo:          repo,
		txnRepo:       txnRepo,
		accRepo:       accRepo,
		settingsRepo:  settingsRepo,
		loggingRepo:   loggingRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ ImportServiceInterface = (*ImportService)(nil)

func (s *ImportService) updateDailyCash(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, txnType string, amt decimal.Decimal, snapshot bool) error {
	if err := s.accRepo.EnsureDailyBalanceRow(ctx, tx, acc.ID, asOf, acc.Currency); err != nil {
		return err
	}

	amt = amt.Round(4)
	column := map[string]string{
		"expense": "cash_outflows",
		"income":  "cash_inflows",
	}[strings.ToLower(txnType)]

	err := s.accRepo.AddToDailyBalance(ctx, tx, acc.ID, asOf, column, amt)
	if err != nil {
		return err
	}

	if snapshot {
		if err := s.accRepo.UpsertSnapshotsFromBalances(
			ctx,
			tx,
			acc.UserID,
			acc.ID,
			acc.Currency,
			asOf.UTC().Truncate(24*time.Hour),
			time.Now().UTC().Truncate(24*time.Hour),
		); err != nil {
			return err
		}

	}

	return nil
}

func (s *ImportService) frontfillBalances(ctx context.Context, tx *gorm.DB, userID, accountID int64, currency string, from time.Time) error {
	from = from.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.accRepo.FrontfillBalances(ctx, tx, accountID, currency, from); err != nil {
		return err
	}

	if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accountID, currency, from, today); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) markImportFailed(ctx context.Context, importID int64, cause error) {

	msg := ""
	if cause != nil {
		msg = cause.Error()
	}

	_ = s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "failed",
		"completed_at": nil,
		"error":        msg,
	})
}

func (s *ImportService) ValidateCustomImport(ctx context.Context, payload *models.TxnImportPayload, step string) ([]string, int, error) {
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
		set = payload.InvestmentTransfers
	case "saving", "savings":
		set = payload.SavingsTransfers
	case "repayment", "repayments":
		set = payload.RepaymentTransfers
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
	case "saving", "savings":
		allowed["savings"] = true
	case "repayment", "repayments":
		allowed["repayments"] = true
	default:
		allowed["income"] = true
		allowed["expense"] = true
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

func (s *ImportService) FetchImportsByImportType(ctx context.Context, userID int64, importType string) ([]models.Import, error) {
	return s.repo.FindImportsByImportType(ctx, nil, userID, importType)
}

func (s *ImportService) FetchImportByID(ctx context.Context, id, userID int64, importType string) (*models.Import, error) {
	return s.repo.FindImportByID(ctx, nil, id, userID, importType)
}

func (s *ImportService) ImportTransactions(ctx context.Context, userID, checkID int64, payload models.TxnImportPayload) error {

	sourceAcc, err := s.accRepo.FindAccountByID(ctx, nil, checkID, userID, false)
	if err != nil {
		return err
	}

	openedYear := sourceAcc.OpenedAt.Year()

	var first time.Time
	for _, t := range payload.Txns {
		if !t.TxnDate.IsZero() {
			first = t.TxnDate
			break
		}
	}
	if first.IsZero() {
		for _, t := range payload.InvestmentTransfers {
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
	importName := fmt.Sprintf("txns_%s_generated_%s", payload.Identifier, todayStr)

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
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
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
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, importID, nil)
			panic(p)
		}
	}()

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
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

	for _, txn := range payload.Txns {

		amount, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, importID, err)
			return fmt.Errorf("invalid amount %q: %w", txn.Amount, err)
		}

		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

		var category models.Category
		var found bool

		for _, m := range payload.CategoryMappings {
			if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(txn.Category)) {
				if m.CategoryID != nil {
					category, err = s.txnRepo.FindCategoryByID(ctx, tx, *m.CategoryID, &userID, false)
					if err != nil {
						tx.Rollback()
						s.markImportFailed(ctx, importID, err)
						return fmt.Errorf(" %d: %w", *m.CategoryID, err)
					}
					found = true
				}
				break
			}
		}

		// Fallback if no mapping or category_id is nil
		if !found {
			category, err = s.txnRepo.FindCategoryByClassification(ctx, tx, "uncategorized", &userID)
			if err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, importID, err)
				return fmt.Errorf(": %w", err)
			}
		}

		if txn.TransactionType == "income" || txn.TransactionType == "expense" {

			t := models.Transaction{
				UserID:          userID,
				AccountID:       sourceAcc.ID,
				CategoryID:      &category.ID,
				TransactionType: txn.TransactionType,
				Amount:          amount,
				Currency:        models.DefaultCurrency,
				TxnDate:         txDay,
				Description:     &txn.Category,
				ImportID:        &importID,
			}

			if _, err := s.txnRepo.InsertTransaction(ctx, tx, &t); err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, importID, err)
				return err
			}

			if err := s.updateDailyCash(ctx, tx, sourceAcc, t.TxnDate, t.TransactionType, t.Amount, true); err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, importID, err)
				return err
			}

		}
	}

	frontfillFrom := utils.LocalMidnightUTC(payload.Txns[0].TxnDate, loc)

	if err := s.frontfillBalances(
		ctx,
		tx,
		userID,
		sourceAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}

	// Write payload to the reserved temp file
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Promote the temp file to final
	if err := os.Rename(tmpPath, finalPath); err != nil {
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "investments",
		"completed_at": time.Now().UTC(),
		"error":        "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "transactions", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", sourceAcc.Name, changes, "source_account")
	utils.CompareChanges("", models.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Txns)), changes, "transactions_count")

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *ImportService) ImportAccounts(ctx context.Context, userID int64, payload models.AccImportPayload, useBalances bool) error {

	accCount, err := s.accRepo.CountAccounts(ctx, nil, userID, nil, false, nil)
	if err != nil {
		return err
	}

	maxAcc, err := s.settingsRepo.FetchMaxAccountsForUser(ctx, nil)
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
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
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
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	for _, acc := range payload.Accounts {

		openedAt := acc.OpenedAt
		if openedAt.IsZero() {
			openedAt = time.Now().UTC()
		}
		openedDay := utils.LocalMidnightUTC(openedAt, loc)

		accType, err := s.accRepo.FindAccountTypeByType(ctx, tx, acc.AccountType.Type, acc.AccountType.SubType)
		if err != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			return fmt.Errorf("can't find account_type from schema %w", err)
		}

		account := &models.Account{
			Name:              acc.Name,
			Currency:          models.DefaultCurrency,
			AccountTypeID:     accType.ID,
			UserID:            userID,
			ImportID:          &importID,
			OpenedAt:          openedDay,
			ExpectedBalance:   decimal.NewFromInt(0),
			BalanceProjection: "fixed",
		}

		accountID, err := s.accRepo.InsertAccount(ctx, tx, account)
		if err != nil {
			s.markImportFailed(ctx, importID, err)
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

		_, err = s.accRepo.InsertBalance(ctx, tx, balance)
		if err != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			return err
		}

		// seed snapshots from opened day to today
		if err := s.accRepo.UpsertSnapshotsFromBalances(
			ctx,
			tx,
			userID,
			accountID,
			models.DefaultCurrency,
			asOf,
			time.Now().UTC().Truncate(24*time.Hour),
		); err != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			return err
		}

	}

	// Write payload to the reserved temp file
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Promote the temp file to final
	if err := os.Rename(tmpPath, finalPath); err != nil {
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "end",
		"completed_at": time.Now().UTC(),
		"error":        "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "accounts", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", models.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Accounts)), changes, "accounts_count")

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *ImportService) ImportCategories(ctx context.Context, userID int64, payload models.CategoryImportPayload) error {

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
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
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
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	for _, cat := range payload.Categories {

		if cat.IsDefault {

			exCat, err := s.txnRepo.FindCategoryByName(ctx, tx, cat.Name, nil)
			if err != nil {
				s.markImportFailed(ctx, importID, err)
				tx.Rollback()
				return err
			}

			upCat := models.Category{
				ID:             exCat.ID,
				DisplayName:    cat.DisplayName,
				Classification: exCat.Classification,
			}

			_, err = s.txnRepo.UpdateCategory(ctx, tx, upCat)
			if err != nil {
				s.markImportFailed(ctx, importID, err)
				tx.Rollback()
				return fmt.Errorf("can't find category from schema %w", err)
			}
			continue
		}

		parent, err := s.txnRepo.FindCategoryByName(ctx, tx, cat.Classification, &userID)
		if err != nil {
			tx.Rollback()
			return err
		}

		category := &models.Category{
			UserID:         &userID,
			Name:           cat.Name,
			DisplayName:    cat.DisplayName,
			Classification: cat.Classification,
			ParentID:       &parent.ID,
			IsDefault:      false,
			ImportID:       &importID,
		}

		_, err = s.txnRepo.InsertCategory(ctx, tx, category)
		if err != nil {
			s.markImportFailed(ctx, importID, err)
			tx.Rollback()
			return err
		}

	}

	// Write payload to the reserved temp file
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = tx.Rollback()
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// Promote the temp file to final
	if err := os.Rename(tmpPath, finalPath); err != nil {
		s.markImportFailed(ctx, importID, err)
		return err
	}

	if err := s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "end",
		"completed_at": time.Now().UTC(),
		"error":        "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	// Log
	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "categories", changes, "sub_type")
	utils.CompareChanges("", importName, changes, "name")
	utils.CompareChanges("", strconv.Itoa(len(payload.Categories)), changes, "categories_count")

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *ImportService) TransferInvestmentsFromImport(ctx context.Context, userID int64, payload models.InvestmentTransferPayload) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
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

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	sort.SliceStable(txnPayload.InvestmentTransfers, func(i, j int) bool {
		return txnPayload.InvestmentTransfers[i].TxnDate.Before(txnPayload.InvestmentTransfers[j].TxnDate)
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
		acc, err := s.accRepo.FindAccountByID(ctx, tx, id, userID, true)
		if err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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

	for _, txn := range txnPayload.InvestmentTransfers {
		if txn.TransactionType != "investments" {
			continue
		}

		// find mapped destination by category
		toAccID, ok := catToAccID[txn.Category]
		if !ok {
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
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		if err := s.updateDailyCash(ctx, tx, checkingAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAccount, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		// record earliest touched date
		touch(checkingAcc.ID, txDay)
		touch(toAccount.ID, txDay)
	}

	// frontfill balances
	frontfillFrom := utils.LocalMidnightUTC(txnPayload.Txns[0].TxnDate, loc)
	if err := s.frontfillBalances(
		ctx,
		tx,
		userID,
		checkingAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, models.DefaultCurrency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accID, models.DefaultCurrency, from, today); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
	}

	if err := s.repo.UpdateImport(ctx, tx, payload.ImportID, map[string]interface{}{
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

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
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

func (s *ImportService) TransferSavingsFromImport(ctx context.Context, userID int64, payload models.SavingTransferPayload) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.SavingsTransferred {
		return errors.New("savings have already been transferred for this import")
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

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	sort.SliceStable(txnPayload.SavingsTransfers, func(i, j int) bool {
		return txnPayload.SavingsTransfers[i].TxnDate.Before(txnPayload.SavingsTransfers[j].TxnDate)
	})

	catToAccID := make(map[string]int64, len(payload.SavingsMappings))
	distinctAccIDs := make(map[int64]struct{})
	for _, m := range payload.SavingsMappings {
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
		acc, err := s.accRepo.FindAccountByID(ctx, tx, id, userID, true)
		if err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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

	for _, txn := range txnPayload.SavingsTransfers {
		if txn.TransactionType != "savings" {
			continue
		}

		// find mapped destination by category
		cAccID, ok := catToAccID[txn.Category]
		if !ok {
			continue
		}

		toAccount, ok := accCache[cAccID]
		if !ok {
			_ = tx.Rollback()
			return fmt.Errorf("account %d not cached (internal error)", cAccID)
		}

		amt, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("invalid amount '%s': %w", txn.Amount, err)
		}

		// normalize date
		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

		// Determine transfer direction based on amount sign
		var fromAccID, toAccID int64
		var fromAcc, toAcc *models.Account

		if amt.IsNegative() {
			fromAccID = toAccount.ID
			toAccID = checkingAcc.ID
			fromAcc = toAccount
			toAcc = checkingAcc
			amt = amt.Abs()
		} else {
			fromAccID = checkingAcc.ID
			toAccID = toAccount.ID
			fromAcc = checkingAcc
			toAcc = toAccount
		}

		desc := txn.Description
		expense := models.Transaction{
			UserID:          userID,
			AccountID:       fromAccID,
			TransactionType: "expense",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccID,
			TransactionType: "income",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		if err := s.updateDailyCash(ctx, tx, fromAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAcc, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		// record earliest touched date
		touch(fromAccID, txDay)
		touch(toAccID, txDay)
	}

	// frontfill balances
	frontfillFrom := utils.LocalMidnightUTC(txnPayload.Txns[0].TxnDate, loc)
	if err := s.frontfillBalances(
		ctx,
		tx,
		userID,
		checkingAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, models.DefaultCurrency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accID, models.DefaultCurrency, from, today); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
	}

	if err := s.repo.UpdateImport(ctx, tx, payload.ImportID, map[string]interface{}{
		"status":              "success",
		"step":                "end",
		"savings_transferred": true,
		"error":               "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", payload.ImportID, err)
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", imp.Name, changes, "import_name")
	utils.CompareChanges("", checkingAcc.Name, changes, "source_account")
	utils.CompareChanges("", strconv.Itoa(len(payload.SavingsMappings)), changes, "savings_mappings_count")

	// collect destination account names for readability
	var destNames []string
	for _, acc := range accCache {
		destNames = append(destNames, acc.Name)
	}
	utils.CompareChanges("", strings.Join(destNames, ", "), changes, "destination_accounts")

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "transfer_savings",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) TransferRepaymentsFromImport(ctx context.Context, userID int64, payload models.RepaymentTransferPayload) error {

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find source account %w", err)
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.RepaymentsTransferred {
		return errors.New("debt repayments have already been transferred for this import")
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

	settings, err := s.settingsRepo.FetchUserSettings(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	sort.SliceStable(txnPayload.RepaymentTransfers, func(i, j int) bool {
		return txnPayload.RepaymentTransfers[i].TxnDate.Before(txnPayload.RepaymentTransfers[j].TxnDate)
	})

	catToAccID := make(map[string]int64, len(payload.RepaymentMappings))
	distinctAccIDs := make(map[int64]struct{})
	for _, m := range payload.RepaymentMappings {
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
		acc, err := s.accRepo.FindAccountByID(ctx, tx, id, userID, true)
		if err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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

	for _, txn := range txnPayload.RepaymentTransfers {
		if txn.TransactionType != "repayments" {
			continue
		}

		// find mapped destination by category
		cAccID, ok := catToAccID[txn.Category]
		if !ok {
			continue
		}

		toAccount, ok := accCache[cAccID]
		if !ok {
			_ = tx.Rollback()
			return fmt.Errorf("account %d not cached (internal error)", cAccID)
		}

		amt, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("invalid amount '%s': %w", txn.Amount, err)
		}

		// normalize date
		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

		// Determine transfer direction based on amount sign
		var fromAccID, toAccID int64
		var fromAcc, toAcc *models.Account

		if amt.IsNegative() {
			fromAccID = toAccount.ID
			toAccID = checkingAcc.ID
			fromAcc = toAccount
			toAcc = checkingAcc
			amt = amt.Abs()
		} else {
			fromAccID = checkingAcc.ID
			toAccID = toAccount.ID
			fromAcc = checkingAcc
			toAcc = toAccount
		}

		desc := txn.Description
		expense := models.Transaction{
			UserID:          userID,
			AccountID:       fromAccID,
			TransactionType: "expense",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccID,
			TransactionType: "income",
			Amount:          amt,
			Currency:        models.DefaultCurrency,
			TxnDate:         txDay,
			Description:     &desc,
			IsTransfer:      true,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
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
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		if err := s.updateDailyCash(ctx, tx, fromAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAcc, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}

		// record earliest touched date
		touch(fromAccID, txDay)
		touch(toAccID, txDay)
	}

	// frontfill balances
	frontfillFrom := utils.LocalMidnightUTC(txnPayload.Txns[0].TxnDate, loc)
	if err := s.frontfillBalances(
		ctx,
		tx,
		userID,
		checkingAcc.ID,
		models.DefaultCurrency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, payload.ImportID, err)
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, models.DefaultCurrency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
		if err := s.accRepo.UpsertSnapshotsFromBalances(ctx, tx, userID, accID, models.DefaultCurrency, from, today); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, payload.ImportID, err)
			return err
		}
	}

	if err := s.repo.UpdateImport(ctx, tx, payload.ImportID, map[string]interface{}{
		"status":                 "success",
		"step":                   "end",
		"repayments_transferred": true,
		"error":                  "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", payload.ImportID, err)
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", imp.Name, changes, "import_name")
	utils.CompareChanges("", checkingAcc.Name, changes, "source_account")
	utils.CompareChanges("", strconv.Itoa(len(payload.RepaymentMappings)), changes, "repayments_mappings_count")

	// collect destination account names for readability
	var destNames []string
	for _, acc := range accCache {
		destNames = append(destNames, acc.Name)
	}
	utils.CompareChanges("", strings.Join(destNames, ", "), changes, "destination_accounts")

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "transfer_repayments",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) DeleteImport(ctx context.Context, userID, id int64) error {

	imp, err := s.FetchImportByID(nil, id, userID, "custom")
	if err != nil {
		return err
	}

	switch imp.SubType {
	case "transactions":
		err = s.DeleteTxnImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "accounts":
		err = s.DeleteAccImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "categories":
		err = s.DeleteCatImport(ctx, userID, imp)
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

	if err := s.jobDispatcher.Dispatch(&jobs.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "import",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	}); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) DeleteTxnImport(ctx context.Context, userID int64, imp *models.Import) error {

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

	// Collect all transactions and transfers to reverse their balance effects
	txns, err := s.txnRepo.FindTransactionsByImportID(ctx, tx, imp.ID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find transactions: %w", err)
	}

	trs, err := s.txnRepo.FindTransfersByImportID(ctx, tx, imp.ID, userID)
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
		inflow, err := s.txnRepo.FindTransactionByID(ctx, tx, tr.TransactionInflowID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find inflow transaction %w", err)
		}
		outflow, err := s.txnRepo.FindTransactionByID(ctx, tx, tr.TransactionOutflowID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find outflow transaction %w", err)
		}

		skipTxn[inflow.ID] = struct{}{}
		skipTxn[outflow.ID] = struct{}{}

		fromAcc, err := s.accRepo.FindAccountByID(ctx, tx, outflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find source account %w", err)
		}
		toAcc, err := s.accRepo.FindAccountByID(ctx, tx, inflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("can't find destination account %w", err)
		}

		touch(fromAcc, outflow.TxnDate)
		touch(toAcc, outflow.TxnDate)

		// Reverse the balance changes
		if err := s.updateDailyCash(ctx, tx, fromAcc, outflow.TxnDate, "expense", outflow.Amount.Neg(), false); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAcc, outflow.TxnDate, "income", outflow.Amount.Neg(), false); err != nil {
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

		acc, err := s.accRepo.FindAccountByID(ctx, tx, t.AccountID, userID, false)
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
		if err := s.updateDailyCash(ctx, tx, acc, t.TxnDate, kind, amt, false); err != nil {
			tx.Rollback()
			return err
		}
	}

	// hard delete the data
	if _, err := s.txnRepo.PurgeImportedTransfers(ctx, tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := s.txnRepo.PurgeImportedTransactions(ctx, tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Recompute balances and snapshots for all touched accounts
	for _, at := range touched {
		if at == nil || at.acc == nil || at.minAs.IsZero() {
			continue
		}

		if err := s.frontfillBalances(ctx, tx, at.acc.UserID, at.acc.ID, at.acc.Currency, at.minAs); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Delete import row
	if err := s.repo.DeleteImport(ctx, tx, imp.ID, userID); err != nil {
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

func (s *ImportService) DeleteAccImport(ctx context.Context, userID int64, imp *models.Import) error {

	// Check if any transactions exist for accounts linked to this import
	txnCount, err := s.repo.CountTransactionsForImport(ctx, userID, imp.ID)
	if err != nil {
		return fmt.Errorf("failed to check transactions: %w", err)
	}

	if txnCount > 0 {
		return errors.New("account import cannot be deleted, transactions linked to same import")
	}

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

	// hard delete the data
	if err := s.accRepo.PurgeImportedAccounts(ctx, tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete import row
	if err := s.repo.DeleteImport(ctx, tx, imp.ID, userID); err != nil {
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

func (s *ImportService) DeleteCatImport(ctx context.Context, userID int64, imp *models.Import) error {

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

	// hard delete the data
	if _, err := s.txnRepo.PurgeImportedCategories(ctx, tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

	// revert names for all default categories
	categories, err := s.txnRepo.FindAllCategories(ctx, tx, nil, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, cat := range categories {
		if err := s.txnRepo.RestoreCategoryName(ctx, tx, cat.ID, &userID, cat.Name); err != nil {
			tx.Rollback()
			return err
		}
	}

	// Delete import row
	if err := s.repo.DeleteImport(ctx, tx, imp.ID, userID); err != nil {
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
