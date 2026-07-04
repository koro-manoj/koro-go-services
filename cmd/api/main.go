package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/koro/koro-go-services/internal/auth"
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

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("ping redis", "error", err)
		os.Exit(1)
	}

	tokens := auth.NewTokenService(env.JWTSecret, time.Hour)
	queue := webhooks.NewRedisQueue(rdb, webhooks.QueueKey)
	webhookHandler := webhooks.NewHandler(queue)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(pool, rdb))
	mux.HandleFunc("POST /auth/token", tokenHandler(tokens))
	mux.Handle("GET /me", auth.Middleware(tokens)(http.HandlerFunc(webhookHandler.Me)))
	mux.HandleFunc("POST /webhooks/{source}", func(w http.ResponseWriter, r *http.Request) {
		source := r.PathValue("source")
		webhookHandler.Receive(source)(w, r)
	})

	server := &http.Server{
		Addr:         env.HTTPAddr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", env.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func healthHandler(pool *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK

		if err := pool.Ping(r.Context()); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

type tokenRequest struct {
	Subject string `json:"subject"`
	Role    string `json:"role"`
}

func tokenHandler(tokens *auth.TokenService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Subject == "" {
			req.Subject = "demo-user"
		}
		if req.Role == "" {
			req.Role = "operator"
		}

		token, expiresAt, err := tokens.Issue(req.Subject, req.Role)
		if err != nil {
			http.Error(w, "issue token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_at":   expiresAt,
		})
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
