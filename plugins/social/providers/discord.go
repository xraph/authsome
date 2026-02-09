package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// DiscordProvider implements OAuth for Discord.
type DiscordProvider struct {
	*BaseProvider
}

// NewDiscordProvider creates a new Discord OAuth provider.
func NewDiscordProvider(config ProviderConfig) *DiscordProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}

	bp := NewBaseProvider(
		"discord",
		"Discord",
		"https://discord.com/api/oauth2/authorize",
		"https://discord.com/api/oauth2/token",
		"https://discord.com/api/users/@me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &DiscordProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Discord API.
func (d *DiscordProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := d.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, d.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Discord user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	if verified, ok := raw["verified"].(bool); ok {
		userInfo.EmailVerified = verified
	}

	if username, ok := raw["username"].(string); ok {
		userInfo.Username = username
	}

	if globalName, ok := raw["global_name"].(string); ok {
		userInfo.Name = globalName
	} else {
		userInfo.Name = userInfo.Username
	}

	// Discord avatar construction
	if avatar, ok := raw["avatar"].(string); ok && avatar != "" {
		userInfo.Avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userInfo.ID, avatar)
	}

	return userInfo, nil
}
