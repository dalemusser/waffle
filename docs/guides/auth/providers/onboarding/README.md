# Authentication Provider Onboarding Guides

*Step-by-step guides for obtaining credentials from authentication providers.*

These guides are designed for **non-technical staff** who will handle the vendor registration and onboarding process. Each guide explains what the provider is, what you need before starting, and walks through the process step-by-step.

---

## How to Use These Guides

1. **Review the Prerequisites Checklist** — Gather required information before starting
2. **Follow the Step-by-Step Process** — Complete the vendor registration
3. **Hand Off to Technical Team** — Provide the credentials and technical documentation link

---

## K-12 Authentication Providers

Ranked by priority for K-12 educational applications:

| Priority | Provider | Coverage | Guide |
|----------|----------|----------|-------|
| 1 | [**Google**](./google.md) | Chromebook schools, Google Workspace for Education | Available |
| 2 | [**Clever**](./clever.md) | Major K-12 SSO platform | Available |
| 3 | [**ClassLink**](./classlink.md) | Second major K-12 SSO platform | Available |
| 4 | [**Microsoft Education**](./microsoft-education.md) | Microsoft 365 Education schools | Available |
| 5 | [**Schoology**](./schoology.md) | LMS with SSO capabilities (OAuth 1.0a) | Available |

---

## Start Here: Preparation Checklist

**[📋 Preparation Checklist](./preparation-checklist.md)** — Master list of everything you need to gather before registering with any provider. This includes privacy policy, terms of service, application name, logo, and more. Review this first to ensure all approvals and assets are ready before starting the registration process.

---

## What Each Guide Contains

- **Overview**: What the provider is and why you need it
- **Prerequisites Checklist**: What to gather before starting
- **Step-by-Step Process**: Detailed instructions with screenshots descriptions
- **Verification Requirements**: What approval processes to expect
- **School/District Requirements**: What participating schools need to do
- **Handoff to Technical Team**: What credentials to provide developers
- **Timeline Expectations**: How long each step typically takes
- **Troubleshooting**: Common issues and solutions

---

## Before You Start Any Provider

You'll typically need these for most providers:

### Organization Information
- Organization legal name
- Organization type (company, educational institution, etc.)
- Organization website URL
- Organization address

### Application Information
- Application name (what users will see)
- Application description (1-2 sentences about what it does)
- Application website URL
- Privacy Policy URL (publicly accessible)
- Terms of Service URL (recommended)
- Support contact email

### Technical Information
Get these from your development team:
- Redirect URI(s) / Callback URL(s)
- Any specific technical requirements

### For Education Vendors
Some K-12 providers may ask for:
- Student Data Privacy Agreement or SDPC membership
- Information about your data handling practices
- Description of how you'll use student data (typically: authentication only, no student data access)

---

## General Timeline

| Phase | Typical Duration |
|-------|------------------|
| Self-service setup (Google, basic OAuth) | 1-2 hours |
| Vendor application review (Clever, ClassLink) | 1-2 weeks |
| Additional verification if required | 1-4 weeks |
| School/district approval | Varies by school |

---

## Related Documentation

- [Authentication Providers Index](../README.md) — Technical documentation for all providers
- [Authentication Protocols](../protocols.md) — Understanding OAuth2, OIDC, SAML, etc.
- [Core Concepts](../core-concepts.md) — Technical implementation patterns

---

[← Back to Auth Providers](../README.md)
