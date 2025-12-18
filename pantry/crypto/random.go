// crypto/random.go
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"math/big"
)

// RandomBytes generates n cryptographically secure random bytes.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// RandomHex generates a random hex string of the specified byte length.
// The returned string will be 2*n characters long.
func RandomHex(n int) (string, error) {
	b, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RandomBase64 generates a random base64 string of the specified byte length.
func RandomBase64(n int) (string, error) {
	b, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// RandomBase64URL generates a random URL-safe base64 string.
func RandomBase64URL(n int) (string, error) {
	b, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RandomString generates a random string of the specified length
// using the given character set.
func RandomString(length int, charset string) (string, error) {
	if len(charset) == 0 {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}

// RandomAlphanumeric generates a random alphanumeric string.
func RandomAlphanumeric(length int) (string, error) {
	return RandomString(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
}

// RandomAlpha generates a random alphabetic string (letters only).
func RandomAlpha(length int) (string, error) {
	return RandomString(length, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// RandomNumeric generates a random numeric string (digits only).
func RandomNumeric(length int) (string, error) {
	return RandomString(length, "0123456789")
}

// RandomInt generates a random integer in the range [0, max).
func RandomInt(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

// RandomIntRange generates a random integer in the range [min, max].
func RandomIntRange(min, max int64) (int64, error) {
	if min > max {
		min, max = max, min
	}
	n, err := RandomInt(max - min + 1)
	if err != nil {
		return 0, err
	}
	return min + n, nil
}

// Common token generators.

// GenerateToken generates a secure random token suitable for sessions, CSRF, etc.
// Returns a 32-byte token as a 64-character hex string.
func GenerateToken() (string, error) {
	return RandomHex(32)
}

// GenerateShortToken generates a shorter token (16 bytes, 32 hex chars).
func GenerateShortToken() (string, error) {
	return RandomHex(16)
}

// GenerateAPIKey generates a random API key.
// Format: prefix_base64url (e.g., "sk_a1b2c3d4e5f6...")
func GenerateAPIKey(prefix string) (string, error) {
	token, err := RandomBase64URL(24)
	if err != nil {
		return "", err
	}
	if prefix != "" {
		return prefix + "_" + token, nil
	}
	return token, nil
}

// GenerateOTP generates a numeric one-time password of the specified length.
func GenerateOTP(length int) (string, error) {
	return RandomNumeric(length)
}

// GenerateVerificationCode generates a user-friendly verification code.
// Uses only uppercase letters and digits, excluding similar-looking characters.
func GenerateVerificationCode(length int) (string, error) {
	// Exclude 0, O, I, 1, L to avoid confusion
	return RandomString(length, "ABCDEFGHJKMNPQRSTUVWXYZ23456789")
}
