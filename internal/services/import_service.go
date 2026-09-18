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
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/config"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/utils"

	"github.com/nootey/wealth-warden-prepper/pkg/bank"
	"github.com/nootey/wealth-warden-prepper/pkg/format"
	"github.com/nootey/wealth-warden-prepper/pkg/pdftext"
	"github.com/nootey/wealth-warden-prepper/pkg/statement"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ImportServiceInterface interface {
	ValidateCustomImport(ctx context.Context, payload *models.TxnImportPayload, step string) ([]string, int, error)
	FetchImportsByImportType(ctx context.Context, userID int64, importType string) ([]models.Import, error)
	FetchImportByID(ctx context.Context, id, userID int64, importType string) (*models.Import, error)
	ImportTransactions(ctx context.Context, userID, checkID int64, source string, payload models.TxnImportPayload) (int64, error)
	ImportAccounts(ctx context.Context, userID int64, payload models.AccImportPayload, useBalances bool) (int64, error)
	ImportCategories(ctx context.Context, userID int64, payload models.CategoryImportPayload) (int64, error)
	ImportRules(ctx context.Context, userID int64, payload models.RuleImportPayload) (int64, error)
	TransferInvestmentsFromImport(ctx context.Context, userID int64, payload models.InvestmentTransferPayload) error
	TransferSavingsFromImport(ctx context.Context, userID int64, payload models.SavingTransferPayload) error
	TransferRepaymentsFromImport(ctx context.Context, userID int64, payload models.RepaymentTransferPayload) error
	TransferInvestmentsTrades(ctx context.Context, userID int64, txnBytes []byte, payload models.InvestmentTradesPayload) error
	DeleteImport(ctx context.Context, userID, id int64) error
	ParseBankStatements(bankName string, files []models.BankStatementFile) (models.TxnImportPayload, error)
	ApplyBankRowOverrides(payload *models.TxnImportPayload, rowCategories []models.RowCategory, skipRows []int) error
}

type ImportService struct {
	logger         *zap.Logger
	repo           repositories.ImportRepositoryInterface
	txnRepo        repositories.TransactionRepositoryInterface
	accRepo        repositories.AccountRepositoryInterface
	balanceRepo    repositories.BalanceRepositoryInterface
	investmentRepo repositories.InvestmentRepositoryInterface
	settingsRepo   repositories.SettingsRepositoryInterface
	rulesRepo      repositories.RulesRepositoryInterface
	jobDispatcher  jobqueue.Dispatcher
}

func NewImportService(
	logger *zap.Logger,
	repo *repositories.ImportRepository,
	txnRepo *repositories.TransactionRepository,
	accRepo *repositories.AccountRepository,
	balanceRepo *repositories.BalanceRepository,
	investmentRepo *repositories.InvestmentRepository,
	settingsRepo *repositories.SettingsRepository,
	rulesRepo *repositories.RulesRepository,
	jobDispatcher jobqueue.Dispatcher,
) *ImportService {
	return &ImportService{
		logger:         logger,
		repo:           repo,
		txnRepo:        txnRepo,
		accRepo:        accRepo,
		balanceRepo:    balanceRepo,
		investmentRepo: investmentRepo,
		settingsRepo:   settingsRepo,
		rulesRepo:      rulesRepo,
		jobDispatcher:  jobDispatcher,
	}
}

var _ ImportServiceInterface = (*ImportService)(nil)

var (
	ErrImportFileExists       = apperr.New(apperr.Conflict, "An import with that name already exists")
	ErrInvestmentsTransferred = apperr.New(apperr.Conflict, "Investments have already been transferred for this import")
	ErrSavingsTransferred     = apperr.New(apperr.Conflict, "Savings have already been transferred for this import")
	ErrRepaymentsTransferred  = apperr.New(apperr.Conflict, "Debt repayments have already been transferred for this import")
)

func (s *ImportService) updateDailyCash(ctx context.Context, tx *gorm.DB, acc *models.Account, asOf time.Time, direction models.TransactionDirection, amt decimal.Decimal, snapshot bool) error {
	amt = amt.Round(4)
	if direction == models.TxnDirectionExpense {
		amt = amt.Neg()
	}

	if err := s.balanceRepo.ApplyDelta(ctx, tx, acc.ID, amt); err != nil {
		return err
	}

	if snapshot {
		if err := s.balanceRepo.RebuildDailyRange(
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

	if err := s.balanceRepo.RebuildBalances(ctx, tx, userID, accountID, currency, from); err != nil {
		return err
	}

	return nil
}

func (s *ImportService) applyRules(ctx context.Context, tx *gorm.DB, userID int64, rules []models.Rule, cache map[int64]models.Category, desc string, amount decimal.Decimal, direction models.TransactionDirection) (models.Category, bool, error) {
	for _, rule := range rules {
		if !rule.Matches(desc, amount, direction) {
			continue
		}
		categoryID, ok := rule.CategoryID()
		if !ok {
			continue
		}
		if c, ok := cache[categoryID]; ok {
			return c, true, nil
		}
		c, err := s.txnRepo.FindCategoryByID(ctx, tx, categoryID, userID, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return models.Category{}, false, apperr.Wrap(apperr.Internal, fmt.Sprintf("failed to find rule category %d", categoryID), err)
		}
		cache[categoryID] = c
		return c, true, nil
	}
	return models.Category{}, false, nil
}

func (s *ImportService) markImportFailed(ctx context.Context, userID, importID int64, cause error, extra ...zap.Field) {

	var appErr *apperr.Error
	isAppErr := errors.As(cause, &appErr)

	// Store only a client-safe reason. A client-facing apperr keeps its message;
	// internal or unexpected errors fall back to the generic one. The full cause
	// still reaches the logs below.
	msg := ""
	if cause != nil {
		msg = apperr.GenericMessage
		if isAppErr && appErr.Kind != apperr.Internal {
			msg = appErr.Message
		}
	}

	fields := append([]zap.Field{
		zap.Int64("user_id", userID),
		zap.Int64("import_id", importID),
	}, extra...)

	if err := s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "failed",
		"completed_at": nil,
		"error":        msg,
	}); err != nil {
		s.logger.Error("failed to mark import as failed", append(fields, zap.Error(err))...)
		return
	}

	if cause == nil {
		return
	}

	log := s.logger.Warn
	if !isAppErr || appErr.Kind == apperr.Internal {
		log = s.logger.Error
	}
	log("import failed", append(fields, zap.Error(cause))...)
}

func (s *ImportService) importFilePath(userID int64, name string) string {
	return filepath.Join("storage", "imports", fmt.Sprintf("%d", userID), name+".json")
}

func (s *ImportService) writeImportPayload(userID int64, name string, payload any) error {
	dir := filepath.Join("storage", "imports", fmt.Sprintf("%d", userID))
	finalPath := filepath.Join(dir, name+".json")
	tmpPath := finalPath + ".tmp"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(finalPath); err == nil {
		return ErrImportFileExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Exclusive temp file reserves the name against a concurrent write.
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrImportFileExists
		}
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (s *ImportService) DiscardStagedImport(ctx context.Context, userID, importID int64) error {
	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		return err
	}
	if err := os.Remove(s.importFilePath(userID, imp.Name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *ImportService) ValidateCustomImport(ctx context.Context, payload *models.TxnImportPayload, step string) ([]string, int, error) {
	if payload.GeneratedAt.IsZero() {
		return nil, 0, apperr.New(apperr.Validation, "The file is missing a valid generated_at field")
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
	case "investment_trades":
		set = payload.TradeTransfers
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
	case "investment_trades":
		allowed["buy"] = true
		allowed["sell"] = true
	default:
		allowed["income"] = true
		allowed["expense"] = true
	}

	for _, t := range set {
		if strings.TrimSpace(t.TransactionType) == "" {
			return nil, 0, apperr.New(apperr.Validation, "A row is missing its transaction_type")
		}
		tt := strings.ToLower(strings.TrimSpace(t.TransactionType))
		if !allowed[tt] {
			return nil, 0, apperr.New(apperr.Validation, "A row has a transaction_type that does not belong to this step")
		}
		if strings.TrimSpace(t.Amount) == "" {
			return nil, 0, apperr.New(apperr.Validation, "A row is missing its amount")
		}
		if t.TxnDate.IsZero() {
			return nil, 0, apperr.New(apperr.Validation, "A row is missing a valid txn_date")
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

func (s *ImportService) ParseBankStatements(bankName string, files []models.BankStatementFile) (models.TxnImportPayload, error) {
	auto := bankName == "auto"
	var parser statement.Parser
	if !auto {
		p, ok := bank.Get(bankName)
		if !ok {
			return models.TxnImportPayload{}, apperr.New(apperr.Invalid, fmt.Sprintf("Unsupported bank %q", bankName))
		}
		parser = p
	}
	if len(files) == 0 {
		return models.TxnImportPayload{}, apperr.New(apperr.Invalid, "At least one statement file is required")
	}

	kind := ""
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".csv" && ext != ".pdf" {
			return models.TxnImportPayload{}, apperr.New(apperr.Invalid, fmt.Sprintf("%s: only CSV and PDF statements are supported", f.Name))
		}
		if kind == "" {
			kind = ext
		} else if kind != ext {
			return models.TxnImportPayload{}, apperr.New(apperr.Invalid, "Upload either CSV exports or PDF statements in one import, not both")
		}
	}

	label := bankName
	var txns []statement.Transaction
	for _, f := range files {
		var parsed []statement.Transaction
		var err error
		if auto {
			var detected string
			parsed, detected, err = bank.ParseAuto(f.Reader, kind)
			if detected != "" {
				label = detected
			}
		} else if kind == ".csv" {
			parsed, err = parser.ParseCSV(f.Reader)
		} else {
			parsed, err = parser.ParsePDF(f.Reader)
		}
		if err != nil {
			if errors.Is(err, pdftext.ErrNotInstalled) {
				return models.TxnImportPayload{}, apperr.Wrap(apperr.Internal, "PDF parsing is not available on this server", err)
			}
			return models.TxnImportPayload{}, apperr.Wrap(apperr.Validation, fmt.Sprintf("%s could not be parsed", f.Name), err)
		}
		txns = append(txns, parsed...)
	}

	txns = statement.Dedupe(txns)
	sort.SliceStable(txns, func(i, j int) bool { return txns[i].Date.Before(txns[j].Date) })
	if len(txns) == 0 {
		return models.TxnImportPayload{}, apperr.New(apperr.Validation, "No transactions were found in the statements")
	}

	// Month granularity so two ranges from one year can be imported on the same day.
	first, last := txns[0].Date.Format("2006-01"), txns[len(txns)-1].Date.Format("2006-01")
	identifier := fmt.Sprintf("%s_%s", label, first)
	if first != last {
		identifier = fmt.Sprintf("%s_%s_%s", label, first, last)
	}

	now := time.Now().UTC()
	built := format.Build(identifier, txns, now)

	payload := models.TxnImportPayload{
		Identifier:  built.Identifier,
		GeneratedAt: now,
		Txns:        make([]models.JSONTxn, 0, len(built.Transactions)),
	}
	for i, t := range built.Transactions {
		var externalID *string
		if t.ExternalID != "" {
			id := t.ExternalID
			externalID = &id
		}
		payload.Txns = append(payload.Txns, models.JSONTxn{
			TransactionType: t.TransactionType,
			Amount:          t.Amount,
			Currency:        t.Currency,
			TxnDate:         txns[i].Date,
			Category:        t.Category,
			Description:     t.Description,
			ExternalTxnID:   externalID,
		})
	}
	return payload, nil
}

func (s *ImportService) ApplyBankRowOverrides(payload *models.TxnImportPayload, rowCategories []models.RowCategory, skipRows []int) error {
	for _, rc := range rowCategories {
		if rc.Row < 0 || rc.Row >= len(payload.Txns) {
			return apperr.New(apperr.Invalid, fmt.Sprintf("row_categories points at row %d, but the statements hold %d rows", rc.Row, len(payload.Txns)))
		}
		id := rc.CategoryID
		payload.Txns[rc.Row].CategoryID = &id
	}

	if len(skipRows) == 0 {
		return nil
	}

	drop := make(map[int]bool, len(skipRows))
	for _, row := range skipRows {
		if row < 0 || row >= len(payload.Txns) {
			return apperr.New(apperr.Invalid, fmt.Sprintf("skip_rows points at row %d, but the statements hold %d rows", row, len(payload.Txns)))
		}
		drop[row] = true
	}
	kept := make([]models.JSONTxn, 0, len(payload.Txns)-len(drop))
	for i, t := range payload.Txns {
		if !drop[i] {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return apperr.New(apperr.Invalid, "Every row was skipped, so there is nothing to import")
	}
	payload.Txns = kept

	return nil
}

func (s *ImportService) FetchImportsByImportType(ctx context.Context, userID int64, importType string) ([]models.Import, error) {
	return s.repo.FindImportsByImportType(ctx, nil, userID, importType)
}

func (s *ImportService) FetchImportByID(ctx context.Context, id, userID int64, importType string) (*models.Import, error) {
	return s.repo.FindImportByID(ctx, nil, id, userID, importType)
}

func (s *ImportService) ImportTransactions(ctx context.Context, userID, checkID int64, source string, payload models.TxnImportPayload) (int64, error) {

	if source != models.ImportTypeCustom && source != models.ImportTypeBank {
		return 0, apperr.New(apperr.Invalid, fmt.Sprintf("Unsupported import source %q", source))
	}

	sourceAcc, err := s.accRepo.FindAccountByID(ctx, nil, checkID, userID, false)
	if err != nil {
		return 0, err
	}
	if err := utils.ValidateAccount(sourceAcc, ""); err != nil {
		return 0, err
	}

	openedDate := sourceAcc.OpenedAt.Truncate(24 * time.Hour)

	var earliest time.Time
	for _, t := range payload.Txns {
		if !t.TxnDate.IsZero() && (earliest.IsZero() || t.TxnDate.Before(earliest)) {
			earliest = t.TxnDate
		}
	}
	for _, t := range payload.InvestmentTransfers {
		if !t.TxnDate.IsZero() && (earliest.IsZero() || t.TxnDate.Before(earliest)) {
			earliest = t.TxnDate
		}
	}
	if earliest.IsZero() {
		return 0, apperr.New(apperr.Validation, "No row carries a valid txn_date, so the import date range cannot be read")
	}
	importDate := earliest.Truncate(24 * time.Hour)

	if importDate.Before(openedDate) {
		return 0, apperr.New(apperr.Conflict, fmt.Sprintf("The account opened on %s, so it cannot take data before that date", openedDate.Format("2006-01-02")))
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("txns_%s_generated_%s", payload.Identifier, todayStr)

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return 0, err
	}

	if err := s.writeImportPayload(userID, importName, payload); err != nil {
		return 0, err
	}

	// create the import as PENDING
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
		Name:      importName,
		UserID:    userID,
		Type:      source,
		SubType:   "transactions",
		Status:    "pending",
		Step:      "cash",
		Currency:  settings.DefaultCurrency,
		StartedAt: &started,
	})
	if err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID:   importID,
		UserID:     userID,
		SubType:    "transactions",
		Source:     source,
		CheckAccID: checkID,
	}); err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		s.markImportFailed(ctx, userID, importID, err)
		return 0, err
	}

	return importID, nil
}

func (s *ImportService) RunImportTransactions(ctx context.Context, userID, importID, checkID int64, source string) error {

	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	data, err := os.ReadFile(s.importFilePath(userID, imp.Name))
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	var payload models.TxnImportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	sourceAcc, err := s.accRepo.FindAccountByID(ctx, nil, checkID, userID, false)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, importID, nil)
			panic(p)
		}
	}()

	sort.SliceStable(payload.Txns, func(i, j int) bool {
		return payload.Txns[i].TxnDate.Before(payload.Txns[j].TxnDate)
	})

	rules, err := s.rulesRepo.FindRules(ctx, tx, userID, true)
	if err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	ruleCategories := map[int64]models.Category{}

	skipped := 0
	for i, txn := range payload.Txns {

		if source == models.ImportTypeBank && txn.ExternalTxnID != nil && *txn.ExternalTxnID != "" {
			_, err := s.txnRepo.FindTransactionByExternalID(ctx, tx, sourceAcc.ID, *txn.ExternalTxnID)
			if err == nil {
				skipped++
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				tx.Rollback()
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("external_txn_id", *txn.ExternalTxnID))
				return err
			}
		}

		amount, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i))
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid amount: %q", txn.Amount), err)
		}

		txDay := utils.LocalMidnightUTC(txn.TxnDate, loc)

		var category models.Category
		var found bool

		if txn.CategoryID != nil {
			category, err = s.txnRepo.FindCategoryByID(ctx, tx, *txn.CategoryID, userID, false)
			if err != nil {
				tx.Rollback()
				failure := apperr.Wrap(apperr.Internal, fmt.Sprintf("failed to find row category %d", *txn.CategoryID), err)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					failure = ErrInvalidCategoryID
				}
				s.markImportFailed(ctx, userID, importID, failure, zap.Int("row", i), zap.Int64("category_id", *txn.CategoryID))
				return failure
			}
			found = true
		}

		for _, m := range payload.CategoryMappings {
			if found {
				break
			}
			if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(txn.Category)) {
				if m.CategoryID != nil {
					category, err = s.txnRepo.FindCategoryByID(ctx, tx, *m.CategoryID, userID, false)
					if err != nil {
						tx.Rollback()
						failure := apperr.Wrap(apperr.Internal, fmt.Sprintf("failed to find mapped category %d", *m.CategoryID), err)
						if errors.Is(err, gorm.ErrRecordNotFound) {
							failure = ErrInvalidCategoryID
						}
						s.markImportFailed(ctx, userID, importID, failure, zap.Int("row", i), zap.Int64("category_id", *m.CategoryID))
						return failure
					}
					found = true
				}
				break
			}
		}

		// Custom files carry the bank's category text as the only label; bank statements carry a real description.
		desc := txn.Category
		if source == models.ImportTypeBank && txn.Description != "" {
			desc = txn.Description
		}

		if !found {
			category, found, err = s.applyRules(ctx, tx, userID, rules, ruleCategories, desc, amount, models.TransactionDirection(txn.TransactionType))
			if err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i))
				return err
			}
		}

		// Fallback if no manual choice or rule matched
		if !found {
			category, err = s.txnRepo.EnsureRootCategory(ctx, tx, "uncategorized", userID)
			if err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i))
				return apperr.Wrap(apperr.Internal, "failed to find uncategorized category", err)
			}
		}

		if txn.TransactionType == "income" || txn.TransactionType == "expense" {

			t := models.Transaction{
				UserID:        userID,
				AccountID:     sourceAcc.ID,
				CategoryID:    &category.ID,
				Direction:     models.TransactionDirection(txn.TransactionType),
				Amount:        amount,
				Currency:      sourceAcc.Currency,
				TxnDate:       txDay,
				Description:   &desc,
				ExternalTxnID: txn.ExternalTxnID,
				ImportID:      &importID,
			}

			if _, err := s.txnRepo.InsertTransaction(ctx, tx, &t); err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", sourceAcc.ID), zap.Int64("category_id", category.ID))
				return err
			}

			if err := s.updateDailyCash(ctx, tx, sourceAcc, t.TxnDate, t.Direction, t.Amount, true); err != nil {
				tx.Rollback()
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", sourceAcc.ID))
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
		sourceAcc.Currency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, userID, importID, err, zap.Int64("account_id", sourceAcc.ID))
		return err
	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(ctx, userID, importID, err)
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
	utils.CompareChanges("", source, changes, "type")
	utils.CompareChanges("", "transactions", changes, "sub_type")
	utils.CompareChanges("", imp.Name, changes, "name")
	utils.CompareChanges("", sourceAcc.Name, changes, "source_account")
	utils.CompareChanges("", settings.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Txns)-skipped), changes, "transactions_count")
	utils.CompareChanges("", strconv.Itoa(skipped), changes, "skipped_count")

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) ImportAccounts(ctx context.Context, userID int64, payload models.AccImportPayload, useBalances bool) (int64, error) {

	accCount, err := s.accRepo.CountAccounts(ctx, nil, userID, nil, false, nil)
	if err != nil {
		return 0, err
	}

	maxAcc, err := s.settingsRepo.FetchMaxAccountsForUser(ctx, nil)
	if err != nil {
		return 0, err
	}

	if accCount >= maxAcc {
		return 0, apperr.New(apperr.Conflict, fmt.Sprintf("You can only have %d active accounts", maxAcc))
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_accounts_generated_%s", todayStr)

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	if err := s.writeImportPayload(userID, importName, payload); err != nil {
		return 0, err
	}

	// create the import as PENDING
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "accounts",
		Status:    "pending",
		Step:      "accounts",
		Currency:  settings.DefaultCurrency,
		StartedAt: &started,
	})
	if err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID:    importID,
		UserID:      userID,
		SubType:     "accounts",
		Source:      "custom",
		UseBalances: useBalances,
	}); err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		s.markImportFailed(ctx, userID, importID, err)
		return 0, err
	}

	return importID, nil
}

func (s *ImportService) RunImportAccounts(ctx context.Context, userID, importID int64, useBalances bool) error {

	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	data, err := os.ReadFile(s.importFilePath(userID, imp.Name))
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	var payload models.AccImportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return fmt.Errorf("can't fetch user settings %w", err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			s.markImportFailed(ctx, userID, importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// The opening row is user editable, and the edit form needs a category on it.
	openingCategory, err := s.txnRepo.EnsureRootCategory(ctx, tx, "adjustment", userID)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		tx.Rollback()
		return fmt.Errorf("can't find adjustment category: %w", err)
	}

	skipped := 0
	for i, acc := range payload.Accounts {

		if _, err := s.accRepo.FindAccountByName(ctx, tx, userID, acc.Name); err == nil {
			skipped++
			s.logger.Info("skipping duplicate account on import", zap.Int("row", i), zap.String("account_name", acc.Name))
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("account_name", acc.Name))
			tx.Rollback()
			return err
		}

		openedAt := acc.OpenedAt
		if openedAt.IsZero() {
			openedAt = time.Now().UTC()
		}
		openedDay := utils.LocalMidnightUTC(openedAt, loc)

		accType, err := s.accRepo.FindAccountTypeByType(ctx, tx, acc.AccountType.Type, acc.AccountType.SubType)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("account_name", acc.Name))
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperr.New(apperr.Validation, fmt.Sprintf("Account %q has an unrecognized account type (%s / %s)",
					acc.Name, acc.AccountType.Type, acc.AccountType.SubType))
			}
			return apperr.Wrap(apperr.Internal, "failed to find account type from schema", err)
		}

		account := &models.Account{
			Name:              acc.Name,
			Currency:          settings.DefaultCurrency,
			AccountTypeID:     accType.ID,
			UserID:            userID,
			ImportID:          &importID,
			OpenedAt:          openedDay,
			ExpectedBalance:   decimal.NewFromInt(0),
			BalanceProjection: "fixed",
		}

		accountID, err := s.accRepo.InsertAccount(ctx, tx, account)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("account_name", acc.Name))
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

		account.ID = accountID
		openingTxn := models.NewOpeningTransaction(userID, accountID, &openingCategory.ID, account.Currency, openedDay, amount)
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &openingTxn); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", accountID))
			tx.Rollback()
			return fmt.Errorf("failed to post the opening transaction: %w", err)
		}

		if err := s.updateDailyCash(ctx, tx, account, openedDay, openingTxn.Direction, openingTxn.Amount, true); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", accountID))
			tx.Rollback()
			return err
		}

	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(ctx, userID, importID, err)
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
	utils.CompareChanges("", imp.Name, changes, "name")
	utils.CompareChanges("", settings.DefaultCurrency, changes, "currency")
	utils.CompareChanges("", strconv.Itoa(len(payload.Accounts)-skipped), changes, "accounts_count")
	utils.CompareChanges("", strconv.Itoa(skipped), changes, "accounts_skipped_count")

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) ImportCategories(ctx context.Context, userID int64, payload models.CategoryImportPayload) (int64, error) {

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_categories_generated_%s", todayStr)

	catSettings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	if err := s.writeImportPayload(userID, importName, payload); err != nil {
		return 0, err
	}

	// create the import as PENDING
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "categories",
		Status:    "pending",
		Step:      "categories",
		Currency:  catSettings.DefaultCurrency,
		StartedAt: &started,
	})
	if err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID: importID,
		UserID:   userID,
		SubType:  "categories",
		Source:   "custom",
	}); err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		s.markImportFailed(ctx, userID, importID, err)
		return 0, err
	}

	return importID, nil
}

func (s *ImportService) RunImportCategories(ctx context.Context, userID, importID int64) error {

	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	data, err := os.ReadFile(s.importFilePath(userID, imp.Name))
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	var payload models.CategoryImportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			s.markImportFailed(ctx, userID, importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	skipped := 0
	for i, cat := range payload.Categories {

		if cat.IsDefault {

			exCat, err := s.txnRepo.FindCategoryByName(ctx, tx, cat.Name, userID)
			if err != nil {
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("category_name", cat.Name))
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
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("category_id", exCat.ID))
				tx.Rollback()
				return apperr.Wrap(apperr.Internal, fmt.Sprintf("failed to update category %d", exCat.ID), err)
			}
			continue
		}

		if existing, err := s.txnRepo.FindCategoryByName(ctx, tx, cat.Name, userID); err == nil {
			if existing.Classification == cat.Classification {
				skipped++
				s.logger.Info("skipping duplicate category on import", zap.Int("row", i), zap.String("category_name", cat.Name))
				continue
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("category_name", cat.Name))
			tx.Rollback()
			return err
		}

		parent, err := s.txnRepo.EnsureRootCategory(ctx, tx, cat.Classification, userID)
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
			if utils.IsUniqueViolation(err) {
				err = apperr.New(apperr.Conflict, fmt.Sprintf("a category named %q already exists", cat.DisplayName))
			}
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("category_name", cat.Name))
			tx.Rollback()
			return err
		}

	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(ctx, userID, importID, err)
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
	utils.CompareChanges("", imp.Name, changes, "name")
	utils.CompareChanges("", strconv.Itoa(len(payload.Categories)-skipped), changes, "categories_count")
	utils.CompareChanges("", strconv.Itoa(skipped), changes, "categories_skipped_count")

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) buildRuleFromImport(ctx context.Context, tx *gorm.DB, userID int64, r models.RuleExport) (models.Rule, error) {
	rule := models.Rule{
		UserID:        userID,
		Name:          r.Name,
		IsActive:      r.IsActive,
		MatchType:     r.MatchType,
		EffectiveDate: r.EffectiveDate,
	}

	conditions, err := buildRuleConditions(ruleConditionReqsFromExport(r.Conditions), 0)
	if err != nil {
		return rule, err
	}
	rule.Conditions = conditions

	for _, a := range r.Actions {
		value := a.Value
		if a.ActionType == models.RuleActionSetCategory {
			cat, err := s.txnRepo.FindCategoryByName(ctx, tx, a.Value, userID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return rule, apperr.New(apperr.Validation, fmt.Sprintf("The category %q from the rule %q was not found", a.Value, r.Name))
				}
				return rule, err
			}
			value = strconv.FormatInt(cat.ID, 10)
		}
		rule.Actions = append(rule.Actions, models.RuleAction{ActionType: a.ActionType, Value: value})
	}

	return rule, nil
}

// nestConditions rebuilds the group/child tree from a flat, DB-loaded condition list
// (RuleCondition.Children is never populated by gorm) so it can be compared against a
// freshly built rule's nested Conditions.
func nestConditions(flat []models.RuleCondition) []models.RuleCondition {
	byParent := map[int64][]models.RuleCondition{}
	var roots []models.RuleCondition
	for _, c := range flat {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			byParent[*c.ParentID] = append(byParent[*c.ParentID], c)
		}
	}

	var attach func(nodes []models.RuleCondition) []models.RuleCondition
	attach = func(nodes []models.RuleCondition) []models.RuleCondition {
		out := make([]models.RuleCondition, len(nodes))
		for i, n := range nodes {
			n.Children = attach(byParent[n.ID])
			out[i] = n
		}
		return out
	}
	return attach(roots)
}

func conditionSignature(c models.RuleCondition) string {
	if c.IsGroup {
		children := make([]string, 0, len(c.Children))
		for _, ch := range c.Children {
			children = append(children, conditionSignature(ch))
		}
		sort.Strings(children)
		return fmt.Sprintf("group:%s[%s]", c.MatchType, strings.Join(children, ","))
	}
	return fmt.Sprintf("leaf:%s:%s:%s", c.Field, c.Operator, c.Value)
}

func conditionsSignature(conditions []models.RuleCondition) string {
	sigs := make([]string, 0, len(conditions))
	for _, c := range conditions {
		sigs = append(sigs, conditionSignature(c))
	}
	sort.Strings(sigs)
	return strings.Join(sigs, "|")
}

func actionsSignature(actions []models.RuleAction) string {
	sigs := make([]string, 0, len(actions))
	for _, a := range actions {
		sigs = append(sigs, fmt.Sprintf("%s:%s", a.ActionType, a.Value))
	}
	sort.Strings(sigs)
	return strings.Join(sigs, "|")
}

// ruleSignature defines a duplicate rule as one with the same match type and the same
// set of conditions and actions, regardless of insertion order or row IDs.
func ruleSignature(matchType string, conditions []models.RuleCondition, actions []models.RuleAction) string {
	return fmt.Sprintf("%s|%s|%s", matchType, conditionsSignature(conditions), actionsSignature(actions))
}

func ruleConditionReqsFromExport(conditions []models.RuleConditionExport) []models.RuleConditionReq {
	out := make([]models.RuleConditionReq, 0, len(conditions))
	for _, c := range conditions {
		out = append(out, models.RuleConditionReq{
			IsGroup:    c.IsGroup,
			MatchType:  c.MatchType,
			Conditions: ruleConditionReqsFromExport(c.Conditions),
			Field:      c.Field,
			Operator:   c.Operator,
			Value:      c.Value,
		})
	}
	return out
}

func (s *ImportService) ImportRules(ctx context.Context, userID int64, payload models.RuleImportPayload) (int64, error) {

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("custom_rules_generated_%s", todayStr)

	ruleSettings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return 0, fmt.Errorf("can't fetch user settings %w", err)
	}

	if err := s.writeImportPayload(userID, importName, payload); err != nil {
		return 0, err
	}

	// create the import as PENDING
	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "rules",
		Status:    "pending",
		Step:      "rules",
		Currency:  ruleSettings.DefaultCurrency,
		StartedAt: &started,
	})
	if err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		return 0, err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID: importID,
		UserID:   userID,
		SubType:  "rules",
		Source:   "custom",
	}); err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		s.markImportFailed(ctx, userID, importID, err)
		return 0, err
	}

	return importID, nil
}

func (s *ImportService) RunImportRules(ctx context.Context, userID, importID int64) error {

	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	data, err := os.ReadFile(s.importFilePath(userID, imp.Name))
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	var payload models.RuleImportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			s.markImportFailed(ctx, userID, importID, err)
			tx.Rollback()
			panic(p)
		}
	}()

	existingRules, err := s.rulesRepo.FindRules(ctx, tx, userID, false)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		tx.Rollback()
		return err
	}
	seenSignatures := make(map[string]bool, len(existingRules))
	for _, er := range existingRules {
		seenSignatures[ruleSignature(er.MatchType, nestConditions(er.Conditions), er.Actions)] = true
	}

	skipped := 0
	for i, r := range payload.Rules {
		var rule models.Rule
		rule, err = s.buildRuleFromImport(ctx, tx, userID, r)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("rule_name", r.Name))
			tx.Rollback()
			return err
		}

		sig := ruleSignature(rule.MatchType, rule.Conditions, rule.Actions)
		if seenSignatures[sig] {
			skipped++
			s.logger.Info("skipping duplicate rule on import", zap.Int("row", i), zap.String("rule_name", r.Name))
			continue
		}

		rule.ImportID = &importID

		if _, err := s.rulesRepo.InsertRule(ctx, tx, &rule); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.String("rule_name", r.Name))
			tx.Rollback()
			return err
		}
		seenSignatures[sig] = true
	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(ctx, userID, importID, err)
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
	utils.CompareChanges("", "rules", changes, "sub_type")
	utils.CompareChanges("", imp.Name, changes, "name")
	utils.CompareChanges("", strconv.Itoa(len(payload.Rules)-skipped), changes, "rules_count")
	utils.CompareChanges("", strconv.Itoa(skipped), changes, "rules_skipped_count")

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

	imp, err := s.repo.FindImportByID(ctx, nil, payload.ImportID, userID, "custom")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}
	if imp.InvestmentsTransferred {
		return ErrInvestmentsTransferred
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID:   payload.ImportID,
		UserID:     userID,
		SubType:    "investments",
		CheckAccID: payload.CheckingAccID,
		Mappings:   payload.InvestmentMappings,
	})
}

func (s *ImportService) RunTransferInvestments(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error {

	payload := models.InvestmentTransferPayload{ImportID: importID, CheckingAccID: checkingAccID, InvestmentMappings: mappings}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return apperr.Wrap(apperr.Validation, "The source account does not exist", err)
	}
	if err := utils.ValidateAccount(checkingAcc, "source"); err != nil {
		tx.Rollback()
		return err
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.InvestmentsTransferred {
		return ErrInvestmentsTransferred
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
		s.markImportFailed(ctx, userID, payload.ImportID, err)
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
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("Destination account %d does not exist", id), err)
		}
		if err := utils.ValidateAccount(acc, "destination"); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return err
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

	for i, txn := range txnPayload.InvestmentTransfers {
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
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid amount: %q", txn.Amount), err)
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
			Direction:       "expense",
			Amount:          amt,
			Currency:        checkingAcc.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", checkingAcc.ID))
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccount.ID,
			Direction:       "income",
			Amount:          amt,
			Currency:        toAccount.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccount.ID))
			return err
		}

		transfer := models.Transfer{
			UserID:               userID,
			TransactionInflowID:  income.ID,
			TransactionOutflowID: expense.ID,
			Amount:               amt,
			Currency:             checkingAcc.Currency,
			Status:               "success",
			CreatedAt:            txDay,
			ImportID:             &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("from_account_id", checkingAcc.ID), zap.Int64("to_account_id", toAccount.ID))
			return err
		}

		if err := s.updateDailyCash(ctx, tx, checkingAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", checkingAcc.ID))
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAccount, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccount.ID))
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
		checkingAcc.Currency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", checkingAcc.ID))
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, checkingAcc.Currency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", accID))
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

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

	imp, err := s.repo.FindImportByID(ctx, nil, payload.ImportID, userID, "custom")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}
	if imp.SavingsTransferred {
		return ErrSavingsTransferred
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID:   payload.ImportID,
		UserID:     userID,
		SubType:    "savings",
		CheckAccID: payload.CheckingAccID,
		Mappings:   payload.SavingsMappings,
	})
}

func (s *ImportService) RunTransferSavings(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error {

	payload := models.SavingTransferPayload{ImportID: importID, CheckingAccID: checkingAccID, SavingsMappings: mappings}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return apperr.Wrap(apperr.Validation, "The source account does not exist", err)
	}
	if err := utils.ValidateAccount(checkingAcc, "source"); err != nil {
		tx.Rollback()
		return err
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.SavingsTransferred {
		return ErrSavingsTransferred
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
		s.markImportFailed(ctx, userID, payload.ImportID, err)
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
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("Destination account %d does not exist", id), err)
		}
		if err := utils.ValidateAccount(acc, "destination"); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return err
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

	for i, txn := range txnPayload.SavingsTransfers {
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
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid amount: %q", txn.Amount), err)
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
			Direction:       "expense",
			Amount:          amt,
			Currency:        fromAcc.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", fromAccID))
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccID,
			Direction:       "income",
			Amount:          amt,
			Currency:        toAcc.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccID))
			return err
		}

		transfer := models.Transfer{
			UserID:               userID,
			TransactionInflowID:  income.ID,
			TransactionOutflowID: expense.ID,
			Amount:               amt,
			Currency:             fromAcc.Currency,
			Status:               "success",
			CreatedAt:            txDay,
			ImportID:             &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("from_account_id", fromAccID), zap.Int64("to_account_id", toAccID))
			return err
		}

		if err := s.updateDailyCash(ctx, tx, fromAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", fromAccID))
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAcc, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccID))
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
		checkingAcc.Currency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", checkingAcc.ID))
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, checkingAcc.Currency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", accID))
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

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

	imp, err := s.repo.FindImportByID(ctx, nil, payload.ImportID, userID, "custom")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}
	if imp.RepaymentsTransferred {
		return ErrRepaymentsTransferred
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID:   payload.ImportID,
		UserID:     userID,
		SubType:    "repayments",
		CheckAccID: payload.CheckingAccID,
		Mappings:   payload.RepaymentMappings,
	})
}

func (s *ImportService) RunTransferRepayments(ctx context.Context, userID, importID, checkingAccID int64, mappings []models.TransferMapping) error {

	payload := models.RepaymentTransferPayload{ImportID: importID, CheckingAccID: checkingAccID, RepaymentMappings: mappings}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, nil)
			panic(p)
		}
	}()

	checkingAcc, err := s.accRepo.FindAccountByID(ctx, tx, payload.CheckingAccID, userID, true)
	if err != nil {
		tx.Rollback()
		return apperr.Wrap(apperr.Validation, "The source account does not exist", err)
	}
	if err := utils.ValidateAccount(checkingAcc, "source"); err != nil {
		tx.Rollback()
		return err
	}

	imp, err := s.repo.FindImportByID(ctx, tx, payload.ImportID, userID, "custom")
	if err != nil {
		return err
	}

	if imp.RepaymentsTransferred {
		return ErrRepaymentsTransferred
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
		s.markImportFailed(ctx, userID, payload.ImportID, err)
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
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("Destination account %d does not exist", id), err)
		}
		if err := utils.ValidateAccount(acc, "destination"); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", id))
			return err
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

	for i, txn := range txnPayload.RepaymentTransfers {
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
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid amount: %q", txn.Amount), err)
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
			Direction:       "expense",
			Amount:          amt,
			Currency:        fromAcc.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &expense); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", fromAccID))
			return err
		}

		income := models.Transaction{
			UserID:          userID,
			AccountID:       toAccID,
			Direction:       "income",
			Amount:          amt,
			Currency:        toAcc.Currency,
			TxnDate:         txDay,
			Description:     &desc,
			TransactionType: models.TxnTypeTransfer,
			ImportID:        &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransaction(ctx, tx, &income); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccID))
			return err
		}

		transfer := models.Transfer{
			UserID:               userID,
			TransactionInflowID:  income.ID,
			TransactionOutflowID: expense.ID,
			Amount:               amt,
			Currency:             fromAcc.Currency,
			Status:               "success",
			CreatedAt:            txDay,
			ImportID:             &imp.ID,
		}
		if _, err := s.txnRepo.InsertTransfer(ctx, tx, &transfer); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("from_account_id", fromAccID), zap.Int64("to_account_id", toAccID))
			return err
		}

		if err := s.updateDailyCash(ctx, tx, fromAcc, txDay, "expense", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", fromAccID))
			return err
		}
		if err := s.updateDailyCash(ctx, tx, toAcc, txDay, "income", amt, true); err != nil {
			tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int("row", i), zap.Int64("account_id", toAccID))
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
		checkingAcc.Currency,
		frontfillFrom,
	); err != nil {
		tx.Rollback()
		s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", checkingAcc.ID))
		return err
	}

	// Frontfill & refresh snapshots for each affected account from its earliest date
	for accID, from := range earliest {
		if err := s.frontfillBalances(ctx, tx, userID, accID, checkingAcc.Currency, from); err != nil {
			_ = tx.Rollback()
			s.markImportFailed(ctx, userID, payload.ImportID, err, zap.Int64("account_id", accID))
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

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) TransferInvestmentsTrades(ctx context.Context, userID int64, txnBytes []byte, payload models.InvestmentTradesPayload) error {

	var txnPayload models.TxnImportPayload
	if err := json.Unmarshal(txnBytes, &txnPayload); err != nil {
		return err
	}

	todayStr := time.Now().UTC().Format("2006-01-02")
	importName := fmt.Sprintf("trades_%s_generated_%s", txnPayload.Identifier, todayStr)

	settings, err := s.settingsRepo.FetchUserSettings(ctx, nil, userID)
	if err != nil {
		return err
	}

	if err := s.writeImportPayload(userID, importName, txnPayload); err != nil {
		return err
	}

	started := time.Now().UTC()

	importID, err := s.repo.InsertImport(ctx, nil, models.Import{
		Name:      importName,
		UserID:    userID,
		Type:      "custom",
		SubType:   "trades",
		Status:    "pending",
		Step:      "investments",
		Currency:  settings.DefaultCurrency,
		StartedAt: &started,
	})
	if err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		return err
	}

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ImportArgs{
		ImportID: importID,
		UserID:   userID,
		SubType:  "trades",
		Source:   "custom",
		Mappings: payload.TradeMappings,
	}); err != nil {
		_ = os.Remove(s.importFilePath(userID, importName))
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	return nil
}

func (s *ImportService) RunTransferTrades(ctx context.Context, userID, importID int64, mappings []models.TransferMapping) error {

	payload := models.InvestmentTradesPayload{TradeMappings: mappings}

	imp, err := s.FetchImportByID(ctx, importID, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	b, err := os.ReadFile(s.importFilePath(userID, imp.Name))
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}
	var txnPayload models.TxnImportPayload
	if err := json.Unmarshal(b, &txnPayload); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	tradeToAccID := make(map[string]int64, len(payload.TradeMappings))
	distinctAccIDs := make(map[int64]struct{})
	for _, m := range payload.TradeMappings {
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
		tradeToAccID[m.Name] = id
		distinctAccIDs[id] = struct{}{}
	}

	accCache := make(map[int64]*models.Account, len(distinctAccIDs))
	for id := range distinctAccIDs {
		acc, err := s.accRepo.FindAccountByID(ctx, tx, id, userID, true)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int64("account_id", id))
			_ = tx.Rollback()
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("Destination account %d does not exist", id), err)
		}
		if err := utils.ValidateAccount(acc, "destination"); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int64("account_id", id))
			_ = tx.Rollback()
			return err
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

	cfg, err := config.LoadConfig(nil)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		_ = tx.Rollback()
		return fmt.Errorf("failed to load config: %w", err)
	}
	client, err := finance.NewPriceFetchClient(cfg.FinanceAPIBaseURL)
	if err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		_ = tx.Rollback()
		return fmt.Errorf("couldn't fetch price client: %w", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i, txn := range txnPayload.TradeTransfers {

		cAccID, ok := tradeToAccID[txn.Category]
		if !ok {
			continue
		}

		toAccount, ok := accCache[cAccID]
		if !ok {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID))
			_ = tx.Rollback()
			return fmt.Errorf("account %d not cached (internal error)", cAccID)
		}

		amt, err := decimal.NewFromString(txn.Amount)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID))
			_ = tx.Rollback()
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid amount: %q", txn.Amount), err)
		}

		fee, err := decimal.NewFromString(*txn.Fee)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID))
			_ = tx.Rollback()
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid fee: %q", txn.Amount), err)
		}

		// Dates in the import JSON are UTC - extract the date component directly
		// without timezone conversion so the trade date matches the source exactly.
		t := txn.TxnDate.UTC()
		txDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		txDayAdjusted := utils.AdjustToWeekday(txDay)

		var asset models.InvestmentAsset

		// Determine investment type and format ticker
		var investmentType models.InvestmentType
		rawTicker := txn.Category
		if toAccount.AccountType.Type == "crypto" {
			investmentType = models.InvestmentCrypto
		} else {
			investmentType = models.InvestmentETF
			// This importer targets Amsterdam-listed ETFs; default the exchange when absent.
			if !strings.Contains(rawTicker, ".") {
				rawTicker = rawTicker + ".AS"
			}
		}

		formattedTicker, err := finance.NormalizeTicker(rawTicker, investmentType)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID), zap.String("ticker", rawTicker))
			_ = tx.Rollback()
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid ticker: %q", txn.Category), err)
		}

		// Check if asset exists by ticker and account
		asset, err = s.investmentRepo.FindAssetByTicker(ctx, tx, formattedTicker, cAccID, userID)
		if err != nil {
			// Asset doesn't exist, create it
			newAsset := models.InvestmentAsset{
				UserID:          userID,
				AccountID:       cAccID,
				ImportID:        &importID,
				InvestmentType:  investmentType,
				Name:            "Asset " + txn.Category,
				Ticker:          formattedTicker,
				Quantity:        decimal.Zero,
				Currency:        txn.Currency,
				AverageBuyPrice: decimal.Zero,
			}

			assetID, err := s.investmentRepo.InsertAsset(ctx, tx, &newAsset)
			if err != nil {
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID), zap.String("ticker", formattedTicker))
				_ = tx.Rollback()
				return fmt.Errorf("failed to create asset: %w", err)
			}
			newAsset.ID = assetID
			asset = newAsset
		}

		effectiveQuantity := amt
		var valueAtBuy decimal.Decimal
		var pricePerUnit decimal.Decimal

		priceData, err := client.GetAssetPriceOnDate(ctx, asset.Ticker, asset.InvestmentType, txDayAdjusted)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.String("ticker", asset.Ticker))
			_ = tx.Rollback()
			return fmt.Errorf("failed to fetch price for %s on %s: %w", asset.Ticker, txDayAdjusted.Format("2006-01-02"), err)
		}

		if txn.TradePrice != nil {
			p, err := decimal.NewFromString(*txn.TradePrice)
			if err != nil {
				s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.String("ticker", asset.Ticker))
				_ = tx.Rollback()
				return apperr.Wrap(apperr.Validation, fmt.Sprintf("A row has an invalid trade_price: %q", *txn.TradePrice), err)
			}
			pricePerUnit = p
		} else {
			pricePerUnit = decimal.NewFromFloat(priceData.Price)
		}

		if asset.InvestmentType == models.InvestmentCrypto {
			effectiveQuantity = amt.Sub(fee)
			valueAtBuy = effectiveQuantity.Mul(pricePerUnit)
		} else {
			valueAtBuy = amt.Mul(pricePerUnit)
		}

		// Fetch current price for the asset
		currentPriceData, err := client.GetAssetPrice(ctx, asset.Ticker, asset.InvestmentType)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.String("ticker", asset.Ticker))
			_ = tx.Rollback()
			return fmt.Errorf("failed to fetch current price for %s: %w", asset.Ticker, err)
		}

		currentPrice := decimal.NewFromFloat(currentPriceData.Price)

		exchangeRate := decimal.NewFromFloat(1.0)
		rate, err := client.GetExchangeRateOnDate(ctx, txn.Currency, "USD", txDayAdjusted)
		if err == nil {
			exchangeRate = decimal.NewFromFloat(rate)
		}

		var txnRealizedValue decimal.Decimal
		if models.TradeType(txn.TransactionType) == models.InvestmentSell {
			if asset.InvestmentType == models.InvestmentCrypto {
				txnRealizedValue = effectiveQuantity.Mul(pricePerUnit)
			} else {
				txnRealizedValue = effectiveQuantity.Mul(pricePerUnit).Sub(fee)
			}
			// A sell stores the basis it removes, not the sale proceeds
			valueAtBuy = asset.AverageBuyPrice.Mul(effectiveQuantity)
		}

		trade := models.InvestmentTrade{
			UserID:            userID,
			AssetID:           asset.ID,
			ImportID:          &importID,
			TxnDate:           txDayAdjusted,
			TradeType:         models.TradeType(txn.TransactionType),
			Quantity:          effectiveQuantity,
			PricePerUnit:      pricePerUnit,
			Fee:               fee,
			ValueAtBuy:        valueAtBuy,
			RealizedValue:     txnRealizedValue,
			Currency:          txn.Currency,
			ExchangeRateToUSD: exchangeRate,
			Description:       nil,
		}

		tradeID, err := s.investmentRepo.InsertInvestmentTrade(ctx, tx, &trade)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.String("ticker", asset.Ticker))
			_ = tx.Rollback()
			return fmt.Errorf("failed to insert trade: %w", err)
		}

		// Update asset after trade
		err = s.investmentRepo.UpdateAssetAfterTrade(
			ctx, tx, asset.ID, effectiveQuantity,
			models.TradeType(txn.TransactionType), valueAtBuy, fee,
		)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.Int64("trade_id", tradeID))
			_ = tx.Rollback()
			return fmt.Errorf("failed to update asset: %w", err)
		}

		// Upsert price history: trade date (historical) + today (current)
		priceEntries := []models.TickerPriceHistory{
			{Ticker: asset.Ticker, AsOf: txDayAdjusted, Price: pricePerUnit, Currency: priceData.Currency},
		}
		if !txDayAdjusted.Equal(today) {
			priceEntries = append(priceEntries, models.TickerPriceHistory{
				Ticker:   asset.Ticker,
				AsOf:     today,
				Price:    currentPrice,
				Currency: currentPriceData.Currency,
			})
		}
		if err := s.investmentRepo.UpsertTickerPrice(ctx, tx, priceEntries); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("asset_id", asset.ID), zap.String("ticker", asset.Ticker))
			_ = tx.Rollback()
			return fmt.Errorf("failed to upsert asset price history for %s: %w", asset.Ticker, err)
		}

		accCashRate := decimal.NewFromFloat(1.0)
		if txn.Currency != toAccount.Currency {
			r, rErr := client.GetExchangeRateOnDate(ctx, txn.Currency, toAccount.Currency, txDayAdjusted)
			if rErr == nil {
				accCashRate = decimal.NewFromFloat(r)
			}
		}

		tradeType := models.TradeType(txn.TransactionType)
		cashAmount := txnRealizedValue.Mul(accCashRate)
		if tradeType == models.InvestmentBuy {
			cashAmount = valueAtBuy.Add(fee).Mul(accCashRate)
		}

		cashCategory, err := s.txnRepo.EnsureRootCategory(ctx, tx, "uncategorized", userID)
		if err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID))
			_ = tx.Rollback()
			return fmt.Errorf("failed to find uncategorized category: %w", err)
		}

		cashTxn := models.NewTradeCashTransaction(userID, cAccID, &cashCategory.ID, asset.Ticker, toAccount.Currency, tradeType, txDayAdjusted, cashAmount)
		if err := linkTradeCashTransaction(ctx, tx, s.txnRepo, s.investmentRepo, tradeID, cashTxn); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int("row", i), zap.Int64("account_id", cAccID), zap.Int64("trade_id", tradeID))
			_ = tx.Rollback()
			return err
		}

		// record earliest touched date
		touch(cAccID, txDayAdjusted)
	}

	for accID, from := range earliest {
		acc := accCache[accID]

		if err := s.balanceRepo.RebuildBalances(ctx, tx, userID, accID, acc.Currency, from); err != nil {
			s.markImportFailed(ctx, userID, importID, err, zap.Int64("account_id", accID))
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	// Populate market_value on the new snapshots from the committed price history
	if err := s.balanceRepo.UpdateSnapshotMarketValues(ctx, nil, userID, nil); err != nil {
		s.markImportFailed(ctx, userID, importID, err)
		return err
	}

	if err := s.repo.UpdateImport(ctx, nil, importID, map[string]interface{}{
		"status":       "success",
		"step":         "completed",
		"completed_at": time.Now().UTC(),
		"error":        "",
	}); err != nil {
		return fmt.Errorf("marking import %d successful failed: %w", importID, err)
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", "custom", changes, "type")
	utils.CompareChanges("", "trades", changes, "sub_type")
	utils.CompareChanges("", imp.Name, changes, "name")
	utils.CompareChanges("", strconv.Itoa(len(payload.TradeMappings)), changes, "trade_mappings_count")
	utils.CompareChanges("", strconv.Itoa(len(txnPayload.TradeTransfers)), changes, "trades_imported_count")

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) DeleteImport(ctx context.Context, userID, id int64) error {

	// Both custom and bank imports are listed for deletion, so no type filter here.
	imp, err := s.FetchImportByID(ctx, id, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	// Surface a blocked delete now, not as a failed background job.
	if err := s.assertImportDeletable(ctx, userID, imp); err != nil {
		return err
	}

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ImportDeleteArgs{ImportID: id, UserID: userID})
}

func (s *ImportService) assertImportDeletable(ctx context.Context, userID int64, imp *models.Import) error {
	if imp.SubType != "accounts" {
		return nil
	}

	txnCount, err := s.repo.CountTransactionsForImport(ctx, userID, imp.ID)
	if err != nil {
		return fmt.Errorf("failed to check transactions: %w", err)
	}
	if txnCount > 0 {
		return apperr.New(apperr.Conflict, "account import cannot be deleted, transactions linked to same import")
	}
	return nil
}

func (s *ImportService) RunImportDelete(ctx context.Context, userID, id int64) error {

	// Both custom and bank imports are listed for deletion, so no type filter here.
	imp, err := s.FetchImportByID(ctx, id, userID, "")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.New(apperr.NotFound, "Import not found")
		}
		return err
	}

	switch imp.SubType {
	case "transactions":
		err = s.deleteTxnImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "accounts":
		err = s.deleteAccImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "categories":
		err = s.deleteCatImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "rules":
		err = s.deleteRulesImport(ctx, userID, imp)
		if err != nil {
			return err
		}
	case "trades":
		err = s.deleteTradesImport(ctx, userID, imp)
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

	if err := s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
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

func (s *ImportService) deleteTxnImport(ctx context.Context, userID int64, imp *models.Import) error {

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
			return apperr.Wrap(apperr.Internal, "failed to find inflow transaction for import transfer reversal", err)
		}
		outflow, err := s.txnRepo.FindTransactionByID(ctx, tx, tr.TransactionOutflowID, userID, false)
		if err != nil {
			tx.Rollback()
			return apperr.Wrap(apperr.Internal, "failed to find outflow transaction for import transfer reversal", err)
		}

		skipTxn[inflow.ID] = struct{}{}
		skipTxn[outflow.ID] = struct{}{}

		fromAcc, err := s.accRepo.FindAccountByID(ctx, tx, outflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return apperr.Wrap(apperr.Internal, "failed to find source account for import transfer reversal", err)
		}
		if err := utils.ValidateAccount(fromAcc, "source"); err != nil {
			tx.Rollback()
			return err
		}
		toAcc, err := s.accRepo.FindAccountByID(ctx, tx, inflow.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return apperr.Wrap(apperr.Internal, "failed to find destination account for import transfer reversal", err)
		}
		if err := utils.ValidateAccount(toAcc, "destination"); err != nil {
			tx.Rollback()
			return err
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
		if t.TransactionType == models.TxnTypeTransfer {
			continue
		}

		acc, err := s.accRepo.FindAccountByID(ctx, tx, t.AccountID, userID, false)
		if err != nil {
			tx.Rollback()
			return apperr.Wrap(apperr.Internal, "failed to find account for import transaction reversal", err)
		}
		if err := utils.ValidateAccount(acc, ""); err != nil {
			tx.Rollback()
			return err
		}

		touch(acc, t.TxnDate)

		// Reverse cash
		amt := t.Amount.Neg()
		if err := s.updateDailyCash(ctx, tx, acc, t.TxnDate, t.Direction, amt, false); err != nil {
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

func (s *ImportService) deleteAccImport(ctx context.Context, userID int64, imp *models.Import) error {

	// Re-check under the worker: a transaction may have linked since staging.
	if err := s.assertImportDeletable(ctx, userID, imp); err != nil {
		return err
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

func (s *ImportService) deleteRulesImport(ctx context.Context, userID int64, imp *models.Import) error {

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
	if _, err := s.rulesRepo.PurgeImportedRules(ctx, tx, imp.ID, userID); err != nil {
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

func (s *ImportService) deleteCatImport(ctx context.Context, userID int64, imp *models.Import) error {

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

	// revert names for this user's default categories
	categories, err := s.txnRepo.FindAllCategories(ctx, tx, userID, false)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, cat := range categories {
		if !cat.IsDefault {
			continue
		}
		if err := s.txnRepo.RestoreCategoryName(ctx, tx, cat.ID, userID, cat.Name); err != nil {
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

func (s *ImportService) deleteTradesImport(ctx context.Context, userID int64, imp *models.Import) error {

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

	affectedAccountIDs := map[int64]bool{}

	assets, err := s.investmentRepo.FindInvestmentAssetsByImportID(ctx, tx, imp.ID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find assets: %w", err)
	}

	for _, asset := range assets {
		affectedAccountIDs[asset.AccountID] = true
		if err := s.investmentRepo.DeleteAllTradesForAsset(ctx, tx, asset.ID, userID); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.investmentRepo.DeleteInvestmentAsset(ctx, tx, asset.ID); err != nil {
			tx.Rollback()
			return err
		}
	}

	trades, err := s.investmentRepo.FindInvestmentTradesByImportID(ctx, tx, imp.ID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find trades: %w", err)
	}

	for _, trade := range trades {
		// Capture account ID before deleting the trade.
		asset, err := s.investmentRepo.FindInvestmentAssetByID(ctx, tx, trade.AssetID, userID)
		if err == nil {
			affectedAccountIDs[asset.AccountID] = true
		}
		if err := s.investmentRepo.DeleteInvestmentTrade(ctx, tx, trade.ID); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.investmentRepo.RecalculateAssetFromTrades(ctx, tx, trade.AssetID, userID); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := s.repo.DeleteImport(ctx, tx, imp.ID, userID); err != nil {
		tx.Rollback()
		return err
	}

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

	accountIDList := make([]int64, 0, len(affectedAccountIDs))
	for id := range affectedAccountIDs {
		accountIDList = append(accountIDList, id)
	}
	return s.backfillInvestmentCashFlows(ctx, userID, accountIDList)
}

func (s *ImportService) backfillInvestmentCashFlows(ctx context.Context, userID int64, accountIDs []int64) error {
	if len(accountIDs) == 0 {
		return nil
	}

	bfTx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			bfTx.Rollback()
			panic(p)
		}
	}()

	affectedSet := make(map[int64]bool, len(accountIDs))
	for _, id := range accountIDs {
		affectedSet[id] = true
	}

	// Load affected accounts to get currency and opening date.
	allAccounts, err := s.accRepo.FindAllAccounts(ctx, bfTx, userID, true, false)
	if err != nil {
		bfTx.Rollback()
		return err
	}
	type accInfo struct {
		currency string
		opening  time.Time
	}
	affected := make(map[int64]accInfo, len(accountIDs))
	for _, acc := range allAccounts {
		if !affectedSet[acc.ID] {
			continue
		}
		opening, err := s.accRepo.GetAccountOpeningAsOf(ctx, bfTx, acc.ID)
		if err != nil {
			bfTx.Rollback()
			return fmt.Errorf("failed to get opening date for account %d: %w", acc.ID, err)
		}
		affected[acc.ID] = accInfo{currency: acc.Currency, opening: opening}
	}

	for id, info := range affected {
		if err := s.balanceRepo.RebuildBalances(ctx, bfTx, userID, id, info.currency, info.opening); err != nil {
			bfTx.Rollback()
			return err
		}
	}

	if err := bfTx.Commit().Error; err != nil {
		return err
	}

	return s.balanceRepo.UpdateSnapshotMarketValues(ctx, nil, userID, nil)
}
