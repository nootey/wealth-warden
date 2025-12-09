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
