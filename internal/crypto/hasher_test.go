package crypto

import (
	"bytes"
	"testing"
)

func TestSHA256Hasher(t *testing.T) {
	hasher := NewSHA256Hasher()
	input := []byte("hello fudou distributed backup")
	expected := "3f604beecd5731ac839bbf4337cd0c16586c8fc09661286bab01a3a307a62c45"

	hash := hasher.Hash(input)
	if hash != expected {
		t.Fatalf("expected hash %s, got %s", expected, hash)
	}

	reader := bytes.NewReader(input)
	hashFromReader, err := hasher.HashReader(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hashFromReader != expected {
		t.Fatalf("expected reader hash %s, got %s", expected, hashFromReader)
	}

	if !VerifyChecksum(input, expected) {
		t.Fatalf("checksum verification failed")
	}

	if VerifyChecksum(input, "wrong-checksum") {
		t.Fatalf("expected verification failure on mismatch")
	}
}
