package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
)

// githubProvider implements Provider for GitHub OAuth2.
type githubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider creates a new GitHub OAuth2 provider.
func NewGitHubProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:email", "read:user"}
	}
	return &githubProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthgithub.Endpoint,
		},
	}
}

func (p *githubProvider) Name() string                 { return "github" }
func (p *githubProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *githubProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: github: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: github: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: github: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: github: decode user: %w", err)
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	return &ProviderUser{
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		Email:          info.Email,
		FirstName:      name,
		AvatarURL:      info.AvatarURL,
	}, nil
}
