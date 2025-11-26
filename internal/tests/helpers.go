package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	wwHttp "wealth-warden/internal/http"
)

func AddAuth(req *http.Request, accessToken, refreshToken string) {

	req.AddCookie(&http.Cookie{
		Name:     "access",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
	})
	req.AddCookie(&http.Cookie{
		Name:     "refresh",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
	})
}

func CreateTestLedgerAccount(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name, balance string) {
	t.Helper()
	requestBody := map[string]interface{}{
		"name":            name,
		"account_type_id": 1,
		"type":            "bank",
		"sub_type":        "checking",
		"classification":  "asset",
		"balance":         balance,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/accounts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	AddAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Failed to create account. Status: %d, Body: %s", w.Code, w.Body.String())
		t.Fatalf("Could not create test account")
	}
}

func FindLedgerAccountByName(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name string, includeInactive bool) map[string]interface{} {
	t.Helper()

	path := "/api/accounts/all"
	if includeInactive {
		path += "?inactive=true"
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	AddAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d %s", w.Code, w.Body.String())
	}

	var accounts []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &accounts); err != nil {
		t.Fatalf("parse list failed: %v", err)
	}

	for _, a := range accounts {
		if a["name"] == name {
			if _, ok := a["id"].(float64); ok {
				return a
			}
		}
	}
	t.Fatalf("account %q not found in list", name)
	return nil
}

func GetLedgerAccountByName(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name string) map[string]interface{} {
	t.Helper()

	path := fmt.Sprintf("/api/accounts/name/%s", name)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	AddAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d %s", w.Code, w.Body.String())
	}

	var account map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &account); err != nil {
		t.Fatalf("parse account failed: %v", err)
	}

	return account
}

func CreateTestTransaction(t *testing.T, s *wwHttp.Server, accessToken, refreshToken string, accID, catID int64, txnType string, amount string, description *string) {

	requestBody := map[string]interface{}{
		"account_id":       accID,
		"transaction_type": txnType,
		"amount":           amount,
		"currency":         "EUR",
		"category_id":      catID,
		"description":      &description,
		"txn_date":         time.Now(),
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/transactions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	AddAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("Failed to create transaction. Status: %d, Body: %s", w.Code, w.Body.String())
	}
}

func CreateTestTransfer(t *testing.T, s *wwHttp.Server, accessToken, refreshToken string, sourceID, destID int64, amount string, notes *string) {
	requestBody := map[string]interface{}{
		"source_id":      sourceID,
		"destination_id": destID,
		"amount":         amount,
		"notes":          notes,
		"created_at":     time.Now(),
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/transactions/transfers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	AddAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("Failed to create transfer. Status: %d, Body: %s", w.Code, w.Body.String())
	}
}
