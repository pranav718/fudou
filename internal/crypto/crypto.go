package crypto

type Encryptor interface {
	Encrypt(plaintext []byte, key []byte) ([]byte, []byte, error)
	Decrypt(ciphertext []byte, key []byte, nonce []byte) ([]byte, error)
}

type Hasher interface {
	Hash(data []byte) string
}
