package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// GitLabProvider implements OAuth for GitLab.
type GitLabProvider struct {
	*BaseProvider
}

// NewGitLabProvider creates a new GitLab OAuth provider.
func NewGitLabProvider(config ProviderConfig) *GitLabProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read_user", "email"}
	}

	bp := NewBaseProvider(
		"gitlab",
		"GitLab",
		"https://gitlab.com/oauth/authorize",
		"https://gitlab.com/oauth/token",
		"https://gitlab.com/api/v4/user",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &GitLabProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from GitLab API.
func (g *GitLabProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := g.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, g.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch GitLab user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(float64); ok {
		userInfo.ID = fmt.Sprintf("%.0f", id)
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	if confirmed, ok := raw["confirmed_at"].(string); ok && confirmed != "" {
		userInfo.EmailVerified = true
	}

	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}

	if username, ok := raw["username"].(string); ok {
		userInfo.Username = username
	}

	if avatar, ok := raw["avatar_url"].(string); ok {
		userInfo.Avatar = avatar
	}

	return userInfo, nil
}
