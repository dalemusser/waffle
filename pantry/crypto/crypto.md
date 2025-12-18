# crypto

Cryptographic utilities for WAFFLE applications.

## Overview

The `crypto` package provides:
- **Password hashing** — Argon2id (recommended) and bcrypt
- **Encryption** — AES-GCM symmetric encryption
- **Hashing** — SHA-256, SHA-512, HMAC
- **Random generation** — Tokens, keys, OTPs, verification codes

## Import

```go
import "github.com/dalemusser/waffle/crypto"
```

---

## Password Hashing

### Argon2id (Recommended)

Argon2id is the recommended algorithm for password hashing. It's resistant to GPU and side-channel attacks.

**Location:** `password.go`

```go
// Hash a password
hash, err := crypto.HashPassword("user-password", nil)
// $argon2id$v=19$m=65536,t=3,p=2$salt$hash

// Verify a password
err := crypto.VerifyPassword("user-password", hash)
if err == crypto.ErrMismatchedPassword {
    // Invalid password
}

// Boolean version
if crypto.CheckPasswordHash("user-password", hash) {
    // Valid
}
```

### Custom Parameters

```go
// Default parameters (64 MiB memory)
params := crypto.DefaultArgon2Params()

// Low memory environments (32 MiB)
params := crypto.LowMemoryArgon2Params()

// Custom parameters
params := &crypto.Argon2Params{
    Memory:      64 * 1024, // 64 MiB
    Iterations:  3,
    Parallelism: 2,
    SaltLength:  16,
    KeyLength:   32,
}

hash, err := crypto.HashPassword("password", params)
```

### Rehashing

Check if a hash needs to be updated with new parameters:

```go
if crypto.NeedsRehash(hash, crypto.DefaultArgon2Params()) {
    newHash, _ := crypto.HashPassword(password, nil)
    // Update stored hash
}
```

### Bcrypt (Legacy/Compatibility)

For compatibility with existing systems using bcrypt:

```go
// Hash with bcrypt
hash, err := crypto.HashPasswordBcrypt("password", crypto.BcryptDefaultCost)

// Verify
err := crypto.VerifyPasswordBcrypt("password", hash)

// Check if rehash needed
if crypto.NeedsRehashBcrypt(hash, crypto.BcryptDefaultCost) {
    // Rehash with current cost
}
```

---

## Password Generation & Validation

### Generate Passwords

```go
// Generate a random password (min 8 chars)
password, err := crypto.GeneratePassword(16)
// e.g., "Kx9$mP2nQ!4wR7tY"
```

### Check Strength

```go
strength := crypto.CheckPasswordStrength("MyP@ssw0rd!")
// crypto.PasswordStrong

switch strength {
case crypto.PasswordWeak:
    // Reject
case crypto.PasswordFair:
    // Warn user
case crypto.PasswordGood, crypto.PasswordStrong:
    // Accept
}
```

### Password Policy

```go
policy := crypto.DefaultPasswordPolicy()
// Or customize:
policy := crypto.PasswordPolicy{
    MinLength:     12,
    MaxLength:     128,
    RequireUpper:  true,
    RequireLower:  true,
    RequireDigit:  true,
    RequireSymbol: true,
    MinStrength:   crypto.PasswordGood,
}

violations := policy.Validate("weak")
if len(violations) > 0 {
    // violations: ["password must be at least 12 characters", ...]
}
```

---

## Encryption

### AES-GCM Encryption

**Location:** `encrypt.go`

```go
// Generate a key (32 bytes for AES-256)
key, _ := crypto.GenerateKey(32)

// Create encryptor
enc, err := crypto.NewEncryptor(key)

// Encrypt
ciphertext, err := enc.Encrypt([]byte("secret data"))

// Decrypt
plaintext, err := enc.Decrypt(ciphertext)
```

### String Encryption

```go
// Generate key as base64 string
keyStr, _ := crypto.GenerateKeyString(32)

// Create encryptor from string key
enc, _ := crypto.NewEncryptorFromString(keyStr)

// Encrypt/decrypt strings (base64 encoded)
encrypted, _ := enc.EncryptString("secret message")
decrypted, _ := enc.DecryptString(encrypted)
```

### Convenience Functions

```go
// One-off encryption
keyStr, _ := crypto.GenerateKeyString(32)

ciphertext, _ := crypto.EncryptString(keyStr, "secret")
plaintext, _ := crypto.DecryptString(keyStr, ciphertext)
```

### Key Sizes

| Size | Algorithm |
|------|-----------|
| 16 bytes | AES-128 |
| 24 bytes | AES-192 |
| 32 bytes | AES-256 (recommended) |

---

## Hashing

### SHA-256

**Location:** `hash.go`

```go
// Hash bytes
hash := crypto.SHA256(data)
hashHex := crypto.SHA256Hex(data)
hashB64 := crypto.SHA256Base64(data)

// Hash string
hashHex := crypto.SHA256String("hello")
```

### SHA-512

```go
hash := crypto.SHA512(data)
hashHex := crypto.SHA512Hex(data)
hashStr := crypto.SHA512String("hello")
```

### HMAC

```go
// HMAC-SHA256
mac := crypto.HMACSHA256(key, data)
macHex := crypto.HMACSHA256Hex(key, data)
macB64 := crypto.HMACSHA256Base64(key, data)

// HMAC-SHA512
mac := crypto.HMACSHA512(key, data)

// Verify HMAC (constant-time)
valid := crypto.VerifyHMACSHA256(key, data, signature)
valid := crypto.VerifyHMACSHA256Hex(key, data, signatureHex)
```

### File Hashing

```go
hash, err := crypto.HashFile("/path/to/file")
hashHex, err := crypto.HashFileHex("/path/to/file")

// From reader
hashHex, err := crypto.HashReaderHex(reader)
```

---

## Random Generation

### Random Bytes & Strings

**Location:** `random.go`

```go
// Random bytes
bytes, _ := crypto.RandomBytes(32)

// Hex string (64 chars for 32 bytes)
hex, _ := crypto.RandomHex(32)

// Base64
b64, _ := crypto.RandomBase64(32)
b64url, _ := crypto.RandomBase64URL(32)

// Custom charset
str, _ := crypto.RandomString(16, "ABCDEF0123456789")

// Alphanumeric
str, _ := crypto.RandomAlphanumeric(16)

// Letters only
str, _ := crypto.RandomAlpha(16)

// Digits only
str, _ := crypto.RandomNumeric(6)
```

### Random Numbers

```go
// Random int [0, max)
n, _ := crypto.RandomInt(100)

// Random int [min, max]
n, _ := crypto.RandomIntRange(10, 20)
```

### Common Tokens

```go
// Session/CSRF token (64 hex chars)
token, _ := crypto.GenerateToken()

// Short token (32 hex chars)
token, _ := crypto.GenerateShortToken()

// API key with prefix
apiKey, _ := crypto.GenerateAPIKey("sk")
// "sk_a1b2c3d4e5f6g7h8i9j0..."

// Numeric OTP
otp, _ := crypto.GenerateOTP(6)
// "847293"

// User-friendly verification code (no confusing chars)
code, _ := crypto.GenerateVerificationCode(6)
// "K7M3NP"
```

---

## WAFFLE Integration

### User Registration

```go
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Validate password
    policy := crypto.DefaultPasswordPolicy()
    if violations := policy.Validate(req.Password); len(violations) > 0 {
        // Return validation errors
        return
    }

    // Hash password
    hash, err := crypto.HashPassword(req.Password, nil)
    if err != nil {
        http.Error(w, "internal error", 500)
        return
    }

    // Store user with hash
    user := User{
        Email:        req.Email,
        PasswordHash: hash,
    }
    db.CreateUser(user)
}
```

### User Login

```go
func loginHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    user, err := db.GetUserByEmail(req.Email)
    if err != nil {
        http.Error(w, "invalid credentials", 401)
        return
    }

    // Verify password
    if err := crypto.VerifyPassword(req.Password, user.PasswordHash); err != nil {
        http.Error(w, "invalid credentials", 401)
        return
    }

    // Check if rehash needed (password upgrade)
    if crypto.NeedsRehash(user.PasswordHash, nil) {
        newHash, _ := crypto.HashPassword(req.Password, nil)
        db.UpdatePasswordHash(user.ID, newHash)
    }

    // Create session/token...
}
```

### Encrypting Sensitive Data

```go
// At startup, load encryption key from environment
key := os.Getenv("ENCRYPTION_KEY")
encryptor, _ := crypto.NewEncryptorFromString(key)

// Encrypt before storing
func storeAPICredential(userID string, credential string) error {
    encrypted, err := encryptor.EncryptString(credential)
    if err != nil {
        return err
    }
    return db.SaveCredential(userID, encrypted)
}

// Decrypt when retrieving
func getAPICredential(userID string) (string, error) {
    encrypted, err := db.GetCredential(userID)
    if err != nil {
        return "", err
    }
    return encryptor.DecryptString(encrypted)
}
```

### Webhook Signature Verification

```go
func webhookHandler(secret []byte) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        signature := r.Header.Get("X-Signature")
        body, _ := io.ReadAll(r.Body)

        if !crypto.VerifyHMACSHA256Hex(secret, body, signature) {
            http.Error(w, "invalid signature", 401)
            return
        }

        // Process webhook...
    }
}
```

### Email Verification

```go
func sendVerificationEmail(user User) error {
    code, _ := crypto.GenerateVerificationCode(6)

    // Store code with expiration
    db.SaveVerificationCode(user.ID, code, time.Now().Add(15*time.Minute))

    // Send email with code
    return email.Send(user.Email, "Verify your account",
        fmt.Sprintf("Your verification code is: %s", code))
}
```

---

## Security Recommendations

1. **Passwords**: Always use Argon2id for new applications. Use bcrypt only for compatibility.

2. **Keys**: Generate keys using `crypto.GenerateKey()`, never from passwords directly.

3. **Storage**: Store encryption keys in environment variables or secret managers, never in code.

4. **Comparison**: Always use constant-time comparison for secrets (`VerifyHMAC`, etc.).

5. **Rehashing**: Implement password rehashing on login to keep hashes up-to-date.

---

## Errors

```go
crypto.ErrInvalidHash          // Hash format is invalid
crypto.ErrIncompatibleVersion  // Argon2 version mismatch
crypto.ErrMismatchedPassword   // Password doesn't match
crypto.ErrInvalidKey           // Key size is invalid
crypto.ErrInvalidCiphertext    // Ciphertext is malformed
crypto.ErrDecryptionFailed     // Decryption failed (wrong key or tampered)
```

---

## See Also

- [auth/jwt](../auth/jwt/jwt.md) — JWT authentication
- [session](../session/session.md) — Server-side sessions
- [auth/apikey](../auth/apikey/apikey.md) — API key authentication
