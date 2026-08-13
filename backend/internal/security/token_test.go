package security

import (
	"testing"
)

func TestGenerateRandomToken(t *testing.T) {
	token1, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken failed: %v", err)
	}
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}

	token2, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken failed: %v", err)
	}
	if token1 == token2 {
		t.Fatal("expected different random tokens")
	}
}

func TestHashToken(t *testing.T) {
	token := "sample-token-123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Errorf("expected consistent hash output, got %s and %s", hash1, hash2)
	}

	hashDiff := HashToken("different-token")
	if hash1 == hashDiff {
		t.Error("expected different hashes for different tokens")
	}
}
