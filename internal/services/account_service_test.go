package services_test

import (
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
		Type:           "liability",
		Subtype:        "credit_card",
		Classification: "current",
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
		Type:           "liability",
		Subtype:        "credit_card",
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
