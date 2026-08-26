package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

const (
	AES256KeySize = 32
	GCMNonceSize  = 12
)

func GenerateAESKey() ([]byte, error) {
	key := make([]byte, AES256KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func GenerateAESKeyHex() (string, error) {
	key, err := GenerateAESKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

func DecodeHexKey(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}
