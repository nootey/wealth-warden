package services_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type ImportServiceSuite struct {
	tests.ServiceIntegrationSuite
}

func TestImportServiceSuite(t *testing.T) {
	suite.Run(t, new(ImportServiceSuite))
}

func (s *ImportServiceSuite) bankPayload(identifier string, ids ...string) models.TxnImportPayload {
	lastYear := time.Date(time.Now().Year()-1, 3, 10, 0, 0, 0, 0, time.UTC)
	p := models.TxnImportPayload{Identifier: identifier, GeneratedAt: time.Now().UTC()}
	for i, id := range ids {
		extID := id
		p.Txns = append(p.Txns, models.JSONTxn{
			TransactionType: "expense",
			Amount:          "10.00",
			Currency:        "EUR",
			TxnDate:         lastYear.AddDate(0, 0, i),
			Category:        "(uncategorized)",
			Description:     "SHOP " + id,
			ExternalTxnID:   &extID,
		})
	}
	return p
}

// Re-importing a statement must skip rows the account already holds and keep the bank's description.
func (s *ImportServiceSuite) TestBankImportSkipsKnownExternalIDs() {
	// The service writes the payload under ./storage relative to the test's cwd.
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-2, 0, 0),
	})
	s.Require().NoError(err)

	id1, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, s.bankPayload("nlb_a", "TX1", "TX2"))
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, id1, accID, models.ImportTypeBank))

	id2, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, s.bankPayload("nlb_b", "TX2", "TX3"))
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, id2, accID, models.ImportTypeBank))

	// TX2 repeats across both imports, so only three unique rows must land.
	var txns []models.Transaction
	s.Require().NoError(s.TC.DB.Where("account_id = ? AND external_txn_id IS NOT NULL", accID).Order("external_txn_id").Find(&txns).Error)
	s.Require().Len(txns, 3)
	s.Equal("TX1", *txns[0].ExternalTxnID)
	s.Equal("SHOP TX1", *txns[0].Description)

	var imp models.Import
	s.Require().NoError(s.TC.DB.Where("name LIKE ?", "txns_nlb_b%").First(&imp).Error)
	s.Equal(models.ImportTypeBank, imp.Type)
}

// A transaction dated on the account's opening day must be allowed, not just the day after.
func (s *ImportServiceSuite) TestBankImportAllowsTxnOnAccountOpenDate() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	openedAt := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      openedAt,
	})
	s.Require().NoError(err)

	extID := "TX1"
	payload := models.TxnImportPayload{
		Identifier:  "nlb_open_day",
		GeneratedAt: time.Now().UTC(),
		Txns: []models.JSONTxn{{
			TransactionType: "expense",
			Amount:          "10.00",
			Currency:        "EUR",
			TxnDate:         openedAt,
			Category:        "(uncategorized)",
			Description:     "SHOP TX1",
			ExternalTxnID:   &extID,
		}},
	}

	impID, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, payload)
	s.Require().NoError(err, "a txn dated the same day the account opened must be allowed")
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, impID, accID, models.ImportTypeBank))
}

// The earliest txn_date must be found across all rows, not just the first one in the array,
// since a bank export or a hand-edited custom import isn't guaranteed to arrive sorted.
func (s *ImportServiceSuite) TestBankImportRejectsOutOfOrderTxnBeforeAccountOpen() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	openedAt := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      openedAt,
	})
	s.Require().NoError(err)

	extID1, extID2 := "TX1", "TX2"
	payload := models.TxnImportPayload{
		Identifier:  "nlb_unsorted",
		GeneratedAt: time.Now().UTC(),
		Txns: []models.JSONTxn{
			{
				TransactionType: "expense",
				Amount:          "10.00",
				Currency:        "EUR",
				TxnDate:         openedAt.AddDate(0, 0, 5),
				Description:     "SHOP TX1",
				ExternalTxnID:   &extID1,
			},
			{
				TransactionType: "expense",
				Amount:          "10.00",
				Currency:        "EUR",
				TxnDate:         openedAt.AddDate(0, 0, -1),
				Description:     "SHOP TX2",
				ExternalTxnID:   &extID2,
			},
		},
	}

	_, err = s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, payload)
	s.Require().Error(err, "a later row dated before the account opened must still be caught")
	status, _ := apperr.Resolve(err)
	s.Equal(http.StatusConflict, status)
}

// Re-importing the custom accounts export must not re-create an account the user already has.
func (s *ImportServiceSuite) TestImportAccountsSkipsDuplicateName() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	balance := decimal.NewFromInt(500)
	_, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-1, 0, 0),
	})
	s.Require().NoError(err)

	payload := models.AccImportPayload{
		GeneratedAt: time.Now().UTC(),
		Accounts: []models.AccountExport{
			{Name: "Checking", Balance: decimal.NewFromInt(999), Currency: "EUR", OpenedAt: time.Now().UTC()},
			{Name: "Savings", Balance: decimal.NewFromInt(200), Currency: "EUR", OpenedAt: time.Now().UTC()},
		},
	}
	payload.Accounts[0].AccountType.Type, payload.Accounts[0].AccountType.SubType = "cash", "checking"
	payload.Accounts[1].AccountType.Type, payload.Accounts[1].AccountType.SubType = "cash", "checking"

	accImpID, err := s.TC.App.ImportService.ImportAccounts(s.Ctx, seedUserID, payload, true)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportAccounts(s.Ctx, seedUserID, accImpID, true))

	var checkingCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Account{}).Where("user_id = ? AND name = ?", seedUserID, "Checking").Count(&checkingCount).Error)
	s.Equal(int64(1), checkingCount, "the duplicate account must not be re-created")

	var savingsCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Account{}).Where("user_id = ? AND name = ?", seedUserID, "Savings").Count(&savingsCount).Error)
	s.Equal(int64(1), savingsCount, "the new account must still be created")
}

// Re-importing the custom categories export must not re-create, or touch, a category the user already has.
func (s *ImportServiceSuite) TestImportCategoriesSkipsDuplicateNameAndClassification() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	_, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Household", Classification: "expense"})
	s.Require().NoError(err)

	payload := models.CategoryImportPayload{
		GeneratedAt: time.Now().UTC(),
		Categories: []models.CategoryExport{
			{Name: "household", DisplayName: "Household Renamed", Classification: "expense"},
			{Name: "utilities", DisplayName: "Utilities", Classification: "expense"},
		},
	}

	catImpID, err := s.TC.App.ImportService.ImportCategories(s.Ctx, seedUserID, payload)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportCategories(s.Ctx, seedUserID, catImpID))

	var household models.Category
	s.Require().NoError(s.TC.DB.Where("user_id = ? AND name = ?", seedUserID, "household").First(&household).Error)
	s.Equal("Household", household.DisplayName, "the existing category must not be overwritten by the duplicate import row")

	var householdCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Category{}).Where("user_id = ? AND name = ?", seedUserID, "household").Count(&householdCount).Error)
	s.Equal(int64(1), householdCount, "no duplicate household category must be created")

	var utilCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Category{}).Where("user_id = ? AND name = ?", seedUserID, "utilities").Count(&utilCount).Error)
	s.Equal(int64(1), utilCount, "the new category must still be created")
}

const nlbCSVHeader = "Opis/Description;Kategorija/Category;+/-;Znesek/Amount;Valuta/Currency;Datum placila/Value date;Naziv/Counter party name;Racun/Counter party Account;Status/Status;BIC koda/BIC Code;Tecaj/Foreign Exchange rate;Referenca/Creditor Reference;Datum poravnave/Settlement date;Stroski/Additional Charges;Naslov/Counter party Address;ID transakcije/Transaction ID;Namen/Purpose"

func nlbCSV(rows ...string) *strings.Reader {
	return strings.NewReader("\xEF\xBB\xBF" + nlbCSVHeader + "\r\n" + strings.Join(rows, "\r\n") + "\r\n")
}

// An inflow and an outflow export overlap on the bank ID; the merged payload must hold each row once and span both months.
func (s *ImportServiceSuite) TestParseBankStatementsMergesFiles() {
	files := []models.BankStatementFile{
		{Name: "inflow.csv", Reader: nlbCSV(
			"PLACA;;+;1000,00;EUR;10/01/2017;ACME;SI56;;;;NRC;10/01/2017;;;TX1;PLACA",
			"NADOMESTILO;;-;0,26;EUR;15/02/2017;;;;;;NRC;15/02/2017;;;TX2;NADOMESTILO",
		)},
		{Name: "outflow.csv", Reader: nlbCSV(
			"NADOMESTILO;;-;0,26;EUR;15/02/2017;;;;;;NRC;15/02/2017;;;TX2;NADOMESTILO",
			"TRGOVINA;;-;19,90;EUR;03/03/2017;SPAR;;;;;NRC;03/03/2017;;;TX3;TRGOVINA",
		)},
	}

	payload, err := s.TC.App.ImportService.ParseBankStatements("nlb", files)
	s.Require().NoError(err)
	s.Equal("nlb_2017-01_2017-03", payload.Identifier)
	s.Require().Len(payload.Txns, 3)
	s.Equal("TX1", *payload.Txns[0].ExternalTxnID)
	s.Equal("TX3", *payload.Txns[2].ExternalTxnID)
}

// Bank imports are listed next to custom ones, so deleting one must remove its rows and the import itself.
func (s *ImportServiceSuite) TestDeleteBankImport() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-2, 0, 0),
	})
	s.Require().NoError(err)

	delImpID, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, s.bankPayload("nlb_del", "TX1", "TX2"))
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, delImpID, accID, models.ImportTypeBank))

	var imp models.Import
	s.Require().NoError(s.TC.DB.Where("name LIKE ?", "txns_nlb_del%").First(&imp).Error)

	s.Require().NoError(s.TC.App.ImportService.DeleteImport(s.Ctx, seedUserID, imp.ID))
	s.Require().NoError(s.TC.App.ImportService.RunImportDelete(s.Ctx, seedUserID, imp.ID))

	var count int64
	s.Require().NoError(s.TC.DB.Model(&models.Transaction{}).Where("import_id = ?", imp.ID).Count(&count).Error)
	s.Equal(int64(0), count)
	s.Require().NoError(s.TC.DB.Model(&models.Import{}).Where("id = ?", imp.ID).Count(&count).Error)
	s.Equal(int64(0), count)

	err = s.TC.App.ImportService.DeleteImport(s.Ctx, seedUserID, imp.ID)
	s.Require().Error(err)
	status, _ := apperr.Resolve(err)
	s.Equal(http.StatusNotFound, status)
}

// A category set by hand on a row wins over a rule, a rule wins over the fallback, and the rest stay uncategorized.
func (s *ImportServiceSuite) TestBankImportRowCategoryThenRules() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	mappedCat, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Mapped", Classification: "expense"})
	s.Require().NoError(err)
	ruleCat, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Ruled", Classification: "expense"})
	s.Require().NoError(err)
	_, err = s.TC.App.RulesService.InsertRule(s.Ctx, seedUserID, &models.RuleReq{
		Name:       "spar",
		MatchType:  models.RuleMatchAll,
		Conditions: []models.RuleConditionReq{{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "spar"}},
		Actions:    []models.RuleActionReq{{ActionType: models.RuleActionSetCategory, Value: strconv.FormatInt(ruleCat, 10)}},
	})
	s.Require().NoError(err)

	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-2, 0, 0),
	})
	s.Require().NoError(err)

	payload := s.bankPayload("nlb_rules", "TX1", "TX2", "TX3")
	payload.Txns[0].CategoryID = &mappedCat
	payload.Txns[0].Description = "SPAR by hand"
	payload.Txns[1].Description = "SPAR ruled"
	payload.Txns[2].Description = "PETROL"

	rulesImpID, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, payload)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, rulesImpID, accID, models.ImportTypeBank))

	var txns []models.Transaction
	s.Require().NoError(s.TC.DB.Where("account_id = ? AND external_txn_id IS NOT NULL", accID).Order("external_txn_id").Find(&txns).Error)
	s.Require().Len(txns, 3)
	s.Equal(mappedCat, *txns[0].CategoryID)
	s.Equal(ruleCat, *txns[1].CategoryID)

	var uncategorized models.Category
	s.Require().NoError(s.TC.DB.Where("classification = ?", "uncategorized").First(&uncategorized).Error)
	s.Equal(uncategorized.ID, *txns[2].CategoryID)
}

// A rule matching on direction must only apply to rows on that side, even when the description matches too.
func (s *ImportServiceSuite) TestBankImportRuleMatchesOnDirection() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	refundCat, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Amazon Refunds", Classification: "income"})
	s.Require().NoError(err)
	_, err = s.TC.App.RulesService.InsertRule(s.Ctx, seedUserID, &models.RuleReq{
		Name:      "amazon refunds",
		MatchType: models.RuleMatchAll,
		Conditions: []models.RuleConditionReq{
			{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "amazon"},
			{Field: models.RuleFieldDirection, Operator: models.RuleOpEquals, Value: "income"},
		},
		Actions: []models.RuleActionReq{{ActionType: models.RuleActionSetCategory, Value: strconv.FormatInt(refundCat, 10)}},
	})
	s.Require().NoError(err)

	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-2, 0, 0),
	})
	s.Require().NoError(err)

	payload := s.bankPayload("nlb_direction", "TX1", "TX2")
	payload.Txns[0].Description = "AMAZON order"
	payload.Txns[0].TransactionType = "expense"
	payload.Txns[1].Description = "AMAZON refund"
	payload.Txns[1].TransactionType = "income"

	dirImpID, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, payload)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportTransactions(s.Ctx, seedUserID, dirImpID, accID, models.ImportTypeBank))

	var txns []models.Transaction
	s.Require().NoError(s.TC.DB.Where("account_id = ? AND external_txn_id IS NOT NULL", accID).Order("external_txn_id").Find(&txns).Error)
	s.Require().Len(txns, 2)

	var uncategorized models.Category
	s.Require().NoError(s.TC.DB.Where("classification = ?", "uncategorized").First(&uncategorized).Error)
	s.Equal(uncategorized.ID, *txns[0].CategoryID)
	s.Equal(refundCat, *txns[1].CategoryID)
}

func (s *ImportServiceSuite) TestParseBankStatementsRejectsMixedKinds() {
	files := []models.BankStatementFile{
		{Name: "a.csv", Reader: strings.NewReader("")},
		{Name: "b.pdf", Reader: strings.NewReader("")},
	}
	_, err := s.TC.App.ImportService.ParseBankStatements("nlb", files)
	s.Require().Error(err)
	s.Contains(err.Error(), "not both")
}

// A rule with a nested condition group must round-trip through export and import: conditions
// come back as the same tree, and the set_category action resolves by category name, not id.
func (s *ImportServiceSuite) TestExportThenImportRulesRoundTrip() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	groceriesCat, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Groceries", Classification: "expense"})
	s.Require().NoError(err)

	isActive := true
	originalRuleID, err := s.TC.App.RulesService.InsertRule(s.Ctx, seedUserID, &models.RuleReq{
		Name:      "grocery stores",
		IsActive:  &isActive,
		MatchType: models.RuleMatchAll,
		Conditions: []models.RuleConditionReq{
			{IsGroup: true, MatchType: models.RuleMatchAny, Conditions: []models.RuleConditionReq{
				{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "MERCATOR"},
				{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "SPAR"},
			}},
		},
		Actions: []models.RuleActionReq{
			{ActionType: models.RuleActionSetCategory, Value: strconv.FormatInt(groceriesCat, 10)},
		},
	})
	s.Require().NoError(err)

	export, err := s.TC.App.ExportService.CreateExport(s.Ctx, seedUserID)
	s.Require().NoError(err)
	err = s.TC.App.ExportService.RunExport(s.Ctx, export.ID, seedUserID)
	s.Require().NoError(err)
	zipData, err := s.TC.App.ExportService.DownloadExport(s.Ctx, export.ID, seedUserID)
	s.Require().NoError(err)

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	s.Require().NoError(err)
	var rulesJSON []byte
	for _, f := range zr.File {
		if f.Name != "rules.json" {
			continue
		}
		rc, err := f.Open()
		s.Require().NoError(err)
		rulesJSON, err = io.ReadAll(rc)
		s.Require().NoError(err)
		_ = rc.Close()
	}
	s.Require().NotEmpty(rulesJSON)

	var payload models.RuleImportPayload
	s.Require().NoError(json.Unmarshal(rulesJSON, &payload))

	var exported *models.RuleExport
	for i, r := range payload.Rules {
		if r.Name == "grocery stores" {
			exported = &payload.Rules[i]
		}
	}
	s.Require().NotNil(exported, "the rule created for this test must be in the export")
	s.Require().Len(exported.Conditions, 1)
	s.True(exported.Conditions[0].IsGroup)
	s.Require().Len(exported.Conditions[0].Conditions, 2)
	s.Require().Len(exported.Actions, 1)
	s.Equal("groceries", exported.Actions[0].Value) // category name, not id

	// The dedup guard skips a rule with the same conditions and actions as one that already
	// exists, so drop the original before importing its export back in.
	s.Require().NoError(s.TC.App.RulesService.DeleteRule(s.Ctx, seedUserID, originalRuleID))

	importPayload := models.RuleImportPayload{GeneratedAt: payload.GeneratedAt, Rules: []models.RuleExport{*exported}}
	ruleImpID, err := s.TC.App.ImportService.ImportRules(s.Ctx, seedUserID, importPayload)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportRules(s.Ctx, seedUserID, ruleImpID))

	var imported models.Rule
	s.Require().NoError(s.TC.DB.
		Preload("Conditions").
		Preload("Actions").
		Where("user_id = ? AND name = ? AND import_id IS NOT NULL", seedUserID, "grocery stores").
		First(&imported).Error)

	s.Require().Len(imported.Actions, 1)
	s.Equal(strconv.FormatInt(groceriesCat, 10), imported.Actions[0].Value)

	s.Require().Len(imported.Conditions, 3)
	var group models.RuleCondition
	leafCount := 0
	for _, c := range imported.Conditions {
		if c.IsGroup {
			group = c
		}
	}
	s.Require().NotZero(group.ID)
	for _, c := range imported.Conditions {
		if c.ParentID != nil && *c.ParentID == group.ID {
			leafCount++
		}
	}
	s.Equal(2, leafCount)
}

// A rule with the same conditions and actions as an existing one must be skipped on import,
// even under a different name; a genuinely different rule must still be imported.
func (s *ImportServiceSuite) TestImportRulesSkipsDuplicateConditionsAndActions() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	fuelCat, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Fuel", Classification: "expense"})
	s.Require().NoError(err)

	_, err = s.TC.App.RulesService.InsertRule(s.Ctx, seedUserID, &models.RuleReq{
		Name:       "spar rule",
		MatchType:  models.RuleMatchAll,
		Conditions: []models.RuleConditionReq{{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "spar"}},
		Actions:    []models.RuleActionReq{{ActionType: models.RuleActionSetCategory, Value: strconv.FormatInt(fuelCat, 10)}},
	})
	s.Require().NoError(err)

	payload := models.RuleImportPayload{
		GeneratedAt: time.Now().UTC(),
		Rules: []models.RuleExport{
			{
				Name:       "duplicate spar rule",
				IsActive:   true,
				MatchType:  models.RuleMatchAll,
				Conditions: []models.RuleConditionExport{{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "spar"}},
				Actions:    []models.RuleActionExport{{ActionType: models.RuleActionSetCategory, Value: "fuel"}},
			},
			{
				Name:       "petrol rule",
				IsActive:   true,
				MatchType:  models.RuleMatchAll,
				Conditions: []models.RuleConditionExport{{Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: "petrol"}},
				Actions:    []models.RuleActionExport{{ActionType: models.RuleActionSetCategory, Value: "fuel"}},
			},
		},
	}

	dupRuleImpID, err := s.TC.App.ImportService.ImportRules(s.Ctx, seedUserID, payload)
	s.Require().NoError(err)
	s.Require().NoError(s.TC.App.ImportService.RunImportRules(s.Ctx, seedUserID, dupRuleImpID))

	var dupCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Rule{}).Where("user_id = ? AND name = ?", seedUserID, "duplicate spar rule").Count(&dupCount).Error)
	s.Zero(dupCount, "a rule with the same conditions and actions must not be imported again")

	var newCount int64
	s.Require().NoError(s.TC.DB.Model(&models.Rule{}).Where("user_id = ? AND name = ?", seedUserID, "petrol rule").Count(&newCount).Error)
	s.Equal(int64(1), newCount, "a genuinely new rule must still be imported")
}

// An imported rule with a malformed condition must fail the same validation the API
// enforces on InsertRule, not silently persist a rule that can never match.
func (s *ImportServiceSuite) TestImportRulesRejectsInvalidCondition() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	payload := models.RuleImportPayload{
		GeneratedAt: time.Now().UTC(),
		Rules: []models.RuleExport{
			{
				Name:      "bad amount rule",
				IsActive:  true,
				MatchType: models.RuleMatchAll,
				Conditions: []models.RuleConditionExport{
					{Field: models.RuleFieldAmount, Operator: models.RuleOpEquals, Value: "not-a-number"},
				},
			},
		},
	}

	badRuleImpID, err := s.TC.App.ImportService.ImportRules(s.Ctx, seedUserID, payload)
	s.Require().NoError(err, "the invalid condition is caught when the job runs, not at stage time")
	err = s.TC.App.ImportService.RunImportRules(s.Ctx, seedUserID, badRuleImpID)
	s.Require().Error(err)
	status, _ := apperr.Resolve(err)
	s.Equal(http.StatusUnprocessableEntity, status)

	var count int64
	s.Require().NoError(s.TC.DB.Model(&models.Rule{}).
		Where("user_id = ? AND name = ?", seedUserID, "bad amount rule").
		Count(&count).Error)
	s.Zero(count, "an invalid rule must not be persisted")
}
