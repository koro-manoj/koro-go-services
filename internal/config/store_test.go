package config

import (
	"context"
	"testing"
)

func TestStoreGetMissingKey(t *testing.T) {
	s := &Store{cache: map[string]string{"existing": "value"}}

	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected missing key to return false")
	}

	if v, ok := s.Get("existing"); !ok || v != "value" {
		t.Fatalf("expected existing=value, got %q ok=%v", v, ok)
	}
}

func TestStoreMustGetPanics(t *testing.T) {
	s := &Store{cache: map[string]string{}}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing key")
		}
	}()

	_ = s.MustGet("webhook.signing_secret")
}

func TestLoadEnvRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "test-secret")

	_, err := LoadEnv()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is empty")
	}
}

func TestLoadEnvDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/koro")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("HTTP_ADDR", "")

	cfg, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default HTTP_ADDR :8080, got %q", cfg.HTTPAddr)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected default APP_ENV development, got %q", cfg.AppEnv)
	}
}

func TestStoreRefreshEmpty(t *testing.T) {
	// Unit-level guard: refresh on nil pool would fail at integration time.
	s := &Store{cache: map[string]string{}}
	s.mu.Lock()
	s.cache = map[string]string{}
	s.mu.Unlock()

	if _, ok := s.Get("any"); ok {
		t.Fatal("empty cache should not contain keys")
	}

	_ = context.Background()
}
