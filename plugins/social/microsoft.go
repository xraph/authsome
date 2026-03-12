package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type microsoftProvider struct {
	config *oauth2.Config
}

// NewMicrosoftProvider creates a Microsoft OAuth2 provider.
// Uses the Microsoft identity platform (v2.0) with the "common" tenant
// (supports both personal and work/school accounts).
func NewMicrosoftProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile", "User.Read"}
	}

	return &microsoftProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     microsoft.AzureADEndpoint("common"),
		},
	}
}

func (p *microsoftProvider) Name() string { return "microsoft" }

func (p *microsoftProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *microsoftProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/me", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("microsoft: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024)) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("microsoft: API error %d: %s", resp.StatusCode, string(body))
	}

	var profile struct {
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		GivenName         string `json:"givenName"`
		Surname           string `json:"surname"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("microsoft: decode profile: %w", err)
	}

	email := profile.Mail
	if email == "" {
		email = profile.UserPrincipalName
	}

	firstName := profile.GivenName
	lastName := profile.Surname
	if firstName == "" && lastName == "" {
		firstName = profile.DisplayName
	}

	return &ProviderUser{
		ProviderUserID: profile.ID,
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
	}, nil
}
