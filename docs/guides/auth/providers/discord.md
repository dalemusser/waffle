# Discord Authentication

*Gaming and community platform with server/guild access.*

Discord OAuth2 provides authentication for Discord users with access to guilds (servers), roles, and the Discord API. Popular for gaming communities and developer tools.

---

## 1. Create Discord Application

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **New Application**
3. Enter your application name and create
4. Go to **OAuth2** → **General**
5. Add **Redirect**: `https://yourapp.com/auth/discord/callback`
6. Note your **Client ID** and **Client Secret**

## 2. Add to AppConfig

```go
type AppConfig struct {
    // ... existing fields

    DiscordClientID     string `conf:"discord_client_id"`
    DiscordClientSecret string `conf:"discord_client_secret"`
    DiscordRedirectURL  string `conf:"discord_redirect_url"`
}
```

## 3. Wire Up in BuildHandler

```go
// Create Discord OAuth provider
discordAuth, err := oauth2.Discord(oauth2.DiscordConfig{
    ClientID:     appCfg.DiscordClientID,
    ClientSecret: appCfg.DiscordClientSecret,
    RedirectURL:  appCfg.DiscordRedirectURL,
    Scopes:       []string{"identify", "email", "guilds"},
    SessionStore: sessionStore,
    StateStore:   stateStore,
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
if err != nil {
    return nil, err
}

// Discord auth routes
r.Get("/auth/discord/login", discordAuth.LoginHandler())
r.Get("/auth/discord/callback", discordAuth.CallbackHandler())
r.Get("/auth/discord/logout", discordAuth.LogoutHandler())
```

## User Information

The `User.Extra` map contains Discord-specific fields:

| Field | Description |
|-------|-------------|
| `id` | Discord user ID (snowflake) |
| `username` | Username |
| `discriminator` | 4-digit discriminator (legacy) |
| `global_name` | Display name |
| `avatar` | Avatar hash |
| `banner` | Banner hash |
| `accent_color` | Profile accent color |
| `locale` | User's locale |
| `mfa_enabled` | Whether MFA is enabled |
| `premium_type` | Nitro subscription type |
| `public_flags` | Public user flags |
| `verified` | Whether email is verified |

## Helper Functions

```go
// Get user identifiers
discordID := oauth2.GetDiscordUserID(user)
username := oauth2.GetDiscordUsername(user)
globalName := oauth2.GetDiscordGlobalName(user)

// Get avatar URL
avatarURL := oauth2.GetDiscordAvatarURL(user)
// Returns: https://cdn.discordapp.com/avatars/{id}/{hash}.png

// Check Nitro status
if oauth2.HasDiscordNitro(user) { ... }
if oauth2.HasDiscordNitroBasic(user) { ... }

// Check verification
if oauth2.IsDiscordVerified(user) { ... }
if oauth2.IsDiscordMFAEnabled(user) { ... }
```

## API Client

For Discord API calls:

```go
func (h *Handler) UserGuilds(w http.ResponseWriter, r *http.Request) {
    user := oauth2.UserFromContext(r.Context())

    discordClient := oauth2.NewDiscordAPIClient(user.AccessToken)

    guilds, err := discordClient.GetUserGuilds(r.Context())
    if err != nil {
        http.Error(w, "Failed to fetch guilds", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(guilds)
}
```

## API Client Methods

| Method | Description |
|--------|-------------|
| `GetUser(ctx)` | Get authenticated user |
| `GetUserGuilds(ctx)` | Get user's guilds |
| `GetGuildMember(ctx, guildID)` | Get user's membership in guild |
| `GetUserConnections(ctx)` | Get user's connected accounts |

## Guild Membership

Check guild membership for access control:

```go
discordAuth, err := oauth2.Discord(oauth2.DiscordConfig{
    // ... other config ...
    Scopes: []string{"identify", "email", "guilds"},
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        discordClient := oauth2.NewDiscordAPIClient(user.AccessToken)
        guilds, _ := discordClient.GetUserGuilds(r.Context())

        hasAccess := false
        for _, guild := range guilds {
            if guild.ID == "your-guild-id" {
                hasAccess = true
                break
            }
        }

        if !hasAccess {
            http.Error(w, "Must be member of guild", http.StatusForbidden)
            return
        }

        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
```

## Guild Role Check

For role-based access, use `guilds.members.read`:

```go
discordAuth, err := oauth2.Discord(oauth2.DiscordConfig{
    // ... other config ...
    Scopes: []string{"identify", "guilds.members.read"},
    GuildID: "your-guild-id",  // For guild.members.read
    OnSuccess: func(w http.ResponseWriter, r *http.Request, user *oauth2.User) {
        discordClient := oauth2.NewDiscordAPIClient(user.AccessToken)
        member, _ := discordClient.GetGuildMember(r.Context(), "your-guild-id")

        // Check for required role
        hasRole := false
        for _, roleID := range member.Roles {
            if roleID == "required-role-id" {
                hasRole = true
                break
            }
        }

        if !hasRole {
            http.Error(w, "Missing required role", http.StatusForbidden)
            return
        }

        http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
    },
}, logger)
```

## Common Scopes

| Scope | Description |
|-------|-------------|
| `identify` | User identity (id, username, avatar) |
| `email` | User's email address |
| `guilds` | List of user's guilds |
| `guilds.members.read` | Guild member info (requires guild ID) |
| `connections` | User's connected accounts |
| `bot` | Add bot to guild (bot flow) |

## Important Notes

- Discord IDs are snowflakes (string representation of 64-bit integers)
- Avatar URLs require constructing from hash
- Discriminators are being phased out (new username system)
- `guilds.members.read` requires specifying a guild ID in the OAuth URL
- Rate limits apply (check Discord documentation)
- Consider using Discord bots for server-side automation

---

[← Back to OAuth2 Providers](./README.md)
