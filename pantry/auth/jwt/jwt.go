// auth/jwt/jwt.go
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Standard claims as defined in RFC 7519.
type Claims struct {
	// Issuer identifies the principal that issued the JWT.
	Issuer string `json:"iss,omitempty"`

	// Subject identifies the principal that is the subject of the JWT.
	Subject string `json:"sub,omitempty"`

	// Audience identifies the recipients that the JWT is intended for.
	Audience Audience `json:"aud,omitempty"`

	// ExpiresAt is the expiration time after which the JWT must not be accepted.
	ExpiresAt *Time `json:"exp,omitempty"`

	// NotBefore is the time before which the JWT must not be accepted.
	NotBefore *Time `json:"nbf,omitempty"`

	// IssuedAt is the time at which the JWT was issued.
	IssuedAt *Time `json:"iat,omitempty"`

	// ID is a unique identifier for the JWT.
	ID string `json:"jti,omitempty"`
}

// Valid checks if the standard claims are valid.
func (c *Claims) Valid() error {
	now := time.Now()

	if c.ExpiresAt != nil && now.After(c.ExpiresAt.Time) {
		return ErrTokenExpired
	}

	if c.NotBefore != nil && now.Before(c.NotBefore.Time) {
		return ErrTokenNotYetValid
	}

	return nil
}

// Time wraps time.Time for JSON marshaling as Unix timestamp.
type Time struct {
	time.Time
}

// NewTime creates a Time from a time.Time.
func NewTime(t time.Time) *Time {
	return &Time{Time: t}
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Unix())
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var unix int64
	if err := json.Unmarshal(data, &unix); err != nil {
		return err
	}
	t.Time = time.Unix(unix, 0)
	return nil
}

// Audience handles both single string and array of strings.
type Audience []string

// MarshalJSON implements json.Marshaler.
func (a Audience) MarshalJSON() ([]byte, error) {
	if len(a) == 1 {
		return json.Marshal(a[0])
	}
	return json.Marshal([]string(a))
}

// UnmarshalJSON implements json.Unmarshaler.
func (a *Audience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = Audience{single}
		return nil
	}

	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return err
	}
	*a = Audience(multiple)
	return nil
}

// Contains checks if the audience contains a specific value.
func (a Audience) Contains(aud string) bool {
	for _, v := range a {
		if v == aud {
			return true
		}
	}
	return false
}

// Token represents a parsed JWT.
type Token[T any] struct {
	Header Header
	Claims T
	Valid  bool
}

// Header is the JWT header.
type Header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// Common errors.
var (
	ErrInvalidToken      = errors.New("jwt: invalid token")
	ErrInvalidSignature  = errors.New("jwt: invalid signature")
	ErrTokenExpired      = errors.New("jwt: token expired")
	ErrTokenNotYetValid  = errors.New("jwt: token not yet valid")
	ErrMissingSecret     = errors.New("jwt: missing secret")
	ErrUnsupportedAlg    = errors.New("jwt: unsupported algorithm")
	ErrInvalidClaims     = errors.New("jwt: invalid claims")
)

// Signer creates and verifies JWTs.
type Signer struct {
	secret []byte
	alg    string
}

// NewHS256 creates a signer using HMAC-SHA256.
func NewHS256(secret []byte) (*Signer, error) {
	if len(secret) == 0 {
		return nil, ErrMissingSecret
	}
	return &Signer{
		secret: secret,
		alg:    "HS256",
	}, nil
}

// NewHS256String creates a signer from a string secret.
func NewHS256String(secret string) (*Signer, error) {
	return NewHS256([]byte(secret))
}

// Sign creates a signed JWT from claims.
func (s *Signer) Sign(claims any) (string, error) {
	header := Header{
		Algorithm: s.alg,
		Type:      "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	signature := s.sign([]byte(signingInput))
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64, nil
}

// Verify parses and verifies a JWT, returning the token with claims.
func (s *Signer) Verify(tokenString string, claims any) error {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	// Decode header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidToken
	}

	var header Header
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return ErrInvalidToken
	}

	// Verify algorithm matches
	if header.Algorithm != s.alg {
		return ErrUnsupportedAlg
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ErrInvalidToken
	}

	if !s.verify([]byte(signingInput), signature) {
		return ErrInvalidSignature
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidToken
	}

	if err := json.Unmarshal(claimsJSON, claims); err != nil {
		return ErrInvalidClaims
	}

	// Validate standard claims if present
	if validator, ok := claims.(interface{ Valid() error }); ok {
		if err := validator.Valid(); err != nil {
			return err
		}
	}

	return nil
}

// Parse parses a JWT without verifying the signature.
// Use only for debugging or when signature is verified elsewhere.
func Parse(tokenString string, claims any) error {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidToken
	}

	return json.Unmarshal(claimsJSON, claims)
}

// sign creates a signature using HMAC-SHA256.
func (s *Signer) sign(data []byte) []byte {
	h := hmac.New(sha256.New, s.secret)
	h.Write(data)
	return h.Sum(nil)
}

// verify checks a signature using HMAC-SHA256.
func (s *Signer) verify(data, signature []byte) bool {
	expected := s.sign(data)
	return hmac.Equal(signature, expected)
}
