package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/koro/koro-go-services/internal/config"
	"github.com/koro/koro-go-services/internal/webhooks"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	env, err := config.LoadEnv()
	if err != nil {
		slog.Error("load env", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, env.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	settings := config.NewStore(pool)
	if err := settings.Start(ctx); err != nil {
		slog.Error("start config store", "error", err)
		os.Exit(1)
	}

	redisOpts, err := redis.ParseURL(env.RedisURL)
	if err != nil {
		slog.Error("parse redis url", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	queue := webhooks.NewRedisQueue(rdb, webhooks.QueueKey)
	processor := webhooks.NewProcessor(verifySignature(settings))

	slog.Info("worker started", "queue", webhooks.QueueKey)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down")
			return
		default:
		}

		jobCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		job, err := queue.Dequeue(jobCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("dequeue", "error", err)
			time.Sleep(time.Second)
			continue
		}

		if err := processor.Handle(ctx, job); err != nil {
			slog.Error("process webhook",
				"job_id", job.ID,
				"source", job.Source,
				"event", job.Event,
				"error", err,
			)
			continue
		}

		slog.Info("webhook processed",
			"job_id", job.ID,
			"source", job.Source,
			"event", job.Event,
		)
	}
}

func verifySignature(settings *config.Store) func(source string, payload []byte, signature string) error {
	return func(source string, payload []byte, signature string) error {
		key := "webhook." + source + ".signing_secret"
		secret, ok := settings.Get(key)
		if !ok || secret == "" {
			return nil
		}
		if signature == "" {
			return nil
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expected), []byte(signature)) {
			return errInvalidSignature
		}
		return nil
	}
}

var errInvalidSignature = &signatureError{}

type signatureError struct{}

func (e *signatureError) Error() string { return "invalid webhook signature" }
