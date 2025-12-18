# Shibboleth/SAML Authentication

*Federated identity for higher education (InCommon, eduGAIN).*

Shibboleth is a SAML-based federated identity system widely used in higher education and research institutions. It enables Single Sign-On (SSO) across organizations through identity federations like InCommon (US) and eduGAIN (international).

**Note**: SAML is not OAuth2, but WAFFLE provides a unified interface for SAML authentication.

---

## 1. Register Your Service Provider

### For InCommon Federation:

1. Register your institution with [InCommon](https://www.incommon.org/)
2. Configure your Service Provider (SP) metadata
3. Register in the InCommon Federation Manager
4. Exchange metadata with your institution's Identity Provider (IdP)

### For Institution-Specific Setup:

1. Contact your institution's **Identity Management team**
2. Request SP registration
3. Provide your SP metadata URL: `https://yourapp.com/auth/saml/metadata`
4. Receive IdP metadata or configuration

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    // Service Provider settings
    SAMLEntityID        string `conf:"saml_entity_id"`        // Your SP entity ID
    SAMLCertPath        string `conf:"saml_cert_path"`        // Path to SP certificate
    SAMLKeyPath         string `conf:"saml_key_path"`         // Path to SP private key
    SAMLACSUrl          string `conf:"saml_acs_url"`          // Assertion Consumer Service URL

    // Identity Provider settings
    SAMLIdPMetadataURL  string `conf:"saml_idp_metadata_url"` // IdP metadata URL
    SAMLIdPEntityID     string `conf:"saml_idp_entity_id"`    // IdP entity ID (optional)
}
```

## 3. Wire Up in BuildHandler

```go
// Load SP certificate and key
spCert, _ := loadCertificate(appCfg.SAMLCertPath)
spKey, _ := loadPrivateKey(appCfg.SAMLKeyPath)

// Create Shibboleth/SAML provider
samlAuth, err := oauth2.Shibboleth(oauth2.ShibbolethConfig{
    EntityID:         appCfg.SAMLEntityID,
    ACSUrl:           appCfg.SAMLACSUrl,
    Certificate:      spCert,
    PrivateKey:       spKey,
    IdPMetadataURL:   appCfg.SAMLIdPMetadataURL,
    IdPEntityID:      appCfg.SAMLIdPEntityID,
    SessionStore:     sessionStore,
    AllowIDPInitiated: false,  // Security: prefer SP-initiated
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        if oauth2.IsShibbolethFaculty(user) {
            http.Redirect(w, r, "/faculty/dashboard", http.StatusTemporaryRedirect)
        } else if oauth2.IsShibbolethStudent(user) {
            http.Redirect(w, r, "/student/dashboard", http.StatusTemporaryRedirect)
        } else {
            http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
        }
    },
}, logger)
if err != nil {
    return nil, err
}

// SAML routes
r.Get("/auth/saml/login", samlAuth.LoginHandler())
r.Post("/auth/saml/acs", samlAuth.ACSHandler())  // Assertion Consumer Service
r.Get("/auth/saml/metadata", samlAuth.MetadataHandler())
r.Get("/auth/saml/logout", samlAuth.LogoutHandler())
r.Post("/auth/saml/slo", samlAuth.SLOHandler())  // Single Logout
```

## User Information

The `User.Extra` map contains SAML-specific fields:

| Field | Description |
|-------|-------------|
| `subject_id` | Subject identifier |
| `persistent_id` | Persistent (targeted) ID |
| `transient_id` | Transient ID |
| `eppn` | eduPersonPrincipalName |
| `affiliation` | eduPersonAffiliation |
| `scoped_affiliation` | eduPersonScopedAffiliation |
| `entitlement` | eduPersonEntitlement |
| `primary_affiliation` | eduPersonPrimaryAffiliation |
| `org_unit_dn` | eduPersonOrgUnitDN |
| `display_name` | displayName |
| `given_name` | givenName |
| `sn` | surname |
| `mail` | email address |
| `idp` | Identity Provider entity ID |

## eduPerson Attributes

Common eduPerson schema attributes:

| Attribute | Description |
|-----------|-------------|
| `eduPersonPrincipalName (eppn)` | Unique identifier (user@institution.edu) |
| `eduPersonAffiliation` | Relationship to institution (student, faculty, staff, etc.) |
| `eduPersonScopedAffiliation` | Affiliation with scope (student@institution.edu) |
| `eduPersonEntitlement` | Authorization entitlements |
| `eduPersonTargetedID` | Persistent pseudonymous identifier |

## Helper Functions

```go
// Check user affiliation
if oauth2.IsShibbolethStudent(user) { ... }
if oauth2.IsShibbolethFaculty(user) { ... }
if oauth2.IsShibbolethStaff(user) { ... }
if oauth2.IsShibbolethEmployee(user) { ... }  // faculty or staff
if oauth2.IsShibbolethMember(user) { ... }
if oauth2.IsShibbolethAffiliate(user) { ... }
if oauth2.IsShibbolethAlum(user) { ... }

// Get attributes
eppn := oauth2.GetShibbolethEPPN(user)
affiliations := oauth2.GetShibbolethAffiliations(user)  // []string
scopedAffiliations := oauth2.GetShibbolethScopedAffiliations(user)
entitlements := oauth2.GetShibbolethEntitlements(user)
idp := oauth2.GetShibbolethIdP(user)

// Check for specific entitlement
if oauth2.HasShibbolethEntitlement(user, "urn:mace:example.edu:entitlement:library") { ... }

// Check for specific affiliation
if oauth2.HasShibbolethAffiliation(user, "faculty") { ... }
```

## Multi-IdP Support

For supporting multiple Identity Providers:

```go
// Discovery Service (WAYF - Where Are You From)
r.Get("/auth/saml/discovery", samlAuth.DiscoveryHandler())

// Or handle IdP selection yourself
r.Get("/auth/saml/login", func(w http.ResponseWriter, r *http.Request) {
    idpEntityID := r.URL.Query().Get("idp")
    if idpEntityID == "" {
        // Show IdP selection page
        renderIdPSelector(w, r)
        return
    }
    samlAuth.LoginHandlerWithIdP(idpEntityID).ServeHTTP(w, r)
})
```

## Federation Support

```go
// InCommon metadata aggregator
samlAuth, err := oauth2.Shibboleth(oauth2.ShibbolethConfig{
    // ... other config ...
    IdPMetadataURL: "https://md.incommon.org/InCommon/InCommon-metadata.xml",
    // Or for eduGAIN
    // IdPMetadataURL: "https://mds.edugain.org/edugain-v2.xml",
})
```

## SP Metadata

Your application automatically generates SP metadata:

```xml
<!-- Available at /auth/saml/metadata -->
<EntityDescriptor entityID="https://yourapp.com/saml">
  <SPSSODescriptor>
    <KeyDescriptor use="signing">...</KeyDescriptor>
    <AssertionConsumerService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://yourapp.com/auth/saml/acs"/>
    <SingleLogoutService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://yourapp.com/auth/saml/slo"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

## Important Notes

- SAML uses XML signatures; keep your SP keys secure
- IdP-initiated SSO can be a security risk; prefer SP-initiated
- eduPerson attributes may vary by institution
- Federation metadata should be refreshed periodically
- Single Logout (SLO) is optional but recommended
- Target/persistent IDs are privacy-preserving identifiers
- InCommon R&S (Research & Scholarship) category provides attribute release

---

[← Back to OAuth2 Providers](./README.md)
