package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

const QueueKey = "webhooks:incoming"

// Job is the payload enqueued for asynchronous webhook processing.
type Job struct {
	ID        string            `json:"id"`
	Source    string            `json:"source"`
	Event     string            `json:"event"`
	Payload   json.RawMessage   `json:"payload"`
	Headers   map[string]string `json:"headers,omitempty"`
	Received  time.Time         `json:"received_at"`
}

// Enqueuer pushes webhook jobs onto Redis.
type Enqueuer interface {
	Enqueue(ctx context.Context, job Job) error
}

// Processor handles a dequeued webhook job.
type Processor struct {
	verify func(source string, payload []byte, signature string) error
}

func NewProcessor(verify func(source string, payload []byte, signature string) error) *Processor {
	return &Processor{verify: verify}
}

func (p *Processor) Handle(ctx context.Context, job Job) error {
	if job.ID == "" {
		return fmt.Errorf("job id is required")
	}
	if job.Source == "" {
		return fmt.Errorf("job source is required")
	}

	sig := ""
	if job.Headers != nil {
		sig = job.Headers["X-Webhook-Signature"]
	}

	if p.verify != nil {
		if err := p.verify(job.Source, job.Payload, sig); err != nil {
			return fmt.Errorf("verify webhook: %w", err)
		}
	}

	switch job.Event {
	case "ping":
		slog.Info("webhook.ping", "source", job.Source, "job_id", job.ID)
		return nil
	case "payment.succeeded", "checkout.session.completed":
		var payload map[string]any
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("decode payment payload: %w", err)
		}
		slog.Info("webhook.payment", "source", job.Source, "event", job.Event, "job_id", job.ID)
		return nil
	case "order.created":
		slog.Info("webhook.order", "source", job.Source, "job_id", job.ID)
		return nil
	default:
		slog.Info("webhook.received", "source", job.Source, "event", job.Event, "job_id", job.ID)
		return nil
	}
}
