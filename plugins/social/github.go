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

	// GitHub's /user endpoint only returns the user's *public* email (and
	// without a verification flag). Fetch the full address list from
	// /user/emails so we get every verified address and can pick the primary.
	// This requires the "user:email" scope (already requested by default).
	emails, _ := p.fetchEmails(ctx, client) //nolint:errcheck // best-effort; falls back to public email

	// Choose the primary email for the back-compat Email field: prefer the
	// provider's primary+verified address, else any verified, else the public
	// /user email (verification unknown -> treated as unverified).
	var primaryEmail string
	var primaryVerified bool
	for _, e := range emails {
		if e.Primary && e.Verified {
			primaryEmail, primaryVerified = e.Email, true
			break
		}
	}
	if primaryEmail == "" {
		for _, e := range emails {
			if e.Verified {
				primaryEmail, primaryVerified = e.Email, true
				break
			}
		}
	}
	if primaryEmail == "" && info.Email != "" {
		// Public email with unknown verification — record it but unverified so
		// it can't auto-link to an existing account.
		primaryEmail = info.Email
		emails = append(emails, ProviderEmail{Email: info.Email, Verified: false, Primary: true})
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	return &ProviderUser{
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		Email:          primaryEmail,
		EmailVerified:  primaryVerified,
		Emails:         emails,
		FirstName:      name,
		AvatarURL:      info.AvatarURL,
	}, nil
}

// fetchEmails calls GitHub's /user/emails endpoint and returns every address
// on the account with its verified/primary flags. Requires the "user:email"
// OAuth scope. Returns nil (no error) when the endpoint is unavailable.
func (p *githubProvider) fetchEmails(ctx context.Context, client *http.Client) ([]ProviderEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: github: create emails request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: github: fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: github: fetch emails: status %d: %s", resp.StatusCode, body)
	}

	var raw []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("social: github: decode emails: %w", err)
	}

	out := make([]ProviderEmail, 0, len(raw))
	for _, e := range raw {
		out = append(out, ProviderEmail{Email: e.Email, Verified: e.Verified, Primary: e.Primary})
	}
	return out, nil
}
