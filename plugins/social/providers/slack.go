package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/slack"
)

// SlackProvider implements OAuth for Slack
type SlackProvider struct {
	*BaseProvider
}

// NewSlackProvider creates a new Slack OAuth provider
func NewSlackProvider(config ProviderConfig) *SlackProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identity.basic", "identity.email", "identity.avatar"}
	}

	bp := NewBaseProvider(
		"slack",
		"Slack",
		slack.Endpoint.AuthURL,
		slack.Endpoint.TokenURL,
		"https://slack.com/api/users.identity",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &SlackProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Slack API
func (s *SlackProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := s.oauth2Config.Client(ctx, token)

	var response struct {
		OK   bool                   `json:"ok"`
		User map[string]interface{} `json:"user"`
	}

	if err := FetchJSON(ctx, client, s.userInfoURL, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch Slack user info: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API returned ok=false")
	}

	raw := response.User
	userInfo := &UserInfo{
		Raw:           raw,
		EmailVerified: true, // Slack emails are verified
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}
	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}
	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}

	// Slack avatar in image fields
	if image192, ok := raw["image_192"].(string); ok {
		userInfo.Avatar = image192
	} else if image512, ok := raw["image_512"].(string); ok {
		userInfo.Avatar = image512
	}

	return userInfo, nil
}
