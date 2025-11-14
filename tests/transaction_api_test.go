package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/database/seeders/workers"

	"github.com/stretchr/testify/assert"
)

const txnApiEndpoint = "/api/transactions"

func getDefaultCategory(t *testing.T) int64 {
	t.Helper()

	ctx := context.Background()

	if err := workers.SeedCategories(ctx, testDB, testLogger, testCfg); err != nil {
		t.Fatalf("Failed to create categories: %v", err)
	}

	var count int64
	testDB.Model(&models.Category{}).Count(&count)
	if count == 0 {
		t.Fatal("No categories were created")
	}

	var category models.Category
	result := testDB.Where("name = ?", "(uncategorized)").First(&category)
	if result.Error != nil {
		t.Fatalf("Failed to find 'uncategorized' category: %v", result.Error)
	}

	return category.ID
}

func TestInsertValidTransaction(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)
	catID := getDefaultCategory(t)

	// Create account
	const name = "Test"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name, "0")
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accID := int64(acc["id"].(float64))
	accBalance := acc["balance"].(map[string]interface{})
	startBalance := accBalance["end_balance"].(string)

	// Create txn
	createTestTransaction(t, s, accessToken, refreshToken, accID, catID, "income", "100", nil)

	// Check new balance state
	acc1 := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accBalance1 := acc1["balance"].(map[string]interface{})
	endBalance := accBalance1["end_balance"].(string)

	if startBalance == endBalance {
		t.Errorf("Expected balance to change after transaction, but it remained %s", startBalance)
	}
}

func TestGetTransactions(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	// Create a txn
	const name = "Test"
	desc := "Test Transaction - Unique Description 12345"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name, "0")
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accID := int64(acc["id"].(float64))
	catID := getDefaultCategory(t)
	createTestTransaction(t, s, accessToken, refreshToken, accID, catID, "income", "100", &desc)

	// Test getting txns
	req := httptest.NewRequest(http.MethodGet, txnApiEndpoint, nil)
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("Status: %d", w.Code)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	txns, ok := response["data"].([]interface{})
	assert.True(t, ok, "Response should contain 'data' array")
	assert.NotEmpty(t, txns, "Data array should not be empty")
	assert.GreaterOrEqual(t, len(txns), 1, "Should have at least 1 transaction")

	// Find the transaction by unique description
	found := false
	for _, txn := range txns {
		txnMap := txn.(map[string]interface{})
		if d, ok := txnMap["description"].(string); ok && d == desc {
			found = true
			// Verify key fields match
			assert.Equal(t, float64(accID), txnMap["account_id"].(float64))
			assert.Equal(t, float64(catID), txnMap["category_id"].(float64))
			assert.Equal(t, "income", txnMap["transaction_type"])
			assert.Equal(t, "100", txnMap["amount"])
			assert.Equal(t, "EUR", txnMap["currency"])
			assert.False(t, txnMap["is_adjustment"].(bool))
			assert.False(t, txnMap["is_transfer"].(bool))

			t.Logf("Found transaction with ID: %.0f", txnMap["id"].(float64))
			break
		}
	}
	assert.True(t, found, "Created transaction with unique description should be in the results")
}

func TestDeleteTransaction(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)
	catID := getDefaultCategory(t)

	const name = "Test"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name, "0")
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accID := int64(acc["id"].(float64))
	desc := "Transaction to Delete - Unique 67890"
	createTestTransaction(t, s, accessToken, refreshToken, accID, catID, "income", "100", &desc)

	listReq := httptest.NewRequest(http.MethodGet, txnApiEndpoint, nil)
	addAuth(listReq, accessToken, refreshToken)
	listW := httptest.NewRecorder()
	s.Router.ServeHTTP(listW, listReq)

	assert.Equal(t, http.StatusOK, listW.Code)

	var listResponse map[string]interface{}
	err := json.Unmarshal(listW.Body.Bytes(), &listResponse)
	assert.NoError(t, err)

	txns := listResponse["data"].([]interface{})
	assert.NotEmpty(t, txns, "Should have at least one transaction")

	// Find the transaction by description
	var txnID int64
	found := false
	for _, txn := range txns {
		txnMap := txn.(map[string]interface{})
		if d, ok := txnMap["description"].(string); ok && d == desc {
			txnID = int64(txnMap["id"].(float64))
			found = true
			t.Logf("Found transaction ID to delete: %d", txnID)
			break
		}
	}
	assert.True(t, found, "Should find the created transaction")

	// Delete the transaction
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%d", txnApiEndpoint, txnID), nil)
	addAuth(delReq, accessToken, refreshToken)
	delW := httptest.NewRecorder()
	s.Router.ServeHTTP(delW, delReq)

	t.Logf("Delete status: %d", delW.Code)
	assert.True(t, delW.Code == http.StatusOK || delW.Code == http.StatusNoContent, "Expected 200 or 204")

	// Verify transaction no longer appears in the list (soft-deleted records should be excluded)
	listReq2 := httptest.NewRequest(http.MethodGet, txnApiEndpoint, nil)
	addAuth(listReq2, accessToken, refreshToken)
	listW2 := httptest.NewRecorder()
	s.Router.ServeHTTP(listW2, listReq2)

	assert.Equal(t, http.StatusOK, listW2.Code)

	var listResponse2 map[string]interface{}
	err = json.Unmarshal(listW2.Body.Bytes(), &listResponse2)
	assert.NoError(t, err)

	txns2 := listResponse2["data"].([]interface{})

	// Verify the deleted transaction is not in the list
	foundAfterDelete := false
	for _, txn := range txns2 {
		txnMap := txn.(map[string]interface{})
		if d, ok := txnMap["description"].(string); ok && d == desc {
			foundAfterDelete = true
			break
		}
	}
	assert.False(t, foundAfterDelete, "Soft-deleted transaction should not appear in transaction list")
}

func TestPositiveTransfer(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	// Create source account
	createTestLedgerAccount(t, s, accessToken, refreshToken, "Source", "600")
	sourceAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceID := int64(sourceAcc["id"].(float64))
	sourceBalance := sourceAcc["balance"].(map[string]interface{})
	startBalance := sourceBalance["end_balance"].(string)

	if startBalance != "600" {
		t.Fatalf("Expected source start balance 600, got %s", startBalance)
	}

	// Create destination account
	createTestLedgerAccount(t, s, accessToken, refreshToken, "Destination", "0")
	destAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destID := int64(destAcc["id"].(float64))

	// Create transfer of 100 from source to destination
	notes := "Test transfer"
	createTestTransfer(t, s, accessToken, refreshToken, sourceID, destID, "100", &notes)

	// Check new balance state
	sourceAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceBalance = sourceAcc["balance"].(map[string]interface{})
	sourceEndBalance := sourceBalance["end_balance"].(string)

	if sourceEndBalance != "500" {
		t.Errorf("Expected source balance 500, got %s", sourceEndBalance)
	}

	// Check destination account balance - should have 100
	destAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destBalance := destAcc["balance"].(map[string]interface{})
	destEndBalance := destBalance["end_balance"].(string)

	if destEndBalance != "100" {
		t.Errorf("Expected destination balance 100, got %s", destEndBalance)
	}

}

func TestNegativeTransfer(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)
	catID := getDefaultCategory(t)

	// Create source account
	createTestLedgerAccount(t, s, accessToken, refreshToken, "Source", "600")
	sourceAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceID := int64(sourceAcc["id"].(float64))
	sourceBalance := sourceAcc["balance"].(map[string]interface{})
	startBalance := sourceBalance["end_balance"].(string)

	if startBalance != "600" {
		t.Fatalf("Expected source start balance 600, got %s", startBalance)
	}

	// Create destination account
	createTestLedgerAccount(t, s, accessToken, refreshToken, "Destination", "0")
	destAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destID := int64(destAcc["id"].(float64))

	// Create negative balance
	createTestTransaction(t, s, accessToken, refreshToken, destID, catID, "expense", "600", nil)

	// Create transfer of 100 from source to destination
	notes := "Test transfer"
	createTestTransfer(t, s, accessToken, refreshToken, sourceID, destID, "300", &notes)

	// Check new balance state
	sourceAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceBalance = sourceAcc["balance"].(map[string]interface{})
	sourceEndBalance := sourceBalance["end_balance"].(string)

	if sourceEndBalance != "300" {
		t.Errorf("Expected source balance 300, got %s", sourceEndBalance)
	}

	// Check destination account balance - should have -300
	destAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destBalance := destAcc["balance"].(map[string]interface{})
	destEndBalance := destBalance["end_balance"].(string)

	if destEndBalance != "-300" {
		t.Errorf("Expected destination balance -300, got %s", destEndBalance)
	}

}

func TestDeleteTransfer(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	createTestLedgerAccount(t, s, accessToken, refreshToken, "Source", "600")
	sourceAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceID := int64(sourceAcc["id"].(float64))
	sourceBalance := sourceAcc["balance"].(map[string]interface{})
	startBalance := sourceBalance["end_balance"].(string)

	if startBalance != "600" {
		t.Fatalf("Expected source start balance 600, got %s", startBalance)
	}

	createTestLedgerAccount(t, s, accessToken, refreshToken, "Destination", "0")
	destAcc := getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destID := int64(destAcc["id"].(float64))

	notes := "Test transfer to delete - Unique 12345"
	createTestTransfer(t, s, accessToken, refreshToken, sourceID, destID, "100", &notes)

	// Check balances after transfer
	sourceAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceBalance = sourceAcc["balance"].(map[string]interface{})
	sourceEndBalance := sourceBalance["end_balance"].(string)

	if sourceEndBalance != "500" {
		t.Errorf("Expected source balance after transfer 500, got %s", sourceEndBalance)
	}

	destAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destBalance := destAcc["balance"].(map[string]interface{})
	destEndBalance := destBalance["end_balance"].(string)

	if destEndBalance != "100" {
		t.Errorf("Expected destination balance after transfer 100, got %s", destEndBalance)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/transactions/transfers", nil)
	addAuth(listReq, accessToken, refreshToken)
	listW := httptest.NewRecorder()
	s.Router.ServeHTTP(listW, listReq)

	assert.Equal(t, http.StatusOK, listW.Code)

	var listResponse map[string]interface{}
	err := json.Unmarshal(listW.Body.Bytes(), &listResponse)
	assert.NoError(t, err)

	transfers := listResponse["data"].([]interface{})
	assert.NotEmpty(t, transfers, "Should have at least one transfer")

	// Find the transfer by notes
	var transferID int64
	found := false
	for _, transfer := range transfers {
		transferMap := transfer.(map[string]interface{})
		if n, ok := transferMap["notes"].(string); ok && n == notes {
			transferID = int64(transferMap["id"].(float64))
			found = true
			t.Logf("Found transfer ID to delete: %d", transferID)
			break
		}
	}
	assert.True(t, found, "Should find the created transfer")

	// Delete the transfer
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/transactions/transfers/%d", transferID), nil)
	addAuth(delReq, accessToken, refreshToken)
	delW := httptest.NewRecorder()
	s.Router.ServeHTTP(delW, delReq)

	t.Logf("Delete status: %d", delW.Code)
	assert.True(t, delW.Code == http.StatusOK || delW.Code == http.StatusNoContent, "Expected 200 or 204")

	// Check balances are reversed - source should be back to 600
	sourceAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Source")
	sourceBalance = sourceAcc["balance"].(map[string]interface{})
	sourceEndBalance = sourceBalance["end_balance"].(string)

	if sourceEndBalance != "600" {
		t.Errorf("Expected source balance after delete 600, got %s", sourceEndBalance)
	}

	// Destination should be back to 0
	destAcc = getLedgerAccountByName(t, s, accessToken, refreshToken, "Destination")
	destBalance = destAcc["balance"].(map[string]interface{})
	destEndBalance = destBalance["end_balance"].(string)

	if destEndBalance != "0" {
		t.Errorf("Expected destination balance after delete 0, got %s", destEndBalance)
	}

	// Verify transfer no longer appears in the list
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/transactions/transfers", nil)
	addAuth(listReq2, accessToken, refreshToken)
	listW2 := httptest.NewRecorder()
	s.Router.ServeHTTP(listW2, listReq2)

	assert.Equal(t, http.StatusOK, listW2.Code)

	var listResponse2 map[string]interface{}
	err = json.Unmarshal(listW2.Body.Bytes(), &listResponse2)
	assert.NoError(t, err)

	transfers2 := listResponse2["data"].([]interface{})

	foundAfterDelete := false
	for _, transfer := range transfers2 {
		transferMap := transfer.(map[string]interface{})
		if n, ok := transferMap["notes"].(string); ok && n == notes {
			foundAfterDelete = true
			break
		}
	}
	assert.False(t, foundAfterDelete, "Soft-deleted transfer should not appear in transfer list")
}
