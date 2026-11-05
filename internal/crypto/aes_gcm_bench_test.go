package crypto

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func BenchmarkAESGCM_Encrypt(b *testing.B) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		b.Fatalf("failed to generate key: %v", err)
	}
	encryptor := NewAESGCMEncryptor()

	sizes := []struct {
		name string
		size int
	}{
		{"64KB", 64 * 1024},
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			plaintext := bytes.Repeat([]byte{0xab}, tc.size)
			b.SetBytes(int64(tc.size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, err := encryptor.Encrypt(plaintext, key)
				if err != nil {
					b.Fatalf("encryption failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkAESGCM_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		b.Fatalf("failed to generate key: %v", err)
	}
	encryptor := NewAESGCMEncryptor()

	sizes := []struct {
		name string
		size int
	}{
		{"64KB", 64 * 1024},
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			plaintext := bytes.Repeat([]byte{0xcd}, tc.size)
			ciphertext, nonce, err := encryptor.Encrypt(plaintext, key)
			if err != nil {
				b.Fatalf("prep encryption failed: %v", err)
			}

			b.SetBytes(int64(tc.size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := encryptor.Decrypt(ciphertext, key, nonce)
				if err != nil {
					b.Fatalf("decryption failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkSHA256_Hash(b *testing.B) {
	hasher := NewSHA256Hasher()

	sizes := []struct {
		name string
		size int
	}{
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
		{"16MB", 16 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			data := bytes.Repeat([]byte{0xef}, tc.size)
			b.SetBytes(int64(tc.size))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = hasher.Hash(data)
			}
		})
	}
}
