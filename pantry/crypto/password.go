// crypto/password.go
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Password hashing errors.
var (
	ErrInvalidHash     = errors.New("crypto: invalid password hash format")
	ErrIncompatibleVersion = errors.New("crypto: incompatible argon2 version")
	ErrMismatchedPassword  = errors.New("crypto: password does not match")
)

// Argon2Params contains the parameters for Argon2id hashing.
type Argon2Params struct {
	// Memory is the amount of memory used in KiB.
	Memory uint32

	// Iterations is the number of passes over the memory.
	Iterations uint32

	// Parallelism is the number of threads to use.
	Parallelism uint8

	// SaltLength is the length of the random salt in bytes.
	SaltLength uint32

	// KeyLength is the length of the generated key in bytes.
	KeyLength uint32
}

// DefaultArgon2Params returns secure default parameters for Argon2id.
// These are suitable for most web applications.
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// LowMemoryArgon2Params returns parameters for memory-constrained environments.
func LowMemoryArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      32 * 1024, // 32 MiB
		Iterations:  4,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword hashes a password using Argon2id with the given parameters.
// If params is nil, DefaultArgon2Params() is used.
// Returns a string in the format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
func HashPassword(password string, params *Argon2Params) (string, error) {
	if params == nil {
		params = DefaultArgon2Params()
	}

	// Generate random salt
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Encode to standard format
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// VerifyPassword checks if a password matches the given hash.
// Returns nil if the password matches, ErrMismatchedPassword if not.
func VerifyPassword(password, encodedHash string) error {
	params, salt, hash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return err
	}

	// Hash the password with the same parameters
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(hash, otherHash) != 1 {
		return ErrMismatchedPassword
	}

	return nil
}

// CheckPasswordHash is an alias for VerifyPassword that returns a bool.
func CheckPasswordHash(password, hash string) bool {
	return VerifyPassword(password, hash) == nil
}

// decodeArgon2Hash parses an encoded Argon2id hash string.
func decodeArgon2Hash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Argon2Params{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// NeedsRehash checks if a hash was created with outdated parameters.
// Returns true if the hash should be regenerated with new parameters.
func NeedsRehash(encodedHash string, params *Argon2Params) bool {
	if params == nil {
		params = DefaultArgon2Params()
	}

	current, _, _, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return true
	}

	return current.Memory != params.Memory ||
		current.Iterations != params.Iterations ||
		current.Parallelism != params.Parallelism ||
		current.KeyLength != params.KeyLength
}

// Bcrypt functions for compatibility with existing systems.

// BcryptCost is the cost parameter for bcrypt hashing.
type BcryptCost int

const (
	BcryptMinCost     BcryptCost = 4
	BcryptDefaultCost BcryptCost = 10
	BcryptMaxCost     BcryptCost = 31
)

// HashPasswordBcrypt hashes a password using bcrypt.
// If cost is 0, BcryptDefaultCost is used.
func HashPasswordBcrypt(password string, cost BcryptCost) (string, error) {
	if cost == 0 {
		cost = BcryptDefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), int(cost))
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPasswordBcrypt checks if a password matches a bcrypt hash.
func VerifyPasswordBcrypt(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return ErrMismatchedPassword
	}
	return err
}

// CheckPasswordHashBcrypt is an alias that returns a bool.
func CheckPasswordHashBcrypt(password, hash string) bool {
	return VerifyPasswordBcrypt(password, hash) == nil
}

// NeedsRehashBcrypt checks if a bcrypt hash needs rehashing.
func NeedsRehashBcrypt(hash string, cost BcryptCost) bool {
	if cost == 0 {
		cost = BcryptDefaultCost
	}
	hashCost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return hashCost != int(cost)
}

// GeneratePassword creates a random password of the given length.
// The password contains uppercase, lowercase, digits, and symbols.
func GeneratePassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		symbols   = "!@#$%^&*()-_=+[]{}|;:,.<>?"
		all       = lowercase + uppercase + digits + symbols
	)

	// Ensure at least one of each type
	password := make([]byte, length)

	// Fill with random characters
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for i := range password {
		password[i] = all[int(randomBytes[i])%len(all)]
	}

	// Ensure complexity by replacing first 4 chars with one from each set
	sets := []string{lowercase, uppercase, digits, symbols}
	for i, set := range sets {
		if i >= length {
			break
		}
		b := make([]byte, 1)
		rand.Read(b)
		password[i] = set[int(b[0])%len(set)]
	}

	// Shuffle the password
	shuffled := make([]byte, length)
	perm := make([]byte, length)
	rand.Read(perm)
	indices := make([]int, length)
	for i := range indices {
		indices[i] = i
	}
	for i := length - 1; i > 0; i-- {
		j := int(perm[i]) % (i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}
	for i, idx := range indices {
		shuffled[i] = password[idx]
	}

	return string(shuffled), nil
}

// PasswordStrength evaluates password strength.
type PasswordStrength int

const (
	PasswordWeak PasswordStrength = iota
	PasswordFair
	PasswordGood
	PasswordStrong
)

func (s PasswordStrength) String() string {
	switch s {
	case PasswordWeak:
		return "weak"
	case PasswordFair:
		return "fair"
	case PasswordGood:
		return "good"
	case PasswordStrong:
		return "strong"
	default:
		return "unknown"
	}
}

// CheckPasswordStrength evaluates the strength of a password.
func CheckPasswordStrength(password string) PasswordStrength {
	var score int

	// Length
	switch {
	case len(password) >= 16:
		score += 2
	case len(password) >= 12:
		score += 1
	case len(password) < 8:
		return PasswordWeak
	}

	// Character types
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	if hasLower {
		score++
	}
	if hasUpper {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSymbol {
		score++
	}

	switch {
	case score >= 6:
		return PasswordStrong
	case score >= 4:
		return PasswordGood
	case score >= 2:
		return PasswordFair
	default:
		return PasswordWeak
	}
}

// PasswordPolicy defines requirements for passwords.
type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSymbol  bool
	MinStrength    PasswordStrength
}

// DefaultPasswordPolicy returns a sensible default policy.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:     8,
		MaxLength:     128,
		RequireUpper:  true,
		RequireLower:  true,
		RequireDigit:  true,
		RequireSymbol: false,
		MinStrength:   PasswordFair,
	}
}

// Validate checks if a password meets the policy requirements.
// Returns a list of violations, or nil if the password is valid.
func (p PasswordPolicy) Validate(password string) []string {
	var violations []string

	if len(password) < p.MinLength {
		violations = append(violations, "password must be at least "+strconv.Itoa(p.MinLength)+" characters")
	}
	if p.MaxLength > 0 && len(password) > p.MaxLength {
		violations = append(violations, "password must be at most "+strconv.Itoa(p.MaxLength)+" characters")
	}

	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	if p.RequireLower && !hasLower {
		violations = append(violations, "password must contain a lowercase letter")
	}
	if p.RequireUpper && !hasUpper {
		violations = append(violations, "password must contain an uppercase letter")
	}
	if p.RequireDigit && !hasDigit {
		violations = append(violations, "password must contain a digit")
	}
	if p.RequireSymbol && !hasSymbol {
		violations = append(violations, "password must contain a symbol")
	}

	if CheckPasswordStrength(password) < p.MinStrength {
		violations = append(violations, "password is too weak")
	}

	return violations
}
