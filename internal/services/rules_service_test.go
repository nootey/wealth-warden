package services_test

import (
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type RulesServiceSuite struct {
	tests.ServiceIntegrationSuite
}

func TestRulesServiceSuite(t *testing.T) {
	suite.Run(t, new(RulesServiceSuite))
}

func (s *RulesServiceSuite) assertValidation(err error) {
	status, _ := apperr.Resolve(err)
	s.Equal(http.StatusUnprocessableEntity, status)
}

func (s *RulesServiceSuite) ruleReq(field, op, value string, categoryID int64) *models.RuleReq {
	return &models.RuleReq{
		Name:       field + " " + op,
		Conditions: []models.RuleConditionReq{{Field: field, Operator: op, Value: value}},
		Actions:    []models.RuleActionReq{{ActionType: models.RuleActionSetCategory, Value: strconv.FormatInt(categoryID, 10)}},
	}
}

func (s *RulesServiceSuite) TestInsertRuleRejectsBadPairsAndForeignCategory() {
	svc := s.TC.App.RulesService
	catID, err := s.TC.App.TransactionService.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Rule Cat", Classification: "expense"})
	s.Require().NoError(err)

	_, err = svc.InsertRule(s.Ctx, seedUserID, s.ruleReq("description", "gt", "x", catID))
	s.Require().Error(err)
	s.assertValidation(err)

	_, err = svc.InsertRule(s.Ctx, seedUserID, s.ruleReq("amount", "gt", "abc", catID))
	s.Require().Error(err)
	s.assertValidation(err)

	_, err = svc.InsertRule(s.Ctx, seedUserID, s.ruleReq("description", "contains", "x", 999999))
	s.Require().Error(err)
	s.assertValidation(err)

	id, err := svc.InsertRule(s.Ctx, seedUserID, s.ruleReq("description", "contains", "spar", catID))
	s.Require().NoError(err)

	rule, err := svc.FetchRuleByID(s.Ctx, seedUserID, id)
	s.Require().NoError(err)
	s.True(rule.IsActive)
	s.Len(rule.Conditions, 1)
	s.Len(rule.Actions, 1)
}

// A bank import must categorize rows through active rules and leave the rest uncategorized.
func (s *RulesServiceSuite) TestBankImportAppliesRules() {
	s.T().Cleanup(func() { _ = os.RemoveAll("storage") })

	txnSvc := s.TC.App.TransactionService
	groceriesID, err := txnSvc.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Rule Groceries", Classification: "expense"})
	s.Require().NoError(err)
	bigID, err := txnSvc.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Rule Big Spend", Classification: "expense"})
	s.Require().NoError(err)
	offID, err := txnSvc.InsertCategory(s.Ctx, seedUserID, &models.CategoryReq{DisplayName: "Rule Inactive", Classification: "expense"})
	s.Require().NoError(err)

	rulesSvc := s.TC.App.RulesService
	_, err = rulesSvc.InsertRule(s.Ctx, seedUserID, s.ruleReq("description", "contains", "spar", groceriesID))
	s.Require().NoError(err)
	_, err = rulesSvc.InsertRule(s.Ctx, seedUserID, s.ruleReq("amount", "gte", "500", bigID))
	s.Require().NoError(err)
	inactive := s.ruleReq("description", "contains", "hofer", offID)
	off := false
	inactive.IsActive = &off
	_, err = rulesSvc.InsertRule(s.Ctx, seedUserID, inactive)
	s.Require().NoError(err)

	balance := decimal.NewFromInt(1000)
	accID, err := s.TC.App.AccountService.InsertAccount(s.Ctx, seedUserID, &models.AccountReq{
		Name:          "Rules Checking",
		AccountTypeID: checkingTypeID,
		Balance:       &balance,
		OpenedAt:      time.Now().UTC().AddDate(-2, 0, 0),
	})
	s.Require().NoError(err)

	day := time.Date(time.Now().Year()-1, 3, 10, 0, 0, 0, 0, time.UTC)
	row := func(id, desc, amount string) models.JSONTxn {
		extID := id
		return models.JSONTxn{TransactionType: "expense", Amount: amount, Currency: "EUR", TxnDate: day, Category: "(uncategorized)", Description: desc, ExternalTxnID: &extID}
	}
	payload := models.TxnImportPayload{Identifier: "nlb_rules", GeneratedAt: time.Now().UTC(), Txns: []models.JSONTxn{
		row("R1", "SPAR LJUBLJANA - nakup", "12.30"),
		row("R2", "HOFER - nakup", "8.00"),
		row("R3", "FURNITURE STORE", "750.00"),
		row("R4", "SPAR - big trip", "600.00"),
	}}

	_, err = s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, payload)
	s.Require().NoError(err)

	var txns []models.Transaction
	s.Require().NoError(s.TC.DB.Where("account_id = ? AND external_txn_id IS NOT NULL", accID).Order("external_txn_id").Find(&txns).Error)
	s.Require().Len(txns, 4)

	var uncategorized models.Category
	s.Require().NoError(s.TC.DB.Where("classification = ?", "uncategorized").First(&uncategorized).Error)

	s.Equal(groceriesID, *txns[0].CategoryID)      // description rule
	s.Equal(uncategorized.ID, *txns[1].CategoryID) // inactive rule is skipped
	s.Equal(bigID, *txns[2].CategoryID)            // amount rule
	s.Equal(groceriesID, *txns[3].CategoryID)      // first matching rule wins
}
