package services_test

import (
	"os"
	"strings"
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

func (s *ImportServiceSuite) TestParseBankStatementsRejectsMixedKinds() {
	files := []models.BankStatementFile{
		{Name: "a.csv", Reader: strings.NewReader("")},
		{Name: "b.pdf", Reader: strings.NewReader("")},
	}
	_, err := s.TC.App.ImportService.ParseBankStatements("nlb", files)
	s.Require().Error(err)
	s.Contains(err.Error(), "not both")
}
