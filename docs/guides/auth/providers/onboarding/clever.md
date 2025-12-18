# Clever Authentication - Vendor Onboarding Guide

*Step-by-step guide for registering as a Clever partner and obtaining SSO credentials.*

---

## Overview

### What is Clever?

Clever is the leading Single Sign-On (SSO) platform for K-12 education in the United States. It provides:

- **Universal SSO**: One login for students and teachers to access all their educational apps
- **Automatic rostering**: Syncs student/teacher data from school information systems (optional—not needed for auth-only)
- **77% US K-12 coverage**: Clever is used by 77% of US K-12 schools and 95 of the top 100 districts

Clever acts as an intermediary between schools and educational applications, simplifying the login experience for students of all ages (including Clever Badges for young students who can't type passwords).

### Why You Need This

- **Largest K-12 SSO platform**: Most schools already have Clever accounts for their students and teachers
- **No school-by-school integration**: One Clever integration works with all Clever schools
- **Simple for students**: Click the app in Clever Portal—no additional passwords
- **Free SSO integration**: Clever SSO is free for basic authentication

### What You'll Get

At the end of this process, you will have:
- A **Client ID** (identifies your application)
- A **Client Secret** (private key—keep this secure)
- A **Development Sandbox** with test users for building your integration
- After certification: **Production access** to live school districts

### Important: Clever Certification

Unlike Google (which is self-service), Clever requires:
1. **Registration** as a developer (this guide—Part 1)
2. **Building** your SSO integration using their sandbox
3. **Certification** by Clever's Partner Engineering team before live school access

This guide covers the **registration and onboarding** process. Your technical team handles the build and certification process.

---

## Prerequisites Checklist

Before starting, gather the following:

### Required

- [ ] **Business Email**: Your organization email address (not personal email)
- [ ] **Company Name**: Your organization's legal or doing-business-as name
- [ ] **Company Website**: Your publicly accessible website URL
- [ ] **Application Name**: The name students/teachers will see (e.g., "My Learning Portal")
- [ ] **Redirect URL**: The URL where Clever sends users after authentication. Get this from your technical team—typically something like `https://yourapp.com/auth/clever/callback`

### Recommended

- [ ] **Application Description**: 1-2 sentences explaining what your app does
- [ ] **Application Logo**: Square image for display in Clever Portal
- [ ] **Contact Phone**: For Clever partner team to reach you
- [ ] **Privacy Policy URL**: A publicly accessible page with your privacy policy

### For K-12 Education Vendors

- [ ] **Data Privacy Information**: Be prepared to describe how you handle student data. For authentication-only: "We only use Clever for authentication. We do not access, store, or process student roster data."
- [ ] **SDPC Membership** (optional): Membership in the Student Data Privacy Consortium can speed approvals

---

## The Onboarding Process

### Part 1: Developer Registration

#### Step 1: Start the Partner Signup

1. Go to [Clever Partner Signup](https://www.clever.com/developer-signup)
2. You'll see a form to begin the registration process

**What you'll see**: A form asking about your business and what you're looking to do with Clever.

---

#### Step 2: Complete the Signup Form

Fill in the required information:

1. **Name**: Your first and last name
2. **Business Email**: Use your organization email (e.g., `you@yourcompany.com`), not a personal Gmail
3. **Company Name**: Your organization name (e.g., "Acme Learning")
4. **Website**: Your company website (e.g., `https://yourcompany.com`)
5. **Phone**: Contact phone number (optional)
6. **Inquiry Category**: Select the option that best describes your goal:
   - "Accelerating market expansion" — if you're looking to reach more schools
   - "Increasing product engagement" — if you want easier logins for existing users
   - "Securing identities and data" — if data security is your primary focus
   - "Eliminating integration complexity" — if you're simplifying your school integrations
   - "Retaining customers and growing revenue" — if you're focused on school retention

7. **Solutions to Explore**: Check **Clever SSO** (and optionally Clever Complete if interested in rostering)

8. **Additional Context**: Add a brief description of your application and needs:
   > "We are building an educational application for K-12 students and teachers. We need Clever SSO to authenticate users—we do not need roster sync."

9. Click **Submit**

---

#### Step 3: Create Your Developer Account

After submitting the partner inquiry, you'll need to create a developer account:

1. Go to [Clever Application Developer Signup](https://apps.clever.com/signup)
2. Create your account with:
   - **Email**: Same business email used in Step 2
   - **Password**: Create a secure password
   - **Application Name**: Name of your app (e.g., "My Learning Portal")

3. Complete the registration

**Checkpoint**: After registration, you can log in to [apps.clever.com](https://apps.clever.com) to access your developer dashboard.

---

#### Step 4: Access Your Developer Dashboard

1. Log in at [apps.clever.com](https://apps.clever.com)
2. You'll see your development application listed
3. Click on your application to view settings

**What you'll find**:
- **Client ID**: Your application's identifier
- **Client Secret**: Your private key (keep this secure!)
- **Settings**: Configure redirect URIs, supported user types, etc.

---

#### Step 5: Configure SSO Settings

1. In your app dashboard, go to **Settings**
2. Find the **Instant Login** or **SSO** section
3. Configure:
   - **Redirect URI**: Enter your callback URL (e.g., `https://yourapp.com/auth/clever/callback`)
   - **Supported User Types**: Select which types of users can log in:
     - ☑ Students
     - ☑ Teachers
     - ☐ District Admins (usually not needed for educational apps)
     - ☐ School Admins (usually not needed for educational apps)

4. Save your settings

**Note**: The first redirect URI in your list becomes your "primary" redirect URI—make sure it's correct.

---

#### Step 6: Access Your Sandbox District

Clever provides a sandbox district with test data:

1. In your dashboard, find the **Sandbox** or **Test District** section
2. You'll have access to test users:
   - Test students
   - Test teachers
   - Sample school/class data

3. Note the sandbox credentials. Default test credentials are often:
   - **Email**: `demo-teacher@clever.com` or similar
   - **Password**: `clever` or provided in dashboard

Your technical team will use this sandbox to build and test the integration.

---

### Part 2: What Happens Next

After registration, the process continues:

| Phase | Who Does This | What Happens |
|-------|---------------|--------------|
| Registration | You (Non-technical) | Complete Steps 1-6 above |
| Development | Technical Team | Build SSO integration using sandbox |
| Testing | Technical Team | Test with sandbox users |
| Certification | Clever + Technical Team | Submit for review, address feedback |
| Production | Everyone | Access live school districts |

---

## Certification Requirements

Your technical team must complete certification before accessing live districts.

### What Clever Reviews

The Partner Engineering team checks that your integration:

- ✅ Uses OAuth 2.0 authorization grant flow
- ✅ Uses Clever API v3.0 or v3.1
- ✅ Uses Clever ID as the primary user identifier
- ✅ Provides clear identification of login accounts
- ✅ Uses HTTPS for redirect URIs
- ✅ Displays friendly error messages for failed logins
- ✅ Includes a logout button
- ✅ Follows shared device security guidelines

### Certification Process

1. **Build**: Technical team completes the integration
2. **Test**: Verify with sandbox district
3. **Submit**: Technical team submits for certification review
4. **Review**: Clever Partner Engineering evaluates the integration
5. **Feedback**: If needed, Clever schedules a call to discuss
6. **Approval**: Receive certification confirmation

### Timeline

- **Registration**: Same day (Steps 1-6)
- **Development**: Varies based on your team (days to weeks)
- **Certification review**: Typically 1-2 weeks after submission
- **Total time to production**: Usually 2-4 weeks for SSO-only

---

## What Schools Need to Do

After you're certified, schools can connect to your app through Clever.

### How School Connections Work

Unlike Google (where IT admins must manually approve each app), Clever works differently:

1. **Your app appears in Clever**: Once certified, schools can find your app
2. **District approves connection**: School/district admin approves sharing data with your app
3. **Automatic access**: Students and teachers see your app in their Clever Portal

### For Authentication-Only Apps

When schools connect:
- You receive basic user info (name, email, Clever ID, school, role)
- You do **not** receive full roster data unless you requested Secure Sync
- Students can log in immediately after the district approves

### What to Tell School Administrators

Provide them with:
1. **Your application name** (as it appears in Clever)
2. **What data you access**: "Authentication only—we receive name, email, and role to verify users. We do not access roster data."
3. **How to connect**: "Search for [App Name] in your Clever Dashboard and approve the connection"

---

## Handoff to Technical Team

### Credentials to Provide

Give your technical team:

| Item | Description | Where to Find |
|------|-------------|---------------|
| **Client ID** | Application identifier | Dashboard → Your App |
| **Client Secret** | Private key | Dashboard → Your App |
| **Redirect URI** | The callback URL configured | Dashboard → Settings |
| **Sandbox Credentials** | Test users for development | Dashboard → Sandbox |

### Security Notes

- **Never share the Client Secret publicly** (don't put it in emails, chat, or code repositories)
- Consider using a secure password manager to transfer credentials
- The Client ID is semi-public, but the Client Secret must stay private

### Technical Documentation

Direct your technical team to:
- [Clever Authentication - Technical Implementation](../clever.md)
- [Clever Developer Documentation](https://dev.clever.com/)
- [Getting Started with Clever SSO](https://dev.clever.com/docs/getting-started-with-clever-sso)

---

## Timeline Expectations

| Step | Typical Duration |
|------|------------------|
| Partner inquiry form | 10 minutes |
| Developer account creation | 10 minutes |
| SSO configuration | 15-30 minutes |
| **Total registration time** | **30-60 minutes** |
| Development (technical team) | 1-2 weeks |
| Certification review | 1-2 weeks |
| School connections | Varies by school |

**Note**: Your first five schools are free, allowing you to pilot without cost barriers.

---

## Troubleshooting

### Form Submission Issues

**Problem**: Partner signup form doesn't submit

**Solution**:
- Ensure you're using a business email, not personal
- Check all required fields are completed
- Try a different browser if issues persist

### Can't Access Developer Dashboard

**Problem**: Login doesn't work or dashboard is empty

**Solution**:
- Verify you created an account at apps.clever.com (separate from partner inquiry)
- Check your email for verification link
- Contact partners@clever.com for assistance

### Sandbox Connection Issues

**Problem**: Can't log in with test credentials

**Solution**:
- Clear any existing Clever sessions (log out completely)
- Verify you're using the correct sandbox credentials from your dashboard
- Ensure your redirect URI is correctly configured

### Certification Rejected

**Problem**: Technical team's integration wasn't approved

**Solution**:
- Review feedback from Clever Partner Engineering
- Address specific issues mentioned
- Resubmit after fixes
- Schedule a call with Clever if feedback is unclear

---

## Contact Information

| Purpose | Contact |
|---------|---------|
| Partner inquiries | partners@clever.com |
| Technical support | dev.clever.com (documentation) |
| General questions | [Clever Support](https://support.clever.com/) |

---

## Additional Resources

- [Clever Developer Portal](https://dev.clever.com/)
- [Clever Partner Signup](https://www.clever.com/developer-signup)
- [Application Developer Signup](https://apps.clever.com/signup)
- [Getting Started with Clever SSO](https://dev.clever.com/docs/getting-started-with-clever-sso)
- [District SSO Certification Guide](https://dev.clever.com/docs/district-sso-certification-guide)

---

## Summary

1. Submit partner inquiry form at clever.com/developer-signup
2. Create developer account at apps.clever.com/signup
3. Configure SSO settings in your dashboard
4. Provide credentials to technical team
5. Technical team builds and tests integration
6. Submit for Clever certification
7. After certification, schools can connect to your app

---

[← Back to Onboarding Guides](./README.md) | [Clever Technical Documentation](../clever.md)
