// notify/vapid.go
package notify

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// VAPIDKeys holds a VAPID key pair for Web Push authentication.
type VAPIDKeys struct {
	// PublicKey is the base64url-encoded public key (for client subscription).
	PublicKey string

	// PrivateKey is the base64url-encoded private key (for signing).
	PrivateKey string

	// Subject is the contact URI (mailto: or https:).
	Subject string

	// parsed keys
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

// GenerateVAPIDKeys generates a new VAPID key pair.
func GenerateVAPIDKeys() (*VAPIDKeys, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to generate VAPID keys: %w", err)
	}

	return keysFromECDSA(privateKey)
}

// keysFromECDSA creates VAPIDKeys from an ECDSA private key.
func keysFromECDSA(privateKey *ecdsa.PrivateKey) (*VAPIDKeys, error) {
	// Encode public key (uncompressed point format)
	pubBytes := elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)
	publicKeyB64 := base64.RawURLEncoding.EncodeToString(pubBytes)

	// Encode private key (just the D value, 32 bytes)
	privBytes := privateKey.D.Bytes()
	// Pad to 32 bytes if necessary
	if len(privBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privBytes):], privBytes)
		privBytes = padded
	}
	privateKeyB64 := base64.RawURLEncoding.EncodeToString(privBytes)

	return &VAPIDKeys{
		PublicKey:  publicKeyB64,
		PrivateKey: privateKeyB64,
		publicKey:  &privateKey.PublicKey,
		privateKey: privateKey,
	}, nil
}

// NewVAPIDKeys creates VAPIDKeys from base64url-encoded keys.
func NewVAPIDKeys(publicKey, privateKey, subject string) (*VAPIDKeys, error) {
	keys := &VAPIDKeys{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Subject:    subject,
	}

	if err := keys.parse(); err != nil {
		return nil, err
	}

	return keys, nil
}

// LoadVAPIDKeysFromPEM loads VAPID keys from PEM-encoded private key.
func LoadVAPIDKeysFromPEM(pemData []byte, subject string) (*VAPIDKeys, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, ErrInvalidPEM
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("notify: failed to parse private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, ErrInvalidKeyType
		}
	}

	if privateKey.Curve != elliptic.P256() {
		return nil, ErrInvalidCurve
	}

	keys, err := keysFromECDSA(privateKey)
	if err != nil {
		return nil, err
	}
	keys.Subject = subject

	return keys, nil
}

// parse parses the base64url-encoded keys into ECDSA keys.
func (v *VAPIDKeys) parse() error {
	// Parse public key
	pubBytes, err := base64.RawURLEncoding.DecodeString(v.PublicKey)
	if err != nil {
		return fmt.Errorf("notify: invalid public key encoding: %w", err)
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), pubBytes)
	if x == nil {
		return ErrInvalidPublicKey
	}
	v.publicKey = &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	// Parse private key
	privBytes, err := base64.RawURLEncoding.DecodeString(v.PrivateKey)
	if err != nil {
		return fmt.Errorf("notify: invalid private key encoding: %w", err)
	}

	v.privateKey = &ecdsa.PrivateKey{
		PublicKey: *v.publicKey,
		D:         new(big.Int).SetBytes(privBytes),
	}

	return nil
}

// WithSubject sets the subject (contact URI) for the VAPID keys.
func (v *VAPIDKeys) WithSubject(subject string) *VAPIDKeys {
	v.Subject = subject
	return v
}

// ExportPEM exports the private key as PEM-encoded data.
func (v *VAPIDKeys) ExportPEM() ([]byte, error) {
	if v.privateKey == nil {
		if err := v.parse(); err != nil {
			return nil, err
		}
	}

	der, err := x509.MarshalECPrivateKey(v.privateKey)
	if err != nil {
		return nil, fmt.Errorf("notify: failed to marshal private key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}), nil
}

// ExportJSON exports the keys as JSON.
func (v *VAPIDKeys) ExportJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"publicKey":  v.PublicKey,
		"privateKey": v.PrivateKey,
		"subject":    v.Subject,
	})
}

// LoadVAPIDKeysFromJSON loads VAPID keys from JSON.
func LoadVAPIDKeysFromJSON(data []byte) (*VAPIDKeys, error) {
	var keys struct {
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
		Subject    string `json:"subject"`
	}

	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("notify: failed to parse VAPID keys JSON: %w", err)
	}

	return NewVAPIDKeys(keys.PublicKey, keys.PrivateKey, keys.Subject)
}

// ApplicationServerKey returns the public key in the format needed for
// PushManager.subscribe() on the client side.
func (v *VAPIDKeys) ApplicationServerKey() string {
	return v.PublicKey
}

// vapidToken creates a VAPID JWT token for the given audience.
func (v *VAPIDKeys) vapidToken(audience string, expiry time.Duration) (string, error) {
	if v.privateKey == nil {
		if err := v.parse(); err != nil {
			return "", err
		}
	}

	if v.Subject == "" {
		return "", ErrSubjectRequired
	}

	now := time.Now()
	exp := now.Add(expiry)

	// JWT header
	header := map[string]string{
		"typ": "JWT",
		"alg": "ES256",
	}

	// JWT claims
	claims := map[string]any{
		"aud": audience,
		"exp": exp.Unix(),
		"sub": v.Subject,
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	// Sign with ES256
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, v.privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("notify: failed to sign VAPID token: %w", err)
	}

	// Convert to fixed-size signature (32 bytes each for r and s)
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):], sBytes)

	signatureB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + signatureB64, nil
}

// authorizationHeader creates the Authorization header value for VAPID.
func (v *VAPIDKeys) authorizationHeader(audience string, expiry time.Duration) (string, error) {
	token, err := v.vapidToken(audience, expiry)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("vapid t=%s, k=%s", token, v.PublicKey), nil
}
