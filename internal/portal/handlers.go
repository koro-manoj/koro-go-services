package portal

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/koro/koro-go-services/internal/auth"
	"github.com/koro/koro-go-services/internal/config"
)

type demoUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Handlers exposes portal-compatible REST endpoints for the Next.js dashboard.
type Handlers struct {
	settings *config.Store
	tokens   *auth.TokenService
}

func NewHandlers(settings *config.Store, tokens *auth.TokenService) *Handlers {
	return &Handlers{settings: settings, tokens: tokens}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	user, ok := h.authenticate(req.Email, req.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, _, err := h.tokens.Issue(user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"token":      token,
			"token_type": "Bearer",
			"expires_in": 3600,
			"user":       user,
		},
	})
}

func (h *Handlers) Logout(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user := demoUser{
		ID:    1,
		Name:  h.settings.GetDefault("auth.demo.name", "Alex Rivera"),
		Email: claims.Subject,
		Role:  claims.Role,
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

func (h *Handlers) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.ClaimsFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"stats": map[string]any{
				"revenue":    statBlock(284750, 12.4),
				"orders":     statBlock(1842, 8.1),
				"customers":  statBlock(9673, 3.2),
				"conversion": statBlock(4.28, -0.3),
			},
			"recent_activity": []map[string]any{
				{"id": "1", "type": "order", "title": "Order #48291 fulfilled", "description": "Enterprise plan — Acme Industries", "timestamp": time.Now().Add(-12 * time.Minute).UTC().Format(time.RFC3339), "meta": map[string]any{"amount": 2499}},
				{"id": "2", "type": "customer", "title": "New customer onboarded", "description": "Meridian Labs signed up for Pro tier", "timestamp": time.Now().Add(-45 * time.Minute).UTC().Format(time.RFC3339)},
				{"id": "3", "type": "payment", "title": "Payment received", "description": "Invoice #INV-0892 — $12,400", "timestamp": time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339), "meta": map[string]any{"amount": 12400}},
			},
		},
	})
}

func (h *Handlers) authenticate(email, password string) (demoUser, bool) {
	expectedEmail := h.settings.GetDefault("auth.demo.email", "demo@koro.io")
	expectedPassword := h.settings.GetDefault("auth.demo.password", "password")
	expectedName := h.settings.GetDefault("auth.demo.name", "Alex Rivera")
	expectedRole := h.settings.GetDefault("auth.demo.role", "admin")

	if strings.EqualFold(strings.TrimSpace(email), expectedEmail) && password == expectedPassword {
		return demoUser{ID: 1, Name: expectedName, Email: expectedEmail, Role: expectedRole}, true
	}

	return demoUser{}, false
}

func statBlock(value float64, change float64, _ ...bool) map[string]any {
	return map[string]any{
		"value":  value,
		"change": change,
		"period": "vs last month",
	}
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]any{"message": message})
}
