// crypto/identifier.go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/hkdf"
)

// IdentifierHasher provides secure, deterministic hashing for identifiers
// like email addresses, usernames, or other PII that needs to be stored
// in a way that allows lookup but protects the original value.
//
// It uses HMAC-SHA256 with key derivation to ensure:
// - Same identifier + key always produces the same hash (deterministic)
// - Without the key, hashes cannot be reversed or precomputed
// - Different domains produce different hashes for the same identifier
type IdentifierHasher struct {
	key    []byte
	domain string
	format OutputFormat
}

// OutputFormat specifies how hash output is encoded.
type OutputFormat int

const (
	// FormatHex outputs lowercase hexadecimal (64 chars for SHA-256).
	FormatHex OutputFormat = iota

	// FormatBase64 outputs standard base64 (44 chars for SHA-256).
	FormatBase64

	// FormatBase64URL outputs URL-safe base64 without padding.
	FormatBase64URL
)

// IdentifierHasherConfig configures the identifier hasher.
type IdentifierHasherConfig struct {
	// Key is the secret key for HMAC. Required.
	// Should be at least 32 bytes for security.
	Key []byte

	// Domain provides separation between different identifier types.
	// For example, "email" vs "username" will produce different hashes
	// for the same input. Default: "" (no domain separation).
	Domain string

	// Format specifies the output encoding. Default: FormatHex.
	Format OutputFormat

	// DeriveKey enables key derivation using HKDF.
	// When true, the provided key is used as input key material (IKM)
	// and a domain-specific key is derived. Default: true.
	DeriveKey bool
}

// NewIdentifierHasher creates a hasher with default settings.
// Key should be at least 32 bytes.
func NewIdentifierHasher(key []byte) *IdentifierHasher {
	return NewIdentifierHasherWithConfig(IdentifierHasherConfig{
		Key:       key,
		DeriveKey: true,
	})
}

// NewIdentifierHasherFromString creates a hasher from a base64-encoded key.
func NewIdentifierHasherFromString(keyBase64 string) (*IdentifierHasher, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	return NewIdentifierHasher(key), nil
}

// NewIdentifierHasherWithConfig creates a hasher with custom configuration.
func NewIdentifierHasherWithConfig(cfg IdentifierHasherConfig) *IdentifierHasher {
	key := cfg.Key

	// Derive a domain-specific key if enabled
	if cfg.DeriveKey && len(key) > 0 {
		key = deriveKey(cfg.Key, cfg.Domain)
	}

	return &IdentifierHasher{
		key:    key,
		domain: cfg.Domain,
		format: cfg.Format,
	}
}

// deriveKey uses HKDF to derive a domain-specific key.
func deriveKey(ikm []byte, domain string) []byte {
	info := []byte("waffle-identifier-hash")
	if domain != "" {
		info = append(info, ':')
		info = append(info, []byte(domain)...)
	}

	reader := hkdf.New(sha256.New, ikm, nil, info)
	derived := make([]byte, 32)
	reader.Read(derived)
	return derived
}

// Hash computes a deterministic hash of the identifier.
// The identifier is normalized (trimmed and lowercased) before hashing.
func (h *IdentifierHasher) Hash(identifier string) string {
	normalized := h.normalize(identifier)
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(normalized))
	hash := mac.Sum(nil)
	return h.encode(hash)
}

// HashRaw computes a hash without normalizing the identifier.
// Use this when you need case-sensitive or whitespace-sensitive hashing.
func (h *IdentifierHasher) HashRaw(identifier string) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(identifier))
	hash := mac.Sum(nil)
	return h.encode(hash)
}

// HashBytes computes a hash of raw bytes.
func (h *IdentifierHasher) HashBytes(data []byte) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write(data)
	hash := mac.Sum(nil)
	return h.encode(hash)
}

// Verify checks if an identifier matches a hash.
// Uses constant-time comparison to prevent timing attacks.
func (h *IdentifierHasher) Verify(identifier, hash string) bool {
	computed := h.Hash(identifier)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// VerifyRaw checks if an identifier matches a hash without normalization.
func (h *IdentifierHasher) VerifyRaw(identifier, hash string) bool {
	computed := h.HashRaw(identifier)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// normalize prepares an identifier for hashing.
func (h *IdentifierHasher) normalize(identifier string) string {
	return strings.ToLower(strings.TrimSpace(identifier))
}

// encode converts hash bytes to the configured output format.
func (h *IdentifierHasher) encode(hash []byte) string {
	switch h.format {
	case FormatBase64:
		return base64.StdEncoding.EncodeToString(hash)
	case FormatBase64URL:
		return base64.RawURLEncoding.EncodeToString(hash)
	default:
		return hex.EncodeToString(hash)
	}
}

// WithDomain creates a new hasher with a different domain.
// This is useful when you need to hash different types of identifiers
// (emails, usernames, phone numbers) with the same master key.
func (h *IdentifierHasher) WithDomain(domain string) *IdentifierHasher {
	return &IdentifierHasher{
		key:    deriveKey(h.key, domain),
		domain: domain,
		format: h.format,
	}
}

// WithFormat creates a new hasher with a different output format.
func (h *IdentifierHasher) WithFormat(format OutputFormat) *IdentifierHasher {
	return &IdentifierHasher{
		key:    h.key,
		domain: h.domain,
		format: format,
	}
}

// Convenience functions for simple usage.

// HashIdentifier hashes an identifier with the given key.
// Uses default settings (hex output, key derivation enabled).
func HashIdentifier(key []byte, identifier string) string {
	h := NewIdentifierHasher(key)
	return h.Hash(identifier)
}

// HashIdentifierWithDomain hashes an identifier with domain separation.
func HashIdentifierWithDomain(key []byte, domain, identifier string) string {
	h := NewIdentifierHasherWithConfig(IdentifierHasherConfig{
		Key:       key,
		Domain:    domain,
		DeriveKey: true,
	})
	return h.Hash(identifier)
}

// VerifyIdentifier checks if an identifier matches a hash.
func VerifyIdentifier(key []byte, identifier, hash string) bool {
	h := NewIdentifierHasher(key)
	return h.Verify(identifier, hash)
}

// VerifyIdentifierWithDomain verifies with domain separation.
func VerifyIdentifierWithDomain(key []byte, domain, identifier, hash string) bool {
	h := NewIdentifierHasherWithConfig(IdentifierHasherConfig{
		Key:       key,
		Domain:    domain,
		DeriveKey: true,
	})
	return h.Verify(identifier, hash)
}

// BlindIndex creates a truncated hash suitable for database indexing.
// The shorter length reduces storage while maintaining enough uniqueness
// for efficient lookups. Collisions are possible but rare.
//
// Recommended lengths:
// - 8 bytes (16 hex chars): ~4 billion unique values before 50% collision chance
// - 12 bytes (24 hex chars): ~280 trillion unique values
// - 16 bytes (32 hex chars): ~18 quintillion unique values
func (h *IdentifierHasher) BlindIndex(identifier string, length int) string {
	if length <= 0 || length > 32 {
		length = 16 // Default to 16 bytes (32 hex chars)
	}

	normalized := h.normalize(identifier)
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(normalized))
	hash := mac.Sum(nil)

	// Truncate
	truncated := hash[:length]

	switch h.format {
	case FormatBase64:
		return base64.StdEncoding.EncodeToString(truncated)
	case FormatBase64URL:
		return base64.RawURLEncoding.EncodeToString(truncated)
	default:
		return hex.EncodeToString(truncated)
	}
}
