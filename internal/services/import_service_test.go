package services_test

import (
	"os"
	"testing"
	"time"
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

	skipped, err := s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, s.bankPayload("nlb_a", "TX1", "TX2"))
	s.Require().NoError(err)
	s.Equal(0, skipped)

	skipped, err = s.TC.App.ImportService.ImportTransactions(s.Ctx, seedUserID, accID, models.ImportTypeBank, s.bankPayload("nlb_b", "TX2", "TX3"))
	s.Require().NoError(err)
	s.Equal(1, skipped)

	var txns []models.Transaction
	s.Require().NoError(s.TC.DB.Where("account_id = ? AND external_txn_id IS NOT NULL", accID).Order("external_txn_id").Find(&txns).Error)
	s.Require().Len(txns, 3)
	s.Equal("TX1", *txns[0].ExternalTxnID)
	s.Equal("SHOP TX1", *txns[0].Description)

	var imp models.Import
	s.Require().NoError(s.TC.DB.Where("name LIKE ?", "txns_nlb_b%").First(&imp).Error)
	s.Equal(models.ImportTypeBank, imp.Type)
}
