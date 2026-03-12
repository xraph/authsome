package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

// facebookProvider implements Provider for Facebook OAuth2.
type facebookProvider struct {
	config *oauth2.Config
}

// NewFacebookProvider creates a new Facebook OAuth2 provider.
func NewFacebookProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"email", "public_profile"}
	}
	return &facebookProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     facebook.Endpoint,
		},
	}
}

func (p *facebookProvider) Name() string                 { return "facebook" }
func (p *facebookProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *facebookProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.facebook.com/me?fields=id,name,email,picture.type(large)", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: facebook: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: facebook: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: facebook: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: facebook: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: info.ID,
		Email:          info.Email,
		FirstName:      info.Name,
		AvatarURL:      info.Picture.Data.URL,
	}, nil
}
