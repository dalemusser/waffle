# Schoology Authentication - Vendor Onboarding Guide

*Step-by-step guide for registering as a Schoology developer and obtaining OAuth credentials for your application.*

---

## Overview

### What is Schoology?

Schoology (now part of PowerSchool) is a Learning Management System (LMS) used by K-12 schools and districts worldwide. It provides:

- **Course Management**: Teachers create courses, assignments, and assessments
- **Student Portal**: Students access coursework, grades, and resources
- **Parent Access**: Parents view their children's progress
- **App Integration**: Third-party apps can integrate via the Schoology App Center

Schoology serves as a central hub for classroom activity and can authenticate users for third-party educational applications.

### Why You Need This

- **LMS with built-in SSO**: Schools already using Schoology can authenticate users through it
- **Widespread adoption**: Used by many K-12 districts, especially in the US
- **App Center ecosystem**: Schoology has an established platform for educational apps
- **Student data access**: Can optionally provide course/grade data (beyond basic auth)

### Important: OAuth 1.0a Protocol

**Schoology uses OAuth 1.0a**, not the more common OAuth 2.0. This is an older protocol with some differences:
- More complex signature generation
- Different authentication flow
- Your technical team should be aware of this requirement

### What You'll Get

At the end of this process, you will have:
- A **Developer Sandbox Account** for testing
- **Consumer Key** (identifies your application)
- **Consumer Secret** (private key—keep this secure)
- Access to the Schoology API and App Platform

### Important Context

Schoology was acquired by PowerSchool, so:
- Documentation is spread across `developers.schoology.com` and PowerSchool resources
- Support is handled through the PowerSchool Community
- The developer registration process may involve PowerSchool partner programs

---

## Prerequisites Checklist

Before starting, gather the following:

### Required

- [ ] **Business Email**: Your organization email address (not personal email)
- [ ] **Company/Organization Name**: Your organization's legal name
- [ ] **Application Name**: The name users will see (e.g., "My Learning Portal")
- [ ] **Application Description**: Brief description of what your app does
- [ ] **Callback URL**: The URL where Schoology sends users after authentication. Get this from your technical team—typically something like `https://yourapp.com/auth/schoology/callback`

### Recommended

- [ ] **Company Website**: Your publicly accessible website URL
- [ ] **Privacy Policy URL**: A publicly accessible page with your privacy policy
- [ ] **Application Logo**: Image for display in the Schoology App Center
- [ ] **Technical Contact**: Someone who can handle OAuth 1.0a implementation

### For K-12 Education Vendors

- [ ] **Data Privacy Information**: Be prepared to describe how you handle student data. For authentication-only: "We only use Schoology for authentication. We do not access course, grade, or assignment data."

---

## The Onboarding Process

### Part 1: Developer Registration

#### Step 1: Request Developer Access

1. Go to [Schoology Developer Request Form](https://app.schoology.com/apps/developer/request)
2. Fill out the developer application form with:
   - Your name and contact information
   - Organization name
   - Description of your intended integration
   - Type of application you're building

3. Submit the request

**What to expect**: Developer registration may take time for approval. PowerSchool reviews requests before granting sandbox access.

---

#### Step 2: Wait for Approval

After submitting your request:

1. Check your email for confirmation
2. You may receive follow-up questions from PowerSchool
3. Approval typically includes:
   - Access to a Developer Sandbox (test environment)
   - Developer account credentials
   - Access to developer documentation

**Timeline**: Approval can take several days to weeks. If you don't hear back within 2-3 weeks:
- Email: partnerprogram@powerschool.com
- Check the [PowerSchool Community](https://help.powerschool.com/) for support

---

#### Step 3: Access Your Developer Account

Once approved:

1. Log in at [Schoology](https://app.schoology.com/) with your developer credentials
2. Access the [Developer Portal](https://developers.schoology.com/)
3. Review the documentation for:
   - [Apps Platform](https://developers.schoology.com/app-platform/)
   - [API Documentation](https://developers.schoology.com/api/)
   - [Authentication Guide](https://developers.schoology.com/api-documentation/authentication/)

---

#### Step 4: Create Your Application

In your Schoology developer account:

1. Navigate to the Apps section
2. Create a new application with:
   - **Application Name**: Your app's display name
   - **Description**: What your app does
   - **Logo**: Your application icon
   - **Callback URL**: Your OAuth callback endpoint

3. Configure installation options:
   - **User realm**: App installed per-user
   - **Course realm**: App installed per-course
   - **Group realm**: App installed per-group

For authentication-only, **User realm** is typically appropriate.

---

### Part 2: OAuth Configuration

#### Step 5: Obtain Consumer Credentials

After creating your application, you'll receive:

1. **Consumer Key**: Your application identifier
   - Example: `abc123def456`

2. **Consumer Secret**: Your private key
   - Example: `xyz789secret123`
   - **Keep this secure!** Never share publicly

**Save both values securely** — you'll need them for integration.

---

#### Step 6: Understand the OAuth 1.0a Flow

Schoology uses **three-legged OAuth 1.0a**:

```
1. User clicks "Login with Schoology"
        ↓
2. Your app requests a "request token" from Schoology
        ↓
3. User is redirected to Schoology to authorize
        ↓
4. User approves, Schoology redirects back with verifier
        ↓
5. Your app exchanges for "access token"
        ↓
6. Access token used for API calls
```

**Key Endpoints**:

| Endpoint | URL |
|----------|-----|
| Request Token | `https://api.schoology.com/v1/oauth/request_token` |
| Authorization | `https://api.schoology.com/v1/oauth/authorize` |
| Access Token | `https://api.schoology.com/v1/oauth/access_token` |
| API Base | `https://api.schoology.com/v1/` |

---

#### Step 7: Test in Sandbox

Your developer sandbox provides:

1. **Test school environment** with sample data
2. **Test users** (students, teachers, admins)
3. **Safe testing** without affecting real schools

Use the sandbox to:
- Test the OAuth flow
- Verify user information retrieval
- Ensure proper error handling

---

### Part 3: App Center Submission (Optional)

If you want your app listed in the Schoology App Center:

#### Step 8: Prepare for Submission

Before submitting, ensure:
- [ ] OAuth flow works correctly
- [ ] Error handling is implemented
- [ ] Privacy policy is accessible
- [ ] App description and screenshots are ready

#### Step 9: Submit Your App

1. In the developer portal, find the submission option
2. Provide required information:
   - App details and description
   - Screenshots
   - Privacy policy URL
   - Support contact information

3. Choose visibility:
   - **Only people in my school**: Skips full review (for internal/testing use)
   - **All Schoology users**: Requires full review process

4. Submit for review

**Note**: For authentication-only use with specific schools, you may not need full App Center listing — schools can install apps directly.

---

## What Schools Need to Do

After your integration is ready, schools can connect your app.

### How School Connections Work

1. **Administrator installs**: School admin adds your app from App Center or direct link
2. **Users authorize**: On first use, students/teachers authorize your app
3. **OAuth tokens saved**: Your app stores tokens for future access
4. **Seamless access**: Users can authenticate without re-authorizing

### For Authentication-Only Apps

When schools connect:
- You receive basic user info (name, email, Schoology ID, role)
- You do **not** receive course or grade data unless specifically requested
- Schools can revoke access at any time

### What to Tell School Administrators

Provide them with:
1. **Your application name** (as registered in Schoology)
2. **What data you access**: "Authentication only—we receive name, email, and role to verify users. We do not access course, grade, or assignment data."
3. **How to install**: Direct link or App Center search instructions

---

## Handoff to Technical Team

### Credentials to Provide

Give your technical team:

| Item | Description | Where to Find |
|------|-------------|---------------|
| **Consumer Key** | Application identifier | Developer portal |
| **Consumer Secret** | Private key | Developer portal |
| **Callback URL** | Your configured callback | Your configuration |
| **Sandbox Credentials** | Test account access | Developer approval email |

### OAuth 1.0a Endpoints

| Endpoint | URL |
|----------|-----|
| Request Token | `https://api.schoology.com/v1/oauth/request_token` |
| Authorization | `https://api.schoology.com/v1/oauth/authorize` |
| Access Token | `https://api.schoology.com/v1/oauth/access_token` |
| User Info | `https://api.schoology.com/v1/users/me` |

### Important Technical Notes

1. **OAuth 1.0a, NOT OAuth 2.0**: Schoology uses the older OAuth 1.0a protocol
   - Requires HMAC-SHA1 signature generation
   - Different from most modern OAuth implementations
   - Many libraries available for OAuth 1.0a

2. **Required OAuth Parameters**:
   - `oauth_consumer_key`
   - `oauth_token`
   - `oauth_signature`
   - `oauth_signature_method` (use `HMAC-SHA1`)
   - `oauth_timestamp`
   - `oauth_nonce`
   - `oauth_version` (use `1.0`)

3. **Token Storage**: Save access tokens to avoid re-authorization

### Security Notes

- **Never share the Consumer Secret publicly**
- The Consumer Key is semi-public, but the Consumer Secret must stay private
- Consider using a secure password manager to transfer credentials

### Technical Documentation

Direct your technical team to:
- [Schoology Authentication - Technical Implementation](../schoology.md)
- [Schoology Developer Portal](https://developers.schoology.com/)
- [Authentication Documentation](https://developers.schoology.com/api-documentation/authentication/)
- [Apps Platform Guide](https://developers.schoology.com/app-platform/)

---

## Timeline Expectations

| Step | Typical Duration |
|------|------------------|
| Developer request submission | 15-30 minutes |
| Developer approval | 1-3 weeks (varies) |
| Application creation | 30-60 minutes |
| **Total registration time** | **1-4 weeks** |
| Development (technical team) | 2-4 weeks (OAuth 1.0a is more complex) |
| App Center review (if applicable) | 1-2 weeks |
| School installations | Varies by school |

**Note**: The approval wait time is the longest part. Start the developer request early!

---

## Troubleshooting

### Developer Registration Issues

**Problem**: No response after submitting developer request

**Solution**:
- Wait at least 2-3 weeks
- Email partnerprogram@powerschool.com
- Check the PowerSchool Community forums
- If working with a district, ask them to escalate through their account manager

### Verification Link Not Working

**Problem**: Email verification link says "not valid"

**Solution**:
- Links may expire — request a new one
- Check if your account is blocked
- Contact PowerSchool support

### OAuth Signature Invalid

**Problem**: OAuth signature errors during authentication

**Solution**:
- Verify Consumer Key and Secret are correct
- Check signature generation follows OAuth 1.0a spec
- Ensure timestamp is current (within 5 minutes)
- Verify nonce is unique
- Use a tested OAuth 1.0a library

### App Not Appearing for Schools

**Problem**: Schools can't find or install your app

**Solution**:
- Verify app is properly registered and approved
- For limited rollout, provide direct installation links
- Ensure visibility settings allow the target schools

---

## Contact Information

| Purpose | Contact |
|---------|---------|
| Partner Program | partnerprogram@powerschool.com |
| Technical Support | [PowerSchool Community](https://help.powerschool.com/) |
| Developer Portal | [developers.schoology.com](https://developers.schoology.com/) |

---

## Additional Resources

- [Schoology Developer Portal](https://developers.schoology.com/)
- [Developer Request Form](https://app.schoology.com/apps/developer/request)
- [API Documentation](https://developers.schoology.com/api/)
- [Authentication Guide](https://developers.schoology.com/api-documentation/authentication/)
- [Apps Platform](https://developers.schoology.com/app-platform/)
- [PowerSchool Community](https://help.powerschool.com/)
- [Developer Terms of Use](https://developers.schoology.com/terms/)

---

## Summary

1. Submit developer request at app.schoology.com/apps/developer/request
2. Wait for approval (may take 1-3 weeks)
3. Access developer sandbox and portal
4. Create your application and obtain Consumer Key/Secret
5. Provide credentials to technical team (note: OAuth 1.0a, not 2.0!)
6. Technical team builds and tests integration
7. Optionally submit to App Center for broader distribution
8. Schools install and users can authenticate

---

[← Back to Onboarding Guides](./README.md) | [Schoology Technical Documentation](../schoology.md)
