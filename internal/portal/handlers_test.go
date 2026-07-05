package portal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/koro/koro-go-services/internal/auth"
	"github.com/koro/koro-go-services/internal/config"
)

func TestPortalLoginAndDashboardOverview(t *testing.T) {
	settings := config.NewMemoryStore(map[string]string{
		"auth.demo.email":    "demo@koro.io",
		"auth.demo.password": "password",
		"auth.demo.name":     "Alex Rivera",
		"auth.demo.role":     "admin",
	})
	tokens := auth.NewTokenService("test-secret-key-at-least-32-chars-long", time.Hour)
	h := NewHandlers(settings, tokens)

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "demo@koro.io",
		"password": "password",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	h.Login(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", loginRec.Code)
	}

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if loginResp.Data.Token == "" {
		t.Fatal("expected token in login response")
	}

	claims, err := tokens.Parse(loginResp.Data.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRec := httptest.NewRecorder()
	h.Me(meRec, meReq.WithContext(auth.WithClaims(meReq.Context(), claims)))

	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want 200", meRec.Code)
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	overviewRec := httptest.NewRecorder()
	h.DashboardOverview(overviewRec, overviewReq.WithContext(auth.WithClaims(overviewReq.Context(), claims)))

	if overviewRec.Code != http.StatusOK {
		t.Fatalf("overview status = %d, want 200", overviewRec.Code)
	}

	var overviewResp struct {
		Data struct {
			Stats map[string]any `json:"stats"`
		} `json:"data"`
	}
	if err := json.NewDecoder(overviewRec.Body).Decode(&overviewResp); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overviewResp.Data.Stats == nil {
		t.Fatal("expected stats in overview response")
	}
}

func TestPortalLoginRejectsInvalidCredentials(t *testing.T) {
	settings := config.NewMemoryStore(map[string]string{})
	tokens := auth.NewTokenService("test-secret-key-at-least-32-chars-long", time.Hour)
	h := NewHandlers(settings, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":    "wrong@example.com",
		"password": "bad",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
