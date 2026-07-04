package config

import (
	"fmt"
	"os"
)

// Env holds bootstrap settings that cannot live in the database (connection strings, signing key).
type Env struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

func LoadEnv() (Env, error) {
	cfg := Env{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    getenv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return Env{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Env{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
