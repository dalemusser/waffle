// crypto/encrypt.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encryption errors.
var (
	ErrInvalidKey        = errors.New("crypto: invalid key size (must be 16, 24, or 32 bytes)")
	ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")
	ErrDecryptionFailed  = errors.New("crypto: decryption failed")
)

// Encryptor provides AES-GCM encryption and decryption.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates an encryptor with the given key.
// Key must be 16 (AES-128), 24 (AES-192), or 32 (AES-256) bytes.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// NewEncryptorFromString creates an encryptor from a base64-encoded key.
func NewEncryptorFromString(keyBase64 string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	return NewEncryptor(key)
}

// Encrypt encrypts plaintext and returns ciphertext with nonce prepended.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext (with prepended nonce) and returns plaintext.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// DecryptString decrypts base64-encoded ciphertext and returns plaintext string.
func (e *Encryptor) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GenerateKey generates a random encryption key of the specified size.
// Size must be 16 (AES-128), 24 (AES-192), or 32 (AES-256).
func GenerateKey(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKey
	}

	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}

// GenerateKeyString generates a random key and returns it base64-encoded.
func GenerateKeyString(size int) (string, error) {
	key, err := GenerateKey(size)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Convenience functions for simple encryption without creating an Encryptor.

// Encrypt encrypts plaintext with the given key using AES-GCM.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	enc, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}
	return enc.Encrypt(plaintext)
}

// Decrypt decrypts ciphertext with the given key using AES-GCM.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	enc, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}
	return enc.Decrypt(ciphertext)
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func EncryptString(keyBase64, plaintext string) (string, error) {
	enc, err := NewEncryptorFromString(keyBase64)
	if err != nil {
		return "", err
	}
	return enc.EncryptString(plaintext)
}

// DecryptString decrypts base64-encoded ciphertext and returns plaintext.
func DecryptString(keyBase64, ciphertextBase64 string) (string, error) {
	enc, err := NewEncryptorFromString(keyBase64)
	if err != nil {
		return "", err
	}
	return enc.DecryptString(ciphertextBase64)
}
