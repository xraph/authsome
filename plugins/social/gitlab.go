package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthgitlab "golang.org/x/oauth2/gitlab"
)

// gitlabProvider implements Provider for GitLab OAuth2.
type gitlabProvider struct {
	config *oauth2.Config
}

// NewGitLabProvider creates a new GitLab OAuth2 provider.
func NewGitLabProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read_user", "openid", "email"}
	}
	return &gitlabProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthgitlab.Endpoint,
		},
	}
}

func (p *gitlabProvider) Name() string                 { return "gitlab" }
func (p *gitlabProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *gitlabProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://gitlab.com/api/v4/user", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: gitlab: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: gitlab: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: gitlab: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: gitlab: decode user: %w", err)
	}

	name := info.Name
	if name == "" {
		name = info.Username
	}

	return &ProviderUser{
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		Email:          info.Email,
		FirstName:      name,
		AvatarURL:      info.AvatarURL,
	}, nil
}
