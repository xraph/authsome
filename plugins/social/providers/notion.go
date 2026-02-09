package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// NotionProvider implements OAuth for Notion.
type NotionProvider struct {
	*BaseProvider
}

// NewNotionProvider creates a new Notion OAuth provider.
func NewNotionProvider(config ProviderConfig) *NotionProvider {
	scopes := config.Scopes
	// Notion doesn't use traditional scopes

	bp := NewBaseProvider(
		"notion",
		"Notion",
		"https://api.notion.com/v1/oauth/authorize",
		"https://api.notion.com/v1/oauth/token",
		"https://api.notion.com/v1/users/me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &NotionProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Notion API.
func (n *NotionProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := n.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, n.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Notion user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}

	// Notion user object structure
	if person, ok := raw["person"].(map[string]any); ok {
		if email, ok := person["email"].(string); ok {
			userInfo.Email = email
			userInfo.EmailVerified = true // Notion emails are verified
		}
	}

	if avatarURL, ok := raw["avatar_url"].(string); ok {
		userInfo.Avatar = avatarURL
	}

	return userInfo, nil
}
