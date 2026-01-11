// auth/oauth2/discord.go
package oauth2

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// DiscordConfig holds configuration for Discord OAuth2 authentication.
type DiscordConfig struct {
	// ClientID is the Discord OAuth2 client ID (Application ID).
	ClientID string

	// ClientSecret is the Discord OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with Discord.
	// Example: "https://myapp.com/auth/discord/callback"
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	// Default: identify, email
	// Common scopes: identify, email, guilds, guilds.members.read
	Scopes []string

	// SessionStore persists user sessions.
	SessionStore SessionStore

	// StateStore persists OAuth2 state for CSRF protection.
	StateStore StateStore

	// SessionDuration controls how long sessions remain valid (in seconds).
	// Default: 24 hours.
	SessionDuration int

	// CookieName is the name of the session cookie.
	// Default: "waffle_session".
	CookieName string

	// CookieSecure sets the Secure flag on cookies (HTTPS only).
	// Default: true.
	CookieSecure bool

	// OnSuccess is called after successful authentication.
	OnSuccess func(w http.ResponseWriter, r *http.Request, user *User)

	// OnError is called when an error occurs during authentication.
	OnError func(w http.ResponseWriter, r *http.Request, err error)

	// Logger for logging authentication events.
	Logger *zap.Logger
}

// Discord creates a new OAuth2 provider configured for Discord authentication.
//
// Discord is a popular communication platform used by gaming communities, developers,
// and various online communities. It provides OAuth2 authentication and access to
// user profiles, guilds (servers), and roles.
//
// Setup in Discord Developer Portal:
//  1. Go to https://discord.com/developers/applications
//  2. Click "New Application" and give it a name
//  3. Go to OAuth2 → General
//  4. Add your redirect URL under "Redirects"
//  5. Copy your Client ID and Client Secret
//
// Usage in BuildHandler:
//
//	discordAuth, err := oauth2.Discord(oauth2.DiscordConfig{
//	    ClientID:     appCfg.DiscordClientID,
//	    ClientSecret: appCfg.DiscordClientSecret,
//	    RedirectURL:  "https://myapp.com/auth/discord/callback",
//	    SessionStore: mySessionStore,
//	    StateStore:   myStateStore,
//	}, logger)
//
//	r.Get("/auth/discord/login", discordAuth.LoginHandler())
//	r.Get("/auth/discord/callback", discordAuth.CallbackHandler())
//	r.Get("/auth/discord/logout", discordAuth.LogoutHandler())
//
// The User.Extra map contains Discord-specific fields:
//   - "discord_id": Discord user ID (snowflake)
//   - "username": Discord username
//   - "discriminator": Legacy discriminator (deprecated, usually "0")
//   - "global_name": User's display name
//   - "avatar": Avatar hash (use GetDiscordAvatarURL for full URL)
//   - "banner": Banner hash
//   - "accent_color": User's banner color
//   - "locale": User's locale
//   - "mfa_enabled": "true" if MFA is enabled
//   - "premium_type": Nitro subscription type (0=none, 1=classic, 2=nitro, 3=basic)
//   - "public_flags": User's public flags (badges)
func Discord(cfg DiscordConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("oauth2/discord: ClientID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("oauth2/discord: ClientSecret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, errors.New("oauth2/discord: RedirectURL is required")
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("oauth2/discord: SessionStore is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("oauth2/discord: StateStore is required")
	}

	// Discord OAuth2 endpoint
	endpoint := oauth2.Endpoint{
		AuthURL:  "https://discord.com/api/oauth2/authorize",
		TokenURL: "https://discord.com/api/oauth2/token",
	}

	// Default scopes for Discord
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"identify",
			"email",
		}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	providerCfg := &Config{
		ProviderName:  "discord",
		OAuth2Config:  oauth2Cfg,
		FetchUserInfo: discordFetchUserInfo,
		SessionStore:  cfg.SessionStore,
		StateStore:    cfg.StateStore,
		CookieName:    cfg.CookieName,
		CookieSecure:  cfg.CookieSecure,
		OnSuccess:     cfg.OnSuccess,
		OnError:       cfg.OnError,
		Logger:        logger,
	}

	if cfg.SessionDuration > 0 {
		providerCfg.SessionDuration = time.Duration(cfg.SessionDuration) * time.Second
	}

	return NewProvider(providerCfg)
}

// discordUserInfo represents the response from Discord's /users/@me endpoint.
type discordUserInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"` // Deprecated, usually "0" now
	GlobalName    string `json:"global_name"`   // Display name
	Avatar        string `json:"avatar"`
	Banner        string `json:"banner"`
	AccentColor   int    `json:"accent_color"`
	Locale        string `json:"locale"`
	MFAEnabled    bool   `json:"mfa_enabled"`
	PremiumType   int    `json:"premium_type"` // 0=none, 1=classic, 2=nitro, 3=basic
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Flags         int    `json:"flags"`
	PublicFlags   int    `json:"public_flags"`
	Bot           bool   `json:"bot"`
	System        bool   `json:"system"`
}

// discordFetchUserInfo fetches user information from Discord's API.
func discordFetchUserInfo(ctx context.Context, token *oauth2.Token) (*User, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	resp, err := client.Get("https://discord.com/api/v10/users/@me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info discordUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Build display name - prefer global_name, fall back to username
	name := info.GlobalName
	if name == "" {
		name = info.Username
	}

	// Build avatar URL
	avatarURL := ""
	if info.Avatar != "" {
		ext := "png"
		if strings.HasPrefix(info.Avatar, "a_") {
			ext = "gif" // Animated avatar
		}
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", info.ID, info.Avatar, ext)
	} else {
		// Default avatar based on discriminator or user ID
		var index int
		if info.Discriminator != "" && info.Discriminator != "0" {
			idx, _ := strconv.Atoi(info.Discriminator)
			index = idx % 5
		} else {
			id, _ := strconv.ParseInt(info.ID, 10, 64)
			index = int((id >> 22) % 6)
		}
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)
	}

	return &User{
		ID:            info.ID,
		Email:         info.Email,
		EmailVerified: info.Verified,
		Name:          name,
		Picture:       avatarURL,
		Raw: map[string]any{
			"id":            info.ID,
			"username":      info.Username,
			"discriminator": info.Discriminator,
			"global_name":   info.GlobalName,
			"avatar":        info.Avatar,
			"banner":        info.Banner,
			"accent_color":  info.AccentColor,
			"locale":        info.Locale,
			"mfa_enabled":   info.MFAEnabled,
			"premium_type":  info.PremiumType,
			"email":         info.Email,
			"verified":      info.Verified,
			"flags":         info.Flags,
			"public_flags":  info.PublicFlags,
		},
		Extra: map[string]string{
			"discord_id":    info.ID,
			"username":      info.Username,
			"discriminator": info.Discriminator,
			"global_name":   info.GlobalName,
			"avatar":        info.Avatar,
			"banner":        info.Banner,
			"accent_color":  strconv.Itoa(info.AccentColor),
			"locale":        info.Locale,
			"mfa_enabled":   strconv.FormatBool(info.MFAEnabled),
			"premium_type":  strconv.Itoa(info.PremiumType),
			"public_flags":  strconv.Itoa(info.PublicFlags),
		},
	}, nil
}

// DiscordAPIClient provides methods to call Discord API endpoints.
// Use this after authentication to fetch guilds, roles, etc.
type DiscordAPIClient struct {
	client  *http.Client
	baseURL string
}

// NewDiscordAPIClient creates a client for calling Discord API.
// Requires a valid OAuth2 access token.
//
// Usage:
//
//	user := oauth2.UserFromContext(r.Context())
//	discordClient := oauth2.NewDiscordAPIClient(user.AccessToken)
//	guilds, err := discordClient.GetUserGuilds(r.Context())
func NewDiscordAPIClient(accessToken string) *DiscordAPIClient {
	return &DiscordAPIClient{
		client: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})),
		baseURL: "https://discord.com/api/v10",
	}
}

// DiscordGuild represents a guild (server) the user is a member of.
type DiscordGuild struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Owner       bool     `json:"owner"`
	Permissions string   `json:"permissions"` // Bitfield as string
	Features    []string `json:"features"`
}

// GetUserGuilds fetches the guilds (servers) the user is a member of.
// Requires the "guilds" scope.
func (c *DiscordAPIClient) GetUserGuilds(ctx context.Context) ([]DiscordGuild, error) {
	resp, err := c.client.Get(c.baseURL + "/users/@me/guilds")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guilds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var guilds []DiscordGuild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, fmt.Errorf("failed to decode guilds: %w", err)
	}

	return guilds, nil
}

// DiscordGuildMember represents a user's membership in a guild.
type DiscordGuildMember struct {
	User         *discordUserInfo `json:"user"`
	Nick         string           `json:"nick"`
	Avatar       string           `json:"avatar"` // Guild-specific avatar
	Roles        []string         `json:"roles"`  // Role IDs
	JoinedAt     string           `json:"joined_at"`
	PremiumSince string           `json:"premium_since"` // Nitro boost time
	Deaf         bool             `json:"deaf"`
	Mute         bool             `json:"mute"`
	Pending      bool             `json:"pending"`
}

// GetGuildMember fetches the user's membership info for a specific guild.
// Requires the "guilds.members.read" scope.
func (c *DiscordAPIClient) GetGuildMember(ctx context.Context, guildID string) (*DiscordGuildMember, error) {
	url := fmt.Sprintf("%s/users/@me/guilds/%s/member", c.baseURL, guildID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guild member: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var member DiscordGuildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode guild member: %w", err)
	}

	return &member, nil
}

// DiscordConnection represents a connected account (Twitch, YouTube, etc.).
type DiscordConnection struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"` // twitch, youtube, spotify, etc.
	Verified     bool   `json:"verified"`
	FriendSync   bool   `json:"friend_sync"`
	ShowActivity bool   `json:"show_activity"`
	Visibility   int    `json:"visibility"` // 0=none, 1=everyone
}

// GetUserConnections fetches the user's connected accounts.
// Requires the "connections" scope.
func (c *DiscordAPIClient) GetUserConnections(ctx context.Context) ([]DiscordConnection, error) {
	resp, err := c.client.Get(c.baseURL + "/users/@me/connections")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch connections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var connections []DiscordConnection
	if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
		return nil, fmt.Errorf("failed to decode connections: %w", err)
	}

	return connections, nil
}

// GetDiscordAvatarURL returns the full URL for a Discord user's avatar.
// Pass the user's ID and avatar hash from User.Extra.
// If avatar is empty, returns the default avatar URL.
func GetDiscordAvatarURL(userID, avatarHash string) string {
	if avatarHash == "" {
		// Default avatar
		id, _ := strconv.ParseInt(userID, 10, 64)
		index := int((id >> 22) % 6)
		return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", index)
	}

	ext := "png"
	if strings.HasPrefix(avatarHash, "a_") {
		ext = "gif" // Animated avatar
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", userID, avatarHash, ext)
}

// GetDiscordGuildIconURL returns the full URL for a guild's icon.
func GetDiscordGuildIconURL(guildID, iconHash string) string {
	if iconHash == "" {
		return ""
	}

	ext := "png"
	if strings.HasPrefix(iconHash, "a_") {
		ext = "gif" // Animated icon
	}
	return fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.%s", guildID, iconHash, ext)
}

// DiscordNitroType represents Discord Nitro subscription types.
type DiscordNitroType int

const (
	DiscordNitroNone    DiscordNitroType = 0
	DiscordNitroClassic DiscordNitroType = 1
	DiscordNitroFull    DiscordNitroType = 2
	DiscordNitroBasic   DiscordNitroType = 3
)

// GetDiscordNitroType returns the user's Nitro subscription type.
func GetDiscordNitroType(user *User) DiscordNitroType {
	premiumType, _ := strconv.Atoi(user.Extra["premium_type"])
	return DiscordNitroType(premiumType)
}

// HasDiscordNitro returns true if the user has any Nitro subscription.
func HasDiscordNitro(user *User) bool {
	return GetDiscordNitroType(user) != DiscordNitroNone
}

// IsDiscordMFAEnabled returns true if the user has MFA enabled.
func IsDiscordMFAEnabled(user *User) bool {
	return user.Extra["mfa_enabled"] == "true"
}

// IsInDiscordGuild checks if the user is a member of a specific guild.
// Requires guilds to be fetched via DiscordAPIClient.GetUserGuilds().
func IsInDiscordGuild(guilds []DiscordGuild, guildID string) bool {
	for _, g := range guilds {
		if g.ID == guildID {
			return true
		}
	}
	return false
}

// IsDiscordGuildOwner checks if the user owns a specific guild.
func IsDiscordGuildOwner(guilds []DiscordGuild, guildID string) bool {
	for _, g := range guilds {
		if g.ID == guildID && g.Owner {
			return true
		}
	}
	return false
}

// HasDiscordRole checks if the user has a specific role in a guild.
// Requires guild member info fetched via DiscordAPIClient.GetGuildMember().
func HasDiscordRole(member *DiscordGuildMember, roleID string) bool {
	for _, r := range member.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}

// HasAnyDiscordRole checks if the user has any of the specified roles.
func HasAnyDiscordRole(member *DiscordGuildMember, roleIDs ...string) bool {
	roleSet := make(map[string]bool)
	for _, r := range member.Roles {
		roleSet[r] = true
	}
	for _, id := range roleIDs {
		if roleSet[id] {
			return true
		}
	}
	return false
}

// HasAllDiscordRoles checks if the user has all of the specified roles.
func HasAllDiscordRoles(member *DiscordGuildMember, roleIDs ...string) bool {
	roleSet := make(map[string]bool)
	for _, r := range member.Roles {
		roleSet[r] = true
	}
	for _, id := range roleIDs {
		if !roleSet[id] {
			return false
		}
	}
	return true
}

// Discord permission flags (partial list of common permissions).
const (
	DiscordPermissionAdministrator    = 1 << 3
	DiscordPermissionManageGuild      = 1 << 5
	DiscordPermissionManageRoles      = 1 << 28
	DiscordPermissionManageChannels   = 1 << 4
	DiscordPermissionKickMembers      = 1 << 1
	DiscordPermissionBanMembers       = 1 << 2
	DiscordPermissionManageMessages   = 1 << 13
	DiscordPermissionMentionEveryone  = 1 << 17
	DiscordPermissionViewAuditLog     = 1 << 7
	DiscordPermissionManageWebhooks   = 1 << 29
	DiscordPermissionManageEmojis     = 1 << 30
	DiscordPermissionModerateMembers  = 1 << 40
)

// HasDiscordGuildPermission checks if the user has a specific permission in a guild.
// The permissions parameter should be the permissions string from DiscordGuild.
func HasDiscordGuildPermission(permissions string, permission int64) bool {
	perms, err := strconv.ParseInt(permissions, 10, 64)
	if err != nil {
		return false
	}
	// Administrator permission grants all permissions
	if perms&DiscordPermissionAdministrator != 0 {
		return true
	}
	return perms&permission != 0
}

// IsDiscordGuildAdmin checks if the user has administrator permission in a guild.
func IsDiscordGuildAdmin(guild *DiscordGuild) bool {
	return HasDiscordGuildPermission(guild.Permissions, DiscordPermissionAdministrator)
}

// IsDiscordGuildModerator checks if the user has moderation permissions in a guild.
// This checks for kick, ban, or moderate members permissions.
func IsDiscordGuildModerator(guild *DiscordGuild) bool {
	perms, err := strconv.ParseInt(guild.Permissions, 10, 64)
	if err != nil {
		return false
	}
	// Administrator grants all permissions
	if perms&DiscordPermissionAdministrator != 0 {
		return true
	}
	// Check for moderation permissions
	modPerms := int64(DiscordPermissionKickMembers | DiscordPermissionBanMembers | DiscordPermissionModerateMembers)
	return perms&modPerms != 0
}
