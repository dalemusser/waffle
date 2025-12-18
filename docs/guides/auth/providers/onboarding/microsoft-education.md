# Microsoft Education Authentication - Vendor Onboarding Guide

*Step-by-step guide for registering your application with Microsoft Entra ID to authenticate Microsoft 365 Education users.*

---

## Overview

### What is Microsoft Education Authentication?

Microsoft 365 Education provides cloud-based productivity and identity services to schools worldwide. Students and teachers use their Microsoft 365 accounts (ending in domains like `@school.edu`) to access educational tools.

Microsoft Entra ID (formerly Azure Active Directory) is the identity platform that handles authentication for all Microsoft 365 accounts, including:

- **Microsoft 365 Education**: Free and paid plans for K-12 and higher education
- **Work/School Accounts**: Organizational accounts managed by schools
- **Personal Microsoft Accounts**: (Optional) Personal Outlook, Hotmail accounts

### Why You Need This

- **Wide adoption**: Microsoft 365 Education is used by many schools worldwide, especially those not on Chromebooks
- **Office integration**: Schools using Office 365, Teams, or OneDrive already have Microsoft accounts
- **Standard protocol**: Uses industry-standard OpenID Connect (OIDC) / OAuth 2.0
- **Free to register**: No cost to register your application

### What You'll Get

At the end of this process, you will have:
- An **Application (Client) ID** (identifies your application)
- A **Client Secret** (private key—keep this secure)
- **Configured redirect URIs** for your application
- Access to authenticate users from any Microsoft 365 Education tenant

### Important Context for K-12

When a school uses Microsoft 365 Education:
1. **Tenant-level control**: Each school/district has their own Microsoft Entra tenant
2. **Admin consent may be required**: School IT admins may need to approve your app
3. **Multi-tenant support**: You'll configure your app to work with any school's tenant

---

## Prerequisites Checklist

Before starting, gather the following:

### Required

- [ ] **Microsoft Account**: A Microsoft account to access the Azure portal. This can be:
  - A personal Microsoft account (outlook.com, hotmail.com)
  - A work/school account with Azure access
  - A new account created for this purpose

- [ ] **Application Name**: The name users will see when signing in (e.g., "My Learning Portal")
- [ ] **Redirect URL**: The URL where Microsoft sends users after authentication. Get this from your technical team—typically something like `https://yourapp.com/auth/microsoft/callback`

### Recommended

- [ ] **Application Logo**: Image for the consent screen
- [ ] **Privacy Policy URL**: A publicly accessible page with your privacy policy
- [ ] **Terms of Service URL**: A publicly accessible page with your terms
- [ ] **Support URL**: Where users can get help with your app

### For Education Applications

- [ ] **Publisher Domain Verification** (optional): Verify your organization's domain for a more trustworthy appearance
- [ ] **Clear Permissions Documentation**: Be prepared to explain what data your app accesses (for authentication only: "We only request basic profile information to verify user identity")

---

## The Onboarding Process

### Step 1: Access the Microsoft Entra Admin Center

1. Go to [Microsoft Entra Admin Center](https://entra.microsoft.com/)
   - Alternative: [Azure Portal](https://portal.azure.com/) → search for "Microsoft Entra ID"
2. Sign in with your Microsoft account
3. If this is your first time, you may need to:
   - Accept terms of service
   - Set up your directory (a "Default Directory" is created automatically)

**What you'll see**: A dashboard with identity and access management options. Don't be intimidated—you only need the App registrations section.

---

### Step 2: Create a New App Registration

1. In the left sidebar, expand **Applications**
2. Click **App registrations**
3. Click **+ New registration** at the top

---

### Step 3: Configure Basic Information

Fill in the registration form:

1. **Name**: Your application name (e.g., "My Learning Portal")
   - This is what users see on the consent screen
   - You can change this later

2. **Supported account types**: Choose who can sign in:

   | Option | Use Case |
   |--------|----------|
   | **Accounts in this organizational directory only** | Only users from your own organization (not recommended for K-12) |
   | **Accounts in any organizational directory (Any Microsoft Entra ID tenant - Multitenant)** | ✅ **Recommended for K-12** — Any school with Microsoft 365 can sign in |
   | **Accounts in any organizational directory and personal Microsoft accounts** | Schools + personal accounts |
   | **Personal Microsoft accounts only** | Only personal accounts (not for education) |

   **For K-12 apps**: Select **"Accounts in any organizational directory"** to allow any school's users to authenticate.

3. **Redirect URI (optional)**: You can add this now or later
   - Platform: **Web**
   - URI: Your callback URL (e.g., `https://yourapp.com/auth/microsoft/callback`)

4. Click **Register**

**Checkpoint**: You're now on the app's Overview page. You'll see your **Application (client) ID**—copy and save this.

---

### Step 4: Copy the Application (Client) ID

On the Overview page:

1. Find **Application (client) ID**
2. Click the copy icon to copy this value
3. Save it somewhere secure—you'll need this later

**Example**: `12345678-abcd-1234-abcd-123456789abc`

---

### Step 5: Create a Client Secret

Now create the secret key your application will use:

1. In the left sidebar (under your app), click **Certificates & secrets**
2. Click the **Client secrets** tab
3. Click **+ New client secret**
4. Fill in:
   - **Description**: A name to identify this secret (e.g., "Production App Secret")
   - **Expires**: Choose an expiration period
     - **24 months** is maximum for standard secrets
     - Set a calendar reminder to rotate before expiration!

5. Click **Add**

**IMPORTANT**:
- The secret **Value** is shown only once—copy it immediately!
- If you refresh or navigate away, you'll only see a masked version
- Store this value securely

**Example**: `abc123~XYZ789~Secret_Value_Here`

---

### Step 6: Configure Redirect URIs (if not done in Step 3)

1. In the left sidebar, click **Authentication**
2. Under **Platform configurations**, click **+ Add a platform**
3. Select **Web**
4. Enter your **Redirect URI**: `https://yourapp.com/auth/microsoft/callback`
   - For development, you can also add: `https://localhost:3000/auth/microsoft/callback`
5. Click **Configure**

**You can add multiple redirect URIs** for different environments (development, staging, production).

---

### Step 7: Configure API Permissions (Optional)

For basic authentication, the default permissions are usually sufficient. Verify you have:

1. In the left sidebar, click **API permissions**
2. You should see **Microsoft Graph** with these permissions:
   - `User.Read` — Sign in and read user profile

For authentication-only apps, this is all you need.

**If User.Read is not present**:
1. Click **+ Add a permission**
2. Select **Microsoft Graph**
3. Select **Delegated permissions**
4. Search for and check `User.Read`
5. Click **Add permissions**

---

### Step 8: Review Your Configuration

Verify your setup:

| Setting | Where to Find | What You Need |
|---------|---------------|---------------|
| Application (client) ID | Overview page | Copied and saved |
| Client Secret | Certificates & secrets | Copied and saved (only shown once) |
| Redirect URI(s) | Authentication | Your callback URL(s) configured |
| Supported account types | Overview → "Supported account types" | Multitenant for K-12 apps |
| API permissions | API permissions | User.Read (delegated) |

---

## OAuth Endpoints

Your technical team needs these endpoints:

### For Multi-tenant Apps (recommended for K-12)

| Endpoint | URL |
|----------|-----|
| Authorization | `https://login.microsoftonline.com/common/oauth2/v2.0/authorize` |
| Token | `https://login.microsoftonline.com/common/oauth2/v2.0/token` |
| OpenID Configuration | `https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration` |

**Note**: The `/common` path works for any Microsoft Entra tenant.

### For Single-tenant Apps

Replace `{tenant-id}` with your specific tenant ID:
- `https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/authorize`

---

## What Schools Need to Do

After your app is registered, schools can allow their users to sign in.

### User Consent vs. Admin Consent

Microsoft has two consent models:

1. **User Consent**: Users can approve basic permissions themselves (like User.Read)
2. **Admin Consent**: IT admin must approve before any user can sign in (if the school has restricted user consent)

### For Authentication-Only Apps

Your app only needs `User.Read` (basic profile), which:
- Usually doesn't require admin consent
- Users can approve themselves on first sign-in
- IT admins can pre-approve for all users if desired

### What to Tell School IT Administrators

Provide them with:
1. **Application Name**: As shown in your registration
2. **Application (Client) ID**: Your app's unique identifier
3. **Publisher**: Your organization name
4. **Permissions Requested**: "User.Read only — we request basic profile information (name, email) for authentication. We do not access any other Microsoft 365 data."

### If Admin Consent is Required

The IT admin can:
1. Go to **Microsoft Entra admin center** → **Enterprise applications**
2. Find your app (may appear after first user tries to sign in)
3. Click **Permissions** → **Grant admin consent for [Organization]**

Or they can visit this URL directly (replace values):
```
https://login.microsoftonline.com/common/adminconsent?client_id=YOUR_CLIENT_ID
```

---

## Handoff to Technical Team

### Credentials to Provide

Give your technical team:

| Item | Description | Where to Find |
|------|-------------|---------------|
| **Application (Client) ID** | Application identifier | Overview page |
| **Client Secret** | Private key (save immediately!) | Certificates & secrets |
| **Tenant Configuration** | "common" for multi-tenant | Your registration choice |
| **Redirect URI(s)** | The callback URLs configured | Authentication page |

### Scopes to Request

For authentication only:
```
openid profile email User.Read
```

### Security Notes

- **Never share the Client Secret publicly** (don't put it in emails, chat, or code repositories)
- **Set a reminder** to rotate the Client Secret before it expires
- Consider using a secure password manager to transfer credentials
- The Application ID is semi-public, but the Client Secret must stay private

### Technical Documentation

Direct your technical team to:
- [Microsoft Education Authentication - Technical Implementation](../microsoft-education.md)
- [Microsoft Identity Platform Documentation](https://learn.microsoft.com/en-us/entra/identity-platform/)
- [OpenID Connect on Microsoft Identity Platform](https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc)

---

## Timeline Expectations

| Step | Typical Duration |
|------|------------------|
| Azure/Entra account setup | 5-10 minutes |
| App registration | 10-15 minutes |
| Client secret creation | 2 minutes |
| Redirect URI configuration | 5 minutes |
| **Total registration time** | **20-35 minutes** |
| Development (technical team) | 1-2 weeks |
| School IT approval | Varies (instant to weeks) |

---

## Troubleshooting

### "AADSTS50011: Reply URL does not match"

**Cause**: The redirect URI in your app doesn't match what's configured in Azure.

**Solution**:
- Verify the exact redirect URI (including http vs https, trailing slashes)
- Add the missing URI in Authentication → Platform configurations
- URIs are case-sensitive and must match exactly

### "Need Admin Approval"

**Cause**: The school requires admin consent for third-party apps.

**Solution**:
- Contact the school's IT administrator
- Provide your Application ID and explain you only need User.Read
- Admin can grant consent in Microsoft Entra admin center

### Can't Sign In with School Account

**Cause**: App may be configured for wrong account types.

**Solution**:
- Verify "Supported account types" is set to multi-tenant
- Check that redirect URIs are properly configured
- Ensure the school hasn't blocked third-party apps entirely

### Client Secret Expired

**Cause**: The secret has passed its expiration date.

**Solution**:
- Create a new client secret in Certificates & secrets
- Update your application with the new secret
- Set a calendar reminder for next expiration

### User Gets "Unverified Publisher" Warning

**Cause**: Your organization's publisher domain isn't verified.

**Solution** (optional):
- Verify your publisher domain in the Microsoft Entra admin center
- This adds a "verified" badge and increases user trust
- For small projects, users can still proceed without verification

---

## Additional Resources

- [Microsoft Entra Admin Center](https://entra.microsoft.com/)
- [Azure Portal](https://portal.azure.com/)
- [Register an Application](https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app)
- [OpenID Connect on Microsoft Identity Platform](https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc)
- [Microsoft 365 Education](https://www.microsoft.com/en-us/education/products/microsoft-365)
- [Admin Consent](https://learn.microsoft.com/en-us/entra/identity/enterprise-apps/grant-admin-consent)

---

## Summary

1. Sign in to Microsoft Entra Admin Center
2. Create a new app registration
3. Configure for multi-tenant (any organizational directory)
4. Add redirect URI(s)
5. Create a client secret (copy immediately!)
6. Verify User.Read permission is granted
7. Provide credentials to technical team
8. Help schools approve your app if admin consent is required

---

[← Back to Onboarding Guides](./README.md) | [Microsoft Education Technical Documentation](../microsoft-education.md)
