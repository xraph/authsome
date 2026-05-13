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

	// GitHub's /user endpoint only returns the user's *public* email. Accounts
	// that keep their email private (the default) return an empty string here.
	// Fall back to /user/emails to fetch the primary verified email. This
	// requires the "user:email" scope (already requested by default above).
	email := info.Email
	if email == "" {
		if primary, emailErr := p.fetchPrimaryEmail(ctx, client); emailErr == nil {
			email = primary
		}
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	return &ProviderUser{
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		Email:          email,
		FirstName:      name,
		AvatarURL:      info.AvatarURL,
	}, nil
}

// fetchPrimaryEmail calls GitHub's /user/emails endpoint and returns the
// user's primary verified email. Requires the "user:email" OAuth scope.
// Returns an empty string (no error) if no verified primary is available.
func (p *githubProvider) fetchPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("social: github: create emails request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("social: github: fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return "", fmt.Errorf("social: github: fetch emails: status %d: %s", resp.StatusCode, body)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("social: github: decode emails: %w", err)
	}

	// Prefer primary + verified.
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	// Fall back to any verified email.
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}
