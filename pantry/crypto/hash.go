// crypto/hash.go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// SHA256 computes the SHA-256 hash of data.
func SHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// SHA256Hex computes the SHA-256 hash and returns it as a hex string.
func SHA256Hex(data []byte) string {
	return hex.EncodeToString(SHA256(data))
}

// SHA256String computes the SHA-256 hash of a string.
func SHA256String(s string) string {
	return SHA256Hex([]byte(s))
}

// SHA256Base64 computes the SHA-256 hash and returns it as base64.
func SHA256Base64(data []byte) string {
	return base64.StdEncoding.EncodeToString(SHA256(data))
}

// SHA512 computes the SHA-512 hash of data.
func SHA512(data []byte) []byte {
	h := sha512.Sum512(data)
	return h[:]
}

// SHA512Hex computes the SHA-512 hash and returns it as a hex string.
func SHA512Hex(data []byte) string {
	return hex.EncodeToString(SHA512(data))
}

// SHA512String computes the SHA-512 hash of a string.
func SHA512String(s string) string {
	return SHA512Hex([]byte(s))
}

// HMACSHA256 computes HMAC-SHA256.
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// HMACSHA256Hex computes HMAC-SHA256 and returns it as hex.
func HMACSHA256Hex(key, data []byte) string {
	return hex.EncodeToString(HMACSHA256(key, data))
}

// HMACSHA256Base64 computes HMAC-SHA256 and returns it as base64.
func HMACSHA256Base64(key, data []byte) string {
	return base64.StdEncoding.EncodeToString(HMACSHA256(key, data))
}

// HMACSHA512 computes HMAC-SHA512.
func HMACSHA512(key, data []byte) []byte {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// HMACSHA512Hex computes HMAC-SHA512 and returns it as hex.
func HMACSHA512Hex(key, data []byte) string {
	return hex.EncodeToString(HMACSHA512(key, data))
}

// VerifyHMAC verifies an HMAC signature using constant-time comparison.
func VerifyHMAC(key, data, signature []byte, hashFunc func() hash.Hash) bool {
	h := hmac.New(hashFunc, key)
	h.Write(data)
	expected := h.Sum(nil)
	return hmac.Equal(signature, expected)
}

// VerifyHMACSHA256 verifies an HMAC-SHA256 signature.
func VerifyHMACSHA256(key, data, signature []byte) bool {
	return VerifyHMAC(key, data, signature, sha256.New)
}

// VerifyHMACSHA256Hex verifies an HMAC-SHA256 signature (hex encoded).
func VerifyHMACSHA256Hex(key, data []byte, signatureHex string) bool {
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return VerifyHMACSHA256(key, data, signature)
}

// HashFile computes the SHA-256 hash of a file.
func HashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// HashFileHex computes the SHA-256 hash of a file and returns it as hex.
func HashFileHex(path string) (string, error) {
	hash, err := HashFile(path)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// HashReader computes the SHA-256 hash of data from a reader.
func HashReader(r io.Reader) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// HashReaderHex computes the SHA-256 hash of a reader and returns it as hex.
func HashReaderHex(r io.Reader) (string, error) {
	hash, err := HashReader(r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}
