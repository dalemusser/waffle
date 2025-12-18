# Authentication Provider Registration - Preparation Checklist

*Master list of items needed to register with K-12 authentication providers.*

---

## Overview

This document consolidates everything you need to prepare before registering your application with K-12 authentication providers (Google, Clever, ClassLink, Microsoft Education, and Schoology). Having these items ready will streamline the registration process across all providers.

**Target Audience**: Project directors and administrative staff responsible for gathering approvals and information before vendor registration begins.

---

## Quick Reference: What's Needed Where

| Item | Google | Clever | ClassLink | Microsoft | Schoology |
|------|:------:|:------:|:---------:|:---------:|:---------:|
| Organization Name | ✓ | ✓ | ✓ | ✓ | ✓ |
| Business Email | ✓ | ✓ | ✓ | ✓ | ✓ |
| Application Name | ✓ | ✓ | ✓ | ✓ | ✓ |
| Application Description | | ✓ | ✓ | | ✓ |
| Website URL | ✓ | ✓ | ✓ | | |
| Privacy Policy URL | ✓ | ✓ | ✓ | ✓ | ✓ |
| Terms of Service URL | ✓ | | | ✓ | |
| Support Email | ✓ | | | ✓ | |
| Application Logo | ✓ | ✓ | ✓ | ✓ | ✓ |
| Redirect URL (from tech team) | ✓ | ✓ | ✓ | ✓ | ✓ |
| Domain Ownership/Verification | ✓ | | | | |

---

## Items Requiring Director Approval

These items may need institutional review, legal approval, or administrative sign-off.

### 1. Privacy Policy

**What it is**: A public document explaining how you collect, use, and protect user data.

**Why it's needed**: All providers require a privacy policy URL. Schools and parents need to know how student data is handled.

**Requirements**:
- Must be publicly accessible via URL (e.g., `https://yourapp.example.com/privacy`)
- Should address:
  - What data you collect (for auth-only: name, email, role)
  - How data is used (authentication/verification only)
  - How data is protected
  - Data retention and deletion policies
  - COPPA/FERPA compliance (for K-12)
  - Contact information for privacy questions

**Action needed**:
- [ ] Draft privacy policy document
- [ ] Get legal/institutional review if required
- [ ] Publish to publicly accessible URL

---

### 2. Terms of Service / Terms of Use

**What it is**: Legal agreement between your application and its users.

**Why it's needed**: Google and Microsoft specifically ask for this. Professional appearance for all providers.

**Requirements**:
- Must be publicly accessible via URL (e.g., `https://yourapp.example.com/terms`)
- Should address:
  - Acceptable use of the application
  - User responsibilities
  - Limitation of liability
  - Governing law/jurisdiction

**Action needed**:
- [ ] Draft terms of service document
- [ ] Get legal/institutional review if required
- [ ] Publish to publicly accessible URL

---

### 3. Data Privacy Statement for Schools

**What it is**: A clear statement explaining your data practices, specifically for K-12 contexts.

**Why it's needed**: School IT administrators will ask what data you access before approving your app.

**Suggested language for authentication-only apps**:
> "We use [Provider] for authentication only. We collect the user's name, email address, and role (student/teacher) solely to verify their identity. We do not access, store, or process student roster data, grades, assignments, or any other educational records."

**Action needed**:
- [ ] Prepare data privacy statement
- [ ] Confirm statement accuracy with technical team

---

## Organization Information

### 4. Official Organization Name

**What it is**: The legal or doing-business-as name of your organization.

**Examples**:
- "Acme Learning Technologies"
- "State University - College of Education"
- "Riverside School District"

**Used by**: All providers

**Decision needed**: Choose which organizational name to use consistently across all providers.

**Action needed**:
- [ ] Confirm official organization name to use

---

### 5. Organization Website URL

**What it is**: Your organization's public website.

**Examples**: `https://yourcompany.com` or `https://education.university.edu`

**Used by**: Google, Clever, ClassLink

**Requirements**:
- Must be a real, accessible website
- Should represent your organization professionally

**Action needed**:
- [ ] Confirm website URL
- [ ] Ensure website is live and professional

---

### 6. Business Email Address

**What it is**: An email address on your organization's domain (not personal Gmail/Outlook).

**Examples**: `admin@yourcompany.com` or `developer@yourorganization.edu` (not a personal Gmail)

**Used by**: All providers (Clever specifically requires non-personal email)

**Requirements**:
- Should be on your organization's domain
- Should be monitored regularly (approval emails will come here)
- Consider using a role-based address that multiple people can access

**Action needed**:
- [ ] Designate business email for registrations
- [ ] Ensure email is monitored

---

### 7. Support/Contact Email

**What it is**: Email where users can reach you with questions or issues.

**Examples**: `support@yourcompany.com` or `help@yourcompany.com`

**Used by**: Google, Microsoft

**Can be**: Same as business email, or a dedicated support address

**Action needed**:
- [ ] Designate support email address

---

## Application Information

### 8. Application Name

**What it is**: The user-facing name of your application.

**Examples**: "My Learning Portal" or "Classroom Helper"

**Used by**: All providers

**Requirements**:
- This is what students and teachers see when logging in
- Should be clear and recognizable
- Can usually be changed later, but consistency helps

**Action needed**:
- [ ] Decide on official application name

---

### 9. Application Description

**What it is**: 1-3 sentences explaining what your application does.

**Example**:
> "An educational platform that helps K-12 students practice math skills through interactive exercises and real-time feedback."

**Used by**: Clever, ClassLink, Schoology

**Action needed**:
- [ ] Write application description

---

### 10. Application Logo

**What it is**: A square image representing your application.

**Used by**: All providers (displayed on login screens and app directories)

**Requirements**:
- Square format recommended (e.g., 120x120 pixels minimum)
- PNG or JPG format
- Professional appearance
- Recognizable at small sizes

**Action needed**:
- [ ] Create or obtain application logo
- [ ] Prepare in appropriate format/size

---

## Technical Information

Get the following from your technical team before starting registration.

### 11. Redirect URL(s)

**What it is**: The URL where users are sent after authenticating.

**Production Redirect URL pattern**:

| Provider | Redirect URL Pattern |
|----------|----------------------|
| **Google** | `https://yourapp.example.com/auth/google/callback` |
| **Clever** | `https://yourapp.example.com/auth/clever/callback` |
| **ClassLink** | `https://yourapp.example.com/auth/classlink/callback` |
| **Microsoft Education** | `https://yourapp.example.com/auth/microsoft/callback` |
| **Schoology** | `https://yourapp.example.com/auth/schoology/callback` |

**Used by**: All providers — enter the appropriate URL when configuring each provider

**Development URLs** (if needed for testing):

| Provider | Development Redirect URL |
|----------|--------------------------|
| Google | `http://localhost:8080/auth/google/callback` |
| Clever | `http://localhost:8080/auth/clever/callback` |
| ClassLink | `http://localhost:8080/auth/classlink/callback` |
| Microsoft | `http://localhost:8080/auth/microsoft/callback` |
| Schoology | `http://localhost:8080/auth/schoology/callback` |

**Note**: Some providers (Google, Microsoft) allow adding multiple redirect URIs, so you can add both production and development URLs.

**Action needed**:
- [ ] Get production redirect URLs from technical team
- [ ] Get development redirect URLs if needed

---

### 12. Application Domain

**What it is**: The domain name(s) where your application runs.

**Example**: `yourapp.example.com`

**Used by**: Google, ClassLink

**Action needed**:
- [ ] Confirm application domain with technical team

---

## Domain Verification (Google Only)

### 13. Domain Ownership Verification

**What it is**: Proof that your organization owns the domain used in the application.

**Why it's needed**: Google requires domain verification for published apps.

**How it works**: Usually done by:
- Adding a DNS TXT record, OR
- Uploading a verification file to your website

**Who does this**: Your IT/web team

**Action needed**:
- [ ] Identify who can modify DNS records or upload files to your domain
- [ ] Be prepared to coordinate with them during Google setup

---

## Account Access

### 14. Account Credentials

**What it is**: Login credentials for the various provider portals.

**You'll need accounts for**:
| Provider | Portal | Account Type |
|----------|--------|--------------|
| Google | Google Cloud Console | Any Google account |
| Clever | apps.clever.com | Created during registration |
| ClassLink | partnerportal.classlink.com | Created during registration |
| Microsoft | entra.microsoft.com | Any Microsoft account |
| Schoology | app.schoology.com | Developer account (requires approval) |

**Recommendation**: Use your organization's business email when creating these accounts, or create a shared account that multiple team members can access.

**Action needed**:
- [ ] Decide who will own/manage these accounts
- [ ] Use organization email for registration
- [ ] Document credentials securely

---

## Summary Checklist

### Must Have Before Starting

- [ ] **Organization name** — Official name to use across all providers
- [ ] **Business email** — On your domain (not personal email)
- [ ] **Application name** — User-facing name for your app
- [ ] **Privacy policy** — Published at a public URL
- [ ] **Website URL** — Your organization or application website
- [ ] **Redirect URLs** — Get from technical team

### Should Have (Highly Recommended)

- [ ] **Terms of service** — Published at a public URL
- [ ] **Application logo** — Square format, for login screens
- [ ] **Application description** — 1-3 sentences about your app
- [ ] **Support email** — For user inquiries
- [ ] **Data privacy statement** — For school IT administrators

### May Need During Process

- [ ] **Domain verification access** — For Google: who can add DNS records or upload files?
- [ ] **Account credentials** — Created during registration with each provider

---

## Recommended Order of Operations

1. **Gather all "Must Have" items first**
2. **Register with Google** (self-service, fastest)
3. **Register with ClassLink** (self-service, no certification required)
4. **Register with Microsoft** (self-service, straightforward)
5. **Register with Clever** (self-service registration, but requires certification)
6. **Request Schoology developer access** (requires approval, longest wait)

Start Schoology early since their approval process can take 1-3 weeks.

---

## Questions for Technical Team

Before starting registration, confirm with your technical team:

1. What are the production redirect URLs for each provider?
2. What is the application domain?
3. Who can help with domain verification for Google? (DNS or file upload access)
4. What port does local development use? (typically 8080 or 3000)
5. What user information do we need from each provider? (typically: name, email, role)

---

## Document Locations

Once created, note where these documents are hosted:

| Document | URL |
|----------|-----|
| Privacy Policy | `https://yourapp.example.com/privacy` |
| Terms of Service | `https://yourapp.example.com/terms` |
| Organization Website | `https://yourcompany.com` |
| Application Logo | (file location or URL) |

---

*Last updated: December 2024*

[← Back to Onboarding Guides](./README.md)
