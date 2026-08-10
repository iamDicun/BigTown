package security

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"
	role := "User"

	token, err := GenerateToken(userID, role, secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}
}

func TestParseToken_Expired(t *testing.T) {
	secret := "test-secret"
	token, err := GenerateToken("user-1", "User", secret, -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token, secret)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken("user-1", "User", "correct-secret", 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseToken_InvalidString(t *testing.T) {
	_, err := ParseToken("not.a.valid.jwt", "secret")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}
