# LTI Authentication

*Learning Tools Interoperability for embedding in LMS platforms.*

LTI (Learning Tools Interoperability) is an IMS Global standard for integrating learning tools with Learning Management Systems. LTI 1.3 uses OpenID Connect (OIDC) for authentication.

**Note**: LTI is not a standard OAuth2 provider—it uses a platform-initiated OIDC flow.

---

## 1. Register Your Tool

For each LMS platform (Canvas, Blackboard, Moodle, etc.):

1. Register your tool in the LMS admin panel
2. Configure your tool's endpoints:
   - **Login Initiation URL**: `https://yourapp.com/lti/login`
   - **Redirect URI**: `https://yourapp.com/lti/launch`
   - **JWKS URL**: `https://yourapp.com/lti/jwks`
   - **Deep Link URL**: `https://yourapp.com/lti/deeplink` (optional)
3. Note the platform's credentials:
   - **Issuer** (platform identifier)
   - **Client ID**
   - **Platform JWKS URL**
   - **Authorization endpoint**
   - **Token endpoint**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    LTIRedirectURL string `conf:"lti_redirect_url"`

    // RSA key pair for JWT signing
    LTIPrivateKeyPath string `conf:"lti_private_key_path"`
    LTIPublicKeyPath  string `conf:"lti_public_key_path"`
    LTIKeyID          string `conf:"lti_key_id"`

    // Platform configurations loaded from database or config
    LTIPlatforms []oauth2.LTIPlatformConfig
}
```

## 3. Wire Up in BuildHandler

```go
// Load RSA keys
privateKey, _ := loadPrivateKey(appCfg.LTIPrivateKeyPath)
publicKey, _ := loadPublicKey(appCfg.LTIPublicKeyPath)

// Create LTI provider
ltiProvider, err := oauth2.LTI(oauth2.LTIConfig{
    RedirectURL:   appCfg.LTIRedirectURL,
    Platforms:     appCfg.LTIPlatforms,
    PrivateKey:    privateKey,
    PublicKey:     publicKey,
    KeyID:         appCfg.LTIKeyID,
    SessionStore:  sessionStore,
    NonceStore:    nonceStore,  // For replay attack prevention
    OnSuccess: func(w http.ResponseWriter, r *http.Request, launch *oauth2.LTILaunch) {
        // Handle successful LTI launch
        if oauth2.IsLTIInstructor(launch.User) {
            http.Redirect(w, r, "/instructor/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        }
    },
    OnDeepLink: func(w http.ResponseWriter, r *http.Request, launch *oauth2.LTILaunch) {
        // Handle deep linking request - return content selection UI
        http.Redirect(w, r, "/lti/content-picker", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// LTI routes
r.Get("/lti/login", ltiProvider.LoginHandler())
r.Post("/lti/login", ltiProvider.LoginHandler())
r.Post("/lti/launch", ltiProvider.LaunchHandler())
r.Get("/lti/jwks", ltiProvider.JWKSHandler())
r.Post("/lti/deeplink", ltiProvider.DeepLinkHandler())
```

## Platform Configuration

```go
platformConfig := oauth2.LTIPlatformConfig{
    Issuer:       "https://canvas.instructure.com",
    ClientID:     "10000000000001",
    AuthURL:      "https://canvas.instructure.com/api/lti/authorize_redirect",
    TokenURL:     "https://canvas.instructure.com/login/oauth2/token",
    JWKSURL:      "https://canvas.instructure.com/api/lti/security/jwks",
    DeploymentID: "1:abc123...",
}
```

## User Information

The `User.Extra` map contains LTI-specific fields:

| Field | Description |
|-------|-------------|
| `lti_sub` | LTI subject (user ID in platform) |
| `lti_deployment_id` | Deployment ID |
| `lti_context_id` | Context (course) ID |
| `lti_context_title` | Context (course) title |
| `lti_context_label` | Context short name |
| `lti_resource_link_id` | Resource link ID |
| `lti_resource_link_title` | Resource link title |
| `lti_roles` | Comma-separated LTI roles |
| `lti_platform` | Platform product family |
| `lti_version` | LTI version |

## LTI Launch Object

The `OnSuccess` callback receives an `LTILaunch` struct:

```go
type LTILaunch struct {
    User           *User
    MessageType    string          // "LtiResourceLinkRequest" or "LtiDeepLinkingRequest"
    Version        string          // "1.3.0"
    DeploymentID   string
    TargetLinkURI  string
    Context        LTIContext
    ResourceLink   LTIResourceLink
    LaunchPresentation LTILaunchPresentation
    Custom         map[string]string
    Services       LTIServices     // NRPS, AGS endpoints
}
```

## Helper Functions

```go
// Check user roles
if oauth2.IsLTIInstructor(user) { ... }
if oauth2.IsLTILearner(user) { ... }
if oauth2.IsLTIAdministrator(user) { ... }
if oauth2.IsLTITA(user) { ... }
if oauth2.IsLTIContentDeveloper(user) { ... }

// Check for specific role
if oauth2.HasLTIRole(user, "http://purl.imsglobal.org/vocab/lis/v2/institution/person#Instructor") { ... }

// Get context info
contextID := oauth2.GetLTIContextID(user)
contextTitle := oauth2.GetLTIContextTitle(user)

// Check service availability
if oauth2.SupportsNRPS(launch) { ... }  // Names and Role Provisioning Service
if oauth2.SupportsAGS(launch) { ... }   // Assignment and Grade Services

// Check launch type
if oauth2.IsDeepLinkingRequest(launch) { ... }
```

## LTI Services

### Names and Role Provisioning Service (NRPS)

```go
// Get class roster from LMS
if oauth2.SupportsNRPS(launch) {
    nrpsClient := oauth2.NewNRPSClient(ltiProvider, launch)
    members, err := nrpsClient.GetMembers(r.Context())
    // Returns list of users with roles in the course
}
```

### Assignment and Grade Services (AGS)

```go
// Submit grades back to LMS
if oauth2.SupportsAGS(launch) {
    agsClient := oauth2.NewAGSClient(ltiProvider, launch)

    // Create line item (grade column)
    lineItem := oauth2.AGSLineItem{
        Label:        "Assignment 1",
        ScoreMaximum: 100,
    }
    lineItemURL, err := agsClient.CreateLineItem(r.Context(), lineItem)

    // Submit score
    score := oauth2.AGSScore{
        UserID:       studentLTIID,
        ScoreGiven:   85,
        ScoreMaximum: 100,
        Comment:      "Good work!",
    }
    err = agsClient.SubmitScore(r.Context(), lineItemURL, score)
}
```

## Deep Linking

Return content items to the LMS:

```go
func (h *Handler) SubmitDeepLink(w http.ResponseWriter, r *http.Request) {
    launch := oauth2.GetLTILaunch(r.Context())

    // Create content items to return
    items := []oauth2.LTIContentItem{
        {
            Type:  "ltiResourceLink",
            Title: "My Learning Activity",
            URL:   "https://yourapp.com/activity/123",
            Custom: map[string]string{
                "activity_id": "123",
            },
        },
    }

    // Generate and return deep link response JWT
    responseJWT, err := ltiProvider.CreateDeepLinkResponse(launch, items)
    // Return form that auto-submits to LMS
}
```

## Important Notes

- LTI 1.3 is platform-initiated (LMS redirects to your tool)
- Each platform requires separate registration and configuration
- Your tool needs RSA keys for signing JWTs
- NRPS and AGS are optional services—check availability before use
- Deep linking allows instructors to add your content to their courses
- Use NonceStore to prevent replay attacks
- LTI roles use URIs from IMS vocabulary

---

[← Back to OAuth2 Providers](./README.md)
