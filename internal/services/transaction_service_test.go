package services_test

import (
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TransactionServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestTransactionServiceSuite(t *testing.T) {
	suite.Run(t, new(TransactionServiceTestSuite))
}

func (s *TransactionServiceTestSuite) TestInsertTransaction() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account first
	initialBalance := decimal.NewFromInt(50000)
	accReq := &models.AccountReq{
		Name:           "Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err, "failed to create account")

	desc := "Test transaction"
	amount := decimal.NewFromInt(10000)

	req := &models.TransactionReq{
		AccountID:       accID,
		CategoryID:      nil,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         time.Now(),
		Description:     &desc,
	}

	_, err = svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().NoError(err)

	var got models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("user_id = ? AND account_id = ? AND amount = ? AND description = ?",
			userID, accID, req.Amount, req.Description).
		First(&got).Error
	s.Require().NoError(err)

	s.Assert().True(req.Amount.Equal(got.Amount))
}
