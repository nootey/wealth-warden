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
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name, false)
	id := int64(acc["id"].(float64))

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

func TestUpdateLedgerAccount(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	const name = "Update Me (test)"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name)
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name, false)
	id := int64(acc["id"].(float64))

	updated := map[string]interface{}{
		"name":            "Updated Name (test)",
		"account_type_id": 1,
		"type":            "bank",
		"sub_type":        "checking",
		"classification":  "asset",
	}
	body, _ := json.Marshal(updated)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/accounts/%d", id), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("update status: %d", w.Code)
	assert.Equal(t, http.StatusOK, w.Code)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%d", id), nil)
	addAuth(getReq, accessToken, refreshToken)
	getW := httptest.NewRecorder()
	s.Router.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)
}

func TestToggleLedgerAccount(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	const name = "Toggle Me (test)"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name)
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name, false)
	origActive, _ := acc["is_active"].(bool)
	id := int64(acc["id"].(float64))

	// toggle
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/active", id), nil)
	addAuth(req, accessToken, refreshToken)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	t.Logf("toggle status: %d", w.Code)
	assert.Equal(t, http.StatusOK, w.Code)

	acc2 := getLedgerAccountByName(t, s, accessToken, refreshToken, name, true)
	newActive, _ := acc2["is_active"].(bool)

	t.Logf("before: %v, after: %v", origActive, newActive)
	assert.NotEqual(t, origActive, newActive, "is_active should be flipped")

}

func TestDeleteLedgerAccount(t *testing.T) {
	s := setupTestServer(t)
	cleanupTestData(t)
	_, accessToken, refreshToken := createRootUser(t)

	const name = "Delete Me (test)"
	createTestLedgerAccount(t, s, accessToken, refreshToken, name)
	acc := getLedgerAccountByName(t, s, accessToken, refreshToken, name, false)
	id := int64(acc["id"].(float64))

	// delete
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/accounts/%d", id), nil)
	addAuth(delReq, accessToken, refreshToken)
	delW := httptest.NewRecorder()
	s.Router.ServeHTTP(delW, delReq)

	t.Logf("delete status: %d", delW.Code)
	assert.True(t, delW.Code == http.StatusOK || delW.Code == http.StatusNoContent, "expected 200 or 204")

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%d", id), nil)
	addAuth(getReq, accessToken, refreshToken)
	getW := httptest.NewRecorder()
	s.Router.ServeHTTP(getW, getReq)
	assert.NotEqual(t, http.StatusOK, getW.Code)
}
