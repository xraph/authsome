package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct {
	*BaseProvider
}

// NewGitHubProvider creates a new GitHub OAuth provider
func NewGitHubProvider(config ProviderConfig) *GitHubProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:email", "read:user"}
	}

	bp := NewBaseProvider(
		"github",
		"GitHub",
		github.Endpoint.AuthURL,
		github.Endpoint.TokenURL,
		"https://api.github.com/user",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &GitHubProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from GitHub
func (gh *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := gh.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	if err := FetchJSON(ctx, client, gh.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}

	// Fetch emails separately if needed
	var emails []map[string]interface{}
	if err := FetchJSON(ctx, client, "https://api.github.com/user/emails", &emails); err == nil {
		// Find primary verified email
		for _, e := range emails {
			if primary, ok := e["primary"].(bool); ok && primary {
				if verified, ok := e["verified"].(bool); ok && verified {
					if email, ok := e["email"].(string); ok {
						raw["email"] = email
						raw["email_verified"] = true
					}
				}
			}
		}
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
	if verified, ok := raw["email_verified"].(bool); ok {
		userInfo.EmailVerified = verified
	}
	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}
	if login, ok := raw["login"].(string); ok {
		userInfo.Username = login
	}
	if avatarURL, ok := raw["avatar_url"].(string); ok {
		userInfo.Avatar = avatarURL
	}

	return userInfo, nil
}
