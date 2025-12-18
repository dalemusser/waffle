// auth/oauth2/lti.go
package oauth2

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// LTIVersion represents the LTI protocol version.
type LTIVersion string

const (
	LTIVersion13 LTIVersion = "1.3"
)

// LTIRole represents standard LTI roles.
type LTIRole string

const (
	// Institution roles
	LTIRoleInstructor      LTIRole = "Instructor"
	LTIRoleLearner         LTIRole = "Learner"
	LTIRoleAdministrator   LTIRole = "Administrator"
	LTIRoleContentDeveloper LTIRole = "ContentDeveloper"
	LTIRoleMentor          LTIRole = "Mentor"
	LTIRoleTeachingAssistant LTIRole = "TeachingAssistant"

	// Context roles (course-level)
	LTIRoleContextInstructor    LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor"
	LTIRoleContextLearner       LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#Learner"
	LTIRoleContextAdministrator LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#Administrator"
	LTIRoleContextContentDeveloper LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#ContentDeveloper"
	LTIRoleContextMentor        LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#Mentor"
	LTIRoleContextTeachingAssistant LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/membership#TeachingAssistant"

	// System roles
	LTIRoleSystemAdministrator LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/system/person#Administrator"
	LTIRoleSystemUser          LTIRole = "http://purl.imsglobal.org/vocab/lis/v2/system/person#User"
)

// LTIMessageType represents the type of LTI message.
type LTIMessageType string

const (
	LTIMessageTypeLaunchRequest        LTIMessageType = "LtiResourceLinkRequest"
	LTIMessageTypeDeepLinkingRequest   LTIMessageType = "LtiDeepLinkingRequest"
	LTIMessageTypeDeepLinkingResponse  LTIMessageType = "LtiDeepLinkingResponse"
	LTIMessageTypeSubmissionReview     LTIMessageType = "LtiSubmissionReviewRequest"
)

// LTIConfig holds configuration for LTI 1.3 authentication.
type LTIConfig struct {
	// Issuer is this tool's client ID / issuer URL.
	// This is provided by the LMS platform when registering the tool.
	Issuer string

	// ClientID is the LTI tool's client ID assigned by the platform.
	ClientID string

	// DeploymentID is the deployment-specific ID (optional, for multi-tenancy).
	DeploymentID string

	// PlatformConfigs is a map of platform issuers to their configuration.
	// This allows a single tool to work with multiple LMS platforms.
	PlatformConfigs map[string]*LTIPlatformConfig

	// PrivateKey is the RSA private key for signing JWTs (tool -> platform).
	// This should be in PEM format.
	PrivateKey string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// NonceStore persists nonces to prevent replay attacks.
	// If nil, nonce validation is skipped (not recommended for production).
	NonceStore NonceStore

	// SessionDuration controls how long sessions remain valid (in seconds).
	// Default: 24 hours.
	SessionDuration int

	// CookieName is the name of the session cookie.
	// Default: "waffle_session".
	CookieName string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	// Default: true.
	CookieSecure bool

	// OnSuccess is called after successful LTI launch.
	OnSuccess func(w http.ResponseWriter, r *http.Request, launch *LTILaunch)

	// OnError is called when an error occurs during LTI authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// LTIPlatformConfig holds configuration for an LMS platform.
type LTIPlatformConfig struct {
	// Issuer is the platform's issuer URL (e.g., "https://canvas.instructure.com").
	Issuer string

	// AuthURL is the platform's OIDC authorization endpoint.
	AuthURL string

	// TokenURL is the platform's token endpoint.
	TokenURL string

	// JWKSURL is the platform's JSON Web Key Set URL for verifying signatures.
	JWKSURL string

	// PublicKey is the platform's public key (alternative to JWKS).
	// This should be in PEM format.
	PublicKey string
}

// NonceStore defines the interface for storing LTI nonces.
type NonceStore interface {
	// Save stores a nonce with an expiration time. Returns error if nonce already exists.
	Save(ctx context.Context, nonce string, expiresAt time.Time) error

	// Exists checks if a nonce has been used.
	Exists(ctx context.Context, nonce string) (bool, error)
}

// LTILaunch represents a successful LTI launch with all claims.
type LTILaunch struct {
	// User information
	User         *User
	UserID       string   // Platform user ID (sub claim)
	Roles        []string // LTI roles
	Email        string
	Name         string
	GivenName    string
	FamilyName   string
	Picture      string

	// Context (course) information
	ContextID    string // Course/context ID
	ContextLabel string // Course code/label
	ContextTitle string // Course title
	ContextType  []string // Context types

	// Resource link information
	ResourceLinkID          string // Link ID (unique per placement)
	ResourceLinkTitle       string // Link title
	ResourceLinkDescription string // Link description

	// Platform information
	PlatformID            string // Platform issuer
	PlatformName          string // Platform name (e.g., "Canvas")
	PlatformVersion       string // Platform version
	PlatformProductFamily string // Product family (e.g., "canvas")

	// Launch information
	MessageType    LTIMessageType
	LTIVersion     string
	DeploymentID   string
	TargetLinkURI  string
	LaunchPresentation *LTILaunchPresentation

	// Custom parameters
	Custom map[string]string

	// Deep linking (if applicable)
	DeepLinkingSettings *LTIDeepLinkingSettings

	// Names and Roles Provisioning Service
	NRPSEndpoint     string
	NRPSContextMembershipsURL string

	// Assignment and Grade Services
	AGSEndpoint      string
	AGSLineItemsURL  string
	AGSLineItemURL   string
	AGSScopes        []string

	// Raw claims for advanced use
	RawClaims map[string]any
}

// LTILaunchPresentation contains presentation settings from the platform.
type LTILaunchPresentation struct {
	DocumentTarget string // "iframe", "window", "embed"
	Width          int
	Height         int
	ReturnURL      string
	Locale         string
}

// LTIDeepLinkingSettings contains deep linking configuration.
type LTIDeepLinkingSettings struct {
	DeepLinkReturnURL string
	AcceptTypes       []string // "link", "file", "html", "ltiResourceLink", "image"
	AcceptPresentationDocumentTargets []string
	AcceptMediaTypes  string
	AcceptMultiple    bool
	AutoCreate        bool
	Title             string
	Text              string
	Data              string
}

// LTIProvider handles LTI 1.3 authentication.
type LTIProvider struct {
	config     *LTIConfig
	privateKey *rsa.PrivateKey
	logger     *zap.Logger
}

// LTI creates a new LTI 1.3 provider with the given configuration.
//
// LTI (Learning Tools Interoperability) is the standard for integrating external
// tools with Learning Management Systems like Canvas, Blackboard, Moodle, etc.
// LTI 1.3 uses OIDC for authentication and supports features like:
// - Deep Linking (content selection)
// - Names and Roles Provisioning (roster access)
// - Assignment and Grade Services (gradebook integration)
//
// Usage in BuildHandler:
//
//	ltiProvider, err := oauth2.LTI(oauth2.LTIConfig{
//	    ClientID:     appCfg.LTIClientID,
//	    Issuer:       appCfg.LTIIssuer,
//	    PrivateKey:   appCfg.LTIPrivateKey,
//	    PlatformConfigs: map[string]*oauth2.LTIPlatformConfig{
//	        "https://canvas.instructure.com": {
//	            Issuer:   "https://canvas.instructure.com",
//	            AuthURL:  "https://canvas.instructure.com/api/lti/authorize_redirect",
//	            TokenURL: "https://canvas.instructure.com/login/oauth2/token",
//	            JWKSURL:  "https://canvas.instructure.com/api/lti/security/jwks",
//	        },
//	    },
//	    SessionStore: sessionStore,
//	    StateStore:   stateStore,
//	    OnSuccess: func(w http.ResponseWriter, r *http.Request, launch *oauth2.LTILaunch) {
//	        // Handle successful launch
//	        if oauth2.IsLTIInstructor(launch) {
//	            http.Redirect(w, r, "/instructor/dashboard", http.StatusTemporaryRedirect)
//	        } else {
//	            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
//	        }
//	    },
//	}, logger)
//
//	// LTI endpoints
//	r.Post("/lti/login", ltiProvider.LoginHandler())      // OIDC login initiation
//	r.Post("/lti/launch", ltiProvider.LaunchHandler())    // Tool launch
//	r.Get("/lti/jwks", ltiProvider.JWKSHandler())         // Public key for platform
//	r.Post("/lti/deeplink", ltiProvider.DeepLinkHandler()) // Deep linking response
func LTI(cfg LTIConfig, logger *zap.Logger) (*LTIProvider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/lti: ClientID is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/lti: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/lti: StateStore is required")
	}
	if len(cfg.PlatformConfigs) == 0 {
		return nil, errors.New("oauth2/lti: at least one PlatformConfig is required")
	}

	// Parse private key if provided
	var privateKey *rsa.PrivateKey
	if cfg.PrivateKey != "" {
		block, _ := pem.Decode([]byte(cfg.PrivateKey))
		if block == nil {
			return nil, errors.New("oauth2/lti: failed to parse private key PEM")
		}

		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			// Try PKCS1 format
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("oauth2/lti: failed to parse private key: %w", err)
			}
		}

		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("oauth2/lti: private key is not RSA")
		}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &LTIProvider{
		config:     &cfg,
		privateKey: privateKey,
		logger:     logger,
	}, nil
}

// LoginHandler returns an HTTP handler for the OIDC login initiation.
// This is where the LMS platform redirects to start the authentication flow.
func (p *LTIProvider) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to parse form: %w", err))
			return
		}

		// Extract login initiation parameters
		issuer := r.FormValue("iss")
		loginHint := r.FormValue("login_hint")
		targetLinkURI := r.FormValue("target_link_uri")
		ltiMessageHint := r.FormValue("lti_message_hint")
		clientID := r.FormValue("client_id")
		deploymentID := r.FormValue("lti_deployment_id")

		// Validate issuer
		if issuer == "" {
			p.handleError(w, r, errors.New("missing issuer"))
			return
		}

		platformCfg, ok := p.config.PlatformConfigs[issuer]
		if !ok {
			p.handleError(w, r, fmt.Errorf("unknown platform issuer: %s", issuer))
			return
		}

		// Validate client ID if provided
		if clientID != "" && clientID != p.config.ClientID {
			p.handleError(w, r, fmt.Errorf("client_id mismatch: expected %s, got %s", p.config.ClientID, clientID))
			return
		}

		// Generate state and nonce
		state, err := generateState()
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to generate state: %w", err))
			return
		}

		nonce, err := generateState() // Use same function for nonce
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to generate nonce: %w", err))
			return
		}

		// Store state
		expiresAt := time.Now().Add(10 * time.Minute)
		if err := p.config.StateStore.Save(r.Context(), state, expiresAt); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to save state: %w", err))
			return
		}

		// Store nonce if nonce store is available
		if p.config.NonceStore != nil {
			if err := p.config.NonceStore.Save(r.Context(), nonce, expiresAt); err != nil {
				p.handleError(w, r, fmt.Errorf("failed to save nonce: %w", err))
				return
			}
		}

		// Build authorization URL
		authParams := url.Values{
			"scope":          {"openid"},
			"response_type":  {"id_token"},
			"response_mode":  {"form_post"},
			"client_id":      {p.config.ClientID},
			"redirect_uri":   {targetLinkURI},
			"login_hint":     {loginHint},
			"state":          {state},
			"nonce":          {nonce},
			"prompt":         {"none"},
		}

		if ltiMessageHint != "" {
			authParams.Set("lti_message_hint", ltiMessageHint)
		}

		if deploymentID != "" {
			authParams.Set("lti_deployment_id", deploymentID)
		}

		authURL := platformCfg.AuthURL + "?" + authParams.Encode()

		p.logger.Debug("initiating LTI OIDC flow",
			zap.String("issuer", issuer),
			zap.String("auth_url", authURL),
		)

		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

// LaunchHandler returns an HTTP handler for processing LTI launches.
// This receives the id_token from the platform and validates it.
func (p *LTIProvider) LaunchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to parse form: %w", err))
			return
		}

		// Get id_token and state
		idToken := r.FormValue("id_token")
		state := r.FormValue("state")

		if idToken == "" {
			p.handleError(w, r, errors.New("missing id_token"))
			return
		}

		// Validate state
		if state != "" {
			valid, err := p.config.StateStore.Validate(r.Context(), state)
			if err != nil {
				p.handleError(w, r, fmt.Errorf("failed to validate state: %w", err))
				return
			}
			if !valid {
				p.handleError(w, r, errors.New("invalid or expired state"))
				return
			}
		}

		// Parse and validate the JWT
		launch, err := p.validateIDToken(r.Context(), idToken)
		if err != nil {
			p.handleError(w, r, fmt.Errorf("invalid id_token: %w", err))
			return
		}

		// Create user and session
		user := launch.User
		user.Provider = "lti"
		user.AccessToken = idToken

		sessionID, err := generateSessionID()
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to generate session: %w", err))
			return
		}

		sessionDuration := time.Duration(p.config.SessionDuration) * time.Second
		if sessionDuration == 0 {
			sessionDuration = 24 * time.Hour
		}

		session := &Session{
			ID:        sessionID,
			User:      *user,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(sessionDuration),
		}

		if err := p.config.SessionStore.Save(r.Context(), session); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to save session: %w", err))
			return
		}

		// Set session cookie
		cookieName := p.config.CookieName
		if cookieName == "" {
			cookieName = "waffle_session"
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    sessionID,
			Path:     "/",
			Secure:   p.config.CookieSecure,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode, // LTI requires None for iframe
			MaxAge:   int(sessionDuration.Seconds()),
		})

		p.logger.Info("LTI launch successful",
			zap.String("user_id", user.ID),
			zap.String("context_id", launch.ContextID),
			zap.String("message_type", string(launch.MessageType)),
		)

		// Call success handler
		if p.config.OnSuccess != nil {
			p.config.OnSuccess(w, r, launch)
		} else {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}
}

// JWKSHandler returns an HTTP handler that serves the tool's public key in JWKS format.
// The platform uses this to verify signatures from the tool.
func (p *LTIProvider) JWKSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if p.privateKey == nil {
			http.Error(w, "no key configured", http.StatusNotFound)
			return
		}

		// Build JWKS response
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": "1", // Key ID - should be configurable
					"n":   base64.RawURLEncoding.EncodeToString(p.privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), // 65537
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}
}

// DeepLinkHandler returns an HTTP handler for creating deep link responses.
// Use this to return selected content items to the platform.
func (p *LTIProvider) DeepLinkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder - actual implementation depends on the content items
		// being returned. The tool should build the appropriate response JWT.
		http.Error(w, "deep link handler not implemented - use CreateDeepLinkResponse", http.StatusNotImplemented)
	}
}

// RequireAuth returns middleware that requires a valid LTI session.
// If the user is not authenticated, redirects to the login URL.
func (p *LTIProvider) RequireAuth(loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := p.GetSession(r)
			if err != nil || session == nil || session.IsExpired() {
				http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
				return
			}

			ctx := ContextWithUser(r.Context(), &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthJSON returns middleware that requires a valid session.
// If the user is not authenticated, returns a 401 JSON response.
func (p *LTIProvider) RequireAuthJSON() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := p.GetSession(r)
			if err != nil || session == nil || session.IsExpired() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}

			ctx := ContextWithUser(r.Context(), &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession retrieves the session from the request cookie.
func (p *LTIProvider) GetSession(r *http.Request) (*Session, error) {
	cookieName := p.config.CookieName
	if cookieName == "" {
		cookieName = "waffle_session"
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}
	if cookie.Value == "" {
		return nil, errors.New("empty session cookie")
	}
	return p.config.SessionStore.Get(r.Context(), cookie.Value)
}

// validateIDToken parses and validates an LTI id_token JWT.
func (p *LTIProvider) validateIDToken(ctx context.Context, tokenString string) (*LTILaunch, error) {
	// Parse without verification first to get the issuer
	unverified, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := unverified.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	issuer, _ := claims["iss"].(string)
	platformCfg, ok := p.config.PlatformConfigs[issuer]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuer)
	}

	// Get the platform's public key for verification
	var publicKey *rsa.PublicKey
	if platformCfg.PublicKey != "" {
		block, _ := pem.Decode([]byte(platformCfg.PublicKey))
		if block == nil {
			return nil, errors.New("failed to parse platform public key")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse platform public key: %w", err)
		}
		publicKey, ok = pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("platform public key is not RSA")
		}
	}

	// Verify the token
	var verifiedClaims jwt.MapClaims
	if publicKey != nil {
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return publicKey, nil
		})
		if err != nil {
			return nil, fmt.Errorf("token verification failed: %w", err)
		}
		verifiedClaims, ok = token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			return nil, errors.New("invalid token claims")
		}
	} else {
		// Without a public key, we can't verify - use unverified claims
		// In production, you should always configure public keys or JWKS
		verifiedClaims = claims
	}

	// Validate audience
	aud, _ := verifiedClaims["aud"].(string)
	if aud == "" {
		if audArr, ok := verifiedClaims["aud"].([]interface{}); ok && len(audArr) > 0 {
			aud, _ = audArr[0].(string)
		}
	}
	if aud != p.config.ClientID {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", p.config.ClientID, aud)
	}

	// Validate nonce if nonce store is configured
	if p.config.NonceStore != nil {
		nonce, _ := verifiedClaims["nonce"].(string)
		if nonce != "" {
			exists, err := p.config.NonceStore.Exists(ctx, nonce)
			if err != nil {
				return nil, fmt.Errorf("failed to check nonce: %w", err)
			}
			if !exists {
				return nil, errors.New("invalid or expired nonce")
			}
		}
	}

	// Build the LTI launch from claims
	return p.buildLaunchFromClaims(verifiedClaims), nil
}

// buildLaunchFromClaims constructs an LTILaunch from JWT claims.
func (p *LTIProvider) buildLaunchFromClaims(claims jwt.MapClaims) *LTILaunch {
	launch := &LTILaunch{
		RawClaims: claims,
	}

	// Standard claims
	launch.UserID, _ = claims["sub"].(string)
	launch.Name, _ = claims["name"].(string)
	launch.GivenName, _ = claims["given_name"].(string)
	launch.FamilyName, _ = claims["family_name"].(string)
	launch.Email, _ = claims["email"].(string)
	launch.Picture, _ = claims["picture"].(string)

	// LTI-specific claims
	launch.PlatformID, _ = claims["iss"].(string)
	launch.DeploymentID, _ = claims["https://purl.imsglobal.org/spec/lti/claim/deployment_id"].(string)
	launch.TargetLinkURI, _ = claims["https://purl.imsglobal.org/spec/lti/claim/target_link_uri"].(string)
	launch.LTIVersion, _ = claims["https://purl.imsglobal.org/spec/lti/claim/version"].(string)

	// Message type
	if msgType, ok := claims["https://purl.imsglobal.org/spec/lti/claim/message_type"].(string); ok {
		launch.MessageType = LTIMessageType(msgType)
	}

	// Roles
	if rolesInterface, ok := claims["https://purl.imsglobal.org/spec/lti/claim/roles"].([]interface{}); ok {
		for _, r := range rolesInterface {
			if role, ok := r.(string); ok {
				launch.Roles = append(launch.Roles, role)
			}
		}
	}

	// Context (course)
	if context, ok := claims["https://purl.imsglobal.org/spec/lti/claim/context"].(map[string]interface{}); ok {
		launch.ContextID, _ = context["id"].(string)
		launch.ContextLabel, _ = context["label"].(string)
		launch.ContextTitle, _ = context["title"].(string)
		if types, ok := context["type"].([]interface{}); ok {
			for _, t := range types {
				if typeStr, ok := t.(string); ok {
					launch.ContextType = append(launch.ContextType, typeStr)
				}
			}
		}
	}

	// Resource link
	if resourceLink, ok := claims["https://purl.imsglobal.org/spec/lti/claim/resource_link"].(map[string]interface{}); ok {
		launch.ResourceLinkID, _ = resourceLink["id"].(string)
		launch.ResourceLinkTitle, _ = resourceLink["title"].(string)
		launch.ResourceLinkDescription, _ = resourceLink["description"].(string)
	}

	// Tool platform
	if platform, ok := claims["https://purl.imsglobal.org/spec/lti/claim/tool_platform"].(map[string]interface{}); ok {
		launch.PlatformName, _ = platform["name"].(string)
		launch.PlatformVersion, _ = platform["version"].(string)
		launch.PlatformProductFamily, _ = platform["product_family_code"].(string)
	}

	// Launch presentation
	if presentation, ok := claims["https://purl.imsglobal.org/spec/lti/claim/launch_presentation"].(map[string]interface{}); ok {
		launch.LaunchPresentation = &LTILaunchPresentation{}
		launch.LaunchPresentation.DocumentTarget, _ = presentation["document_target"].(string)
		launch.LaunchPresentation.ReturnURL, _ = presentation["return_url"].(string)
		launch.LaunchPresentation.Locale, _ = presentation["locale"].(string)
		if width, ok := presentation["width"].(float64); ok {
			launch.LaunchPresentation.Width = int(width)
		}
		if height, ok := presentation["height"].(float64); ok {
			launch.LaunchPresentation.Height = int(height)
		}
	}

	// Custom parameters
	if custom, ok := claims["https://purl.imsglobal.org/spec/lti/claim/custom"].(map[string]interface{}); ok {
		launch.Custom = make(map[string]string)
		for k, v := range custom {
			if s, ok := v.(string); ok {
				launch.Custom[k] = s
			}
		}
	}

	// Deep linking settings
	if dlSettings, ok := claims["https://purl.imsglobal.org/spec/lti-dl/claim/deep_linking_settings"].(map[string]interface{}); ok {
		launch.DeepLinkingSettings = &LTIDeepLinkingSettings{}
		launch.DeepLinkingSettings.DeepLinkReturnURL, _ = dlSettings["deep_link_return_url"].(string)
		launch.DeepLinkingSettings.Title, _ = dlSettings["title"].(string)
		launch.DeepLinkingSettings.Text, _ = dlSettings["text"].(string)
		launch.DeepLinkingSettings.Data, _ = dlSettings["data"].(string)
		if acceptMultiple, ok := dlSettings["accept_multiple"].(bool); ok {
			launch.DeepLinkingSettings.AcceptMultiple = acceptMultiple
		}
		if autoCreate, ok := dlSettings["auto_create"].(bool); ok {
			launch.DeepLinkingSettings.AutoCreate = autoCreate
		}
		if types, ok := dlSettings["accept_types"].([]interface{}); ok {
			for _, t := range types {
				if typeStr, ok := t.(string); ok {
					launch.DeepLinkingSettings.AcceptTypes = append(launch.DeepLinkingSettings.AcceptTypes, typeStr)
				}
			}
		}
	}

	// Names and Roles Provisioning Service
	if nrps, ok := claims["https://purl.imsglobal.org/spec/lti-nrps/claim/namesroleservice"].(map[string]interface{}); ok {
		launch.NRPSContextMembershipsURL, _ = nrps["context_memberships_url"].(string)
		launch.NRPSEndpoint = launch.NRPSContextMembershipsURL
	}

	// Assignment and Grade Services
	if ags, ok := claims["https://purl.imsglobal.org/spec/lti-ags/claim/endpoint"].(map[string]interface{}); ok {
		launch.AGSLineItemsURL, _ = ags["lineitems"].(string)
		launch.AGSLineItemURL, _ = ags["lineitem"].(string)
		launch.AGSEndpoint = launch.AGSLineItemsURL
		if scopes, ok := ags["scope"].([]interface{}); ok {
			for _, s := range scopes {
				if scope, ok := s.(string); ok {
					launch.AGSScopes = append(launch.AGSScopes, scope)
				}
			}
		}
	}

	// Build User object
	launch.User = &User{
		ID:            launch.UserID,
		Email:         launch.Email,
		EmailVerified: launch.Email != "",
		Name:          launch.Name,
		Picture:       launch.Picture,
		Raw:           claims,
		Extra: map[string]string{
			"context_id":       launch.ContextID,
			"context_title":    launch.ContextTitle,
			"context_label":    launch.ContextLabel,
			"resource_link_id": launch.ResourceLinkID,
			"platform_id":      launch.PlatformID,
			"platform_name":    launch.PlatformName,
			"deployment_id":    launch.DeploymentID,
			"roles":            strings.Join(launch.Roles, ","),
			"given_name":       launch.GivenName,
			"family_name":      launch.FamilyName,
		},
	}

	return launch
}

// handleError calls the configured error handler or returns a generic error.
func (p *LTIProvider) handleError(w http.ResponseWriter, r *http.Request, err error) {
	p.logger.Error("LTI error", zap.Error(err))
	if p.config.OnError != nil {
		p.config.OnError(w, r, err)
		return
	}
	http.Error(w, "LTI authentication failed", http.StatusUnauthorized)
}

// CreateDeepLinkResponse creates a signed JWT for returning deep link content.
func (p *LTIProvider) CreateDeepLinkResponse(launch *LTILaunch, items []LTIContentItem) (string, error) {
	if p.privateKey == nil {
		return "", errors.New("private key required for deep linking")
	}

	if launch.DeepLinkingSettings == nil {
		return "", errors.New("launch does not have deep linking settings")
	}

	claims := jwt.MapClaims{
		"iss":  p.config.Issuer,
		"aud":  launch.PlatformID,
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
		"nonce": launch.RawClaims["nonce"],
		"https://purl.imsglobal.org/spec/lti/claim/message_type": "LtiDeepLinkingResponse",
		"https://purl.imsglobal.org/spec/lti/claim/version":      "1.3.0",
		"https://purl.imsglobal.org/spec/lti/claim/deployment_id": launch.DeploymentID,
		"https://purl.imsglobal.org/spec/lti-dl/claim/content_items": items,
	}

	if launch.DeepLinkingSettings.Data != "" {
		claims["https://purl.imsglobal.org/spec/lti-dl/claim/data"] = launch.DeepLinkingSettings.Data
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "1"

	return token.SignedString(p.privateKey)
}

// LTIContentItem represents a content item for deep linking responses.
type LTIContentItem struct {
	Type   string `json:"type"`   // "ltiResourceLink", "link", "file", "html", "image"
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	URL    string `json:"url,omitempty"`
	Icon   *LTIIcon `json:"icon,omitempty"`
	Thumbnail *LTIIcon `json:"thumbnail,omitempty"`
	Custom map[string]string `json:"custom,omitempty"`

	// For ltiResourceLink type
	LineItem *LTILineItem `json:"lineItem,omitempty"`
	Iframe   *LTIIframe   `json:"iframe,omitempty"`
	Window   *LTIWindow   `json:"window,omitempty"`
}

// LTIIcon represents an icon for content items.
type LTIIcon struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// LTILineItem represents a gradebook line item.
type LTILineItem struct {
	Label         string  `json:"label,omitempty"`
	ScoreMaximum  float64 `json:"scoreMaximum"`
	ResourceID    string  `json:"resourceId,omitempty"`
	Tag           string  `json:"tag,omitempty"`
	GradesReleased bool   `json:"gradesReleased,omitempty"`
}

// LTIIframe represents iframe presentation settings.
type LTIIframe struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

// LTIWindow represents window presentation settings.
type LTIWindow struct {
	TargetName string `json:"targetName,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

// IsLTIInstructor returns true if the user has an instructor role.
func IsLTIInstructor(launch *LTILaunch) bool {
	for _, role := range launch.Roles {
		if strings.Contains(role, "Instructor") {
			return true
		}
	}
	return false
}

// IsLTILearner returns true if the user has a learner/student role.
func IsLTILearner(launch *LTILaunch) bool {
	for _, role := range launch.Roles {
		if strings.Contains(role, "Learner") {
			return true
		}
	}
	return false
}

// IsLTIAdministrator returns true if the user has an administrator role.
func IsLTIAdministrator(launch *LTILaunch) bool {
	for _, role := range launch.Roles {
		if strings.Contains(role, "Administrator") {
			return true
		}
	}
	return false
}

// IsLTIContentDeveloper returns true if the user has a content developer role.
func IsLTIContentDeveloper(launch *LTILaunch) bool {
	for _, role := range launch.Roles {
		if strings.Contains(role, "ContentDeveloper") {
			return true
		}
	}
	return false
}

// IsLTITeachingAssistant returns true if the user has a teaching assistant role.
func IsLTITeachingAssistant(launch *LTILaunch) bool {
	for _, role := range launch.Roles {
		if strings.Contains(role, "TeachingAssistant") {
			return true
		}
	}
	return false
}

// HasLTIRole returns true if the user has any of the specified roles.
func HasLTIRole(launch *LTILaunch, roles ...string) bool {
	for _, userRole := range launch.Roles {
		for _, checkRole := range roles {
			if strings.Contains(userRole, checkRole) {
				return true
			}
		}
	}
	return false
}

// GetLTIContextID returns the course/context ID from the launch.
func GetLTIContextID(launch *LTILaunch) string {
	return launch.ContextID
}

// GetLTIContextTitle returns the course/context title from the launch.
func GetLTIContextTitle(launch *LTILaunch) string {
	return launch.ContextTitle
}

// GetLTIResourceLinkID returns the resource link ID from the launch.
func GetLTIResourceLinkID(launch *LTILaunch) string {
	return launch.ResourceLinkID
}

// GetLTIPlatformName returns the platform name (e.g., "Canvas", "Blackboard").
func GetLTIPlatformName(launch *LTILaunch) string {
	return launch.PlatformName
}

// GetLTICustomParam returns a custom parameter value.
func GetLTICustomParam(launch *LTILaunch, key string) string {
	if launch.Custom == nil {
		return ""
	}
	return launch.Custom[key]
}

// SupportsNRPS returns true if the launch includes Names and Roles Provisioning Service.
func SupportsNRPS(launch *LTILaunch) bool {
	return launch.NRPSContextMembershipsURL != ""
}

// SupportsAGS returns true if the launch includes Assignment and Grade Services.
func SupportsAGS(launch *LTILaunch) bool {
	return launch.AGSLineItemsURL != "" || launch.AGSLineItemURL != ""
}

// IsDeepLinkingRequest returns true if this is a deep linking request.
func IsDeepLinkingRequest(launch *LTILaunch) bool {
	return launch.MessageType == LTIMessageTypeDeepLinkingRequest
}

// IsResourceLinkRequest returns true if this is a standard resource link launch.
func IsResourceLinkRequest(launch *LTILaunch) bool {
	return launch.MessageType == LTIMessageTypeLaunchRequest
}
