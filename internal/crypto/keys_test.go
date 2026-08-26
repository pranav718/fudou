package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateAESKey(t *testing.T) {
	key1, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("failed to generate AES key: %v", err)
	}

	if len(key1) != AES256KeySize {
		t.Fatalf("expected key length %d, got %d", AES256KeySize, len(key1))
	}

	key2, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("failed to generate second AES key: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Fatalf("generated keys should be unique and random")
	}
}

func TestGenerateAESKeyHex(t *testing.T) {
	hexKey, err := GenerateAESKeyHex()
	if err != nil {
		t.Fatalf("failed to generate hex key: %v", err)
	}

	if len(hexKey) != AES256KeySize*2 {
		t.Fatalf("expected hex key length %d, got %d", AES256KeySize*2, len(hexKey))
	}

	decoded, err := DecodeHexKey(hexKey)
	if err != nil {
		t.Fatalf("failed to decode valid hex key: %v", err)
	}

	if len(decoded) != AES256KeySize {
		t.Fatalf("expected decoded length %d, got %d", AES256KeySize, len(decoded))
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("failed to generate nonce: %v", err)
	}

	if len(nonce) != GCMNonceSize {
		t.Fatalf("expected nonce size %d, got %d", GCMNonceSize, len(nonce))
	}
}
