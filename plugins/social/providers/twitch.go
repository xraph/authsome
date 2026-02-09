package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// TwitchProvider implements OAuth for Twitch.
type TwitchProvider struct {
	*BaseProvider
}

// NewTwitchProvider creates a new Twitch OAuth provider.
func NewTwitchProvider(config ProviderConfig) *TwitchProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:read:email"}
	}

	bp := NewBaseProvider(
		"twitch",
		"Twitch",
		twitch.Endpoint.AuthURL,
		twitch.Endpoint.TokenURL,
		"https://api.twitch.tv/helix/users",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &TwitchProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Twitch API.
func (t *TwitchProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := t.oauth2Config.Client(ctx, token)

	var response struct {
		Data []map[string]any `json:"data"`
	}

	if err := FetchJSON(ctx, client, t.userInfoURL, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch Twitch user info: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, errs.New(errs.CodeNotFound, "no user data returned from Twitch", http.StatusNotFound)
	}

	raw := response.Data[0]
	userInfo := &UserInfo{
		Raw:           raw,
		EmailVerified: true, // Twitch emails are verified
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	if displayName, ok := raw["display_name"].(string); ok {
		userInfo.Name = displayName
	}

	if login, ok := raw["login"].(string); ok {
		userInfo.Username = login
	}

	if avatar, ok := raw["profile_image_url"].(string); ok {
		userInfo.Avatar = avatar
	}

	return userInfo, nil
}
