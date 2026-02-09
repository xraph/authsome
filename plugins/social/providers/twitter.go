package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// TwitterProvider implements OAuth 2.0 for Twitter (X).
type TwitterProvider struct {
	*BaseProvider
}

// NewTwitterProvider creates a new Twitter OAuth provider.
func NewTwitterProvider(config ProviderConfig) *TwitterProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"tweet.read", "users.read", "offline.access"}
	}

	bp := NewBaseProvider(
		"twitter",
		"Twitter",
		"https://twitter.com/i/oauth2/authorize",
		"https://api.twitter.com/2/oauth2/token",
		"https://api.twitter.com/2/users/me?user.fields=profile_image_url,name,username",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &TwitterProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Twitter API v2.
func (t *TwitterProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := t.oauth2Config.Client(ctx, token)

	var response struct {
		Data map[string]any `json:"data"`
	}

	if err := FetchJSON(ctx, client, t.userInfoURL, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch Twitter user info: %w", err)
	}

	raw := response.Data
	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}

	if username, ok := raw["username"].(string); ok {
		userInfo.Username = username
	}

	if avatar, ok := raw["profile_image_url"].(string); ok {
		userInfo.Avatar = avatar
	}

	// Twitter OAuth 2.0 doesn't provide email by default
	// Would need additional verification step

	return userInfo, nil
}
