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

type AccountServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceTestSuite))
}

// Tests adjusting an account balance upward
// and verifying that an adjustment transaction is created
func (s *AccountServiceTestSuite) TestUpdateAccount_AdjustBalanceUp() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	// Create account with initial balance of 10,000
	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Balance Adjust Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Verify initial snapshot is 10,000
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshotBefore.EndBalance),
		"Initial snapshot should be %s", initialBalance.String())

	// Update account balance to 15,000 (increase by 5,000)
	newBalance := decimal.NewFromInt(15000)
	updateReq := &models.AccountReq{
		Name:           "Balance Adjust Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &newBalance,
		OpenedAt:       time.Now(),
	}

	_, err = svc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().NoError(err)

	// Verify an adjustment transaction was created
	var adjustmentTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND is_adjustment = ?", accID, true).
		First(&adjustmentTxn).Error
	s.Require().NoError(err, "adjustment transaction should exist")

	// Verify adjustment is an income of 5,000
	expectedAdjustment := decimal.NewFromInt(5000)
	s.Assert().Equal("income", adjustmentTxn.TransactionType,
		"adjustment should be income type")
	s.Assert().True(expectedAdjustment.Equal(adjustmentTxn.Amount),
		"adjustment amount should be %s, got %s",
		expectedAdjustment.String(), adjustmentTxn.Amount.String())
	s.Assert().Equal("Manual adjustment", *adjustmentTxn.Description)

	// Verify the adjustment has adjustment category
	var category models.Category
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", *adjustmentTxn.CategoryID).
		First(&category).Error
	s.Require().NoError(err)
	s.Assert().Equal("adjustment", category.Classification)

	// Verify balance record shows the income
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAdjustment.Equal(balance.CashInflows),
		"cash_inflows should be %s, got %s",
		expectedAdjustment.String(), balance.CashInflows.String())

	// Verify snapshot updated to 15,000
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(newBalance.Equal(snapshotAfter.EndBalance),
		"Snapshot should be updated to %s, got %s",
		newBalance.String(), snapshotAfter.EndBalance.String())
}

// Tests adjusting an account balance downward
// and verifying that an expense adjustment transaction is created
func (s *AccountServiceTestSuite) TestUpdateAccount_AdjustBalanceDown() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	// Create account with initial balance of 20,000
	initialBalance := decimal.NewFromInt(20000)
	accReq := &models.AccountReq{
		Name:           "Balance Down Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Verify initial snapshot is 20,000
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshotBefore.EndBalance),
		"Initial snapshot should be %s", initialBalance.String())

	// Update account balance to 12,000 (decrease by 8,000)
	newBalance := decimal.NewFromInt(12000)
	updateReq := &models.AccountReq{
		Name:           "Balance Down Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &newBalance,
		OpenedAt:       time.Now(),
	}

	_, err = svc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().NoError(err)

	// Verify an adjustment transaction was created
	var adjustmentTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND is_adjustment = ?", accID, true).
		First(&adjustmentTxn).Error
	s.Require().NoError(err, "adjustment transaction should exist")

	// Verify adjustment is an expense of 8,000
	expectedAdjustment := decimal.NewFromInt(8000)
	s.Assert().Equal("expense", adjustmentTxn.TransactionType,
		"adjustment should be expense type")
	s.Assert().True(expectedAdjustment.Equal(adjustmentTxn.Amount),
		"adjustment amount should be %s, got %s",
		expectedAdjustment.String(), adjustmentTxn.Amount.String())
	s.Assert().Equal("Manual adjustment", *adjustmentTxn.Description)
	s.Assert().True(adjustmentTxn.IsAdjustment, "transaction should be marked as adjustment")

	// Verify balance record shows the outflow
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAdjustment.Equal(balance.CashOutflows),
		"cash_outflows should be %s, got %s",
		expectedAdjustment.String(), balance.CashOutflows.String())

	// Verify snapshot updated to 12,000
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(newBalance.Equal(snapshotAfter.EndBalance),
		"Snapshot should be updated to %s, got %s",
		newBalance.String(), snapshotAfter.EndBalance.String())
}

// Tests that no adjustment transaction
// is created when the balance doesn't change
func (s *AccountServiceTestSuite) TestUpdateAccount_AdjustBalanceNoChange() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	// Create account with initial balance of 10,000
	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "No Change Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Update account with the same balance (10,000)
	sameBalance := decimal.NewFromInt(10000)
	updateReq := &models.AccountReq{
		Name:           "No Change Account",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &sameBalance,
		OpenedAt:       time.Now(),
	}

	_, err = svc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().NoError(err)

	// Verify no adjustment transaction was created
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ? AND is_adjustment = ?", accID, true).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount,
		"no adjustment transaction should be created when balance doesn't change")

	// Verify balance record still shows only the opening balance (no inflows/outflows)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(balance.CashInflows),
		"cash_inflows should remain 0, got %s", balance.CashInflows.String())
	s.Assert().True(decimal.Zero.Equal(balance.CashOutflows),
		"cash_outflows should remain 0, got %s", balance.CashOutflows.String())

	// Verify snapshot remains at 10,000
	var snapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshot).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshot.EndBalance),
		"Snapshot should remain %s, got %s",
		initialBalance.String(), snapshot.EndBalance.String())
}

// Tests adjusting a liability account balance
// and verifying that signs are correctly handled (liabilities stored as negative)
func (s *AccountServiceTestSuite) TestUpdateAccount_AdjustLiabilityBalance() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	// Create a liability account with initial balance of -5,000
	initialBalance := decimal.NewFromInt(5000)
	accReq := &models.AccountReq{
		Name:           "Credit Card",
		AccountTypeID:  19,
		Type:           "loan",
		Subtype:        "personal",
		Classification: "liability",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Verify initial snapshot (-5,000)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var snapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotBefore).Error
	s.Require().NoError(err)
	expectedInitial := initialBalance.Neg()
	s.Assert().True(expectedInitial.Equal(snapshotBefore.EndBalance),
		"Initial liability snapshot should be %s (negative), got %s",
		expectedInitial.String(), snapshotBefore.EndBalance.String())

	// Update liability balance to -8,000
	newBalance := decimal.NewFromInt(-8000)
	updateReq := &models.AccountReq{
		Name:           "Credit Card",
		AccountTypeID:  19,
		Type:           "loan",
		Subtype:        "personal",
		Classification: "liability",
		Balance:        &newBalance,
		OpenedAt:       time.Now(),
	}

	_, err = svc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().NoError(err)

	// Verify an adjustment transaction was created
	var adjustmentTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND is_adjustment = ?", accID, true).
		First(&adjustmentTxn).Error
	s.Require().NoError(err, "adjustment transaction should exist")

	expectedAdjustment := decimal.NewFromInt(3000)
	s.Assert().Equal("expense", adjustmentTxn.TransactionType,
		"increasing liability debt should be expense type")
	s.Assert().True(expectedAdjustment.Equal(adjustmentTxn.Amount),
		"adjustment amount should be %s, got %s",
		expectedAdjustment.String(), adjustmentTxn.Amount.String())

	// Verify balance record shows the expense (outflow)
	var balance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&balance).Error
	s.Require().NoError(err)
	s.Assert().True(expectedAdjustment.Equal(balance.CashOutflows),
		"cash_outflows should be %s, got %s",
		expectedAdjustment.String(), balance.CashOutflows.String())

	// Verify snapshot updated to -8,000
	var snapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&snapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(newBalance.Equal(snapshotAfter.EndBalance),
		"Snapshot should be updated to %s, got %s",
		newBalance.String(), snapshotAfter.EndBalance.String())
}

// Tests adjusting balance for an account
// that was created in the past, verifying snapshots are correctly created/updated
func (s *AccountServiceTestSuite) TestUpdateAccount_AdjustBalancePastAccount() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	// Create account 5 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -5)
	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Past Account Adjust",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Verify initial snapshots exist from opening date to today (6 days total)
	openMidnight := openDate.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)

	var snapshotsBefore []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			accID, openMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&snapshotsBefore).Error
	s.Require().NoError(err)
	s.Assert().Equal(6, len(snapshotsBefore), "should have 6 snapshots (day -5 to today)")

	// All snapshots should be 10,000 (no transactions yet)
	for _, snap := range snapshotsBefore {
		s.Assert().True(initialBalance.Equal(snap.EndBalance),
			"Snapshot on %s should be %s", snap.AsOf.Format("2006-01-02"), initialBalance.String())
	}

	// Adjust balance TODAY to 15,000 (increase by 5,000)
	newBalance := decimal.NewFromInt(15000)
	updateReq := &models.AccountReq{
		Name:           "Past Account Adjust",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &newBalance,
		OpenedAt:       openDate,
	}

	_, err = svc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().NoError(err)

	// Verify adjustment transaction created today
	var adjustmentTxn models.Transaction
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND is_adjustment = ?", accID, true).
		First(&adjustmentTxn).Error
	s.Require().NoError(err)
	s.Assert().Equal("income", adjustmentTxn.TransactionType)
	s.Assert().True(decimal.NewFromInt(5000).Equal(adjustmentTxn.Amount))

	// Verify balance record on today has the adjustment
	var todayBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&todayBalance).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(5000).Equal(todayBalance.CashInflows),
		"Today's balance should have 5000 inflows")

	// Verify snapshots for past days remain 10,000
	var snapshotDay4 models.AccountDailySnapshot
	day4Midnight := openMidnight.AddDate(0, 0, 1) // Day -4
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, day4Midnight).
		First(&snapshotDay4).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(snapshotDay4.EndBalance),
		"Day -4 snapshot should remain %s, got %s",
		initialBalance.String(), snapshotDay4.EndBalance.String())

	// Verify today's snapshot is 15,000
	var todaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&todaySnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(newBalance.Equal(todaySnapshot.EndBalance),
		"Today's snapshot should be %s, got %s",
		newBalance.String(), todaySnapshot.EndBalance.String())

	// Verify all snapshots still exist (should still be 6)
	var snapshotsAfter []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			accID, openMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&snapshotsAfter).Error
	s.Require().NoError(err)
	s.Assert().Equal(6, len(snapshotsAfter), "should still have 6 snapshots")
}

// Tests closing an account and verifying snapshots/balances are correct
func (s *AccountServiceTestSuite) TestCloseAccount() {
	svc := s.TC.App.AccountService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	// Create account 3 days ago with initial balance of 10,000
	openDate := time.Now().AddDate(0, 0, -3)
	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Account to Close",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       openDate,
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Add a transaction 1 day ago (adds 2,000)
	oneDayAgo := time.Now().AddDate(0, 0, -1)
	txnAmount := decimal.NewFromInt(2000)
	desc := "Test income transaction"
	txnReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          txnAmount,
		TxnDate:         oneDayAgo,
		Description:     &desc,
	}
	_, err = txnSvc.InsertTransaction(s.Ctx, userID, txnReq)
	s.Require().NoError(err)

	// Expected balance after transaction: 10,000 + 2,000 = 12,000
	expectedBalanceAfterTxn := initialBalance.Add(txnAmount)

	// Verify snapshots exist before closing (should be 4 days: -3, -2, -1, 0/today)
	openMidnight := openDate.UTC().Truncate(24 * time.Hour)
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	oneDayAgoMidnight := oneDayAgo.UTC().Truncate(24 * time.Hour)

	var snapshotsBefore []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			accID, openMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&snapshotsBefore).Error
	s.Require().NoError(err)
	s.Assert().Equal(4, len(snapshotsBefore), "should have 4 snapshots before closing")

	// Verify account is active
	var accountBefore models.Account
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", accID).
		First(&accountBefore).Error
	s.Require().NoError(err)
	s.Assert().True(accountBefore.IsActive)
	s.Assert().Nil(accountBefore.ClosedAt)

	// Verify yesterday's snapshot reflects the transaction (12,000)
	var yesterdaySnapshotBefore models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, oneDayAgoMidnight).
		First(&yesterdaySnapshotBefore).Error
	s.Require().NoError(err)
	s.Assert().True(expectedBalanceAfterTxn.Equal(yesterdaySnapshotBefore.EndBalance),
		"Yesterday's snapshot should be %s (after transaction), got %s",
		expectedBalanceAfterTxn.String(), yesterdaySnapshotBefore.EndBalance.String())

	// Close the account TODAY
	err = svc.CloseAccount(s.Ctx, userID, accID)
	s.Require().NoError(err)

	// Verify account is closed
	var accountAfter models.Account
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", accID).
		First(&accountAfter).Error
	s.Require().NoError(err)
	s.Assert().False(accountAfter.IsActive, "account should be inactive")
	s.Assert().NotNil(accountAfter.ClosedAt, "ClosedAt should be set")
	s.Assert().Equal(todayMidnight.Format("2006-01-02"),
		accountAfter.ClosedAt.UTC().Truncate(24*time.Hour).Format("2006-01-02"),
		"ClosedAt should be set to today")

	// Verify all historical snapshots still exist
	var snapshotsAfter []models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of >= ? AND as_of <= ?",
			accID, openMidnight, todayMidnight).
		Order("as_of ASC").
		Find(&snapshotsAfter).Error
	s.Require().NoError(err)
	s.Assert().Equal(4, len(snapshotsAfter), "should still have 4 snapshots after closing")

	// Verify opening day snapshot is still 10,000
	var openSnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, openMidnight).
		First(&openSnapshot).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(openSnapshot.EndBalance),
		"Opening day snapshot should be %s, got %s",
		initialBalance.String(), openSnapshot.EndBalance.String())

	// Verify yesterday's snapshot still reflects the transaction (12,000)
	var yesterdaySnapshotAfter models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, oneDayAgoMidnight).
		First(&yesterdaySnapshotAfter).Error
	s.Require().NoError(err)
	s.Assert().True(expectedBalanceAfterTxn.Equal(yesterdaySnapshotAfter.EndBalance),
		"Yesterday's snapshot should still be %s (after transaction), got %s",
		expectedBalanceAfterTxn.String(), yesterdaySnapshotAfter.EndBalance.String())

	// Verify today's snapshot exists and has correct final balance (12,000)
	var todaySnapshot models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&todaySnapshot).Error
	s.Require().NoError(err, "snapshot should exist for closing date")
	s.Assert().True(expectedBalanceAfterTxn.Equal(todaySnapshot.EndBalance),
		"Today's final snapshot should be %s, got %s",
		expectedBalanceAfterTxn.String(), todaySnapshot.EndBalance.String())

	// Verify today's balance record exists
	var todayBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&todayBalance).Error
	s.Require().NoError(err, "balance record should exist for closing date")

	// Today's balance should carry forward yesterday's ending balance (12,000)
	s.Assert().True(expectedBalanceAfterTxn.Equal(todayBalance.EndBalance),
		"Today's balance end_balance should be %s, got %s",
		expectedBalanceAfterTxn.String(), todayBalance.EndBalance.String())

	// Verify yesterday's balance shows the income transaction
	var yesterdayBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, oneDayAgoMidnight).
		First(&yesterdayBalance).Error
	s.Require().NoError(err)
	s.Assert().True(txnAmount.Equal(yesterdayBalance.CashInflows),
		"Yesterday's balance should show cash_inflows of %s, got %s",
		txnAmount.String(), yesterdayBalance.CashInflows.String())
}

// Tests that inserting a transaction to a closed account fails
func (s *AccountServiceTestSuite) TestInsertTransaction_OnClosedAccount() {
	svc := s.TC.App.AccountService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	// Create account with initial balance of 10,000
	initialBalance := decimal.NewFromInt(10000)
	accReq := &models.AccountReq{
		Name:           "Account to Close",
		AccountTypeID:  1,
		Type:           "asset",
		Subtype:        "cash",
		Classification: "current",
		Balance:        &initialBalance,
		OpenedAt:       time.Now(),
	}
	accID, err := svc.InsertAccount(s.Ctx, userID, accReq)
	s.Require().NoError(err)

	// Close the account
	err = svc.CloseAccount(s.Ctx, userID, accID)
	s.Require().NoError(err)

	// Verify account is closed
	var account models.Account
	err = s.TC.DB.WithContext(s.Ctx).
		Where("id = ?", accID).
		First(&account).Error
	s.Require().NoError(err)
	s.Assert().False(account.IsActive)
	s.Assert().NotNil(account.ClosedAt)

	// Attempt to insert a transaction to the closed account
	txnAmount := decimal.NewFromInt(1000)
	desc := "Transaction on closed account"
	txnReq := &models.TransactionReq{
		AccountID:       accID,
		TransactionType: "income",
		Amount:          txnAmount,
		TxnDate:         time.Now(),
		Description:     &desc,
	}

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, txnReq)
	s.Require().Error(err, "should not allow transaction on closed account")
	s.Assert().Contains(err.Error(), "closed",
		"error message should indicate account is closed")

	// Verify no transaction was created
	var txnCount int64
	err = s.TC.DB.WithContext(s.Ctx).
		Model(&models.Transaction{}).
		Where("account_id = ?", accID).
		Count(&txnCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(0), txnCount,
		"no transactions should exist for closed account")

	// Verify balance hasn't changed
	todayMidnight := time.Now().UTC().Truncate(24 * time.Hour)
	var todayBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", accID, todayMidnight).
		First(&todayBalance).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.Zero.Equal(todayBalance.CashInflows),
		"cash_inflows should remain 0, got %s", todayBalance.CashInflows.String())
}

// Tests that manual balance adjustment is blocked if it would set balance below total investment value
func (s *AccountServiceTestSuite) TestUpdateAccount_BlockedByInvestmentValue() {
	accSvc := s.TC.App.AccountService
	invSvc := s.TC.App.InvestmentService
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

	// Buy 90k of BTC — balance drops to 10k
	_, err = invSvc.InsertInvestmentTrade(ctx2, userID, &models.InvestmentTradeReq{
		AssetID:      assetID,
		TxnDate:      today,
		TradeType:    models.InvestmentBuy,
		Quantity:     decimal.NewFromInt(1),
		PricePerUnit: decimal.NewFromInt(90000),
		Currency:     "EUR",
	})
	if err != nil {
		if errors.Is(ctx2.Err(), context.DeadlineExceeded) {
			s.T().Skip("Skipping test: price fetch timed out")
		}
		s.Require().NoError(err)
	}

	// Try to adjust balance to -5k (negative) — should be blocked
	negativeBalance := decimal.NewFromInt(-5000)
	updateReq := &models.AccountReq{
		Name:          "Investment Account",
		AccountTypeID: 5,
		Balance:       &negativeBalance,
	}

	_, err = accSvc.UpdateAccount(s.Ctx, userID, accID, updateReq)
	s.Require().Error(err, "should not allow negative balance adjustment on asset account")

	// Verify balance unchanged — should still be 10k (100k - 90k buy)
	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)

	expectedBalance := decimal.NewFromInt(10000)
	s.Assert().True(expectedBalance.Equal(latestBalance.EndBalance),
		"balance should remain at %s, got %s",
		expectedBalance.String(), latestBalance.EndBalance.String())
}

// Tests that manual balance adjustment is blocked if it would drop available balance below goal allocations.
func (s *AccountServiceTestSuite) TestUpdateAccount_BlockedByGoalAllocation() {
	accSvc := s.TC.App.AccountService
	savSvc := s.TC.App.SavingsService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	initialBalance := decimal.NewFromInt(1000)

	accID, err := accSvc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Savings Account",
		AccountTypeID: 2,
		Balance:       &initialBalance,
		OpenedAt:      today,
	})
	s.Require().NoError(err)

	alloc := decimal.NewFromInt(500)
	goalID, err := savSvc.InsertGoal(s.Ctx, userID, &models.SavingGoalReq{
		AccountID:         accID,
		Name:              "Holiday Fund",
		TargetAmount:      decimal.NewFromInt(5000),
		MonthlyAllocation: &alloc,
	})
	s.Require().NoError(err)

	goalWithProgress, err := savSvc.FetchGoalByID(s.Ctx, userID, goalID)
	s.Require().NoError(err)

	// Fund goal so $500 is allocated — uncategorized balance = $500
	_, _, err = savSvc.AutoFundGoal(s.Ctx, goalWithProgress.SavingGoal, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)

	// Try to adjust balance down to $200 — removes $800, but only $500 is free
	newBalance := decimal.NewFromInt(200)
	_, err = accSvc.UpdateAccount(s.Ctx, userID, accID, &models.AccountReq{
		Name:          "Savings Account",
		AccountTypeID: 2,
		Balance:       &newBalance,
	})
	s.Require().Error(err, "should block balance adjustment that eats into goal allocations")

	// Balance must remain at $1000
	var latestBalance models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ?", accID).
		Order("as_of DESC").
		First(&latestBalance).Error
	s.Require().NoError(err)
	s.Assert().True(initialBalance.Equal(latestBalance.EndBalance),
		"balance should remain at %s, got %s",
		initialBalance.String(), latestBalance.EndBalance.String())
}

// Merging two cash accounts moves all transactions to the destination
// and closes the source account
func (s *AccountServiceTestSuite) TestMergeAccount_Success() {
	svc := s.TC.App.AccountService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)
	zero := decimal.Zero

	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Source Checking",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Dest Checking",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID:       srcID,
		TransactionType: "income",
		Amount:          decimal.NewFromInt(1000),
		TxnDate:         time.Now(),
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID:       srcID,
		TransactionType: "expense",
		Amount:          decimal.NewFromInt(200),
		TxnDate:         time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Require().NoError(err)

	// Source account should be closed
	var src models.Account
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", srcID).First(&src).Error
	s.Require().NoError(err)
	s.Assert().NotNil(src.ClosedAt, "source account should be closed")

	// Both transactions should now belong to destination
	var count int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.Transaction{}).
		Where("account_id = ? AND deleted_at IS NULL", dstID).
		Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), count, "both transactions should be on destination")

	// Destination balance should reflect the moved transactions
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var bal models.Balance
	err = s.TC.DB.WithContext(s.Ctx).
		Where("account_id = ? AND as_of = ?", dstID, today).
		First(&bal).Error
	s.Require().NoError(err)
	s.Assert().True(bal.CashInflows.Equal(decimal.NewFromInt(1000)),
		"destination should have 1000 inflows, got %s", bal.CashInflows)
	s.Assert().True(bal.CashOutflows.Equal(decimal.NewFromInt(200)),
		"destination should have 200 outflows, got %s", bal.CashOutflows)
}

// Merging an account into itself should return an error
func (s *AccountServiceTestSuite) TestMergeAccount_SameAccount() {
	svc := s.TC.App.AccountService
	userID := int64(1)
	zero := decimal.Zero

	accID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Single Account",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, accID, accID)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "must be different")
}

// A transfer between source and destination gets soft-deleted
// and both its transactions flagged as adjustments
func (s *AccountServiceTestSuite) TestMergeAccount_IntraTransferVoided() {
	svc := s.TC.App.AccountService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)
	zero := decimal.Zero

	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Transfer Source",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Transfer Dest",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID:       srcID,
		TransactionType: "income",
		Amount:          decimal.NewFromInt(500),
		TxnDate:         time.Now(),
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransfer(s.Ctx, userID, &models.TransferReq{
		SourceID:      srcID,
		DestinationID: dstID,
		Amount:        decimal.NewFromInt(500),
		CreatedAt:     time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Require().NoError(err)

	// Transfer should be soft-deleted
	var transfer models.Transfer
	err = s.TC.DB.WithContext(s.Ctx).Where("user_id = ?", userID).First(&transfer).Error
	s.Require().NoError(err)
	s.Assert().NotNil(transfer.DeletedAt, "transfer should be soft-deleted")

	// Both transfer transactions should be flagged as adjustments
	var adjustmentCount int64
	err = s.TC.DB.WithContext(s.Ctx).Model(&models.Transaction{}).
		Where("is_adjustment = true AND user_id = ?", userID).
		Count(&adjustmentCount).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(2), adjustmentCount, "both transfer transactions should be flagged as adjustments")
}

// Investment accounts can only merge into the same type and sub-type
func (s *AccountServiceTestSuite) TestMergeAccount_InvestmentGuard_SubtypeMismatch() {
	svc := s.TC.App.AccountService
	userID := int64(1)
	zero := decimal.Zero

	// investment/brokerage = type ID 5, investment/retirement = type ID 6
	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "My Brokerage",
		AccountTypeID: 5,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "My Retirement",
		AccountTypeID: 6,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "same type and sub-type")
}

// Investment accounts cannot be merged into crypto accounts
func (s *AccountServiceTestSuite) TestMergeAccount_InvestmentGuard_CrossTypeCrypto() {
	svc := s.TC.App.AccountService
	userID := int64(1)
	zero := decimal.Zero

	// investment/brokerage = type ID 5, crypto/wallet = type ID 9
	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "My Brokerage",
		AccountTypeID: 5,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "My Wallet",
		AccountTypeID: 9,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "same type and sub-type")
}

// Liability accounts cannot be merged into asset accounts
func (s *AccountServiceTestSuite) TestMergeAccount_LiabilityGuard() {
	svc := s.TC.App.AccountService
	userID := int64(1)
	zero := decimal.Zero

	// cash/checking = type ID 1, credit_card/credit = type ID 18
	cashAccID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Checking",
		AccountTypeID: 1,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	creditAccID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Credit Card",
		AccountTypeID: 18,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, creditAccID, cashAccID)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "liability")
}

// Merging two accounts with initial balances and transactions on different days
// produces correct balance rows and snapshots on destination, and zeros source.
func (s *AccountServiceTestSuite) TestMergeAccount_BalancesAndSnapshotsWithTransactions() {
	svc := s.TC.App.AccountService
	txnSvc := s.TC.App.TransactionService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	day5ago := today.AddDate(0, 0, -5)
	day2ago := today.AddDate(0, 0, -2)
	openedAt := today.AddDate(0, 0, -10)

	srcInitial := decimal.NewFromInt(1000)
	dstInitial := decimal.NewFromInt(2000)

	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name: "Merge Src Txns", AccountTypeID: 1, Balance: &srcInitial, OpenedAt: openedAt,
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name: "Merge Dst Txns", AccountTypeID: 1, Balance: &dstInitial, OpenedAt: openedAt,
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID: srcID, TransactionType: "income", Amount: decimal.NewFromInt(500), TxnDate: day5ago,
	})
	s.Require().NoError(err)

	_, err = txnSvc.InsertTransaction(s.Ctx, userID, &models.TransactionReq{
		AccountID: srcID, TransactionType: "income", Amount: decimal.NewFromInt(300), TxnDate: day2ago,
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Require().NoError(err)

	// Dest today snapshot = 1000 + 500 + 300 + 2000 = 3800
	expectedTotal := decimal.NewFromInt(3800)
	var dstSnap models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", dstID, today).First(&dstSnap).Error
	s.Require().NoError(err)
	s.Assert().True(expectedTotal.Equal(dstSnap.EndBalance),
		"dest today snapshot should be %s, got %s", expectedTotal, dstSnap.EndBalance)

	// Source today snapshot = 0
	var srcSnap models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", srcID, today).First(&srcSnap).Error
	s.Require().NoError(err)
	s.Assert().True(srcSnap.EndBalance.IsZero(),
		"source today snapshot should be 0, got %s", srcSnap.EndBalance)

	// Dest opening balance row start_balance = 1000 + 2000 = 3000
	var dstOpeningBal models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", dstID, openedAt).First(&dstOpeningBal).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(3000).Equal(dstOpeningBal.StartBalance),
		"dest opening start_balance should be 3000, got %s", dstOpeningBal.StartBalance)

	// Dest day-5 balance row has 500 inflow
	var dstBal5 models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", dstID, day5ago).First(&dstBal5).Error
	s.Require().NoError(err)
	s.Assert().True(decimal.NewFromInt(500).Equal(dstBal5.CashInflows),
		"dest day-5 cash_inflows should be 500, got %s", dstBal5.CashInflows)

	// Source opening balance row start_balance = 0 (transferred to dest)
	var srcOpeningBal models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", srcID, openedAt).First(&srcOpeningBal).Error
	s.Require().NoError(err)
	s.Assert().True(srcOpeningBal.StartBalance.IsZero(),
		"source opening start_balance should be 0, got %s", srcOpeningBal.StartBalance)

	// Source day-5 balance row cash_inflows = 0 (zeroed after move)
	var srcBal5 models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", srcID, day5ago).First(&srcBal5).Error
	s.Require().NoError(err)
	s.Assert().True(srcBal5.CashInflows.IsZero(),
		"source day-5 cash_inflows should be 0, got %s", srcBal5.CashInflows)
}

// Merging a source account that has only an initial balance (no transactions)
// transfers that balance to destination and zeroes source.
func (s *AccountServiceTestSuite) TestMergeAccount_SourceNoTransactions_InitialBalanceTransferred() {
	svc := s.TC.App.AccountService
	userID := int64(1)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	openedAt := today.AddDate(0, 0, -10)

	srcInitial := decimal.NewFromInt(5000)
	dstInitial := decimal.NewFromInt(3000)

	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name: "Merge Src NoTxns", AccountTypeID: 1, Balance: &srcInitial, OpenedAt: openedAt,
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name: "Merge Dst NoTxns", AccountTypeID: 1, Balance: &dstInitial, OpenedAt: openedAt,
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Require().NoError(err)

	// Dest today snapshot = 5000 + 3000 = 8000
	expectedTotal := decimal.NewFromInt(8000)
	var dstSnap models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", dstID, today).First(&dstSnap).Error
	s.Require().NoError(err)
	s.Assert().True(expectedTotal.Equal(dstSnap.EndBalance),
		"dest today snapshot should be %s, got %s", expectedTotal, dstSnap.EndBalance)

	// Source today snapshot = 0
	var srcSnap models.AccountDailySnapshot
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", srcID, today).First(&srcSnap).Error
	s.Require().NoError(err)
	s.Assert().True(srcSnap.EndBalance.IsZero(),
		"source today snapshot should be 0, got %s", srcSnap.EndBalance)

	// Dest opening start_balance = 8000
	var dstOpeningBal models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", dstID, openedAt).First(&dstOpeningBal).Error
	s.Require().NoError(err)
	s.Assert().True(expectedTotal.Equal(dstOpeningBal.StartBalance),
		"dest opening start_balance should be %s, got %s", expectedTotal, dstOpeningBal.StartBalance)

	// Source opening start_balance = 0
	var srcOpeningBal models.Balance
	err = s.TC.DB.WithContext(s.Ctx).Where("account_id = ? AND as_of = ?", srcID, openedAt).First(&srcOpeningBal).Error
	s.Require().NoError(err)
	s.Assert().True(srcOpeningBal.StartBalance.IsZero(),
		"source opening start_balance should be 0, got %s", srcOpeningBal.StartBalance)
}

// Two liability accounts of different sub-types can be merged
func (s *AccountServiceTestSuite) TestMergeAccount_LiabilityToLiability_OK() {
	svc := s.TC.App.AccountService
	userID := int64(1)
	zero := decimal.Zero

	// credit_card/credit = type ID 18, loan/mortgage = type ID 19
	srcID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Old Credit Card",
		AccountTypeID: 18,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	dstID, err := svc.InsertAccount(s.Ctx, userID, &models.AccountReq{
		Name:          "Mortgage",
		AccountTypeID: 19,
		Balance:       &zero,
		OpenedAt:      time.Now(),
	})
	s.Require().NoError(err)

	err = svc.MergeAccount(s.Ctx, userID, srcID, dstID)
	s.Assert().NoError(err)

	var src models.Account
	err = s.TC.DB.WithContext(s.Ctx).Where("id = ?", srcID).First(&src).Error
	s.Require().NoError(err)
	s.Assert().NotNil(src.ClosedAt)
}
