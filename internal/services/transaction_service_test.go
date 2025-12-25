package services_test

import (
	"context"
	"errors"
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

// Tests transaction creation on the current date
func (s *TransactionServiceTestSuite) TestInsertTransaction_CurrentDate() {
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
	now := time.Now()

	req := &models.TransactionReq{
		AccountID:       accID,
		CategoryID:      nil,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         now,
		Description:     &desc,
	}

	txnID, err := svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().NoError(err)
	s.Assert().Greater(txnID, int64(0))

	// Verify transaction was inserted
	var transaction models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND user_id = ?", txnID, userID).
		First(&transaction).Error
	s.Require().NoError(err)
	s.Assert().Equal(accID, transaction.AccountID)
	s.Assert().Equal("expense", transaction.TransactionType)
	s.Assert().True(amount.Equal(transaction.Amount))
	s.Assert().Equal(desc, *transaction.Description)

	// Verify balance record exists for today
	todayMidnight := now.UTC().Truncate(24 * time.Hour)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balance).Error
	s.Require().NoError(err, "balance record should exist for transaction date")

	// Verify cash_outflows was updated
	s.Assert().True(amount.Equal(balance.CashOutflows),
		"cash_outflows should equal transaction amount")

	// Verify snapshot exists for today
	var snapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshot).Error
	s.Require().NoError(err, "snapshot should exist for today")
	s.Assert().Equal(userID, snapshot.UserID)
	s.Assert().Equal(accID, snapshot.AccountID)

}

// Tests transaction creation on a past date
// and verifies that snapshots are created for all days from that date to today
func (s *TransactionServiceTestSuite) TestInsertTransaction_PastDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account with opening balance 10 days ago
	daysInPast := 10
	openDate := time.Now().AddDate(0, 0, -daysInPast)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:           "Past Transaction Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err, "failed to create account")

	// Create transaction 5 days in the past
	txnDaysAgo := 5
	txnDate := time.Now().AddDate(0, 0, -txnDaysAgo)
	amount := decimal.NewFromInt(15000)
	desc := "Past transaction"

	req := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amount,
		TxnDate:         txnDate,
		Description:     &desc,
	}

	txnID, err := svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().NoError(err)
	s.Assert().Greater(txnID, int64(0))

	// Verify transaction and related records were inserted
	var transaction models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txnID).
		First(&transaction).Error
	s.Require().NoError(err)

	txnMidnight := txnDate.UTC().Truncate(24 * time.Hour)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&balance).Error
	s.Require().NoError(err, "balance record should exist for transaction date")

	s.Assert().True(amount.Equal(balance.CashInflows),
		"cash_inflows should equal transaction amount")

	// Verify snapshots exist for all days from transaction date to today
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshots []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			accID, txnMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&snapshots).Error
	s.Require().NoError(err)

	expectedSnapshotCount := txnDaysAgo + 1
	s.Assert().Equal(expectedSnapshotCount, len(snapshots),
		"should have snapshot for each day from transaction date to today (inclusive)")

	// Verify snapshots are consecutive days
	for i, snapshot := range snapshots {
		expectedDate := txnMidnight.AddDate(0, 0, i)
		s.Assert().Equal(expectedDate, snapshot.AsOf.UTC().Truncate(24*time.Hour),
			"snapshot %d should be for date %s", i, expectedDate.Format("2006-01-02"))
		s.Assert().Equal(userID, snapshot.UserID)
		s.Assert().Equal(accID, snapshot.AccountID)
	}
}

// Tests multiple transactions on the same day
func (s *TransactionServiceTestSuite) TestInsertTransaction_SameDayMultiple() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	openDate := time.Now().AddDate(0, 0, -5)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:           "Same Day Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create multiple transactions on the same day
	txnDate := time.Now().AddDate(0, 0, -2)

	amt1 := decimal.NewFromInt(1000)
	desc1 := "Expense 1"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt1,
		TxnDate:         txnDate,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense 2"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txnDate,
		Description:     &desc2,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	amt3 := decimal.NewFromInt(5000)
	desc3 := "Income"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txnDate,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Verify balance accumulates all transactions correctly
	txnMidnight := txnDate.UTC().Truncate(24 * time.Hour)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&balance).Error
	s.Require().NoError(err)

	expectedOutflows := amt1.Add(amt2)
	s.Assert().True(expectedOutflows.Equal(balance.CashOutflows),
		"cash_outflows should be sum of both expenses: expected %s, got %s",
		expectedOutflows.String(), balance.CashOutflows.String())

	s.Assert().True(amt3.Equal(balance.CashInflows),
		"cash_inflows should equal income amount: expected %s, got %s",
		amt3.String(), balance.CashInflows.String())

	// Verify only one balance record exists for that day
	var balanceCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Balance{}).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		Count(&balanceCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), balanceCount, "should have exactly one balance record for the day")

	// Verify all 3 transactions were inserted
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ? AND txn_date >= ? AND txn_date < ?",
			accID, txnMidnight, txnMidnight.AddDate(0, 0, 1)).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(3), txnCount, "should have 3 transactions for that day")
}

// Tests that transactions before opening date are rejected
func (s *TransactionServiceTestSuite) TestInsertTransaction_BeforeOpeningDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	openDate := time.Now().AddDate(0, 0, -5)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Date Check Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Try to create transaction 2 days before opening date
	txnDate := openDate.AddDate(0, 0, -2)
	amount := decimal.NewFromInt(1000)
	desc := "Too early transaction"

	req := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         txnDate,
		Description:     &desc,
	}

	_, err = svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().Error(err, "should reject transaction before opening date")
	s.Assert().Contains(err.Error(), "cannot be before account opening date",
		"error should mention opening date restriction")

	// Verify no transaction was created
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ?", accID).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount, "no transaction should have been inserted")
}

// Tests that future transactions are rejected
func (s *TransactionServiceTestSuite) TestInsertTransaction_FutureDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Future Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Try to create transaction 5 days in the future
	futureDate := time.Now().AddDate(0, 0, 5)
	amount := decimal.NewFromInt(1000)
	desc := "Future transaction"

	req := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         futureDate,
		Description:     &desc,
	}

	_, err = svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().Error(err, "should reject future transaction")
	s.Assert().Contains(err.Error(), "cannot be in the future",
		"error should mention future date restriction")

	// Verify no transaction was created
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ?", accID).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount, "no transaction should have been inserted")
}

// Tests that snapshot end_balance values are correctly calculated
// and forward-filled when transactions are added
func (s *TransactionServiceTestSuite) TestInsertTransaction_SnapshotValuesCorrect() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Snapshot Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Transaction 1: Add income of 5,000 on day -8 (8 days ago)
	// Expected balance after: 10,000 + 5,000 = 15,000
	txn1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(5000)
	desc1 := "Income 8 days ago"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transaction 2: Add expense of 2,000 on day -5 (5 days ago)
	// Expected balance after: 15,000 - 2,000 = 13,000
	txn2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense 5 days ago"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transaction 3: Add income of 3,000 on day -2 (2 days ago)
	// Expected balance after: 13,000 + 3,000 = 16,000
	txn3Date := time.Now().AddDate(0, 0, -2)
	amt3 := decimal.NewFromInt(3000)
	desc3 := "Income 2 days ago"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txn3Date,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Now verify snapshot values are correct for key dates
	openMidnight := openDate.UTC().Truncate(24 * time.Hour)
	txn1Midnight := txn1Date.UTC().Truncate(24 * time.Hour)
	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)
	txn3Midnight := txn3Date.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Check snapshot on opening day (day -10): should be 10,000
	var snapshot1 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, openMidnight).
		First(&snapshot1).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshot1.EndBalance),
		"Opening day snapshot should be %s, got %s",
		initialBalance.String(), snapshot1.EndBalance.String())

	// Check snapshot on txn1 day (day -8): should be 15,000
	var snapshot2 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn1Midnight).
		First(&snapshot2).Error
	s.Require().NoError(err)
	expectedAfterTxn1 := initialBalance.Add(amt1)
	s.Assert().True(expectedAfterTxn1.Equal(snapshot2.EndBalance),
		"Day -8 snapshot should be %s, got %s",
		expectedAfterTxn1.String(), snapshot2.EndBalance.String())

	// Check snapshot on day -7 (between txn1 and txn2): should still be 15,000
	day7Midnight := txn1Midnight.AddDate(0, 0, 1)
	var snapshot3 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, day7Midnight).
		First(&snapshot3).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfterTxn1.Equal(snapshot3.EndBalance),
		"Day -7 snapshot (no txn) should be forward-filled to %s, got %s",
		expectedAfterTxn1.String(), snapshot3.EndBalance.String())

	// Check snapshot on txn2 day (day -5): should be 13,000
	var snapshot4 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshot4).Error
	s.Require().NoError(err)
	expectedAfterTxn2 := expectedAfterTxn1.Sub(amt2)
	s.Assert().True(expectedAfterTxn2.Equal(snapshot4.EndBalance),
		"Day -5 snapshot should be %s, got %s",
		expectedAfterTxn2.String(), snapshot4.EndBalance.String())

	// Check snapshot on txn3 day (day -2): should be 16,000
	var snapshot5 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshot5).Error
	s.Require().NoError(err)
	expectedAfterTxn3 := expectedAfterTxn2.Add(amt3)
	s.Assert().True(expectedAfterTxn3.Equal(snapshot5.EndBalance),
		"Day -2 snapshot should be %s, got %s",
		expectedAfterTxn3.String(), snapshot5.EndBalance.String())

	// Check snapshot today: should still be 16,000
	var snapshot6 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshot6).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfterTxn3.Equal(snapshot6.EndBalance),
		"Today's snapshot should be forward-filled to %s, got %s",
		expectedAfterTxn3.String(), snapshot6.EndBalance.String())
}

// Tests inserting an older transaction after newer transactions already exist,
// verifying backward updates work correctly
func (s *TransactionServiceTestSuite) TestInsertTransaction_BackfillBehavior() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Backfill Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert transaction on day -3 - expense of 2,000
	// Expected balance after: 8,000
	txn1Date := time.Now().AddDate(0, 0, -3)
	amt1 := decimal.NewFromInt(2000)
	desc1 := "Recent expense"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	txn1Midnight := txn1Date.UTC().Truncate(24 * time.Hour)
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn1Midnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedAfterTxn1 := initialBalance.Sub(amt1)
	s.Assert().True(expectedAfterTxn1.Equal(snapshotBefore.EndBalance),
		"Before backfill: day -3 should be %s, got %s",
		expectedAfterTxn1.String(), snapshotBefore.EndBalance.String())

	// Insert older transaction on day -7 - income of 5,000
	// This should backfill and update all snapshots from day -7 onwards
	// Day -10: 10,000 (opening)
	// Day -7: 10,000 + 5,000 = 15,000 (new transaction)
	// Day -3: 15,000 - 2,000 = 13,000 (existing transaction, should be recalculated)
	txn2Date := time.Now().AddDate(0, 0, -7)
	amt2 := decimal.NewFromInt(5000)
	desc2 := "Backdated income"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)
	var balance2 models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&balance2).Error
	s.Require().NoError(err)
	s.Assert().True(amt2.Equal(balance2.CashInflows),
		"Day -7 balance should have cash_inflows of %s", amt2.String())

	var snapshot2 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshot2).Error
	s.Require().NoError(err)
	expectedAfterTxn2 := initialBalance.Add(amt2)
	s.Assert().True(expectedAfterTxn2.Equal(snapshot2.EndBalance),
		"After backfill: day -7 should be %s, got %s",
		expectedAfterTxn2.String(), snapshot2.EndBalance.String())

	// Verify snapshot on day -3 was updated to 13,000 (not still 8,000)
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn1Midnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	expectedAfterBackfill := expectedAfterTxn2.Sub(amt1)
	s.Assert().True(expectedAfterBackfill.Equal(snapshotAfter.EndBalance),
		"After backfill: day -3 should be updated to %s, got %s",
		expectedAfterBackfill.String(), snapshotAfter.EndBalance.String())

	// Verify a day between the two transactions (day -5) was also updated
	day5Midnight := txn2Midnight.AddDate(0, 0, 2)
	var snapshot5 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, day5Midnight).
		First(&snapshot5).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfterTxn2.Equal(snapshot5.EndBalance),
		"Day -5 (between transactions) should be %s, got %s",
		expectedAfterTxn2.String(), snapshot5.EndBalance.String())

	// Verify today's snapshot is also updated correctly
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfterBackfill.Equal(snapshotToday.EndBalance),
		"Today should reflect all transactions: %s, got %s",
		expectedAfterBackfill.String(), snapshotToday.EndBalance.String())
}

// Tests deleting a transaction on the current date
func (s *TransactionServiceTestSuite) TestDeleteTransaction_CurrentDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Delete Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	amount := decimal.NewFromInt(3000)
	desc := "Expense to delete"
	req := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         time.Now(),
		Description:     &desc,
	}
	txnID, err := svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().NoError(err)

	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var balanceBefore models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balanceBefore).Error
	s.Require().NoError(err)
	s.Assert().True(amount.Equal(balanceBefore.CashOutflows),
		"Before delete: cash_outflows should be %s", amount.String())

	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedBefore := initialBalance.Sub(amount)
	s.Assert().True(expectedBefore.Equal(snapshotBefore.EndBalance),
		"Before delete: snapshot should be %s, got %s",
		expectedBefore.String(), snapshotBefore.EndBalance.String())

	// delete the transaction
	err = svc.DeleteTransaction(s.Ctx, userID, txnID)
	s.Require().NoError(err)

	// Verify transaction is soft-deleted
	var deletedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", txnID).
		First(&deletedTxn).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedTxn.DeletedAt, "transaction should be soft-deleted")

	// Verify balance outflow was reversed back to 0
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balanceAfter.CashOutflows),
		"After delete: cash_outflows should be 0, got %s",
		balanceAfter.CashOutflows.String())

	// Verify snapshot is back to initial balance of 10,000
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshotAfter.EndBalance),
		"After delete: snapshot should be back to %s, got %s",
		initialBalance.String(), snapshotAfter.EndBalance.String())
}

// Tests deleting a transaction in the past that sits between other transactions,
// verifying snapshots are recalculated
func (s *TransactionServiceTestSuite) TestDeleteTransaction_PastDateMiddle() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Delete Middle Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert 3 transactions at different dates
	// Transaction 1: Day -8, income of 5,000 -> balance = 15,000
	txn1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(5000)
	desc1 := "Income day -8"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transaction 2: Day -5, expense of 2,000 -> balance = 13,000
	txn2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense day -5 (to delete)"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	txn2ID, err := svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transaction 3: Day -2, income of 3,000 -> balance = 16,000
	txn3Date := time.Now().AddDate(0, 0, -2)
	amt3 := decimal.NewFromInt(3000)
	desc3 := "Income day -2"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txn3Date,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Verify snapshots before deletion
	txn1Midnight := txn1Date.UTC().Truncate(24 * time.Hour)
	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)
	txn3Midnight := txn3Date.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Before delete: verify day -5 snapshot is 13,000
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedBeforeDelete := initialBalance.Add(amt1).Sub(amt2)
	s.Assert().True(expectedBeforeDelete.Equal(snapshotBefore.EndBalance),
		"Before delete: day -5 should be %s, got %s",
		expectedBeforeDelete.String(), snapshotBefore.EndBalance.String())

	// Before delete: verify day -2 snapshot is 16,000
	var snapshot3Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshot3Before).Error
	s.Require().NoError(err)
	expectedDay2Before := expectedBeforeDelete.Add(amt3)
	s.Assert().True(expectedDay2Before.Equal(snapshot3Before.EndBalance),
		"Before delete: day -2 should be %s, got %s",
		expectedDay2Before.String(), snapshot3Before.EndBalance.String())

	// Delete the middle transaction (day -5)
	err = svc.DeleteTransaction(s.Ctx, userID, txn2ID)
	s.Require().NoError(err)

	// Verify transaction 2 is soft-deleted
	var deletedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", txn2ID).
		First(&deletedTxn).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedTxn.DeletedAt)

	// After delete: verify balance on day -5 has cash_outflows = 0
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&balanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balanceAfter.CashOutflows),
		"After delete: day -5 cash_outflows should be 0, got %s",
		balanceAfter.CashOutflows.String())

	// After delete: verify day -5 snapshot is now 15,000
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	expectedAfterDelete := initialBalance.Add(amt1)
	s.Assert().True(expectedAfterDelete.Equal(snapshotAfter.EndBalance),
		"After delete: day -5 should be %s, got %s",
		expectedAfterDelete.String(), snapshotAfter.EndBalance.String())

	// Verify day -2 snapshot was also updated to 18,000
	var snapshot3After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshot3After).Error
	s.Require().NoError(err)
	expectedDay2After := expectedAfterDelete.Add(amt3)
	s.Assert().True(expectedDay2After.Equal(snapshot3After.EndBalance),
		"After delete: day -2 should be updated to %s, got %s",
		expectedDay2After.String(), snapshot3After.EndBalance.String())

	// Verify today's snapshot is also updated to 18,000
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDay2After.Equal(snapshotToday.EndBalance),
		"After delete: today should be %s, got %s",
		expectedDay2After.String(), snapshotToday.EndBalance.String())

	// Verify day -8 snapshot remains unchanged at 15,000 (transaction before deleted one)
	var snapshot1After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn1Midnight).
		First(&snapshot1After).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfterDelete.Equal(snapshot1After.EndBalance),
		"After delete: day -8 should remain %s, got %s",
		expectedAfterDelete.String(), snapshot1After.EndBalance.String())
}

// Tests updating a transaction's amount while keeping it on the same date,
// verifying snapshots recalculate correctly
func (s *TransactionServiceTestSuite) TestUpdateTransaction_SameDateAmountChange() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Update Test Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert 3 transactions
	// Transaction 1: Day -8, income of 5,000 -> balance = 15,000
	txn1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(5000)
	desc1 := "Income day -8"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transaction 2: Day -5, expense of 2,000 -> balance = 13,000
	txn2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense day -5 (to update)"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	txn2ID, err := svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transaction 3: Day -2, income of 3,000 -> balance = 16,000
	txn3Date := time.Now().AddDate(0, 0, -2)
	amt3 := decimal.NewFromInt(3000)
	desc3 := "Income day -2"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txn3Date,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Verify snapshots before update
	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)
	txn3Midnight := txn3Date.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Before update: day -5 should be 13,000
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedBefore := initialBalance.Add(amt1).Sub(amt2)
	s.Assert().True(expectedBefore.Equal(snapshotBefore.EndBalance),
		"Before update: day -5 should be %s, got %s",
		expectedBefore.String(), snapshotBefore.EndBalance.String())

	// Before update: day -2 should be 16,000
	var snapshot3Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshot3Before).Error
	s.Require().NoError(err)
	expectedDay2Before := expectedBefore.Add(amt3)
	s.Assert().True(expectedDay2Before.Equal(snapshot3Before.EndBalance),
		"Before update: day -2 should be %s, got %s",
		expectedDay2Before.String(), snapshot3Before.EndBalance.String())

	// update transaction 2: change amount from 2,000 to 4,000 (still expense, same date)
	newAmt2 := decimal.NewFromInt(4000)
	newDesc2 := "Updated expense day -5"
	updateReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          newAmt2,
		TxnDate:         txn2Date,
		Description:     &newDesc2,
	}
	_, err = svc.UpdateTransaction(s.Ctx, userID, txn2ID, updateReq)
	s.Require().NoError(err)

	// Verify transaction was updated
	var updatedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txn2ID).
		First(&updatedTxn).Error
	s.Require().NoError(err)
	s.Assert().True(newAmt2.Equal(updatedTxn.Amount),
		"Transaction amount should be updated to %s", newAmt2.String())
	s.Assert().Equal(newDesc2, *updatedTxn.Description)

	// Verify balance on day -5: should have 4,000 in outflows (not 2,000)
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&balanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(newAmt2.Equal(balanceAfter.CashOutflows),
		"After update: day -5 cash_outflows should be %s, got %s",
		newAmt2.String(), balanceAfter.CashOutflows.String())

	// After update: day -5 snapshot should be 11,000 (15,000 - 4,000)
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	expectedAfter := initialBalance.Add(amt1).Sub(newAmt2)
	s.Assert().True(expectedAfter.Equal(snapshotAfter.EndBalance),
		"After update: day -5 should be %s, got %s",
		expectedAfter.String(), snapshotAfter.EndBalance.String())

	// Verify day -2 snapshot was updated to 14,000 (11,000 + 3,000) not 16,000
	var snapshot3After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshot3After).Error
	s.Require().NoError(err)
	expectedDay2After := expectedAfter.Add(amt3)
	s.Assert().True(expectedDay2After.Equal(snapshot3After.EndBalance),
		"After update: day -2 should be updated to %s, got %s",
		expectedDay2After.String(), snapshot3After.EndBalance.String())

	// Verify today's snapshot is also updated to 14,000
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDay2After.Equal(snapshotToday.EndBalance),
		"After update: today should be %s, got %s",
		expectedDay2After.String(), snapshotToday.EndBalance.String())
}

// Tests moving a transaction to a later date and verifying that old date snapshots
// revert and new date gets the transaction
func (s *TransactionServiceTestSuite) TestUpdateTransaction_ChangeDateToLater() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Date Change Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert 3 transactions
	// Transaction 1: Day -8, income of 5,000 -> balance becomes 15,000
	txn1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(5000)
	desc1 := "Income day -8"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transaction 2: Day -5, expense of 2,000 -> balance becomes 13,000
	txn2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense to move"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	txn2ID, err := svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transaction 3: Day -3, income of 3,000 -> balance becomes 16,000
	txn3Date := time.Now().AddDate(0, 0, -3)
	amt3 := decimal.NewFromInt(3000)
	desc3 := "Income day -3"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txn3Date,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Before update: verify timeline is 10k -> 15k -> 13k -> 16k
	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)
	txn3Midnight := txn3Date.UTC().Truncate(24 * time.Hour)

	var snapshotDay5Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotDay5Before).Error
	s.Require().NoError(err)
	expectedDay5Before := initialBalance.Add(amt1).Sub(amt2)
	s.Assert().True(expectedDay5Before.Equal(snapshotDay5Before.EndBalance),
		"Before update: day -5 should be %s", expectedDay5Before.String())

	var snapshotDay3Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshotDay3Before).Error
	s.Require().NoError(err)
	expectedDay3Before := expectedDay5Before.Add(amt3)
	s.Assert().True(expectedDay3Before.Equal(snapshotDay3Before.EndBalance),
		"Before update: day -3 should be %s", expectedDay3Before.String())

	// Move transaction 2 from day -5 to day -2 (later)
	newTxn2Date := time.Now().AddDate(0, 0, -2)
	updateReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         newTxn2Date,
		Description:     &desc2,
	}
	_, err = svc.UpdateTransaction(s.Ctx, userID, txn2ID, updateReq)
	s.Require().NoError(err)

	// Verify transaction date was updated
	var updatedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txn2ID).
		First(&updatedTxn).Error
	s.Require().NoError(err)
	newTxn2Midnight := newTxn2Date.UTC().Truncate(24 * time.Hour)
	s.Assert().Equal(newTxn2Midnight, updatedTxn.TxnDate.UTC().Truncate(24*time.Hour))

	// After update: verify old date (day -5) balance has 0 outflows
	var balanceOldDate models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&balanceOldDate).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balanceOldDate.CashOutflows),
		"After update: old date (day -5) should have 0 outflows, got %s",
		balanceOldDate.CashOutflows.String())

	// After update: verify newW date (day -2) balance has the outflows
	var balanceNewDate models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, newTxn2Midnight).
		First(&balanceNewDate).Error
	s.Require().NoError(err)
	s.Assert().True(amt2.Equal(balanceNewDate.CashOutflows),
		"After update: new date (day -2) should have %s outflows, got %s",
		amt2.String(), balanceNewDate.CashOutflows.String())

	// After update: verify snapshots
	// New timeline: 10k -> 15k (day -8) -> 15k (day -5, no txn) -> 18k (day -3, +3k) -> 16k (day -2, -2k)

	// Day -5: should now be 15,000 (no longer has the -2k expense)
	var snapshotDay5After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotDay5After).Error
	s.Require().NoError(err)
	expectedDay5After := initialBalance.Add(amt1)
	s.Assert().True(expectedDay5After.Equal(snapshotDay5After.EndBalance),
		"After update: day -5 should revert to %s, got %s",
		expectedDay5After.String(), snapshotDay5After.EndBalance.String())

	// Day -3: should be 18,000 (15,000 + 3,000, expense moved away)
	var snapshotDay3After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshotDay3After).Error
	s.Require().NoError(err)
	expectedDay3After := expectedDay5After.Add(amt3)
	s.Assert().True(expectedDay3After.Equal(snapshotDay3After.EndBalance),
		"After update: day -3 should be %s, got %s",
		expectedDay3After.String(), snapshotDay3After.EndBalance.String())

	// Day -2 (new date): should be 16,000 (18,000 - 2,000)
	var snapshotDay2After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, newTxn2Midnight).
		First(&snapshotDay2After).Error
	s.Require().NoError(err)
	expectedDay2After := expectedDay3After.Sub(amt2)
	s.Assert().True(expectedDay2After.Equal(snapshotDay2After.EndBalance),
		"After update: day -2 (new date) should be %s, got %s",
		expectedDay2After.String(), snapshotDay2After.EndBalance.String())

	// Today: should be 16,000 (forward-filled from day -2)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDay2After.Equal(snapshotToday.EndBalance),
		"After update: today should be %s, got %s",
		expectedDay2After.String(), snapshotToday.EndBalance.String())
}

// Tests moving a transaction to an earlier date and verifying that snapshots shift backwards correctly
func (s *TransactionServiceTestSuite) TestUpdateTransaction_ChangeDateToEarlier() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Date Earlier Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert 3 transactions
	// Transaction 1: Day -8, income of 5,000 -> balance becomes 15,000
	txn1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(5000)
	desc1 := "Income day -8"
	req1 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt1,
		TxnDate:         txn1Date,
		Description:     &desc1,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transaction 2: Day -5, expense of 2,000 -> balance becomes 13,000
	txn2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(2000)
	desc2 := "Expense to move earlier"
	req2 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         txn2Date,
		Description:     &desc2,
	}
	txn2ID, err := svc.InsertTransaction(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transaction 3: Day -3, income of 3,000 -> balance becomes 16,000
	txn3Date := time.Now().AddDate(0, 0, -3)
	amt3 := decimal.NewFromInt(3000)
	desc3 := "Income day -3"
	req3 := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amt3,
		TxnDate:         txn3Date,
		Description:     &desc3,
	}
	_, err = svc.InsertTransaction(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Before update: verify timeline is 10k -> 15k -> 13k -> 16k
	txn2Midnight := txn2Date.UTC().Truncate(24 * time.Hour)

	var snapshotDay5Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotDay5Before).Error
	s.Require().NoError(err)
	expectedDay5Before := initialBalance.Add(amt1).Sub(amt2)
	s.Assert().True(expectedDay5Before.Equal(snapshotDay5Before.EndBalance),
		"Before update: day -5 should be %s", expectedDay5Before.String())

	// Move transaction 2 from day -5 to day -7 (earlier, between day -8 and day -5)
	newTxn2Date := time.Now().AddDate(0, 0, -7)
	updateReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amt2,
		TxnDate:         newTxn2Date,
		Description:     &desc2,
	}
	_, err = svc.UpdateTransaction(s.Ctx, userID, txn2ID, updateReq)
	s.Require().NoError(err)

	// Verify transaction date was updated
	var updatedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txn2ID).
		First(&updatedTxn).Error
	s.Require().NoError(err)
	newTxn2Midnight := newTxn2Date.UTC().Truncate(24 * time.Hour)
	s.Assert().Equal(newTxn2Midnight, updatedTxn.TxnDate.UTC().Truncate(24*time.Hour))

	// After update: verify old date (day -5) balance has 0 outflows
	var balanceOldDate models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&balanceOldDate).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balanceOldDate.CashOutflows),
		"After update: old date (day -5) should have 0 outflows, got %s",
		balanceOldDate.CashOutflows.String())

	// After update: verify new date (day -7) balance has the outflows
	var balanceNewDate models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, newTxn2Midnight).
		First(&balanceNewDate).Error
	s.Require().NoError(err)
	s.Assert().True(amt2.Equal(balanceNewDate.CashOutflows),
		"After update: new date (day -7) should have %s outflows, got %s",
		amt2.String(), balanceNewDate.CashOutflows.String())

	// After update: verify snapshots
	// New timeline: 10k -> 15k (day -8) -> 13k (day -7, -2k) -> 13k (day -5, no txn) -> 16k (day -3, +3k)

	txn1Midnight := txn1Date.UTC().Truncate(24 * time.Hour)
	txn3Midnight := txn3Date.UTC().Truncate(24 * time.Hour)

	// Day -8: should still be 15,000 (unchanged)
	var snapshotDay8After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn1Midnight).
		First(&snapshotDay8After).Error
	s.Require().NoError(err)
	expectedDay8After := initialBalance.Add(amt1)
	s.Assert().True(expectedDay8After.Equal(snapshotDay8After.EndBalance),
		"After update: day -8 should remain %s, got %s",
		expectedDay8After.String(), snapshotDay8After.EndBalance.String())

	// Day -7 (new date): should be 13,000 (15,000 - 2,000)
	var snapshotDay7After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, newTxn2Midnight).
		First(&snapshotDay7After).Error
	s.Require().NoError(err)
	expectedDay7After := expectedDay8After.Sub(amt2)
	s.Assert().True(expectedDay7After.Equal(snapshotDay7After.EndBalance),
		"After update: day -7 (new date) should be %s, got %s",
		expectedDay7After.String(), snapshotDay7After.EndBalance.String())

	// Day -5 (old date): should be 13,000 (forward-filled, no transaction here anymore)
	var snapshotDay5After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn2Midnight).
		First(&snapshotDay5After).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDay7After.Equal(snapshotDay5After.EndBalance),
		"After update: day -5 should be forward-filled to %s, got %s",
		expectedDay7After.String(), snapshotDay5After.EndBalance.String())

	// Day -3: should be 16,000 (13,000 + 3,000)
	var snapshotDay3After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txn3Midnight).
		First(&snapshotDay3After).Error
	s.Require().NoError(err)
	expectedDay3After := expectedDay7After.Add(amt3)
	s.Assert().True(expectedDay3After.Equal(snapshotDay3After.EndBalance),
		"After update: day -3 should be %s, got %s",
		expectedDay3After.String(), snapshotDay3After.EndBalance.String())

	// Today: should be 16,000 (forward-filled)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDay3After.Equal(snapshotToday.EndBalance),
		"After update: today should be %s, got %s",
		expectedDay3After.String(), snapshotToday.EndBalance.String())
}

// Tests changing transaction type from expense to income and verifying
// that balance columns flip correctly
func (s *TransactionServiceTestSuite) TestUpdateTransaction_ChangeType() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 10 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -10)
	initialBalance := decimal.NewFromInt(10000)

	accReq := &models.AccountReq{
		Name:           "Type Change Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Insert transaction as expense on day -5
	txnDate := time.Now().AddDate(0, 0, -5)
	amount := decimal.NewFromInt(3000)
	desc := "Transaction to flip"
	req := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          amount,
		TxnDate:         txnDate,
		Description:     &desc,
	}
	txnID, err := svc.InsertTransaction(s.Ctx, userID, req)
	s.Require().NoError(err)

	txnMidnight := txnDate.UTC().Truncate(24 * time.Hour)

	// Before update: verify it's an expense with cash_outflows = 3,000
	var balanceBefore models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&balanceBefore).Error
	s.Require().NoError(err)
	s.Assert().True(amount.Equal(balanceBefore.CashOutflows),
		"Before update: should have %s in outflows", amount.String())
	s.Assert().True(decimal.Zero.Equal(balanceBefore.CashInflows),
		"Before update: should have 0 in inflows")

	// Before update: snapshot should be 7,000 (10,000 - 3,000)
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedBefore := initialBalance.Sub(amount)
	s.Assert().True(expectedBefore.Equal(snapshotBefore.EndBalance),
		"Before update: snapshot should be %s, got %s",
		expectedBefore.String(), snapshotBefore.EndBalance.String())

	// Change from expense to income (flip the type)
	updateReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          amount,
		TxnDate:         txnDate,
		Description:     &desc,
	}
	_, err = svc.UpdateTransaction(s.Ctx, userID, txnID, updateReq)
	s.Require().NoError(err)

	// Verify transaction type was updated
	var updatedTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txnID).
		First(&updatedTxn).Error
	s.Require().NoError(err)
	s.Assert().Equal("income", updatedTxn.TransactionType)

	// After update: verify balance flipped - should have 0 outflows, 3,000 inflows
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&balanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balanceAfter.CashOutflows),
		"After update: should have 0 in outflows, got %s",
		balanceAfter.CashOutflows.String())
	s.Assert().True(amount.Equal(balanceAfter.CashInflows),
		"After update: should have %s in inflows, got %s",
		amount.String(), balanceAfter.CashInflows.String())

	// After update: snapshot should be 13,000 (10,000 + 3,000)
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, txnMidnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	expectedAfter := initialBalance.Add(amount)
	s.Assert().True(expectedAfter.Equal(snapshotAfter.EndBalance),
		"After update: snapshot should be %s, got %s",
		expectedAfter.String(), snapshotAfter.EndBalance.String())

	// Verify today's snapshot is also updated to 13,000
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotToday models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotToday).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAfter.Equal(snapshotToday.EndBalance),
		"After update: today should be %s, got %s",
		expectedAfter.String(), snapshotToday.EndBalance.String())
}

// TestInsertTransfer_CurrentDate tests creating a transfer today between two accounts
func (s *TransactionServiceTestSuite) TestInsertTransfer_CurrentDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create source account with balance of 20,000
	srcBalance := decimal.NewFromInt(20000)
	sourceReq := &models.AccountReq{
		Name:           "Source Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &srcBalance,
		OpenedAt:       time.Now(),
	}
	sourceID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	// Create destination account with balance of 5,000
	destBalance := decimal.NewFromInt(5000)
	destReq := &models.AccountReq{
		Name:           "Destination Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &destBalance,
		OpenedAt:       time.Now(),
	}
	destID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Transfer 3,000 from source to destination
	transferAmount := decimal.NewFromInt(3000)
	notes := "Test transfer"
	transferReq := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        transferAmount,
		Notes:         &notes,
		CreatedAt:     time.Now(),
	}

	transferID, err := svc.InsertTransfer(s.Ctx, userID, transferReq)
	s.Require().NoError(err)
	s.Assert().Greater(transferID, int64(0))

	// Verify transfer was created
	var transfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", transferID).
		First(&transfer).Error
	s.Require().NoError(err)
	s.Assert().Equal(userID, transfer.UserID)
	s.Assert().True(transferAmount.Equal(transfer.Amount))
	s.Assert().Equal("success", transfer.Status)

	// Verify two transactions were created (outflow and inflow)
	var outflowTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND is_transfer = ?", transfer.TransactionOutflowID, true).
		First(&outflowTxn).Error
	s.Require().NoError(err)
	s.Assert().Equal(sourceID, outflowTxn.AccountID)
	s.Assert().Equal("expense", outflowTxn.TransactionType)
	s.Assert().True(transferAmount.Equal(outflowTxn.Amount))

	var inflowTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ? AND is_transfer = ?", transfer.TransactionInflowID, true).
		First(&inflowTxn).Error
	s.Require().NoError(err)
	s.Assert().Equal(destID, inflowTxn.AccountID)
	s.Assert().Equal("income", inflowTxn.TransactionType)
	s.Assert().True(transferAmount.Equal(inflowTxn.Amount))

	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Verify source account balance has 3,000 in outflows
	var sourceBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceBalance).Error
	s.Require().NoError(err)
	s.Assert().True(transferAmount.Equal(sourceBalance.CashOutflows),
		"Source account should have %s in outflows, got %s",
		transferAmount.String(), sourceBalance.CashOutflows.String())

	// Verify destination account balance has 3,000 in inflows
	var destBalanceRecord models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destBalanceRecord).Error
	s.Require().NoError(err)
	s.Assert().True(transferAmount.Equal(destBalanceRecord.CashInflows),
		"Destination account should have %s in inflows, got %s",
		transferAmount.String(), destBalanceRecord.CashInflows.String())

	// Verify source account snapshot: 20,000 - 3,000 = 17,000
	var sourceSnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceSnapshot).Error
	s.Require().NoError(err)
	expectedSourceBalance := srcBalance.Sub(transferAmount)
	s.Assert().True(expectedSourceBalance.Equal(sourceSnapshot.EndBalance),
		"Source snapshot should be %s, got %s",
		expectedSourceBalance.String(), sourceSnapshot.EndBalance.String())

	// Verify destination account snapshot: 5,000 + 3,000 = 8,000
	var destSnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destSnapshot).Error
	s.Require().NoError(err)
	expectedDestBalance := destBalance.Add(transferAmount)
	s.Assert().True(expectedDestBalance.Equal(destSnapshot.EndBalance),
		"Destination snapshot should be %s, got %s",
		expectedDestBalance.String(), destSnapshot.EndBalance.String())
}

// Tests creating a transfer in the past and verifying
// that snapshots are created for both accounts from transfer date to today
func (s *TransactionServiceTestSuite) TestInsertTransfer_PastDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create accounts 10 days ago
	openDate := time.Now().AddDate(0, 0, -10)

	// Source account with balance of 30,000
	srcBalance := decimal.NewFromInt(30000)
	sourceReq := &models.AccountReq{
		Name:           "Source Account Past",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &srcBalance,
		OpenedAt:       openDate,
	}
	sourceID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	// Destination account with balance of 10,000
	destBalance := decimal.NewFromInt(10000)
	destReq := &models.AccountReq{
		Name:           "Destination Account Past",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &destBalance,
		OpenedAt:       openDate,
	}
	destID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Create transfer 5 days in the past
	transferDaysAgo := 5
	transferDate := time.Now().AddDate(0, 0, -transferDaysAgo)
	transferAmount := decimal.NewFromInt(4000)
	notes := "Past transfer"

	transferReq := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        transferAmount,
		Notes:         &notes,
		CreatedAt:     transferDate,
	}

	transferID, err := svc.InsertTransfer(s.Ctx, userID, transferReq)
	s.Require().NoError(err)
	s.Assert().Greater(transferID, int64(0))

	// Verify transfer was created
	var transfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", transferID).
		First(&transfer).Error
	s.Require().NoError(err)

	transferMidnight := transferDate.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Verify balance records exist for transfer date on both accounts
	var sourceBalanceRecord models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transferMidnight).
		First(&sourceBalanceRecord).Error
	s.Require().NoError(err, "source balance record should exist for transfer date")
	s.Assert().True(transferAmount.Equal(sourceBalanceRecord.CashOutflows),
		"source should have %s in outflows", transferAmount.String())

	var destBalanceRecord models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transferMidnight).
		First(&destBalanceRecord).Error
	s.Require().NoError(err, "dest balance record should exist for transfer date")
	s.Assert().True(transferAmount.Equal(destBalanceRecord.CashInflows),
		"dest should have %s in inflows", transferAmount.String())

	// Verify snapshots exist for source account from transfer date to today
	var sourceSnapshots []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			sourceID, transferMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&sourceSnapshots).Error
	s.Require().NoError(err)

	expectedSnapshotCount := transferDaysAgo + 1
	s.Assert().Equal(expectedSnapshotCount, len(sourceSnapshots),
		"source should have %d snapshots from transfer date to today", expectedSnapshotCount)

	// Verify snapshots exist for destination account from transfer date to today
	var destSnapshots []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			destID, transferMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&destSnapshots).Error
	s.Require().NoError(err)

	s.Assert().Equal(expectedSnapshotCount, len(destSnapshots),
		"dest should have %d snapshots from transfer date to today", expectedSnapshotCount)

	// Verify snapshot values on transfer date
	var sourceSnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transferMidnight).
		First(&sourceSnapshot).Error
	s.Require().NoError(err)
	expectedSourceBalance := srcBalance.Sub(transferAmount)
	s.Assert().True(expectedSourceBalance.Equal(sourceSnapshot.EndBalance),
		"source snapshot on transfer date should be %s, got %s",
		expectedSourceBalance.String(), sourceSnapshot.EndBalance.String())

	var destSnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transferMidnight).
		First(&destSnapshot).Error
	s.Require().NoError(err)
	expectedDestBalance := destBalance.Add(transferAmount)
	s.Assert().True(expectedDestBalance.Equal(destSnapshot.EndBalance),
		"dest snapshot on transfer date should be %s, got %s",
		expectedDestBalance.String(), destSnapshot.EndBalance.String())

	// Verify today's snapshots are forward-filled correctly
	var sourceTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedSourceBalance.Equal(sourceTodaySnapshot.EndBalance),
		"source snapshot today should be forward-filled to %s, got %s",
		expectedSourceBalance.String(), sourceTodaySnapshot.EndBalance.String())

	var destTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDestBalance.Equal(destTodaySnapshot.EndBalance),
		"dest snapshot today should be forward-filled to %s, got %s",
		expectedDestBalance.String(), destTodaySnapshot.EndBalance.String())
}

// Tests inserting an older transfer after newer transfers already exist,
// verifying snapshots backfill correctly
func (s *TransactionServiceTestSuite) TestInsertTransfer_BetweenMultipleTransfers() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create accounts 10 days ago
	openDate := time.Now().AddDate(0, 0, -10)

	// Source account with balance of 50,000
	srcBalance := decimal.NewFromInt(50000)
	sourceReq := &models.AccountReq{
		Name:           "Source Backfill",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &srcBalance,
		OpenedAt:       openDate,
	}
	sourceID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	// Destination account with balance of 10,000
	destBalance := decimal.NewFromInt(10000)
	destReq := &models.AccountReq{
		Name:           "Dest Backfill",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &destBalance,
		OpenedAt:       openDate,
	}
	destID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Insert first transfer on day -3 (3 days ago) - transfer 5,000
	transfer1Date := time.Now().AddDate(0, 0, -3)
	amt1 := decimal.NewFromInt(5000)
	notes1 := "First transfer"

	req1 := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        amt1,
		Notes:         &notes1,
		CreatedAt:     transfer1Date,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// After first transfer: source = 45,000, dest = 15,000
	transfer1Midnight := transfer1Date.UTC().Truncate(24 * time.Hour)

	var sourceSnapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer1Midnight).
		First(&sourceSnapshotBefore).Error
	s.Require().NoError(err)
	expectedSourceBefore := srcBalance.Sub(amt1)
	s.Assert().True(expectedSourceBefore.Equal(sourceSnapshotBefore.EndBalance),
		"Before backfill: source on day -3 should be %s", expectedSourceBefore.String())

	var destSnapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer1Midnight).
		First(&destSnapshotBefore).Error
	s.Require().NoError(err)
	expectedDestBefore := destBalance.Add(amt1)
	s.Assert().True(expectedDestBefore.Equal(destSnapshotBefore.EndBalance),
		"Before backfill: dest on day -3 should be %s", expectedDestBefore.String())

	// Insert older transfer on day -6 (6 days ago) - transfer 3,000
	// This should trigger backfill and update all snapshots from day -6 forward
	transfer2Date := time.Now().AddDate(0, 0, -6)
	amt2 := decimal.NewFromInt(3000)
	notes2 := "Backdated transfer"

	req2 := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        amt2,
		Notes:         &notes2,
		CreatedAt:     transfer2Date,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// After backfill, the timeline should be:
	// Day -10 (opening): source = 50k, dest = 10k
	// Day -6: source = 47k (50k - 3k), dest = 13k (10k + 3k)
	// Day -3: source = 42k (47k - 5k), dest = 18k (13k + 5k)
	// Today: source = 42k, dest = 18k

	transfer2Midnight := transfer2Date.UTC().Truncate(24 * time.Hour)

	// Verify balance records for day -6
	var sourceBalance2 models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer2Midnight).
		First(&sourceBalance2).Error
	s.Require().NoError(err)
	s.Assert().True(amt2.Equal(sourceBalance2.CashOutflows),
		"Day -6 source should have %s outflows", amt2.String())

	var destBalance2 models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer2Midnight).
		First(&destBalance2).Error
	s.Require().NoError(err)
	s.Assert().True(amt2.Equal(destBalance2.CashInflows),
		"Day -6 dest should have %s inflows", amt2.String())

	// Verify snapshot on day -6
	var sourceSnapshot6 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer2Midnight).
		First(&sourceSnapshot6).Error
	s.Require().NoError(err)
	expectedSource6 := srcBalance.Sub(amt2)
	s.Assert().True(expectedSource6.Equal(sourceSnapshot6.EndBalance),
		"After backfill: source on day -6 should be %s, got %s",
		expectedSource6.String(), sourceSnapshot6.EndBalance.String())

	var destSnapshot6 models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer2Midnight).
		First(&destSnapshot6).Error
	s.Require().NoError(err)
	expectedDest6 := destBalance.Add(amt2)
	s.Assert().True(expectedDest6.Equal(destSnapshot6.EndBalance),
		"After backfill: dest on day -6 should be %s, got %s",
		expectedDest6.String(), destSnapshot6.EndBalance.String())

	// Verify snapshot on day -3 was updated (not 45k/15k anymore)
	var sourceSnapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer1Midnight).
		First(&sourceSnapshotAfter).Error
	s.Require().NoError(err)
	expectedSource3 := expectedSource6.Sub(amt1)
	s.Assert().True(expectedSource3.Equal(sourceSnapshotAfter.EndBalance),
		"After backfill: source on day -3 should be updated to %s, got %s",
		expectedSource3.String(), sourceSnapshotAfter.EndBalance.String())

	var destSnapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer1Midnight).
		First(&destSnapshotAfter).Error
	s.Require().NoError(err)
	expectedDest3 := expectedDest6.Add(amt1) // 13,000 + 5,000 = 18,000
	s.Assert().True(expectedDest3.Equal(destSnapshotAfter.EndBalance),
		"After backfill: dest on day -3 should be updated to %s, got %s",
		expectedDest3.String(), destSnapshotAfter.EndBalance.String())

	// Verify today's snapshots reflect all transfers
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	var sourceTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedSource3.Equal(sourceTodaySnapshot.EndBalance),
		"Today: source should be %s, got %s",
		expectedSource3.String(), sourceTodaySnapshot.EndBalance.String())

	var destTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDest3.Equal(destTodaySnapshot.EndBalance),
		"Today: dest should be %s, got %s",
		expectedDest3.String(), destTodaySnapshot.EndBalance.String())
}

// Tests multiple transfers on the same day
// and verifying that balances accumulate correctly
func (s *TransactionServiceTestSuite) TestInsertTransfer_SameDay() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create three accounts
	openDate := time.Now().AddDate(0, 0, -5)

	// Account A with balance of 100,000
	balanceA := decimal.NewFromInt(100000)
	reqA := &models.AccountReq{
		Name:           "Account A",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &balanceA,
		OpenedAt:       openDate,
	}
	accountAID, err := accSvc.InsertAccount(s.Ctx, userID, reqA)
	s.Require().NoError(err)

	// Account B with balance of 20,000
	balanceB := decimal.NewFromInt(20000)
	reqB := &models.AccountReq{
		Name:           "Account B",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &balanceB,
		OpenedAt:       openDate,
	}
	accountBID, err := accSvc.InsertAccount(s.Ctx, userID, reqB)
	s.Require().NoError(err)

	// Account C with balance of 5,000
	balanceC := decimal.NewFromInt(5000)
	reqC := &models.AccountReq{
		Name:           "Account C",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &balanceC,
		OpenedAt:       openDate,
	}
	accountCID, err := accSvc.InsertAccount(s.Ctx, userID, reqC)
	s.Require().NoError(err)

	// Create multiple transfers on the same day (day -2)
	transferDate := time.Now().AddDate(0, 0, -2)

	// Transfer 1: A -> B (10,000)
	amt1 := decimal.NewFromInt(10000)
	notes1 := "Transfer 1: A to B"
	req1 := &models.TransferReq{
		SourceID:      accountAID,
		DestinationID: accountBID,
		Amount:        amt1,
		Notes:         &notes1,
		CreatedAt:     transferDate,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transfer 2: A -> C (15,000)
	amt2 := decimal.NewFromInt(15000)
	notes2 := "Transfer 2: A to C"
	req2 := &models.TransferReq{
		SourceID:      accountAID,
		DestinationID: accountCID,
		Amount:        amt2,
		Notes:         &notes2,
		CreatedAt:     transferDate,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transfer 3: B -> C (5,000)
	amt3 := decimal.NewFromInt(5000)
	notes3 := "Transfer 3: B to C"
	req3 := &models.TransferReq{
		SourceID:      accountBID,
		DestinationID: accountCID,
		Amount:        amt3,
		Notes:         &notes3,
		CreatedAt:     transferDate,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Expected balances after all transfers:
	// Account A: 100,000 - 10,000 - 15,000 = 75,000
	// Account B: 20,000 + 10,000 - 5,000 = 25,000
	// Account C: 5,000 + 15,000 + 5,000 = 25,000

	transferMidnight := transferDate.UTC().Truncate(24 * time.Hour)

	// Verify Account A balance: 25,000 in outflows (10k + 15k), 0 inflows
	var balanceRecordA models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountAID, transferMidnight).
		First(&balanceRecordA).Error
	s.Require().NoError(err)
	expectedOutflowsA := amt1.Add(amt2) // 10,000 + 15,000 = 25,000
	s.Assert().True(expectedOutflowsA.Equal(balanceRecordA.CashOutflows),
		"Account A should have %s in outflows, got %s",
		expectedOutflowsA.String(), balanceRecordA.CashOutflows.String())
	s.Assert().True(decimal.Zero.Equal(balanceRecordA.CashInflows),
		"Account A should have 0 inflows, got %s",
		balanceRecordA.CashInflows.String())

	// Verify Account B balance: 5,000 in outflows, 10,000 in inflows
	var balanceRecordB models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountBID, transferMidnight).
		First(&balanceRecordB).Error
	s.Require().NoError(err)
	s.Assert().True(amt3.Equal(balanceRecordB.CashOutflows),
		"Account B should have %s in outflows, got %s",
		amt3.String(), balanceRecordB.CashOutflows.String())
	s.Assert().True(amt1.Equal(balanceRecordB.CashInflows),
		"Account B should have %s in inflows, got %s",
		amt1.String(), balanceRecordB.CashInflows.String())

	// Verify Account C balance: 0 in outflows, 20,000 in inflows (15k + 5k)
	var balanceRecordC models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountCID, transferMidnight).
		First(&balanceRecordC).Error
	s.Require().NoError(err)
	expectedInflowsC := amt2.Add(amt3)
	s.Assert().True(decimal.Zero.Equal(balanceRecordC.CashOutflows),
		"Account C should have 0 outflows, got %s",
		balanceRecordC.CashOutflows.String())
	s.Assert().True(expectedInflowsC.Equal(balanceRecordC.CashInflows),
		"Account C should have %s in inflows, got %s",
		expectedInflowsC.String(), balanceRecordC.CashInflows.String())

	// Verify snapshots on transfer date
	var snapshotA models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountAID, transferMidnight).
		First(&snapshotA).Error
	s.Require().NoError(err)
	expectedBalanceA := balanceA.Sub(amt1).Sub(amt2)
	s.Assert().True(expectedBalanceA.Equal(snapshotA.EndBalance),
		"Account A snapshot should be %s, got %s",
		expectedBalanceA.String(), snapshotA.EndBalance.String())

	var snapshotB models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountBID, transferMidnight).
		First(&snapshotB).Error
	s.Require().NoError(err)
	expectedBalanceB := balanceB.Add(amt1).Sub(amt3)
	s.Assert().True(expectedBalanceB.Equal(snapshotB.EndBalance),
		"Account B snapshot should be %s, got %s",
		expectedBalanceB.String(), snapshotB.EndBalance.String())

	var snapshotC models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accountCID, transferMidnight).
		First(&snapshotC).Error
	s.Require().NoError(err)
	expectedBalanceC := balanceC.Add(amt2).Add(amt3)
	s.Assert().True(expectedBalanceC.Equal(snapshotC.EndBalance),
		"Account C snapshot should be %s, got %s",
		expectedBalanceC.String(), snapshotC.EndBalance.String())

	// Verify only one balance record per account for that day
	var countA, countB, countC int64
	s.TC.DB.WithContext(s.Ctx).Model(&models.Balance{}).
		Where("account_id = ? AND as_of = ?", accountAID, transferMidnight).Count(&countA)
	s.Assert().Equal(int64(1), countA, "Account A should have exactly 1 balance record")

	s.TC.DB.WithContext(s.Ctx).Model(&models.Balance{}).
		Where("account_id = ? AND as_of = ?", accountBID, transferMidnight).Count(&countB)
	s.Assert().Equal(int64(1), countB, "Account B should have exactly 1 balance record")

	s.TC.DB.WithContext(s.Ctx).Model(&models.Balance{}).
		Where("account_id = ? AND as_of = ?", accountCID, transferMidnight).Count(&countC)
	s.Assert().Equal(int64(1), countC, "Account C should have exactly 1 balance record")
}

// Tests deleting a transfer on the current date
func (s *TransactionServiceTestSuite) TestDeleteTransfer_CurrentDate() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create source account with balance of 40,000
	srcBalance := decimal.NewFromInt(40000)
	sourceReq := &models.AccountReq{
		Name:           "Source Delete",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &srcBalance,
		OpenedAt:       time.Now(),
	}
	sourceID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	// Create destination account with balance of 10,000
	destBalance := decimal.NewFromInt(10000)
	destReq := &models.AccountReq{
		Name:           "Dest Delete",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &destBalance,
		OpenedAt:       time.Now(),
	}
	destID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Create transfer of 8,000 today
	transferAmount := decimal.NewFromInt(8000)
	notes := "Transfer to delete"
	transferReq := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        transferAmount,
		Notes:         &notes,
		CreatedAt:     time.Now(),
	}

	transferID, err := svc.InsertTransfer(s.Ctx, userID, transferReq)
	s.Require().NoError(err)

	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	// Before delete: verify balances
	var sourceBalanceBefore models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceBalanceBefore).Error
	s.Require().NoError(err)
	s.Assert().True(transferAmount.Equal(sourceBalanceBefore.CashOutflows),
		"Before delete: source should have %s outflows", transferAmount.String())

	var destBalanceBefore models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destBalanceBefore).Error
	s.Require().NoError(err)
	s.Assert().True(transferAmount.Equal(destBalanceBefore.CashInflows),
		"Before delete: dest should have %s inflows", transferAmount.String())

	// Before delete: verify snapshots (source: 32k, dest: 18k)
	var sourceSnapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceSnapshotBefore).Error
	s.Require().NoError(err)
	expectedSourceBefore := srcBalance.Sub(transferAmount)
	s.Assert().True(expectedSourceBefore.Equal(sourceSnapshotBefore.EndBalance),
		"Before delete: source snapshot should be %s", expectedSourceBefore.String())

	var destSnapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destSnapshotBefore).Error
	s.Require().NoError(err)
	expectedDestBefore := destBalance.Add(transferAmount)
	s.Assert().True(expectedDestBefore.Equal(destSnapshotBefore.EndBalance),
		"Before delete: dest snapshot should be %s", expectedDestBefore.String())

	// Delete the transfer
	err = svc.DeleteTransfer(s.Ctx, userID, transferID)
	s.Require().NoError(err)

	// Verify transfer is soft-deleted
	var deletedTransfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", transferID).
		First(&deletedTransfer).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedTransfer.DeletedAt, "transfer should be soft-deleted")

	// Verify both transactions are soft-deleted
	var deletedInflow models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", deletedTransfer.TransactionInflowID).
		First(&deletedInflow).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedInflow.DeletedAt, "inflow transaction should be soft-deleted")

	var deletedOutflow models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", deletedTransfer.TransactionOutflowID).
		First(&deletedOutflow).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedOutflow.DeletedAt, "outflow transaction should be soft-deleted")

	// After delete: verify source balance outflows reversed to 0
	var sourceBalanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceBalanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(sourceBalanceAfter.CashOutflows),
		"After delete: source should have 0 outflows, got %s",
		sourceBalanceAfter.CashOutflows.String())

	// After delete: verify dest balance inflows reversed to 0
	var destBalanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destBalanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(destBalanceAfter.CashInflows),
		"After delete: dest should have 0 inflows, got %s",
		destBalanceAfter.CashInflows.String())

	// After delete: verify snapshots reverted to original balances
	var sourceSnapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceSnapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(srcBalance.Equal(sourceSnapshotAfter.EndBalance),
		"After delete: source snapshot should revert to %s, got %s",
		srcBalance.String(), sourceSnapshotAfter.EndBalance.String())

	var destSnapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destSnapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(destBalance.Equal(destSnapshotAfter.EndBalance),
		"After delete: dest snapshot should revert to %s, got %s",
		destBalance.String(), destSnapshotAfter.EndBalance.String())
}

// Tests deleting a middle transfer in the past
// and verifying that both accounts' snapshots recalculate correctly
func (s *TransactionServiceTestSuite) TestDeleteTransfer_PastDateMiddle() {
	svc := s.TC.App.TransactionService
	accSvc := s.TC.App.AccountService
	userID := int64(1)

	// Create accounts 10 days ago
	openDate := time.Now().AddDate(0, 0, -10)

	// Source account with balance of 100,000
	srcBalance := decimal.NewFromInt(100000)
	sourceReq := &models.AccountReq{
		Name:           "Source Middle Delete",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &srcBalance,
		OpenedAt:       openDate,
	}
	sourceID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	// Destination account with balance of 20,000
	destBalance := decimal.NewFromInt(20000)
	destReq := &models.AccountReq{
		Name:           "Dest Middle Delete",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &destBalance,
		OpenedAt:       openDate,
	}
	destID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Create 3 transfers at different dates
	// Transfer 1: Day -8, transfer 10,000
	transfer1Date := time.Now().AddDate(0, 0, -8)
	amt1 := decimal.NewFromInt(10000)
	notes1 := "Transfer 1"
	req1 := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        amt1,
		Notes:         &notes1,
		CreatedAt:     transfer1Date,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req1)
	s.Require().NoError(err)

	// Transfer 2: Day -5, transfer 5,000
	transfer2Date := time.Now().AddDate(0, 0, -5)
	amt2 := decimal.NewFromInt(5000)
	notes2 := "Transfer 2 to delete"
	req2 := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        amt2,
		Notes:         &notes2,
		CreatedAt:     transfer2Date,
	}
	transfer2ID, err := svc.InsertTransfer(s.Ctx, userID, req2)
	s.Require().NoError(err)

	// Transfer 3: Day -2, transfer 8,000
	transfer3Date := time.Now().AddDate(0, 0, -2)
	amt3 := decimal.NewFromInt(8000)
	notes3 := "Transfer 3"
	req3 := &models.TransferReq{
		SourceID:      sourceID,
		DestinationID: destID,
		Amount:        amt3,
		Notes:         &notes3,
		CreatedAt:     transfer3Date,
	}
	_, err = svc.InsertTransfer(s.Ctx, userID, req3)
	s.Require().NoError(err)

	// Before delete, verify timeline:
	// Day -10: source = 100k, dest = 20k
	// Day -8: source = 90k, dest = 30k (after transfer 1: -10k/+10k)
	// Day -5: source = 85k, dest = 35k (after transfer 2: -5k/+5k)
	// Day -2: source = 77k, dest = 43k (after transfer 3: -8k/+8k)

	transfer2Midnight := transfer2Date.UTC().Truncate(24 * time.Hour)
	transfer3Midnight := transfer3Date.UTC().Truncate(24 * time.Hour)

	// Before delete: verify day -5 snapshots
	var sourceSnapshot5Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer2Midnight).
		First(&sourceSnapshot5Before).Error
	s.Require().NoError(err)
	expectedSource5Before := srcBalance.Sub(amt1).Sub(amt2) // 100k - 10k - 5k = 85k
	s.Assert().True(expectedSource5Before.Equal(sourceSnapshot5Before.EndBalance),
		"Before delete: source day -5 should be %s", expectedSource5Before.String())

	var destSnapshot5Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer2Midnight).
		First(&destSnapshot5Before).Error
	s.Require().NoError(err)
	expectedDest5Before := destBalance.Add(amt1).Add(amt2) // 20k + 10k + 5k = 35k
	s.Assert().True(expectedDest5Before.Equal(destSnapshot5Before.EndBalance),
		"Before delete: dest day -5 should be %s", expectedDest5Before.String())

	// Before delete: verify day -2 snapshots
	var sourceSnapshot2Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer3Midnight).
		First(&sourceSnapshot2Before).Error
	s.Require().NoError(err)
	expectedSource2Before := expectedSource5Before.Sub(amt3) // 85k - 8k = 77k
	s.Assert().True(expectedSource2Before.Equal(sourceSnapshot2Before.EndBalance),
		"Before delete: source day -2 should be %s", expectedSource2Before.String())

	var destSnapshot2Before models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer3Midnight).
		First(&destSnapshot2Before).Error
	s.Require().NoError(err)
	expectedDest2Before := expectedDest5Before.Add(amt3) // 35k + 8k = 43k
	s.Assert().True(expectedDest2Before.Equal(destSnapshot2Before.EndBalance),
		"Before delete: dest day -2 should be %s", expectedDest2Before.String())

	// Delete transfer 2 (middle transfer on day -5)
	err = svc.DeleteTransfer(s.Ctx, userID, transfer2ID)
	s.Require().NoError(err)

	// After delete, timeline should be:
	// Day -10: source = 100k, dest = 20k
	// Day -8: source = 90k, dest = 30k (transfer 1)
	// Day -5: source = 90k, dest = 30k (no transfer, forward-filled)
	// Day -2: source = 82k, dest = 38k (transfer 3, recalculated)

	// Verify transfer 2 is soft-deleted
	var deletedTransfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).
		Unscoped().
		Where("id = ?", transfer2ID).
		First(&deletedTransfer).Error
	s.Require().NoError(err)
	s.Assert().NotNil(deletedTransfer.DeletedAt)

	// After delete: verify day -5 balances reversed to 0
	var sourceBalance5After models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer2Midnight).
		First(&sourceBalance5After).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(sourceBalance5After.CashOutflows),
		"After delete: source day -5 should have 0 outflows, got %s",
		sourceBalance5After.CashOutflows.String())

	var destBalance5After models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer2Midnight).
		First(&destBalance5After).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(destBalance5After.CashInflows),
		"After delete: dest day -5 should have 0 inflows, got %s",
		destBalance5After.CashInflows.String())

	// After delete: verify day -5 snapshots (should be 90k/30k, no longer 85k/35k)
	var sourceSnapshot5After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer2Midnight).
		First(&sourceSnapshot5After).Error
	s.Require().NoError(err)
	expectedSource5After := srcBalance.Sub(amt1)
	s.Assert().True(expectedSource5After.Equal(sourceSnapshot5After.EndBalance),
		"After delete: source day -5 should be %s, got %s",
		expectedSource5After.String(), sourceSnapshot5After.EndBalance.String())

	var destSnapshot5After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer2Midnight).
		First(&destSnapshot5After).Error
	s.Require().NoError(err)
	expectedDest5After := destBalance.Add(amt1)
	s.Assert().True(expectedDest5After.Equal(destSnapshot5After.EndBalance),
		"After delete: dest day -5 should be %s, got %s",
		expectedDest5After.String(), destSnapshot5After.EndBalance.String())

	// After delete: verify day -2 snapshots were updated (not 77k/43k anymore)
	var sourceSnapshot2After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, transfer3Midnight).
		First(&sourceSnapshot2After).Error
	s.Require().NoError(err)
	expectedSource2After := expectedSource5After.Sub(amt3)
	s.Assert().True(expectedSource2After.Equal(sourceSnapshot2After.EndBalance),
		"After delete: source day -2 should be updated to %s, got %s",
		expectedSource2After.String(), sourceSnapshot2After.EndBalance.String())

	var destSnapshot2After models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, transfer3Midnight).
		First(&destSnapshot2After).Error
	s.Require().NoError(err)
	expectedDest2After := expectedDest5After.Add(amt3)
	s.Assert().True(expectedDest2After.Equal(destSnapshot2After.EndBalance),
		"After delete: dest day -2 should be updated to %s, got %s",
		expectedDest2After.String(), destSnapshot2After.EndBalance.String())

	// Verify today's snapshots are also updated correctly
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	var sourceTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", sourceID, todayMidnight).
		First(&sourceTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedSource2After.Equal(sourceTodaySnapshot.EndBalance),
		"After delete: source today should be %s, got %s",
		expectedSource2After.String(), sourceTodaySnapshot.EndBalance.String())

	var destTodaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", destID, todayMidnight).
		First(&destTodaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(expectedDest2After.Equal(destTodaySnapshot.EndBalance),
		"After delete: dest today should be %s, got %s",
		expectedDest2After.String(), destTodaySnapshot.EndBalance.String())
}

// Tests that an expense transaction is blocked if it would reduce balance below total investment value
func (s *TransactionServiceTestSuite) TestInsertTransaction_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create and buy BTC asset worth 60k
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(context.DeadlineExceeded, ctx.Err()) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Get current balance (should be ~100k + unrealized gains)
	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)

	// Try to create an expense that would drop balance below 60k
	// Even if current balance is 110k, we can't spend more than 50k
	expenseAmount := decimal.NewFromInt(55000)

	txnReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          expenseAmount,
		TxnDate:         today,
	}

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, txnReq)
	s.Require().Error(err, "should block expense that would drop balance below investments")

	// Verify no transaction was created
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ?", accID).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount, "no transaction should be created")

	// Verify balance unchanged
	var balanceAfter models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&balanceAfter).Error
	s.Require().NoError(err)
	s.Assert().True(latestBalance.EndBalance.Equal(balanceAfter.EndBalance),
		"balance should remain unchanged")
}

// Tests that updating a transaction is blocked if it would reduce balance below total investment value
func (s *TransactionServiceTestSuite) TestUpdateTransaction_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create a small expense transaction (10k)
	txnReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          decimal.NewFromInt(10000),
		TxnDate:         today,
	}

	txnID, err := txnSvc.InsertTransaction(s.Ctx, userID, txnReq)
	s.Require().NoError(err)

	// Buy BTC worth 50k
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(50000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to update the 10k expense to 60k
	// This would drop balance below investment value
	updateReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          decimal.NewFromInt(60000),
		TxnDate:         today,
	}

	_, err = txnSvc.UpdateTransaction(s.Ctx, userID, txnID, updateReq)
	s.Require().Error(err, "should block update that would drop balance below investments")

	// Verify transaction unchanged
	var txn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", txnID).
		First(&txn).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(10000).Equal(txn.Amount),
		"transaction amount should remain 10000, got %s", txn.Amount.String())
}

// Tests that deleting an income transaction is blocked if it would reduce balance below total investment value
func (s *TransactionServiceTestSuite) TestDeleteTransaction_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(50000)

	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Add 50k income (brings total to 100k)
	incomeReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          decimal.NewFromInt(50000),
		TxnDate:         today,
	}

	incomeID, err := txnSvc.InsertTransaction(s.Ctx, userID, incomeReq)
	s.Require().NoError(err)

	// Buy BTC worth 60k
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to delete the 50k income
	// This would drop balance from ~100k to ~50k, below the 60k+ investment
	err = txnSvc.DeleteTransaction(s.Ctx, userID, incomeID)
	s.Require().Error(err, "should block deleting income that would drop balance below investments")

	// Verify transaction still exists
	var txn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", incomeID).
		First(&txn).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(50000).Equal(txn.Amount),
		"income transaction should still exist")
}

// Tests that a transfer is blocked if it would reduce the source account balance below total investment value
func (s *TransactionServiceTestSuite) TestInsertTransfer_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	// Create investment account
	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	sourceAccID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create destination account
	destReq := &models.AccountReq{
		Name:          "Checking Account",
		AccountTypeID: 1,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	destAccID, err := accSvc.InsertAccount(s.Ctx, userID, destReq)
	s.Require().NoError(err)

	// Buy BTC worth 60k in source account
	assetReq := &models.InvestmentAssetReq{
		AccountID:      sourceAccID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to transfer 50k out of investment account
	// This would drop balance below 60k+ investment value
	transferReq := &models.TransferReq{
		SourceID:      sourceAccID,
		DestinationID: destAccID,
		Amount:        decimal.NewFromInt(50000),
		CreatedAt:     today,
	}

	_, err = txnSvc.InsertTransfer(s.Ctx, userID, transferReq)
	s.Require().Error(err, "should block transfer that would drop source balance below investments")

	// Verify no transfer created
	var transferCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transfer{}).
		Where("user_id = ?", userID).
		Count(&transferCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), transferCount, "no transfer should be created")
}

// Tests that deleting a transfer is blocked if reversing it would reduce the destination account balance below investment value
func (s *TransactionServiceTestSuite) TestDeleteTransfer_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(50000)

	sourceReq := &models.AccountReq{
		Name:          "Checking Account",
		AccountTypeID: 1,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	sourceAccID, err := accSvc.InsertAccount(s.Ctx, userID, sourceReq)
	s.Require().NoError(err)

	invReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	invAccID, err := accSvc.InsertAccount(s.Ctx, userID, invReq)
	s.Require().NoError(err)

	// Transfer 50k from checking to investment (brings investment to 100k)
	transferReq := &models.TransferReq{
		SourceID:      sourceAccID,
		DestinationID: invAccID,
		Amount:        decimal.NewFromInt(50000),
		CreatedAt:     today,
	}

	transferID, err := txnSvc.InsertTransfer(s.Ctx, userID, transferReq)
	s.Require().NoError(err)

	// Buy BTC worth 60k in investment account
	assetReq := &models.InvestmentAssetReq{
		AccountID:      invAccID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(context.DeadlineExceeded, ctx.Err()) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to delete the transfer
	// This would remove 50k from investment account, dropping it to ~50k below 60k+ investment
	err = txnSvc.DeleteTransfer(s.Ctx, userID, transferID)
	s.Require().Error(err, "should block deleting transfer that would drop destination balance below investments")

	// Verify transfer still exists
	var transfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", transferID).
		First(&transfer).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(50000).Equal(transfer.Amount),
		"transfer should still exist")
}

// Tests that restoring a deleted expense is blocked if it would reduce balance below total investment value
func (s *TransactionServiceTestSuite) TestRestoreTransaction_BlockedByInvestments() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(100000)

	accReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &initialBalance,
		OpenedAt:      today,
	}
	accID, err := accSvc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Create expense transaction (50k)
	expenseReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "expense",
		Amount:          decimal.NewFromInt(50000),
		TxnDate:         today,
	}

	expenseID, err := txnSvc.InsertTransaction(s.Ctx, userID, expenseReq)
	s.Require().NoError(err)

	// Delete the expense (balance goes back to 100k)
	err = txnSvc.DeleteTransaction(s.Ctx, userID, expenseID)
	s.Require().NoError(err)

	// Buy BTC worth 60k
	assetReq := &models.InvestmentAssetReq{
		AccountID:      accID,
		InvestmentType: models.InvestmentCrypto,
		Name:           "Bitcoin",
		Ticker:         "BTC-USD",
		Quantity:       decimal.NewFromInt(0),
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	assetID, err := invSvc.InsertAsset(ctx, userID, assetReq)
	if err != nil {
		if errors.Is(context.DeadlineExceeded, ctx.Err()) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	ctx2, cancel2 := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel2()

	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(60000),
		Currency:     "USD",
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to restore the 50k expense
	// This would drop balance from ~100k to ~50k, below 60k+ investment
	err = txnSvc.RestoreTransaction(s.Ctx, userID, expenseID)
	s.Require().Error(err, "should block restoring expense that would drop balance below investments")

	// Verify transaction still deleted
	var txn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).Unscoped().
		Where("id = ?", expenseID).
		First(&txn).Error
	s.Require().NoError(err)
	s.Assert().NotNil(txn.DeletedAt, "transaction should still be deleted")
}
