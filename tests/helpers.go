package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	wwHttp "wealth-warden/internal/http"
)

func createTestLedgerAccount(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name string) {
	t.Helper()

	requestBody := map[string]interface{}{
		"name":            name,
		"account_type_id": 1,
		"type":            "bank",
		"sub_type":        "checking",
		"classification":  "asset",
		"balance":         "0.00",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/accounts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req, accessToken, refreshToken)

	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Failed to create account. Status: %d, Body: %s", w.Code, w.Body.String())
		t.Fatalf("Could not create test account")
	}
}

func findLedgerAccountByName(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name string, includeInactive bool) map[string]interface{} {
	t.Helper()

	path := "/api/v1/accounts/all"
	if includeInactive {
		path += "?inactive=true"
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	addAuth(req, accessToken, refreshToken)

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

func getLedgerAccountByName(t *testing.T, s *wwHttp.Server, accessToken, refreshToken, name string) map[string]interface{} {
	t.Helper()

	path := fmt.Sprintf("/api/v1/accounts/name/%s", name)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	addAuth(req, accessToken, refreshToken)

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

func addAuth(req *http.Request, accessToken, refreshToken string) {

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
