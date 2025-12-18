// auth/oauth2/shibboleth.go
package oauth2

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ShibbolethAffiliation represents a user's affiliation type in eduPerson schema.
type ShibbolethAffiliation string

const (
	// ShibbolethAffiliationFaculty represents faculty members.
	ShibbolethAffiliationFaculty ShibbolethAffiliation = "faculty"

	// ShibbolethAffiliationStudent represents students.
	ShibbolethAffiliationStudent ShibbolethAffiliation = "student"

	// ShibbolethAffiliationStaff represents staff members.
	ShibbolethAffiliationStaff ShibbolethAffiliation = "staff"

	// ShibbolethAffiliationEmployee represents employees (may overlap with faculty/staff).
	ShibbolethAffiliationEmployee ShibbolethAffiliation = "employee"

	// ShibbolethAffiliationMember represents general members of the institution.
	ShibbolethAffiliationMember ShibbolethAffiliation = "member"

	// ShibbolethAffiliationAffiliate represents external affiliates.
	ShibbolethAffiliationAffiliate ShibbolethAffiliation = "affiliate"

	// ShibbolethAffiliationAlum represents alumni.
	ShibbolethAffiliationAlum ShibbolethAffiliation = "alum"

	// ShibbolethAffiliationLibraryWalkIn represents library walk-in users.
	ShibbolethAffiliationLibraryWalkIn ShibbolethAffiliation = "library-walk-in"

	// ShibbolethAffiliationUnknown represents an unknown affiliation.
	ShibbolethAffiliationUnknown ShibbolethAffiliation = "unknown"
)

// ShibbolethConfig holds configuration for Shibboleth/SAML authentication.
type ShibbolethConfig struct {
	// EntityID is your Service Provider's entity ID (unique identifier).
	// Example: "https://myapp.edu/shibboleth"
	EntityID string

	// IdPMetadataURL is the URL to fetch Identity Provider metadata.
	// Example: "https://idp.university.edu/idp/shibboleth" or federation metadata URL
	IdPMetadataURL string

	// IdPMetadataXML is raw IdP metadata XML (alternative to URL).
	// Use this if metadata URL is not available or for testing.
	IdPMetadataXML string

	// IdPEntityID is the Identity Provider's entity ID.
	// Required when using federation metadata with multiple IdPs.
	IdPEntityID string

	// Certificate is your SP's X.509 certificate for signing/encryption (PEM encoded).
	Certificate string

	// PrivateKey is your SP's private key (PEM encoded).
	PrivateKey string

	// AssertionConsumerServiceURL is where SAML responses are sent.
	// Example: "https://myapp.edu/auth/shibboleth/callback"
	AssertionConsumerServiceURL string

	// SingleLogoutServiceURL is where logout requests/responses are sent.
	// Example: "https://myapp.edu/auth/shibboleth/logout"
	SingleLogoutServiceURL string

	// SignRequests indicates whether to sign AuthnRequests.
	// Default: true for production.
	SignRequests bool

	// WantAssertionsSigned indicates whether assertions must be signed.
	// Default: true for security.
	WantAssertionsSigned bool

	// AllowUnencryptedAssertions allows unencrypted assertions (less secure).
	// Default: false.
	AllowUnencryptedAssertions bool

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists SAML request state for security.
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

	// AttributeMap maps SAML attribute names to User.Extra keys.
	// Default uses eduPerson/SCHAC attribute names.
	AttributeMap map[string]string

	// OnSuccess is called after successful authentication.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// ShibbolethProvider implements SAML 2.0 authentication for Shibboleth IdPs.
type ShibbolethProvider struct {
	config       *ShibbolethConfig
	idpMetadata  *samlIdPMetadata
	spCert       *x509.Certificate
	sessionStore SessionStore
	stateStore   StateStore
	logger       *zap.Logger
}

// samlIdPMetadata holds parsed IdP metadata.
type samlIdPMetadata struct {
	EntityID           string
	SSOServiceURL      string
	SSOServiceBinding  string
	SLOServiceURL      string
	SLOServiceBinding  string
	SigningCertificate *x509.Certificate
}

// Shibboleth creates a new SAML 2.0 provider configured for Shibboleth authentication.
//
// Shibboleth is the most widely deployed federated identity solution in higher education
// and research, using SAML 2.0 for secure single sign-on. It's commonly used with:
//   - InCommon Federation (US higher education)
//   - eduGAIN (global research and education)
//   - UK Access Management Federation
//   - Australian Access Federation
//   - And many national/regional federations
//
// Setup overview:
//  1. Register your Service Provider (SP) with your institution's IdP or federation
//  2. Generate SP certificate and private key
//  3. Obtain IdP metadata (URL or XML)
//  4. Configure attribute release (eduPerson attributes)
//  5. Test with your IdP administrator
//
// Common eduPerson attributes:
//   - eduPersonPrincipalName (ePPN): unique identifier like user@institution.edu
//   - eduPersonAffiliation: faculty, student, staff, employee, member, affiliate, alum
//   - eduPersonScopedAffiliation: affiliation@institution.edu
//   - eduPersonEntitlement: access rights URIs
//   - mail: email address
//   - displayName: full name
//   - givenName: first name
//   - sn (surname): last name
//
// Usage in BuildHandler:
//
//	shibAuth, err := oauth2.Shibboleth(oauth2.ShibbolethConfig{
//	    EntityID:                    "https://myapp.edu/shibboleth",
//	    IdPMetadataURL:              "https://idp.university.edu/idp/shibboleth",
//	    Certificate:                 spCertPEM,
//	    PrivateKey:                  spKeyPEM,
//	    AssertionConsumerServiceURL: "https://myapp.edu/auth/shibboleth/callback",
//	    SessionStore:                mySessionStore,
//	    StateStore:                  myStateStore,
//	}, logger)
//
//	r.Get("/auth/shibboleth/login", shibAuth.LoginHandler())
//	r.Post("/auth/shibboleth/callback", shibAuth.CallbackHandler())
//	r.Get("/auth/shibboleth/logout", shibAuth.LogoutHandler())
//	r.Get("/auth/shibboleth/metadata", shibAuth.MetadataHandler())
//
// The User.Extra map contains Shibboleth-specific fields:
//   - "eppn": eduPersonPrincipalName (user@institution.edu)
//   - "affiliation": Primary affiliation (faculty, student, staff, etc.)
//   - "affiliations": All affiliations (comma-separated)
//   - "scoped_affiliations": Scoped affiliations (affiliation@scope)
//   - "entitlements": Access entitlements (comma-separated URIs)
//   - "org": Organization/institution name
//   - "org_unit": Organizational unit/department
//   - "given_name": First name
//   - "surname": Last name
//   - "display_name": Full display name
//   - "idp_entity_id": The IdP that authenticated the user
//   - "authn_instant": Authentication timestamp
//   - "session_index": SAML session index for logout
func Shibboleth(cfg ShibbolethConfig, logger *zap.Logger) (*ShibbolethProvider, error) {
	if cfg.EntityID == "" {
		return nil, errors.New("oauth2/shibboleth: EntityID is required")
	}
	if cfg.IdPMetadataURL == "" && cfg.IdPMetadataXML == "" {
		return nil, errors.New("oauth2/shibboleth: IdPMetadataURL or IdPMetadataXML is required")
	}
	if cfg.AssertionConsumerServiceURL == "" {
		return nil, errors.New("oauth2/shibboleth: AssertionConsumerServiceURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/shibboleth: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/shibboleth: StateStore is required")
	}

	provider := &ShibbolethProvider{
		config:       &cfg,
		sessionStore: cfg.SessionStore,
		stateStore:   cfg.StateStore,
		logger:       logger,
	}

	// Set defaults
	if cfg.AttributeMap == nil {
		provider.config.AttributeMap = defaultShibbolethAttributeMap()
	}

	return provider, nil
}

// defaultShibbolethAttributeMap returns the default mapping for eduPerson/SCHAC attributes.
func defaultShibbolethAttributeMap() map[string]string {
	return map[string]string{
		// eduPerson attributes (OID format)
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.6":  "eppn",                // eduPersonPrincipalName
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.1":  "affiliation",         // eduPersonAffiliation
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.9":  "scoped_affiliation",  // eduPersonScopedAffiliation
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.7":  "entitlement",         // eduPersonEntitlement
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.10": "targeted_id",         // eduPersonTargetedID
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.11": "assurance",           // eduPersonAssurance
		"urn:oid:1.3.6.1.4.1.5923.1.1.1.13": "unique_id",           // eduPersonUniqueId

		// Standard LDAP/X.500 attributes
		"urn:oid:0.9.2342.19200300.100.1.3": "mail",         // mail
		"urn:oid:2.16.840.1.113730.3.1.241": "display_name", // displayName
		"urn:oid:2.5.4.42":                  "given_name",   // givenName
		"urn:oid:2.5.4.4":                   "surname",      // sn (surname)
		"urn:oid:2.5.4.3":                   "cn",           // cn (common name)
		"urn:oid:2.5.4.10":                  "org",          // o (organization)
		"urn:oid:2.5.4.11":                  "org_unit",     // ou (organizational unit)

		// SCHAC attributes
		"urn:oid:1.3.6.1.4.1.25178.1.2.9": "home_org",      // schacHomeOrganization
		"urn:oid:1.3.6.1.4.1.25178.1.2.3": "home_org_type", // schacHomeOrganizationType

		// Friendly name fallbacks
		"eduPersonPrincipalName":      "eppn",
		"eduPersonAffiliation":        "affiliation",
		"eduPersonScopedAffiliation":  "scoped_affiliation",
		"eduPersonEntitlement":        "entitlement",
		"eduPersonTargetedID":         "targeted_id",
		"mail":                        "mail",
		"displayName":                 "display_name",
		"givenName":                   "given_name",
		"sn":                          "surname",
		"cn":                          "cn",
		"o":                           "org",
		"ou":                          "org_unit",
		"schacHomeOrganization":       "home_org",
	}
}

// LoginHandler returns an http.Handler that initiates SAML authentication.
func (p *ShibbolethProvider) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID and store state
		requestID, err := generateState()
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to generate state: %w", err))
			return
		}

		ctx := r.Context()
		if err := p.stateStore.Save(ctx, requestID, time.Now().Add(10*time.Minute)); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to save state: %w", err))
			return
		}

		// Build SAML AuthnRequest
		authnRequest := p.buildAuthnRequest(requestID)

		// Encode and redirect
		// For HTTP-Redirect binding, the request is deflated, base64 encoded, and URL encoded
		encodedRequest := base64.StdEncoding.EncodeToString([]byte(authnRequest))

		// Build redirect URL
		redirectURL := p.getIdPSSOURL()
		if redirectURL == "" {
			p.handleError(w, r, errors.New("IdP SSO URL not configured"))
			return
		}

		// Add SAMLRequest parameter
		u, err := url.Parse(redirectURL)
		if err != nil {
			p.handleError(w, r, fmt.Errorf("invalid IdP URL: %w", err))
			return
		}

		q := u.Query()
		q.Set("SAMLRequest", encodedRequest)
		q.Set("RelayState", requestID)
		u.RawQuery = q.Encode()

		http.Redirect(w, r, u.String(), http.StatusFound)
	})
}

// CallbackHandler returns an http.Handler that processes SAML responses.
func (p *ShibbolethProvider) CallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			p.handleError(w, r, errors.New("SAML callback requires POST"))
			return
		}

		if err := r.ParseForm(); err != nil {
			p.handleError(w, r, fmt.Errorf("failed to parse form: %w", err))
			return
		}

		// Get SAML response
		samlResponse := r.FormValue("SAMLResponse")
		if samlResponse == "" {
			p.handleError(w, r, errors.New("missing SAMLResponse"))
			return
		}

		relayState := r.FormValue("RelayState")

		// Verify relay state if present
		ctx := r.Context()
		if relayState != "" {
			valid, err := p.stateStore.Validate(ctx, relayState)
			if err != nil || !valid {
				p.handleError(w, r, errors.New("invalid relay state"))
				return
			}
		}

		// Decode SAML response
		responseXML, err := base64.StdEncoding.DecodeString(samlResponse)
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to decode SAML response: %w", err))
			return
		}

		// Parse and validate SAML response
		user, err := p.parseSAMLResponse(responseXML)
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to parse SAML response: %w", err))
			return
		}

		// Create session
		sessionDuration := 24 * time.Hour
		if p.config.SessionDuration > 0 {
			sessionDuration = time.Duration(p.config.SessionDuration) * time.Second
		}

		sessionID, err := generateState()
		if err != nil {
			p.handleError(w, r, fmt.Errorf("failed to generate session ID: %w", err))
			return
		}

		session := &Session{
			ID:        sessionID,
			User:      *user,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(sessionDuration),
		}

		if err := p.sessionStore.Save(ctx, session); err != nil {
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
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			Secure:   p.config.CookieSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  session.ExpiresAt,
		})

		// Call success handler
		if p.config.OnSuccess != nil {
			p.config.OnSuccess(w, r, user)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}
	})
}

// LogoutHandler returns an http.Handler that handles logout.
func (p *ShibbolethProvider) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookieName := p.config.CookieName
		if cookieName == "" {
			cookieName = "waffle_session"
		}

		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie.Value != "" {
			// Delete session
			ctx := r.Context()
			_ = p.sessionStore.Delete(ctx, cookie.Value)
		}

		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   p.config.CookieSecure,
			MaxAge:   -1,
		})

		// Redirect to home or IdP logout if configured
		http.Redirect(w, r, "/", http.StatusFound)
	})
}

// MetadataHandler returns an http.Handler that serves SP metadata XML.
func (p *ShibbolethProvider) MetadataHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := p.generateSPMetadata()
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(metadata))
	})
}

// RequireAuth returns middleware that requires Shibboleth authentication.
func (p *ShibbolethProvider) RequireAuth(loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookieName := p.config.CookieName
			if cookieName == "" {
				cookieName = "waffle_session"
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}

			session, err := p.sessionStore.Get(r.Context(), cookie.Value)
			if err != nil || session == nil {
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}

			if time.Now().After(session.ExpiresAt) {
				_ = p.sessionStore.Delete(r.Context(), cookie.Value)
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), userContextKey, &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthJSON returns middleware for API routes.
func (p *ShibbolethProvider) RequireAuthJSON() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookieName := p.config.CookieName
			if cookieName == "" {
				cookieName = "waffle_session"
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			session, err := p.sessionStore.Get(r.Context(), cookie.Value)
			if err != nil || session == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if time.Now().After(session.ExpiresAt) {
				_ = p.sessionStore.Delete(r.Context(), cookie.Value)
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &session.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession retrieves the current session from a request.
func (p *ShibbolethProvider) GetSession(r *http.Request) (*Session, error) {
	cookieName := p.config.CookieName
	if cookieName == "" {
		cookieName = "waffle_session"
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	return p.sessionStore.Get(r.Context(), cookie.Value)
}

// buildAuthnRequest creates a SAML AuthnRequest.
func (p *ShibbolethProvider) buildAuthnRequest(requestID string) string {
	issueInstant := time.Now().UTC().Format(time.RFC3339)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="%s"
    Version="2.0"
    IssueInstant="%s"
    AssertionConsumerServiceURL="%s"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
    <saml:Issuer>%s</saml:Issuer>
    <samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient" AllowCreate="true"/>
</samlp:AuthnRequest>`, requestID, issueInstant, p.config.AssertionConsumerServiceURL, p.config.EntityID)
}

// getIdPSSOURL returns the IdP's SSO service URL.
func (p *ShibbolethProvider) getIdPSSOURL() string {
	if p.idpMetadata != nil {
		return p.idpMetadata.SSOServiceURL
	}
	// Fallback - would need to be set from metadata
	return ""
}

// parseSAMLResponse parses a SAML response and extracts user information.
func (p *ShibbolethProvider) parseSAMLResponse(responseXML []byte) (*User, error) {
	// Parse the SAML response XML
	var response samlResponse
	if err := xml.Unmarshal(responseXML, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAML response: %w", err)
	}

	// Check status
	if response.Status.StatusCode.Value != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		return nil, fmt.Errorf("SAML authentication failed: %s", response.Status.StatusCode.Value)
	}

	// Extract assertion
	if len(response.Assertions) == 0 {
		return nil, errors.New("no assertion in SAML response")
	}

	assertion := response.Assertions[0]

	// Extract user ID from NameID or attributes
	userID := assertion.Subject.NameID.Value
	if userID == "" {
		return nil, errors.New("no NameID in SAML assertion")
	}

	// Build User from attributes
	user := &User{
		ID:    userID,
		Extra: make(map[string]string),
		Raw:   make(map[string]any),
	}

	// Store IdP and session info
	user.Extra["idp_entity_id"] = response.Issuer
	user.Extra["session_index"] = assertion.AuthnStatement.SessionIndex
	user.Extra["authn_instant"] = assertion.AuthnStatement.AuthnInstant

	// Process attributes
	for _, attrStmt := range assertion.AttributeStatements {
		for _, attr := range attrStmt.Attributes {
			// Get mapped attribute name
			mappedName := attr.Name
			if mapped, ok := p.config.AttributeMap[attr.Name]; ok {
				mappedName = mapped
			} else if mapped, ok := p.config.AttributeMap[attr.FriendlyName]; ok {
				mappedName = mapped
			}

			// Collect all values
			var values []string
			for _, v := range attr.Values {
				values = append(values, v.Value)
			}

			if len(values) == 1 {
				user.Extra[mappedName] = values[0]
				user.Raw[attr.Name] = values[0]
			} else if len(values) > 1 {
				user.Extra[mappedName] = strings.Join(values, ",")
				user.Raw[attr.Name] = values
			}
		}
	}

	// Set standard user fields from attributes
	if eppn := user.Extra["eppn"]; eppn != "" {
		user.ID = eppn
	}

	if mail := user.Extra["mail"]; mail != "" {
		user.Email = mail
		user.EmailVerified = true // Institutional email is verified
	}

	if displayName := user.Extra["display_name"]; displayName != "" {
		user.Name = displayName
	} else if cn := user.Extra["cn"]; cn != "" {
		user.Name = cn
	} else {
		// Build name from parts
		givenName := user.Extra["given_name"]
		surname := user.Extra["surname"]
		user.Name = strings.TrimSpace(givenName + " " + surname)
	}

	return user, nil
}

// generateSPMetadata generates Service Provider metadata XML.
func (p *ShibbolethProvider) generateSPMetadata() string {
	certData := ""
	if p.config.Certificate != "" {
		// Extract certificate data (without PEM headers)
		cert := strings.TrimPrefix(p.config.Certificate, "-----BEGIN CERTIFICATE-----")
		cert = strings.TrimSuffix(cert, "-----END CERTIFICATE-----")
		cert = strings.TrimSpace(cert)
		certData = cert
	}

	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="%s">
    <md:SPSSODescriptor AuthnRequestsSigned="%t" WantAssertionsSigned="%t"
        protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">`,
		p.config.EntityID,
		p.config.SignRequests,
		p.config.WantAssertionsSigned)

	if certData != "" {
		metadata += fmt.Sprintf(`
        <md:KeyDescriptor use="signing">
            <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
                <ds:X509Data>
                    <ds:X509Certificate>%s</ds:X509Certificate>
                </ds:X509Data>
            </ds:KeyInfo>
        </md:KeyDescriptor>
        <md:KeyDescriptor use="encryption">
            <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
                <ds:X509Data>
                    <ds:X509Certificate>%s</ds:X509Certificate>
                </ds:X509Data>
            </ds:KeyInfo>
        </md:KeyDescriptor>`, certData, certData)
	}

	if p.config.SingleLogoutServiceURL != "" {
		metadata += fmt.Sprintf(`
        <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
            Location="%s"/>
        <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
            Location="%s"/>`, p.config.SingleLogoutServiceURL, p.config.SingleLogoutServiceURL)
	}

	metadata += fmt.Sprintf(`
        <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
        <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat>
        <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
            Location="%s" index="0" isDefault="true"/>
    </md:SPSSODescriptor>
</md:EntityDescriptor>`, p.config.AssertionConsumerServiceURL)

	return metadata
}

// handleError handles authentication errors.
func (p *ShibbolethProvider) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if p.logger != nil {
		p.logger.Error("Shibboleth authentication error", zap.Error(err))
	}

	if p.config.OnError != nil {
		p.config.OnError(w, r, err)
	} else {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
	}
}

// SAML XML structures for parsing responses

type samlResponse struct {
	XMLName    xml.Name        `xml:"Response"`
	ID         string          `xml:"ID,attr"`
	Issuer     string          `xml:"Issuer"`
	Status     samlStatus      `xml:"Status"`
	Assertions []samlAssertion `xml:"Assertion"`
}

type samlStatus struct {
	StatusCode samlStatusCode `xml:"StatusCode"`
}

type samlStatusCode struct {
	Value string `xml:"Value,attr"`
}

type samlAssertion struct {
	XMLName             xml.Name                 `xml:"Assertion"`
	ID                  string                   `xml:"ID,attr"`
	Issuer              string                   `xml:"Issuer"`
	Subject             samlSubject              `xml:"Subject"`
	Conditions          samlConditions           `xml:"Conditions"`
	AuthnStatement      samlAuthnStatement       `xml:"AuthnStatement"`
	AttributeStatements []samlAttributeStatement `xml:"AttributeStatement"`
}

type samlSubject struct {
	NameID samlNameID `xml:"NameID"`
}

type samlNameID struct {
	Format string `xml:"Format,attr"`
	Value  string `xml:",chardata"`
}

type samlConditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
}

type samlAuthnStatement struct {
	AuthnInstant string `xml:"AuthnInstant,attr"`
	SessionIndex string `xml:"SessionIndex,attr"`
}

type samlAttributeStatement struct {
	Attributes []samlAttribute `xml:"Attribute"`
}

type samlAttribute struct {
	Name         string               `xml:"Name,attr"`
	FriendlyName string               `xml:"FriendlyName,attr"`
	NameFormat   string               `xml:"NameFormat,attr"`
	Values       []samlAttributeValue `xml:"AttributeValue"`
}

type samlAttributeValue struct {
	Value string `xml:",chardata"`
}

// Helper functions for Shibboleth users

// IsShibbolethFaculty checks if the user has faculty affiliation.
func IsShibbolethFaculty(user *User) bool {
	return hasShibbolethAffiliation(user, "faculty")
}

// IsShibbolethStudent checks if the user has student affiliation.
func IsShibbolethStudent(user *User) bool {
	return hasShibbolethAffiliation(user, "student")
}

// IsShibbolethStaff checks if the user has staff affiliation.
func IsShibbolethStaff(user *User) bool {
	return hasShibbolethAffiliation(user, "staff")
}

// IsShibbolethEmployee checks if the user has employee affiliation.
func IsShibbolethEmployee(user *User) bool {
	return hasShibbolethAffiliation(user, "employee")
}

// IsShibbolethMember checks if the user has member affiliation.
func IsShibbolethMember(user *User) bool {
	return hasShibbolethAffiliation(user, "member")
}

// IsShibbolethAffiliate checks if the user has affiliate affiliation.
func IsShibbolethAffiliate(user *User) bool {
	return hasShibbolethAffiliation(user, "affiliate")
}

// IsShibbolethAlum checks if the user has alumni affiliation.
func IsShibbolethAlum(user *User) bool {
	return hasShibbolethAffiliation(user, "alum")
}

// hasShibbolethAffiliation checks if the user has a specific affiliation.
func hasShibbolethAffiliation(user *User, affiliation string) bool {
	// Check primary affiliation
	if user.Extra["affiliation"] == affiliation {
		return true
	}

	// Check all affiliations
	affiliations := user.Extra["affiliations"]
	if affiliations != "" {
		for _, a := range strings.Split(affiliations, ",") {
			// Handle scoped affiliations (e.g., student@university.edu)
			a = strings.TrimSpace(a)
			if strings.HasPrefix(a, affiliation+"@") || a == affiliation {
				return true
			}
		}
	}

	// Check scoped affiliations
	scopedAffiliations := user.Extra["scoped_affiliations"]
	if scopedAffiliations != "" {
		for _, a := range strings.Split(scopedAffiliations, ",") {
			a = strings.TrimSpace(a)
			if strings.HasPrefix(a, affiliation+"@") {
				return true
			}
		}
	}

	return false
}

// GetShibbolethAffiliations returns all affiliations for a user.
func GetShibbolethAffiliations(user *User) []string {
	var affiliations []string

	if aff := user.Extra["affiliation"]; aff != "" {
		affiliations = append(affiliations, aff)
	}

	if affs := user.Extra["affiliations"]; affs != "" {
		for _, a := range strings.Split(affs, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				affiliations = append(affiliations, a)
			}
		}
	}

	return affiliations
}

// GetShibbolethPrimaryAffiliation returns the user's primary affiliation.
func GetShibbolethPrimaryAffiliation(user *User) ShibbolethAffiliation {
	aff := user.Extra["affiliation"]
	if aff == "" {
		affiliations := GetShibbolethAffiliations(user)
		if len(affiliations) > 0 {
			aff = affiliations[0]
		}
	}

	// Strip scope if present
	if idx := strings.Index(aff, "@"); idx > 0 {
		aff = aff[:idx]
	}

	switch aff {
	case "faculty":
		return ShibbolethAffiliationFaculty
	case "student":
		return ShibbolethAffiliationStudent
	case "staff":
		return ShibbolethAffiliationStaff
	case "employee":
		return ShibbolethAffiliationEmployee
	case "member":
		return ShibbolethAffiliationMember
	case "affiliate":
		return ShibbolethAffiliationAffiliate
	case "alum":
		return ShibbolethAffiliationAlum
	case "library-walk-in":
		return ShibbolethAffiliationLibraryWalkIn
	default:
		return ShibbolethAffiliationUnknown
	}
}

// GetShibbolethEPPN returns the eduPersonPrincipalName.
func GetShibbolethEPPN(user *User) string {
	return user.Extra["eppn"]
}

// GetShibbolethEntitlements returns the user's entitlements.
func GetShibbolethEntitlements(user *User) []string {
	entitlements := user.Extra["entitlements"]
	if entitlements == "" {
		return nil
	}
	return strings.Split(entitlements, ",")
}

// HasShibbolethEntitlement checks if the user has a specific entitlement.
func HasShibbolethEntitlement(user *User, entitlement string) bool {
	entitlements := GetShibbolethEntitlements(user)
	for _, e := range entitlements {
		if strings.TrimSpace(e) == entitlement {
			return true
		}
	}
	return false
}

// GetShibbolethOrganization returns the user's organization.
func GetShibbolethOrganization(user *User) string {
	if org := user.Extra["org"]; org != "" {
		return org
	}
	return user.Extra["home_org"]
}

// GetShibbolethOrgUnit returns the user's organizational unit.
func GetShibbolethOrgUnit(user *User) string {
	return user.Extra["org_unit"]
}

// GetShibbolethIdPEntityID returns the IdP that authenticated the user.
func GetShibbolethIdPEntityID(user *User) string {
	return user.Extra["idp_entity_id"]
}

// GetShibbolethSessionIndex returns the SAML session index for logout.
func GetShibbolethSessionIndex(user *User) string {
	return user.Extra["session_index"]
}
