package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultRefreshInterval = 30 * time.Second

// Setting represents a row in app_settings.
type Setting struct {
	Key       string
	Value     string
	Encrypted bool
}

// Store loads application settings from PostgreSQL and keeps an in-memory cache.
// Third-party API keys and feature flags belong here — not in environment variables.
type Store struct {
	pool     *pgxpool.Pool
	mu       sync.RWMutex
	cache    map[string]string
	interval time.Duration
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:     pool,
		cache:    make(map[string]string),
		interval: defaultRefreshInterval,
	}
}

// NewMemoryStore builds an in-memory settings store (tests and local defaults).
func NewMemoryStore(values map[string]string) *Store {
	cache := make(map[string]string, len(values))
	for key, value := range values {
		cache[key] = value
	}

	return &Store{
		cache:    cache,
		interval: defaultRefreshInterval,
	}
}

func (s *Store) Start(ctx context.Context) error {
	if err := s.refresh(ctx); err != nil {
		return err
	}

	go s.poll(ctx)
	return nil
}

func (s *Store) poll(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refresh(ctx)
		}
	}
}

func (s *Store) refresh(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		SELECT key, value, encrypted
		FROM app_settings
		WHERE active = true
	`)
	if err != nil {
		return fmt.Errorf("load app settings: %w", err)
	}
	defer rows.Close()

	next := make(map[string]string)
	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.Encrypted); err != nil {
			return fmt.Errorf("scan setting: %w", err)
		}
		next[setting.Key] = setting.Value
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate settings: %w", err)
	}

	s.mu.Lock()
	s.cache = next
	s.mu.Unlock()
	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	return v, ok
}

func (s *Store) MustGet(key string) string {
	v, ok := s.Get(key)
	if !ok {
		panic(fmt.Sprintf("missing required setting: %s", key))
	}
	return v
}

func (s *Store) GetDefault(key, fallback string) string {
	if v, ok := s.Get(key); ok && v != "" {
		return v
	}
	return fallback
}
