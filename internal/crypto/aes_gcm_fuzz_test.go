package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
)

func FuzzAESGCMDecrypt(f *testing.F) {
	key, _ := GenerateAESKey()
	enc := NewAESGCMEncryptor()

	samplePayloads := [][]byte{
		[]byte(""),
		[]byte("fudou confidential block"),
		bytes.Repeat([]byte{0x7a}, 256),
		bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 1024),
	}

	for _, p := range samplePayloads {
		ct, nonce, _ := enc.Encrypt(p, key)
		f.Add(ct, key, nonce)
	}

	f.Fuzz(func(t *testing.T, ciphertext []byte, k []byte, n []byte) {
		plain, err := enc.Decrypt(ciphertext, k, n)
		if err != nil {
			if !errors.Is(err, ErrInvalidKeySize) && !errors.Is(err, ErrDecryptionFailed) {
				t.Fatalf("unexpected error type: %v", err)
			}
			return
		}

		if len(k) != 32 {
			t.Fatalf("decrypted with invalid key size: %d", len(k))
		}

		ctRecheck, nonceRecheck, err := enc.Encrypt(plain, k)
		if err != nil {
			t.Fatalf("re-encryption failed: %v", err)
		}

		plainRecheck, err := enc.Decrypt(ctRecheck, k, nonceRecheck)
		if err != nil {
			t.Fatalf("re-decryption failed: %v", err)
		}

		if !bytes.Equal(plain, plainRecheck) {
			t.Fatalf("roundtrip plaintexts do not match")
		}
	})
}

func FuzzDecodeHexKey(f *testing.F) {
	validKey, _ := GenerateAESKeyHex()
	f.Add(validKey)
	f.Add("")
	f.Add("00112233")
	f.Add("xyznothex")
	f.Add(validKey + "extra")

	f.Fuzz(func(t *testing.T, raw string) {
		key, err := DecodeHexKey(raw)
		if err != nil {
			return
		}
		if len(key) != 32 {
			t.Fatalf("expected 32 byte key, got %d", len(key))
		}
	})
}

func FuzzAESGCMTamperDetection(f *testing.F) {
	key := make([]byte, 32)
	rand.Read(key)
	enc := NewAESGCMEncryptor()

	f.Add([]byte("critical financial backup data"), 0, byte(0x01))
	f.Add([]byte("health record confidential file"), 5, byte(0xff))
	f.Add(bytes.Repeat([]byte("A"), 1024), 500, byte(0xaa))

	f.Fuzz(func(t *testing.T, plain []byte, flipIndex int, flipMask byte) {
		if len(plain) == 0 {
			return
		}
		ct, nonce, err := enc.Encrypt(plain, key)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		idx := flipIndex % len(ct)
		if idx < 0 {
			idx = -idx
		}
		if flipMask == 0 {
			flipMask = 1
		}

		tampered := make([]byte, len(ct))
		copy(tampered, ct)
		tampered[idx] ^= flipMask

		_, err = enc.Decrypt(tampered, key, nonce)
		if err == nil {
			t.Fatalf("tampered ciphertext at index %d was accepted", idx)
		}
		if !errors.Is(err, ErrDecryptionFailed) {
			t.Fatalf("expected ErrDecryptionFailed, got: %v", err)
		}
	})
}
