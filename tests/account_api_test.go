package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLedgerAccount(t *testing.T) {

	s := setupTestServer(t)
	cleanupTestData(t)

	userID, accessToken, refreshToken := createRootUser(t)
	t.Logf("Test User ID: %d", userID)

	// Prepare the request body
	requestBody := map[string]interface{}{
		"name":            "My Checking Account",
		"account_type_id": 1,
		"type":            "bank",
		"sub_type":        "checking",
		"classification":  "asset",
		"balance":         "100.00",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPut, "/api/v1/accounts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()

	s.Router.ServeHTTP(w, req)

	t.Logf("Status Code: %d", w.Code)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

}

func TestGetAllLedgerAccounts(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	// Create an account first
	createTestLedgerAccount(t, s, accessToken, refreshToken, "Test Account")

	// Test getting all accounts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/all", nil)
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("Status: %d", w.Code)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetLedgerAccountByID(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	const name = "Savings Account (test)"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name)
	id := getLedgerAccountIDByName(t, s, accessToken, refreshToken, name)

	url := fmt.Sprintf("/api/v1/accounts/%d", id)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("Status: %d", w.Code)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreateLedgerAccount_MissingName(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	// Missing "name" field
	requestBody := map[string]interface{}{
		"account_type_id": 1,
		"type":            "bank",
		"sub_type":        "checking",
		"classification":  "asset",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/accounts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("Validation test - Status: %d, Body: %s", w.Code, w.Body.String())
	assert.True(t, w.Code >= 400 && w.Code < 500, "Should return 4xx error")
}
