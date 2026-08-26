package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAESGCMEncryptDecrypt(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	plaintext := []byte("confidential distributed backup payload data")
	ciphertext, nonce, err := enc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatalf("ciphertext should not match plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext, key, nonce)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("expected decrypted %s, got %s", plaintext, decrypted)
	}
}

func TestAESGCMInvalidKey(t *testing.T) {
	enc := NewAESGCMEncryptor()
	shortKey := make([]byte, 16)
	plaintext := []byte("some data")

	_, _, err := enc.Encrypt(plaintext, shortKey)
	if err != ErrInvalidKeySize {
		t.Fatalf("expected ErrInvalidKeySize, got %v", err)
	}

	_, err = enc.Decrypt(plaintext, shortKey, make([]byte, 12))
	if err != ErrInvalidKeySize {
		t.Fatalf("expected ErrInvalidKeySize on decrypt, got %v", err)
	}
}

func TestAESGCMTamperedCiphertext(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte("original data to be tampered")
	ciphertext, nonce, err := enc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	ciphertext[0] ^= 0xFF

	_, err = enc.Decrypt(ciphertext, key, nonce)
	if err != ErrDecryptionFailed {
		t.Fatalf("expected ErrDecryptionFailed for tampered ciphertext, got %v", err)
	}
}
