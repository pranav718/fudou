package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenGenerationAndValidation(t *testing.T) {
	service := NewTokenService("super-secret-backup-key-12345", 1*time.Hour)

	token, err := service.GenerateToken("user-101", "admin")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != "user-101" {
		t.Fatalf("expected user-101, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Fatalf("expected admin, got %s", claims.Role)
	}
}

func TestTokenTampering(t *testing.T) {
	service := NewTokenService("secret-key", 1*time.Hour)

	token, err := service.GenerateToken("user-202", "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	tampered := token + "corrupt"
	_, err = service.ValidateToken(tampered)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenExpired(t *testing.T) {
	service := NewTokenService("secret-key", -1*time.Minute)

	token, err := service.GenerateToken("user-expired", "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = service.ValidateToken(token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
