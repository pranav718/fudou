package auth

import (
	"errors"
	"testing"
	"time"
)

func FuzzTokenValidation(f *testing.F) {
	service := NewTokenService("test-secret-key-12345", 1*time.Hour)
	validToken, _ := service.GenerateToken("user-fuzz", "admin")

	f.Add(validToken)
	f.Add("")
	f.Add("invalid.token.format")
	f.Add("part1.part2")
	f.Add(validToken + "tamper")

	f.Fuzz(func(t *testing.T, token string) {
		claims, err := service.ValidateToken(token)
		if err != nil {
			if !errors.Is(err, ErrInvalidToken) && !errors.Is(err, ErrTokenExpired) {
				t.Fatalf("unexpected validation error: %v", err)
			}
			return
		}
		if claims == nil || claims.UserID == "" {
			t.Fatalf("valid claims returned nil or empty user id")
		}
	})
}
