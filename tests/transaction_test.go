package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/database/seeders/workers"
)

const txnApiEndpoint = "/api/v1/transactions"

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

	t.Logf("Found 'uncategorized' category with ID: %d", category.ID)
	return int64(category.ID)
}

func TestInsertValidTransaction(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)
	catID := getDefaultCategory(t)

	// Create account
	const name = "Test"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name)
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accID := int64(acc["id"].(float64))
	accBalance := acc["balance"].(map[string]interface{})
	startBalance := accBalance["end_balance"].(string)

	// Create txn
	requestBody := map[string]interface{}{
		"account_id":       accID,
		"transaction_type": "income",
		"amount":           "100",
		"currency":         "EUR",
		"category_id":      catID,
		"txn_date":         time.Now(),
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, txnApiEndpoint, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("Failed to create transaction. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	// Check new balance state
	acc1 := getLedgerAccountByName(t, s, accessToken, refreshToken, name)
	accBalance1 := acc1["balance"].(map[string]interface{})
	endBalance := accBalance1["end_balance"].(string)

	if startBalance == endBalance {
		t.Errorf("Expected balance to change after transaction, but it remained %s", startBalance)
	}
}
