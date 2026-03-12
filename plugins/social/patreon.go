package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// patreonEndpoint is the OAuth2 endpoint for Patreon.
var patreonEndpoint = oauth2.Endpoint{ //nolint:gosec // G101: not credentials, OAuth endpoint
	AuthURL:  "https://www.patreon.com/oauth2/authorize",
	TokenURL: "https://www.patreon.com/api/oauth2/token",
}

// patreonProvider implements Provider for Patreon OAuth2.
type patreonProvider struct {
	config *oauth2.Config
}

// NewPatreonProvider creates a new Patreon OAuth2 provider.
func NewPatreonProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identity", "identity[email]"}
	}
	return &patreonProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     patreonEndpoint,
		},
	}
}

func (p *patreonProvider) Name() string                 { return "patreon" }
func (p *patreonProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *patreonProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.patreon.com/api/oauth2/v2/identity?fields%5Buser%5D=email,first_name,last_name,image_url,full_name", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: patreon: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: patreon: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: patreon: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				FullName  string `json:"full_name"`
				ImageURL  string `json:"image_url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("social: patreon: decode user: %w", err)
	}

	firstName := result.Data.Attributes.FirstName
	if firstName == "" {
		firstName = result.Data.Attributes.FullName
	}

	return &ProviderUser{
		ProviderUserID: result.Data.ID,
		Email:          result.Data.Attributes.Email,
		FirstName:      firstName,
		LastName:       result.Data.Attributes.LastName,
		AvatarURL:      result.Data.Attributes.ImageURL,
	}, nil
}
