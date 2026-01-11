// auth/oauth2/apple.go
package oauth2

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Apple OAuth2 endpoints.
var appleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://appleid.apple.com/auth/authorize",
	TokenURL: "https://appleid.apple.com/auth/token",
}

// AppleConfig holds configuration for Apple OAuth2 authentication (Sign in with Apple).
type AppleConfig struct {
	// ClientID is the Apple Services ID (e.g., "com.example.myapp").
	// This is the identifier for your app registered in Apple Developer portal.
	ClientID string

	// TeamID is your Apple Developer Team ID (10-character string).
	TeamID string

	// KeyID is the Key ID for your Sign in with Apple private key.
	KeyID string

	// PrivateKey is the contents of the .p8 private key file downloaded from Apple.
	// This is used to generate the client secret JWT.
	PrivateKey string

	// RedirectURL is the callback URL registered with Apple.
	// Example: "https://myapp.com/auth/apple/callback"
	// Note: Apple requires HTTPS for redirect URLs.
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: name, email
	// Note: Apple only returns name/email on FIRST login.
	Scopes []string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid (in seconds).
	// Default: 24 hours.
	SessionDuration int

	// CookieName is the name of the session cookie.
	// Default: "waffle_session".
	CookieName string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	// Default: true.
	CookieSecure bool

	// OnSuccess is called after successful authentication.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// Apple creates a new OAuth2 provider configured for Sign in with Apple.
//
// Sign in with Apple is required for iOS apps that offer third-party sign-in options.
// It provides privacy-focused authentication with optional email hiding.
//
// Important notes:
//   - Apple only returns the user's name and email on the FIRST authentication.
//     You must store this information in your database on first login.
//   - Apple uses a client secret JWT instead of a static secret.
//   - Requires HTTPS for redirect URLs.
//
// Setup in Apple Developer Portal:
//  1. Go to Certificates, Identifiers & Profiles
//  2. Create an App ID with "Sign in with Apple" capability
//  3. Create a Services ID (this becomes your ClientID)
//  4. Configure the Services ID with your domain and redirect URL
//  5. Create a Key for "Sign in with Apple" and download the .p8 file
//
// Usage in BuildHandler:
//
//	appleAuth, err := oauth2.Apple(oauth2.AppleConfig{
//	    ClientID:    appCfg.AppleClientID,    // Services ID, e.g., "com.example.myapp"
//	    TeamID:      appCfg.AppleTeamID,      // 10-char team ID
//	    KeyID:       appCfg.AppleKeyID,       // Key ID from .p8 file
//	    PrivateKey:  appCfg.ApplePrivateKey,  // Contents of .p8 file
//	    RedirectURL: "https://myapp.com/auth/apple/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/apple/login", appleAuth.LoginHandler())
//	r.Post("/auth/apple/callback", appleAuth.CallbackHandler()) // Apple uses POST
//	r.Get("/auth/apple/logout", appleAuth.LogoutHandler())
//
// The User.Extra map contains Apple-specific fields:
//   - "apple_user_id": Apple's unique user identifier (stable across your apps)
//   - "is_private_email": "true" if user is using Apple's private relay email
//   - "real_user_status": "0" (unsupported), "1" (unknown), "2" (likely real)
//   - "first_name": User's first name (only on first auth)
//   - "last_name": User's last name (only on first auth)
func Apple(cfg AppleConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/apple: ClientID (Services ID) is required")
	}
	if cfg.TeamID == "" {
		return nil, errors.New("oauth2/apple: TeamID is required")
	}
	if cfg.KeyID == "" {
		return nil, errors.New("oauth2/apple: KeyID is required")
	}
	if cfg.PrivateKey == "" {
		return nil, errors.New("oauth2/apple: PrivateKey is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/apple: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/apple: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/apple: StateStore is required")
	}

	// Parse the private key to validate it
	_, err := parseApplePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("oauth2/apple: invalid private key: %w", err)
	}

	// Default scopes for Apple
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"name",
			"email",
		}
	}

	// Generate the client secret JWT
	clientSecret, err := generateAppleClientSecret(cfg.ClientID, cfg.TeamID, cfg.KeyID, cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("oauth2/apple: failed to generate client secret: %w", err)
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     appleEndpoint,
	}

	// Create a fetcher that can handle Apple's ID token
	fetchUserInfo := createAppleUserInfoFetcher(cfg.ClientID)

	providerCfg := &Config{
		ProviderName:  "apple",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: fetchUserInfo,
		SessionStore:  cfg.SessionStore,
		StateStore:    cfg.StateStore,
		CookieName:    cfg.CookieName,
		CookieSecure:  cfg.CookieSecure,
		OnSuccess:     cfg.OnSuccess,
		OnError:       cfg.OnError,
		Logger:        logger,
	}

	if cfg.SessionDuration > 0 {
		providerCfg.SessionDuration = time.Duration(cfg.SessionDuration) * time.Second
	}

	return NewProvider(providerCfg)
}

// parseApplePrivateKey parses a PEM-encoded ECDSA private key from Apple.
func parseApplePrivateKey(privateKeyPEM string) (any, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}

// generateAppleClientSecret generates a JWT client secret for Apple authentication.
// Apple requires a signed JWT instead of a static client secret.
func generateAppleClientSecret(clientID, teamID, keyID, privateKeyPEM string) (string, error) {
	key, err := parseApplePrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": now.Add(180 * 24 * time.Hour).Unix(), // Max 6 months
		"aud": "https://appleid.apple.com",
		"sub": clientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(key)
}

// appleIDTokenClaims represents the claims in Apple's ID token.
type appleIDTokenClaims struct {
	jwt.RegisteredClaims
	Email           string `json:"email"`
	EmailVerified   string `json:"email_verified"` // "true" or "false" as string
	IsPrivateEmail  string `json:"is_private_email"` // "true" or "false"
	RealUserStatus  int    `json:"real_user_status"` // 0, 1, or 2
	NonceSupported  bool   `json:"nonce_supported"`
}

// appleUserName represents the user name from Apple's user object.
type appleUserName struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// appleUserInfo represents the user info Apple sends on first authentication.
type appleUserInfo struct {
	Name  appleUserName `json:"name"`
	Email string        `json:"email"`
}

// createAppleUserInfoFetcher creates a UserInfoFetcher for Apple.
// Apple doesn't have a userinfo endpoint - user info comes from the ID token
// and optionally from the form POST data on first authentication.
func createAppleUserInfoFetcher(clientID string) UserInfoFetcher {
	return func(ctx context.Context, token *oauth2.Token) (*User, error) {
		// Get the ID token from the token extras
		idTokenString, ok := token.Extra("id_token").(string)
		if !ok || idTokenString == "" {
			return nil, errors.New("missing id_token in response")
		}

		// Parse the ID token (without verification for now - in production you'd verify)
		// Apple's ID tokens are JWTs
		claims, err := parseAppleIDToken(idTokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ID token: %w", err)
		}

		// Verify the audience matches our client ID
		if claims.Audience != nil && len(claims.Audience) > 0 {
			found := false
			for _, aud := range claims.Audience {
				if aud == clientID {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.New("ID token audience mismatch")
			}
		}

		emailVerified := claims.EmailVerified == "true"
		isPrivateEmail := claims.IsPrivateEmail == "true"

		// The subject is Apple's unique user identifier
		userID := claims.Subject

		return &User{
			ID:            userID,
			Email:         claims.Email,
			EmailVerified: emailVerified,
			Name:          "", // Name only comes on first auth via form POST
			Picture:       "", // Apple doesn't provide profile pictures
			Raw: map[string]any{
				"sub":              claims.Subject,
				"email":            claims.Email,
				"email_verified":   claims.EmailVerified,
				"is_private_email": claims.IsPrivateEmail,
				"real_user_status": claims.RealUserStatus,
			},
			Extra: map[string]string{
				"apple_user_id":    userID,
				"is_private_email": fmt.Sprintf("%v", isPrivateEmail),
				"real_user_status": fmt.Sprintf("%d", claims.RealUserStatus),
			},
		}, nil
	}
}

// parseAppleIDToken parses an Apple ID token JWT without full verification.
// In production, you should verify the token signature using Apple's public keys.
func parseAppleIDToken(tokenString string) (*appleIDTokenClaims, error) {
	// Split the JWT
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode the payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims appleIDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return &claims, nil
}

// AppleCallbackHandler creates a special callback handler for Apple.
// Apple sends a POST request to the callback URL, not a GET request.
// This handler extracts user info from the form data (only available on first auth).
//
// Usage:
//
//	// Apple uses POST for the callback
//	r.Post("/auth/apple/callback", oauth2.AppleCallbackHandler(appleAuth, func(user *oauth2.User, r *http.Request) {
//	    // Extract name from form data (only on first auth)
//	    if userJSON := r.FormValue("user"); userJSON != "" {
//	        var userInfo appleUserInfo
//	        if json.Unmarshal([]byte(userJSON), &userInfo) == nil {
//	            user.Name = userInfo.Name.FirstName + " " + userInfo.Name.LastName
//	            user.Extra["first_name"] = userInfo.Name.FirstName
//	            user.Extra["last_name"] = userInfo.Name.LastName
//	        }
//	    }
//	}))
func AppleCallbackHandler(provider *Provider, enrichUser func(user *User, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse form data (Apple sends POST)
		if err := r.ParseForm(); err != nil {
			provider.handleError(w, r, fmt.Errorf("failed to parse form: %w", err))
			return
		}

		// Check for errors from Apple
		if errParam := r.FormValue("error"); errParam != "" {
			provider.handleError(w, r, fmt.Errorf("apple error: %s", errParam))
			return
		}

		// Validate state
		state := r.FormValue("state")
		if state == "" {
			provider.handleError(w, r, errors.New("missing state parameter"))
			return
		}

		valid, err := provider.config.StateStore.Validate(r.Context(), state)
		if err != nil || !valid {
			provider.handleError(w, r, errors.New("invalid or expired state"))
			return
		}

		// Exchange code for token
		code := r.FormValue("code")
		if code == "" {
			provider.handleError(w, r, errors.New("missing code parameter"))
			return
		}

		token, err := provider.config.OAuth2Config.Exchange(r.Context(), code)
		if err != nil {
			provider.handleError(w, r, fmt.Errorf("failed to exchange code: %w", err))
			return
		}

		// Fetch user info from ID token
		user, err := provider.config.FetchUserInfo(r.Context(), token)
		if err != nil {
			provider.handleError(w, r, fmt.Errorf("failed to fetch user info: %w", err))
			return
		}

		// Apple sends user info as JSON in the "user" form field (only on first auth)
		if userJSON := r.FormValue("user"); userJSON != "" {
			var userInfo appleUserInfo
			if json.Unmarshal([]byte(userJSON), &userInfo) == nil {
				firstName := userInfo.Name.FirstName
				lastName := userInfo.Name.LastName
				if firstName != "" || lastName != "" {
					user.Name = strings.TrimSpace(firstName + " " + lastName)
					user.Extra["first_name"] = firstName
					user.Extra["last_name"] = lastName
				}
				if userInfo.Email != "" && user.Email == "" {
					user.Email = userInfo.Email
				}
			}
		}

		// Allow caller to further enrich user data
		if enrichUser != nil {
			enrichUser(user, r)
		}

		user.Provider = provider.config.ProviderName
		user.AccessToken = token.AccessToken
		user.RefreshToken = token.RefreshToken
		user.TokenExpiry = token.Expiry

		// Create session
		sessionID, err := generateSessionID()
		if err != nil {
			provider.handleError(w, r, fmt.Errorf("failed to generate session: %w", err))
			return
		}

		session := &Session{
			ID:        sessionID,
			User:      *user,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(provider.config.SessionDuration),
		}

		if err := provider.config.SessionStore.Save(r.Context(), session); err != nil {
			provider.handleError(w, r, fmt.Errorf("failed to save session: %w", err))
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     provider.config.CookieName,
			Value:    sessionID,
			Path:     provider.config.CookiePath,
			Secure:   provider.config.CookieSecure,
			HttpOnly: true,
			SameSite: provider.config.CookieSameSite,
			MaxAge:   int(provider.config.SessionDuration.Seconds()),
		})

		provider.logger.Info("Apple authentication successful",
			zap.String("user_id", user.ID),
			zap.String("email", user.Email),
		)

		// Call success handler
		if provider.config.OnSuccess != nil {
			provider.config.OnSuccess(w, r, user)
		} else {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}
}

// IsApplePrivateEmail returns true if the user is using Apple's private relay email.
func IsApplePrivateEmail(user *User) bool {
	return user.Extra["is_private_email"] == "true"
}

// GetAppleRealUserStatus returns Apple's assessment of whether the user is real.
// Returns:
//   - 0: Unsupported (older iOS versions)
//   - 1: Unknown
//   - 2: Likely real
func GetAppleRealUserStatus(user *User) int {
	switch user.Extra["real_user_status"] {
	case "2":
		return 2
	case "1":
		return 1
	default:
		return 0
	}
}
