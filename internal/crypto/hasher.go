package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

type SHA256Hasher struct{}

func NewSHA256Hasher() *SHA256Hasher {
	return &SHA256Hasher{}
}

func (h *SHA256Hasher) Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (h *SHA256Hasher) HashReader(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func VerifyChecksum(data []byte, expectedHex string) bool {
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	return actual == expectedHex
}
