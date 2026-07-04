package auth

import (
	"testing"
	"time"
)

func TestTokenServiceIssueAndParse(t *testing.T) {
	svc := NewTokenService("test-secret-key-at-least-32-chars", time.Hour)

	token, expiresAt, err := svc.Issue("user-1", "admin")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expected future expiry")
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("expected sub user-1, got %q", claims.Subject)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected role admin, got %q", claims.Role)
	}
}

func TestTokenServiceParseInvalid(t *testing.T) {
	svc := NewTokenService("test-secret-key-at-least-32-chars", time.Hour)

	_, err := svc.Parse("not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestTokenServiceWrongSecret(t *testing.T) {
	issuer := NewTokenService("issuer-secret-key-32-characters!!", time.Hour)
	validator := NewTokenService("other-secret-key-32-characters!!!", time.Hour)

	token, _, err := issuer.Issue("user-1", "viewer")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = validator.Parse(token)
	if err == nil {
		t.Fatal("expected validation failure with different secret")
	}
}
