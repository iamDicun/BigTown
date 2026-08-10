package security

import (
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "strong-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}

	if err := CheckPassword(password, hash); err != nil {
		t.Errorf("CheckPassword failed for correct password: %v", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if err := CheckPassword("wrong-password", hash); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestHashPassword_Empty(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword empty failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash for empty password")
	}
}
