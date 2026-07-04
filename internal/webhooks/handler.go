package webhooks

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/koro/koro-go-services/internal/auth"
)

type Handler struct {
	queue Enqueuer
}

func NewHandler(queue Enqueuer) *Handler {
	return &Handler{queue: queue}
}

func (h *Handler) Receive(source string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		event := r.Header.Get("X-Webhook-Event")
		if event == "" {
			event = "unknown"
		}

		job := Job{
			ID:       r.Header.Get("X-Webhook-ID"),
			Source:   source,
			Event:    event,
			Payload:  json.RawMessage(body),
			Headers:  map[string]string{"X-Webhook-Signature": r.Header.Get("X-Webhook-Signature")},
			Received: time.Now().UTC(),
		}
		if job.ID == "" {
			job.ID = time.Now().UTC().Format("20060102150405.000")
		}

		if err := h.queue.Enqueue(r.Context(), job); err != nil {
			http.Error(w, "enqueue failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"subject": claims.Subject,
		"role":    claims.Role,
	})
}
