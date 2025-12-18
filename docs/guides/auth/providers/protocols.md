# Authentication Protocols

*Understanding the authentication protocols supported by WAFFLE.*

This document explains the different authentication protocols used by providers in the WAFFLE auth package.

---

## Protocol Overview

| Protocol | Type | Token Format | Common Use Cases |
|----------|------|--------------|------------------|
| [OAuth 2.0](#oauth-20) | Authorization | Bearer tokens | Social login, API access |
| [OIDC](#openid-connect-oidc) | Authentication | JWT (ID token) | Enterprise SSO, identity verification |
| [SAML 2.0](#saml-20) | Authentication | XML assertions | Higher ed, enterprise federation |
| [OAuth 1.0a](#oauth-10a) | Authorization | Signed requests | Legacy systems |
| [LTI 1.3](#lti-13) | Authentication | JWT | Learning tool integration |

---

## OAuth 2.0

**OAuth 2.0** is an authorization framework that enables applications to obtain limited access to user accounts on third-party services. It's the most widely used protocol for social login.

### How It Works

```
┌──────────┐     1. Redirect to Provider     ┌──────────────┐
│          │ ──────────────────────────────▶ │              │
│   Your   │                                 │   Provider   │
│   App    │     4. Access Token             │   (Google,   │
│          │ ◀────────────────────────────── │   GitHub)    │
└──────────┘                                 └──────────────┘
     │                                              │
     │  5. API Request                              │
     │     with Token          2. User Login        │
     │ ─────────────────▶           │               │
     │                              ▼               │
     │                        ┌──────────┐          │
     │                        │   User   │          │
     │                        └──────────┘          │
     │                              │               │
     │ ◀────────────────────────────┘               │
     │        3. Authorization Code                 │
```

1. **Authorization Request**: App redirects user to provider with `client_id`, `redirect_uri`, `scope`, and `state`
2. **User Authentication**: User logs in and grants permission
3. **Authorization Code**: Provider redirects back with a temporary code
4. **Token Exchange**: App exchanges code for access token (server-side)
5. **API Access**: App uses token to access user data

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Client ID** | Public identifier for your application |
| **Client Secret** | Private key for server-side token exchange |
| **Redirect URI** | URL where provider sends user after authentication |
| **Scope** | Permissions requested (e.g., `email`, `profile`) |
| **State** | Random value to prevent CSRF attacks |
| **Access Token** | Short-lived token for API access (typically 1 hour) |
| **Refresh Token** | Long-lived token to obtain new access tokens |

### Grant Types

| Grant Type | Use Case |
|------------|----------|
| **Authorization Code** | Web apps with server-side code (most secure) |
| **Authorization Code + PKCE** | Mobile/SPA apps without client secret |
| **Client Credentials** | Server-to-server (no user context) |
| **Implicit** | Deprecated; don't use for new apps |

### WAFFLE Implementation

```go
googleAuth, err := oauth2.Google(oauth2.GoogleConfig{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURL:  "https://yourapp.com/auth/google/callback",
    Scopes:       []string{"openid", "profile", "email"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
}, logger)
```

### Providers Using OAuth 2.0

GitHub, Discord, Canvas, Blackboard, Moodle, PowerSchool, Infinite Campus, Skyward, Clever, GG4L, Ed-Fi, Banner, Workday

---

## OpenID Connect (OIDC)

**OpenID Connect** is an identity layer built on top of OAuth 2.0. While OAuth 2.0 handles *authorization* (what can you access?), OIDC adds *authentication* (who are you?).

### How It Differs from OAuth 2.0

| Feature | OAuth 2.0 | OIDC |
|---------|-----------|------|
| Purpose | Authorization | Authentication + Authorization |
| User Identity | Requires additional API call | Included in ID token |
| Token Format | Opaque access token | JWT ID token + access token |
| Standard Claims | None | `sub`, `email`, `name`, etc. |
| Discovery | Manual configuration | `.well-known/openid-configuration` |

### ID Token

OIDC introduces the **ID Token**, a JWT containing user identity claims:

```json
{
  "iss": "https://accounts.google.com",
  "sub": "110248495921238986420",
  "aud": "your-client-id",
  "exp": 1702500000,
  "iat": 1702496400,
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "picture": "https://..."
}
```

### Standard Claims

| Claim | Description |
|-------|-------------|
| `iss` | Issuer (provider URL) |
| `sub` | Subject (unique user ID) |
| `aud` | Audience (your client ID) |
| `exp` | Expiration time |
| `iat` | Issued at time |
| `email` | User's email |
| `email_verified` | Whether email is verified |
| `name` | Full name |
| `given_name` | First name |
| `family_name` | Last name |
| `picture` | Profile picture URL |

### Discovery Document

OIDC providers publish configuration at `/.well-known/openid-configuration`:

```json
{
  "issuer": "https://accounts.google.com",
  "authorization_endpoint": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_endpoint": "https://oauth2.googleapis.com/token",
  "userinfo_endpoint": "https://openidconnect.googleapis.com/v1/userinfo",
  "jwks_uri": "https://www.googleapis.com/oauth2/v3/certs"
}
```

### WAFFLE Implementation

OIDC providers work the same as OAuth 2.0 in WAFFLE, but automatically parse the ID token:

```go
// OIDC providers (Google, Microsoft, Okta, ClassLink, etc.)
// automatically extract identity from the ID token
oktaAuth, err := oauth2.Okta(oauth2.OktaConfig{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURL:  "https://yourapp.com/auth/okta/callback",
    Domain:       "dev-123456.okta.com",
    Scopes:       []string{"openid", "profile", "email", "groups"},
}, logger)
```

### Providers Using OIDC

Google, Microsoft, Microsoft Education, Okta, ClassLink, LinkedIn, Apple

---

## SAML 2.0

**Security Assertion Markup Language (SAML)** is an XML-based protocol for exchanging authentication data between an Identity Provider (IdP) and a Service Provider (SP). It's the standard for enterprise and higher education federation.

### How It Works

```
┌──────────────┐                              ┌──────────────┐
│   Service    │  1. SAML Request (redirect)  │   Identity   │
│   Provider   │ ────────────────────────────▶│   Provider   │
│   (Your App) │                              │  (Shibboleth,│
│              │  4. SAML Response (POST)     │   ADFS)      │
│              │ ◀────────────────────────────│              │
└──────────────┘                              └──────────────┘
                                                    │
                                              2. User Login
                                                    │
                                                    ▼
                                              ┌──────────┐
                                              │   User   │
                                              └──────────┘
                                                    │
                                              3. Assertion
                                                    │
                                                    ▼
```

1. **SP-Initiated SSO**: User visits your app, redirected to IdP
2. **User Authentication**: User authenticates at IdP
3. **Assertion Creation**: IdP creates signed SAML assertion
4. **Assertion Delivery**: Browser POSTs assertion to your ACS URL

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Identity Provider (IdP)** | Authenticates users (e.g., Shibboleth, ADFS) |
| **Service Provider (SP)** | Your application |
| **Entity ID** | Unique identifier for IdP or SP |
| **Assertion** | XML document containing authentication statement |
| **ACS URL** | Assertion Consumer Service; where IdP sends responses |
| **Metadata** | XML document describing IdP or SP configuration |
| **Signing Certificate** | X.509 certificate for signing assertions |

### SAML Assertion Structure

```xml
<saml:Assertion>
  <saml:Issuer>https://idp.university.edu</saml:Issuer>
  <saml:Subject>
    <saml:NameID>jsmith@university.edu</saml:NameID>
  </saml:Subject>
  <saml:Conditions>
    <saml:AudienceRestriction>
      <saml:Audience>https://yourapp.com</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>
  <saml:AttributeStatement>
    <saml:Attribute Name="eduPersonPrincipalName">
      <saml:AttributeValue>jsmith@university.edu</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="eduPersonAffiliation">
      <saml:AttributeValue>faculty</saml:AttributeValue>
      <saml:AttributeValue>staff</saml:AttributeValue>
    </saml:Attribute>
  </saml:AttributeStatement>
</saml:Assertion>
```

### Federation

SAML enables **federated identity** where multiple organizations trust a common set of IdPs:

| Federation | Region | Use Case |
|------------|--------|----------|
| **InCommon** | US | Higher education and research |
| **eduGAIN** | Global | International research collaboration |
| **UK Federation** | UK | UK higher education |
| **AAF** | Australia | Australian research and education |

### WAFFLE Implementation

```go
shibAuth, err := oauth2.Shibboleth(oauth2.ShibbolethConfig{
    EntityID:                    "https://yourapp.com/shibboleth",
    IdPMetadataURL:              "https://idp.university.edu/metadata",
    Certificate:                 spCertPEM,
    PrivateKey:                  spKeyPEM,
    AssertionConsumerServiceURL: "https://yourapp.com/auth/saml/acs",
    SessionStore:                sessionStore,
}, logger)

// Routes
r.Get("/auth/saml/login", shibAuth.LoginHandler())
r.Post("/auth/saml/acs", shibAuth.CallbackHandler())  // Note: POST
r.Get("/auth/saml/metadata", shibAuth.MetadataHandler())
```

### Providers Using SAML

Shibboleth (InCommon, eduGAIN, institutional IdPs)

---

## OAuth 1.0a

**OAuth 1.0a** is the predecessor to OAuth 2.0. It uses cryptographic signatures instead of HTTPS for security. While largely superseded by OAuth 2.0, some providers still use it.

### How It Works

```
┌──────────┐   1. Request Token    ┌──────────────┐
│          │ ────────────────────▶ │              │
│          │   (signed request)    │              │
│   Your   │                       │   Provider   │
│   App    │   2. Request Token    │              │
│          │ ◀──────────────────── │              │
│          │                       │              │
│          │   3. Redirect User    │              │
│          │ ────────────────────▶ │              │
│          │                       │              │
│          │   5. Access Token     │              │
│          │ ◀──────────────────── │              │
└──────────┘   (signed exchange)   └──────────────┘
                                         │
                                   4. User Auth
                                         │
                                         ▼
                                   ┌──────────┐
                                   │   User   │
                                   └──────────┘
```

### Key Differences from OAuth 2.0

| Feature | OAuth 1.0a | OAuth 2.0 |
|---------|------------|-----------|
| Security | Signature-based | HTTPS required |
| Complexity | Complex signature generation | Simpler |
| Token Types | Request token + Access token | Access token only |
| Secret Required | Always (for signatures) | Server-side only |

### Signature Generation

OAuth 1.0a requires signing every request with:
- Consumer Secret
- Token Secret (when available)
- Request parameters
- Timestamp and nonce

WAFFLE handles this complexity internally.

### WAFFLE Implementation

```go
schoologyAuth, err := oauth2.Schoology(oauth2.SchoologyConfig{
    ConsumerKey:    "your-consumer-key",
    ConsumerSecret: "your-consumer-secret",
    RedirectURL:    "https://yourapp.com/auth/schoology/callback",
    Domain:         "school.schoology.com",
    SessionStore:   sessionStore,
    StateStore:     stateStore,
}, logger)

// Note: Access token secret stored in User.Extra for API calls
user := oauth2.UserFromContext(r.Context())
tokenSecret := user.Extra["access_token_secret"]
```

### Providers Using OAuth 1.0a

Schoology

---

## LTI 1.3

**Learning Tools Interoperability (LTI)** is an IMS Global standard for integrating learning tools with Learning Management Systems. LTI 1.3 uses OIDC for authentication but with a **platform-initiated** flow.

### How It Differs

| Feature | Standard OIDC | LTI 1.3 |
|---------|---------------|---------|
| Initiation | App redirects to provider | Platform redirects to tool |
| Purpose | User authentication | Tool launch with context |
| Context | User identity only | User + course + assignment |
| Services | API access | NRPS, AGS (grades), Deep Linking |

### How It Works

```
┌──────────────┐                              ┌──────────────┐
│     LMS      │  1. Login Initiation         │    Your      │
│   Platform   │ ────────────────────────────▶│    Tool      │
│  (Canvas,    │                              │              │
│   Moodle)    │  2. Auth Request             │              │
│              │ ◀────────────────────────────│              │
│              │                              │              │
│              │  3. ID Token (POST)          │              │
│              │ ────────────────────────────▶│              │
└──────────────┘                              └──────────────┘
      │
      │ User clicks
      │ "Launch Tool"
      ▼
┌──────────────┐
│    User      │
│  (Student/   │
│  Instructor) │
└──────────────┘
```

1. **Login Initiation**: LMS sends user to your login endpoint
2. **Auth Request**: Your tool redirects to LMS authorization endpoint
3. **ID Token**: LMS POSTs signed JWT with launch data to your tool

### LTI Launch Data

The ID token contains rich context about the launch:

```json
{
  "iss": "https://canvas.instructure.com",
  "sub": "user-123",
  "aud": "your-client-id",

  "https://purl.imsglobal.org/spec/lti/claim/message_type": "LtiResourceLinkRequest",
  "https://purl.imsglobal.org/spec/lti/claim/version": "1.3.0",

  "https://purl.imsglobal.org/spec/lti/claim/context": {
    "id": "course-456",
    "title": "Introduction to Computer Science",
    "type": ["CourseSection"]
  },

  "https://purl.imsglobal.org/spec/lti/claim/roles": [
    "http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor"
  ],

  "https://purl.imsglobal.org/spec/lti/claim/resource_link": {
    "id": "resource-789",
    "title": "Week 1 Assignment"
  }
}
```

### LTI Services

| Service | Purpose |
|---------|---------|
| **NRPS** | Names and Role Provisioning - get class roster |
| **AGS** | Assignment and Grade Services - submit grades |
| **Deep Linking** | Let instructors select content to embed |

### WAFFLE Implementation

```go
ltiProvider, err := oauth2.LTI(oauth2.LTIConfig{
    RedirectURL: "https://yourapp.com/lti/launch",
    Platforms: []oauth2.LTIPlatformConfig{
        {
            Issuer:       "https://canvas.instructure.com",
            ClientID:     "10000000000001",
            AuthURL:      "https://canvas.instructure.com/api/lti/authorize_redirect",
            TokenURL:     "https://canvas.instructure.com/login/oauth2/token",
            JWKSURL:      "https://canvas.instructure.com/api/lti/security/jwks",
            DeploymentID: "1:abc123",
        },
    },
    PrivateKey:   privateKey,  // For signing JWTs
    PublicKey:    publicKey,
    KeyID:        "key-1",
    SessionStore: sessionStore,
    NonceStore:   nonceStore,  // Prevent replay attacks
    OnSuccess: func(w http.ResponseWriter, r *http.Request, launch *oauth2.LTILaunch) {
        // launch contains User, Context, ResourceLink, Services
    },
}, logger)

// LTI routes (note: platform initiates, so login accepts GET and POST)
r.Get("/lti/login", ltiProvider.LoginHandler())
r.Post("/lti/login", ltiProvider.LoginHandler())
r.Post("/lti/launch", ltiProvider.LaunchHandler())
r.Get("/lti/jwks", ltiProvider.JWKSHandler())
```

### Providers Using LTI 1.3

LTI 1.3 (integrates with Canvas, Blackboard, Moodle, Schoology, Brightspace, and other LMS platforms)

---

## Choosing a Protocol

| If you need... | Use |
|----------------|-----|
| Simple social login | OAuth 2.0 |
| Verified user identity | OIDC |
| Enterprise/higher ed federation | SAML |
| Integration with legacy system | OAuth 1.0a (if required) |
| LMS tool integration | LTI 1.3 |

---

## See Also

- [Authentication Providers Index](./README.md) — All available providers
- [Core Concepts](./core-concepts.md) — Session stores, middleware, patterns
