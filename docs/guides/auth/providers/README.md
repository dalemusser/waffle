# Authentication Providers

This directory contains documentation for authentication providers supported by WAFFLE.

**Note**: While most providers use OAuth2/OIDC, this package also supports:
- **SAML 2.0**: Shibboleth for federated identity
- **OAuth 1.0a**: Schoology
- **LTI 1.3**: Learning Tools Interoperability (OIDC-based, platform-initiated)

---

## Getting Started

| Document | Description |
|----------|-------------|
| [**Core Concepts**](./core-concepts.md) | Session stores, middleware, common patterns |
| [**Protocols**](./protocols.md) | Understanding OAuth2, OIDC, SAML, OAuth 1.0a, LTI 1.3 |

---

## K-12 SSO Providers

| Provider | Description | Protocol |
|----------|-------------|----------|
| [Clever](./clever.md) | Leading K-12 SSO with student/teacher/admin roles | OAuth2 |
| [ClassLink](./classlink.md) | K-12 SSO with role-based access | OIDC |
| [GG4L](./gg4l.md) | K-12 SSO with OneRoster rostering | OAuth2 |

## Student Information Systems (SIS)

| Provider | Description | Protocol |
|----------|-------------|----------|
| [PowerSchool](./powerschool.md) | Major K-12 SIS with student/teacher/parent roles | OAuth2 |
| [Infinite Campus](./infinite-campus.md) | K-12 SIS with comprehensive student data | OAuth2 |
| [Skyward](./skyward.md) | K-12 SIS with student/staff/family roles | OAuth2 |
| [Ellucian Banner](./banner.md) | Higher ed ERP/SIS for colleges and universities | OAuth2 |
| [Workday Student](./workday.md) | Cloud-based student system for higher ed | OAuth2 |

## Learning Management Systems (LMS)

| Provider | Description | Protocol |
|----------|-------------|----------|
| [Canvas](./canvas.md) | Popular LMS with courses/grades/assignments | OAuth2 |
| [Blackboard](./blackboard.md) | LMS for higher ed and K-12 | OAuth2 |
| [Moodle](./moodle.md) | Open-source LMS with role-based access | OAuth2 |
| [Schoology](./schoology.md) | LMS with courses/grades/assignments | OAuth 1.0a |
| [Google Classroom](./google-classroom.md) | Google's classroom platform | OAuth2 |

## Data Standards & Interoperability

| Provider | Description | Protocol |
|----------|-------------|----------|
| [Ed-Fi](./edfi.md) | K-12 data standard with ODS API | OAuth2 |
| [LTI 1.3](./lti.md) | Learning Tools Interoperability standard | OIDC (platform-initiated) |

## Enterprise & Federated Identity

| Provider | Description | Protocol |
|----------|-------------|----------|
| [Shibboleth/SAML](./shibboleth.md) | Federated identity for higher education | SAML 2.0 |
| [Microsoft](./microsoft.md) | Azure AD, Microsoft 365, personal accounts | OAuth2/OIDC |
| [Microsoft Education](./microsoft-education.md) | Microsoft 365 Education with classes | OAuth2/OIDC |
| [Okta](./okta.md) | Enterprise SSO with group-based authorization | OAuth2/OIDC |

## Social Providers

| Provider | Description | Protocol |
|----------|-------------|----------|
| [Google](./google.md) | Gmail, Google Workspace | OAuth2/OIDC |
| [GitHub](./github.md) | Developer-focused authentication | OAuth2 |
| [Discord](./discord.md) | Gaming/community with guild/role support | OAuth2 |
| [LinkedIn](./linkedin.md) | Professional network authentication | OIDC |
| [Apple](./apple.md) | Sign in with Apple | OAuth2/OIDC |

---

## Package Import

```go
import "github.com/dalemusser/waffle/pantry/auth/oauth2"
```

---

[← Back to Examples](../README.md)
