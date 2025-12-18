# Google Authentication - Vendor Onboarding Guide

*Step-by-step guide for obtaining Google OAuth credentials for your application.*

---

## Overview

### What is Google Authentication?

Google Authentication allows users to sign into your application using their existing Google account. For K-12 schools, most students and teachers already have Google accounts through **Google Workspace for Education** (formerly G Suite for Education). Since Chromebooks require Google accounts, schools using Chromebooks already have this infrastructure in place.

### Why You Need This

- **Chromebook schools**: Students on Chromebooks already have Google accounts—this is their primary identity
- **Broad coverage**: Google Workspace for Education is used by a large percentage of US K-12 schools
- **Simple for users**: Students click "Sign in with Google" and use credentials they already know
- **No cost**: Google OAuth is free to implement

### What You'll Get

At the end of this process, you will have:
- A **Client ID** (a long string that identifies your application)
- A **Client Secret** (a private key—keep this secure)

These credentials allow your application to request authentication from Google on behalf of users.

### Important Context for K-12

When a school uses Google Workspace for Education, their IT administrator controls which third-party apps students can sign into. Your app must be:
1. Registered with Google (this guide)
2. Approved by each school's IT administrator (handled per-school)

The school approval happens after you complete this process. You'll provide schools with your app information so their IT admin can allow it.

---

## Prerequisites Checklist

Before starting, gather the following:

### Required

- [ ] **Google Account**: Any Google account to access Google Cloud Console (can be personal or organizational). This becomes the "owner" of the project.
- [ ] **Application Name**: The name users will see when signing in (e.g., "My Learning Portal")
- [ ] **Application Website URL**: Your publicly accessible website (e.g., `https://yourapp.example.com`)
- [ ] **Privacy Policy URL**: A publicly accessible page with your privacy policy (e.g., `https://yourapp.example.com/privacy`)
- [ ] **Support Email Address**: An email where users can reach you with questions
- [ ] **Redirect URL**: The URL where Google sends users after authentication. Get this from your technical team—it's typically something like `https://yourapp.com/auth/google/callback`

### Recommended

- [ ] **Application Logo**: A square image (preferably 120x120 pixels) for the consent screen
- [ ] **Terms of Service URL**: A publicly accessible page with your terms (e.g., `https://yourapp.example.com/terms`)
- [ ] **Developer Contact Email**: Where Google can reach you about your project (can be same as support email)

### For Verification (if needed)

- [ ] **Domain Ownership**: You must be able to verify you own the domain (typically done by adding a DNS record or uploading a file). Your IT/web team can help with this.

---

## The Onboarding Process

### Step 1: Access Google Cloud Console

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Sign in with your Google account
3. If this is your first time, you may need to agree to terms of service

**What you'll see**: A dashboard with various Google Cloud services. Don't be intimidated—you only need a small part of this.

---

### Step 2: Create a New Project

1. At the top of the page, click the **project dropdown** (it may say "Select a project" or show an existing project name)
2. In the popup, click **New Project** (upper right)
3. Fill in:
   - **Project name**: Something descriptive (e.g., "My Learning Portal")
   - **Organization**: Leave as default or select your organization if shown
   - **Location**: Leave as default
4. Click **Create**
5. Wait a moment, then select your new project from the dropdown

**Checkpoint**: You should see your project name in the top dropdown.

---

### Step 3: Configure the OAuth Consent Screen

This defines what users see when they sign in.

1. In the left sidebar, click **APIs & Services** (you may need to click the hamburger menu ☰ first)
2. Click **OAuth consent screen**
3. Select **User Type**:
   - Choose **External** (this allows any Google user to sign in)
   - Click **Create**

4. Fill in the **App information**:
   - **App name**: Your application name (e.g., "My Learning Portal")
   - **User support email**: Select or enter your support email
   - **App logo**: Upload your logo (optional but recommended)

5. Fill in the **App domain** section:
   - **Application home page**: Your website URL (e.g., `https://yourapp.example.com`)
   - **Application privacy policy link**: Your privacy policy URL
   - **Application terms of service link**: Your terms URL (optional)

6. Fill in **Authorized domains**:
   - Click **Add Domain**
   - Enter your domain without https:// (e.g., `example.com`)
   - If you have multiple domains, add each one

7. Fill in **Developer contact information**:
   - Enter one or more email addresses where Google can reach you

8. Click **Save and Continue**

---

### Step 4: Configure Scopes

Scopes define what information your app can access. For authentication only, you need minimal scopes.

1. Click **Add or Remove Scopes**
2. In the search/filter, find and check these scopes:
   - `openid` — Allows authentication
   - `email` — Access to user's email address
   - `profile` — Access to user's name and profile picture

3. Click **Update**
4. Click **Save and Continue**

**Note**: These are all "non-sensitive" scopes, which means simpler verification requirements.

---

### Step 5: Add Test Users

While your app is in "Testing" mode, only specified test users can sign in.

1. Click **Add Users**
2. Enter email addresses of people who will test the app (your team members)
3. Click **Add**
4. Click **Save and Continue**

**Important**: You can add up to 100 test users. These are for your team during development. Once you publish the app, anyone can sign in.

---

### Step 6: Review and Complete

1. Review the summary of your OAuth consent screen settings
2. Click **Back to Dashboard**

**Checkpoint**: Your OAuth consent screen is now configured.

---

### Step 7: Create OAuth Credentials

Now create the actual credentials your app will use.

1. In the left sidebar, click **Credentials**
2. Click **+ Create Credentials** at the top
3. Select **OAuth client ID**
4. For **Application type**, select **Web application**
5. Fill in:
   - **Name**: A name for this credential set (e.g., "Production Web App")
   - **Authorized JavaScript origins**: Leave blank (not needed for server-side auth)
   - **Authorized redirect URIs**: Click **Add URI** and enter your redirect URL (get this from your technical team—e.g., `https://yourapp.example.com/auth/google/callback`)

6. Click **Create**

**Success!** A popup appears with your credentials:
- **Client ID**: A long string ending in `.apps.googleusercontent.com`
- **Client Secret**: A shorter string (keep this private!)

7. Click **Download JSON** to save a backup of these credentials
8. Click **OK**

---

### Step 8: Publish Your App (When Ready)

While in "Testing" mode:
- Only test users you added can sign in
- Tokens expire after 7 days

To allow all users to sign in:

1. Go to **OAuth consent screen**
2. Under **Publishing status**, click **Publish App**
3. Click **Confirm**

**For apps with only basic scopes (openid, email, profile)**: Publishing is straightforward and doesn't require extensive review.

**Note**: You can stay in Testing mode during development and publish when you're ready for real users.

---

## Verification Requirements

### When Verification is Needed

Google may require verification if:
- Your app uses sensitive or restricted scopes (yours doesn't—you only need basic scopes)
- Your app displays a logo on the consent screen
- Your app is published to external users

### For Basic Authentication (Your Case)

With only `openid`, `email`, and `profile` scopes:
- **No extensive verification required**
- You may need **brand verification** if you display a logo (2-3 business days)
- You need to verify domain ownership

### Verification Process

If verification is required:
1. Google will review your consent screen, privacy policy, and app
2. You may receive emails asking for clarification
3. Timeline: Typically 2-3 business days for basic apps

### Avoiding the "Unverified App" Warning

Until verified, users may see "Google hasn't verified this app" with a warning screen. They can still proceed by clicking "Advanced" → "Go to [App Name]". After verification, this warning disappears.

---

## What Schools Need to Do

After you have your credentials and your app is working, each participating school's IT administrator needs to allow your app.

### What to Tell School IT Administrators

Provide them with:
1. **Your application name** (as it appears in Google)
2. **Your Client ID** (the long string ending in `.apps.googleusercontent.com`)
3. **What permissions you request**: "Sign-in only—we only request basic profile information (name and email) to authenticate users"

### What the School Admin Does

In their Google Admin Console:
1. Go to **Security** → **Access and data control** → **API Controls**
2. Under **App access control**, find or add your app
3. Set access level to **Trusted** or ensure sign-in access is allowed

Schools familiar with educational apps will recognize this process—it's the same for Kahoot, Quizlet, and other educational tools.

---

## Handoff to Technical Team

### Credentials to Provide

Give your technical team:

| Item | Description | Example |
|------|-------------|---------|
| **Client ID** | Long string from Step 7 | `123456789-abc123.apps.googleusercontent.com` |
| **Client Secret** | Shorter private string from Step 7 | `GOCSPX-abc123xyz` |
| **Redirect URI** | The callback URL you configured | `https://yourapp.com/auth/google/callback` |

### Security Notes

- **Never share the Client Secret publicly** (don't put it in emails, chat, or code repositories)
- Consider using a secure password manager or secrets management system to transfer credentials
- The Client ID is semi-public (users can see it), but the Client Secret must stay private

### Technical Documentation

Direct your technical team to: [Google Authentication - Technical Implementation](../google.md)

---

## Timeline Expectations

| Step | Time Required |
|------|---------------|
| Initial setup (Steps 1-7) | 30-60 minutes |
| Testing mode | Immediate |
| Publishing (basic scopes) | Immediate to 24 hours |
| Brand verification (if needed) | 2-3 business days |
| School IT approval | Varies by school (same day to weeks) |

---

## Troubleshooting

### "Access Blocked" Error

**Cause**: The school's Google Workspace admin has restricted third-party app access.

**Solution**: Contact the school's IT administrator and ask them to allow your app (provide your Client ID).

### "Unverified App" Warning

**Cause**: Your app hasn't completed Google's verification process.

**Solution**:
- During testing, click "Advanced" → "Go to [App Name]" to proceed
- For production, submit for verification or ensure you only use basic scopes

### "Redirect URI Mismatch"

**Cause**: The redirect URL in your app doesn't match what you configured in Google Cloud Console.

**Solution**: Verify the exact redirect URI with your technical team and update the Google Cloud Console settings to match.

---

## Additional Resources

- [Google Cloud Console](https://console.cloud.google.com/)
- [Setting up OAuth 2.0](https://support.google.com/cloud/answer/6158849) - Google's official guide
- [OAuth Consent Screen Configuration](https://developers.google.com/workspace/guides/configure-oauth-consent)
- [Control Third-Party App Access](https://support.google.com/a/answer/7281227) - For school administrators

---

## Summary

1. Create a Google Cloud project
2. Configure OAuth consent screen with your app details
3. Add basic scopes (openid, email, profile)
4. Create OAuth credentials
5. Publish when ready for real users
6. Provide credentials to technical team
7. Help schools allow your app through their IT administrators

---

[← Back to Onboarding Guides](./README.md) | [Google Technical Documentation](../google.md)
