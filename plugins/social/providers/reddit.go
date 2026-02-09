package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// RedditProvider implements OAuth for Reddit.
type RedditProvider struct {
	*BaseProvider
}

// NewRedditProvider creates a new Reddit OAuth provider.
func NewRedditProvider(config ProviderConfig) *RedditProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identity"}
	}

	bp := NewBaseProvider(
		"reddit",
		"Reddit",
		"https://www.reddit.com/api/v1/authorize",
		"https://www.reddit.com/api/v1/access_token",
		"https://oauth.reddit.com/api/v1/me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &RedditProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Reddit API.
func (r *RedditProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := r.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, r.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Reddit user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if name, ok := raw["name"].(string); ok {
		userInfo.Username = name
		userInfo.Name = name
	}

	// Reddit doesn't provide email or avatar via OAuth by default
	// Icon images are in icon_img field
	if iconImg, ok := raw["icon_img"].(string); ok {
		userInfo.Avatar = iconImg
	}

	return userInfo, nil
}
