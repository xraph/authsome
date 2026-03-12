package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthamazon "golang.org/x/oauth2/amazon"
)

// amazonProvider implements Provider for Amazon OAuth2.
type amazonProvider struct {
	config *oauth2.Config
}

// NewAmazonProvider creates a new Amazon OAuth2 provider.
func NewAmazonProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"profile"}
	}
	return &amazonProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthamazon.Endpoint,
		},
	}
}

func (p *amazonProvider) Name() string                 { return "amazon" }
func (p *amazonProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *amazonProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.amazon.com/user/profile", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: amazon: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: amazon: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: amazon: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: amazon: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: info.UserID,
		Email:          info.Email,
		FirstName:      info.Name,
	}, nil
}
