# ClassLink Authentication - Vendor Onboarding Guide

*Step-by-step guide for registering with ClassLink and obtaining OIDC credentials for your application.*

---

## Overview

### What is ClassLink?

ClassLink is a leading Single Sign-On (SSO) and rostering platform for K-12 education. It provides:

- **LaunchPad**: A personalized dashboard where students and teachers access all their educational apps
- **OneRoster Rostering**: Automatic sync of student/teacher data from school information systems (optional—not needed for auth-only)
- **Wide K-12 Coverage**: Used by thousands of school districts across the US and internationally

ClassLink acts as an intermediary between schools and educational applications, providing a unified login experience through their LaunchPad portal.

### Why You Need This

- **Major K-12 SSO platform**: Second largest K-12 SSO provider alongside Clever
- **Simple integration**: One ClassLink integration works with all ClassLink districts
- **OIDC standard**: Uses modern OpenID Connect protocol for authentication
- **Free for vendors**: "Interoperability shouldn't cost a dime" — ClassLink provides free SSO integration

### What You'll Get

At the end of this process, you will have:
- A **Client ID** (identifies your application)
- A **Client Secret** (private key—keep this secure)
- **Test OAuth2 Configuration** for development
- **Production OAuth2 Configuration** for live schools

### Partner Tiers

ClassLink offers three partnership levels:

| Tier | Benefits |
|------|----------|
| **Integrated Partner** | SSO + Rostering integration (basic) |
| **Certified Partner** | + Signed MOU, newsletter announcement, custom transition plan |
| **Preferred Partner** | + Press announcement, featured in App Library, shared customer lists, co-marketing |

For authentication-only needs, **Integrated Partner** status is sufficient.

---

## Prerequisites Checklist

Before starting, gather the following:

### Required

- [ ] **Business Email**: Your organization email address (not personal email)
- [ ] **Company Name**: Your organization's legal or doing-business-as name
- [ ] **Application Name**: The name students/teachers will see (e.g., "My Learning Portal")
- [ ] **Application Logo**: Image for display in LaunchPad (recommended: square format)
- [ ] **Redirect URL**: The URL where ClassLink sends users after authentication. Get this from your technical team—typically something like `https://yourapp.com/auth/classlink/callback`

### Recommended

- [ ] **Company Website**: Your publicly accessible website URL
- [ ] **Application Description**: Brief description of what your app does
- [ ] **Privacy Policy URL**: A publicly accessible page with your privacy policy
- [ ] **Domain**: The domain(s) your application runs on

### For K-12 Education Vendors

- [ ] **Data Privacy Information**: Be prepared to describe how you handle student data. For authentication-only: "We only use ClassLink for authentication. We do not access, store, or process student roster data."

---

## The Onboarding Process

### Part 1: Partner Portal Registration

#### Step 1: Access the Partner Portal

1. Go to [ClassLink Partner Portal](https://partnerportal.classlink.com/) or [ClassLink Developer Portal](https://dev.classlink.com/)
2. Click **Sign Up** or **Register** to create a new account
3. If you already have an account, log in

**Note**: If you want to discuss partnership options first, visit [ClassLink Prospective Partners](https://www.classlink.com/company/prospective-partners) and use the "Get in Touch" form.

---

#### Step 2: Create Your Account

Fill in your registration information:

1. **Email**: Your business email address
2. **Password**: Create a secure password
3. **Company/Organization Name**: Your organization's name
4. **Role/Title**: Your position in the organization

Complete the registration and verify your email if required.

**Checkpoint**: After registration, you can access the Partner Portal dashboard.

---

#### Step 3: Create Your Application

Once logged in to the Partner Portal:

1. Navigate to **My Apps** (this is your main dashboard)
2. Click **Add App** (typically in the top left section)
3. Fill in:
   - **App Name**: Your application name (e.g., "My Learning Portal")
   - **Logo**: Upload your application logo or select from existing ones
   - **Domain**: Select an existing domain or add a new one

4. Click **Save** or **Create**

**What you'll have**: Each app in the Partner Portal contains:
- Roster Server Connection (for rostering—optional)
- Production OAuth2 Configuration (for live schools)
- Test OAuth2 Configuration (for development)

---

### Part 2: Configure OAuth2/SSO

#### Step 4: Access SSO Configuration

1. In your app dashboard, navigate to **SSO Connections** or **OAuth2 Configurations**
2. You'll see options for both **Test** and **Production** configurations
3. Start with the **Test OAuth2 Configuration** for development

---

#### Step 5: Create a New SSO Connection

1. Click **Add New SSO Connection** (or similar button)
2. Enter the **Connection Name** (e.g., "My Learning Portal SSO")
3. Select your **Domain** from the dropdown
4. Click **Next** to continue

---

#### Step 6: Configure Authorization Settings

1. **Redirect URI**: Enter your callback URL
   - Example: `https://yourapp.com/auth/classlink/callback`
   - For development, you may use `https://localhost:3000/auth/classlink/callback`

2. **Select Scopes**: Choose the OAuth2 scopes you need:
   - ☑ `openid` — Required for authentication
   - ☑ `profile` — Access to user's name and basic info
   - ☑ `email` — Access to user's email address
   - For authentication only, "full openid profile" is typically sufficient

3. Click **Save**

---

#### Step 7: Retrieve Your Credentials

After saving, ClassLink assigns a unique Application ID to your application:

1. In the **Edit Connection Info** tab (or similar), find:
   - **Client ID**: Your application's identifier
   - **Client Secret**: Your private key (keep this secure!)

2. **Copy both values** and store them securely
3. Note the OAuth2 endpoints:
   - **Authorization Endpoint**: `https://launchpad.classlink.com/oauth2/v2/auth`
   - **Token Endpoint**: `https://launchpad.classlink.com/oauth2/v2/token`
   - **User Info Endpoint**: `https://nodeapi.classlink.com/v2/my/info`

**Checkpoint**: You now have all credentials needed for development.

---

#### Step 8: Test Your Configuration

1. Your technical team can now build the integration using the Test credentials
2. Use the test environment to verify:
   - Login flow works correctly
   - User information is retrieved properly
   - Redirect URIs work as expected

3. Many partners use ClassLink's built-in test functionality to verify the SSO connection

---

### Part 3: Production Setup

#### Step 9: Configure Production OAuth2

Once testing is complete:

1. Go to your app's **Production OAuth2 Configuration**
2. Repeat Steps 5-7 with production settings:
   - Use your production redirect URI (e.g., `https://yourapp.com/auth/classlink/callback`)
   - Copy the production Client ID and Client Secret

3. Store production credentials securely—these are for live schools

---

## What Schools Need to Do

After your integration is working, schools can connect through ClassLink.

### How School Connections Work

ClassLink uses a district-initiated connection model:

1. **School finds your app**: In the ClassLink App Library or by searching
2. **District admin approves**: School/district admin approves the connection
3. **App appears in LaunchPad**: Students and teachers see your app in their ClassLink portal
4. **One-click access**: Users click your app icon to authenticate

### For Authentication-Only Apps

When schools connect:
- You receive basic user info (name, email, ClassLink ID, role)
- You do **not** receive roster data unless you've set up Roster Server Connection
- Authentication is immediate once the district approves

### What to Tell School Administrators

Provide them with:
1. **Your application name** (as it appears in ClassLink)
2. **What data you access**: "Authentication only—we receive name, email, and role to verify users. We do not access roster data."
3. **How to connect**: "Search for [App Name] in the ClassLink App Library or contact ClassLink support to add our app"

---

## Handoff to Technical Team

### Credentials to Provide

Give your technical team:

| Item | Description | Where to Find |
|------|-------------|---------------|
| **Client ID** | Application identifier | Partner Portal → Your App → OAuth2 Config |
| **Client Secret** | Private key | Partner Portal → Your App → OAuth2 Config |
| **Redirect URI** | The callback URL configured | Your configuration settings |
| **Test Credentials** | Separate set for development | Test OAuth2 Configuration |
| **Production Credentials** | For live schools | Production OAuth2 Configuration |

### OAuth2 Endpoints

Your technical team needs these endpoints:

| Endpoint | URL |
|----------|-----|
| Authorization | `https://launchpad.classlink.com/oauth2/v2/auth` |
| Token | `https://launchpad.classlink.com/oauth2/v2/token` |
| User Info | `https://nodeapi.classlink.com/v2/my/info` |

### Security Notes

- **Never share the Client Secret publicly** (don't put it in emails, chat, or code repositories)
- Consider using a secure password manager to transfer credentials
- Keep Test and Production credentials separate

### Technical Documentation

Direct your technical team to:
- [ClassLink Authentication - Technical Implementation](../classlink.md)
- [ClassLink Partner Portal](https://partnerportal.classlink.com/)
- [ClassLink Help Center](https://help.classlink.com/)

---

## Timeline Expectations

| Step | Typical Duration |
|------|------------------|
| Partner Portal registration | 10-15 minutes |
| Application creation | 5-10 minutes |
| SSO configuration | 15-30 minutes |
| **Total registration time** | **30-60 minutes** |
| Development (technical team) | 1-2 weeks |
| School connections | Varies by school |

**Note**: ClassLink integration is generally faster than Clever because there's no formal certification process—you can go live once your integration is working.

---

## Troubleshooting

### Can't Access Partner Portal

**Problem**: Registration or login doesn't work

**Solution**:
- Verify your email address
- Check your spam folder for verification emails
- Try a different browser
- Contact ClassLink at helpdesk@classlink.com or call 888.963.7550 x1

### OAuth2 Connection Fails

**Problem**: Can't authenticate or get tokens

**Solution**:
- Verify redirect URI matches exactly (including trailing slashes)
- Check that Client ID and Secret are correct
- Ensure you're using the right endpoints (v2)
- Verify scopes are properly configured

### User Info Not Returned

**Problem**: Authentication works but no user data

**Solution**:
- Verify you've requested appropriate scopes (openid, profile, email)
- Check that the token exchange completed successfully
- Ensure you're calling the correct user info endpoint

### School Can't Find Your App

**Problem**: District can't locate your app to connect

**Solution**:
- Verify your app is properly configured in the Partner Portal
- Contact ClassLink support to ensure visibility
- Provide the school with your exact app name and any identifiers

---

## Contact Information

| Purpose | Contact |
|---------|---------|
| General Support | helpdesk@classlink.com |
| Phone Support | 888.963.7550 x1 |
| Partner Inquiries | [Prospective Partners Form](https://www.classlink.com/company/prospective-partners) |
| Technical Help | [ClassLink Help Center](https://help.classlink.com/) |

---

## Additional Resources

- [ClassLink Partner Portal](https://partnerportal.classlink.com/)
- [ClassLink Developer Portal](https://dev.classlink.com/)
- [ClassLink Prospective Partners](https://www.classlink.com/company/prospective-partners)
- [ClassLink Partners](https://www.classlink.com/company/partners)
- [ClassLink Help Center](https://help.classlink.com/)

---

## Summary

1. Register for the ClassLink Partner Portal
2. Create your application
3. Configure OAuth2/SSO connection with redirect URI and scopes
4. Retrieve Client ID and Client Secret
5. Provide credentials to technical team
6. Technical team builds and tests integration
7. Configure production credentials when ready
8. Schools can connect through ClassLink App Library

---

[← Back to Onboarding Guides](./README.md) | [ClassLink Technical Documentation](../classlink.md)
