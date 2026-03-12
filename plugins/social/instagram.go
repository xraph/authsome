package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// instagramEndpoint is the OAuth2 endpoint for Instagram (Facebook Login).
var instagramEndpoint = oauth2.Endpoint{ //nolint:gosec // G101: not credentials, OAuth endpoint
	AuthURL:  "https://www.facebook.com/v21.0/dialog/oauth",
	TokenURL: "https://graph.facebook.com/v21.0/oauth/access_token",
}

// instagramProvider implements Provider for Instagram OAuth2 (via Facebook Graph API).
type instagramProvider struct {
	config *oauth2.Config
}

// NewInstagramProvider creates a new Instagram OAuth2 provider.
func NewInstagramProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"instagram_basic", "pages_show_list"}
	}
	return &instagramProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     instagramEndpoint,
		},
	}
}

func (p *instagramProvider) Name() string                 { return "instagram" }
func (p *instagramProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *instagramProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.facebook.com/me?fields=id,name,email,picture.type(large)", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: instagram: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: instagram: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: instagram: fetch user: status %d: %s", resp.StatusCode, body)
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
		return nil, fmt.Errorf("social: instagram: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: info.ID,
		Email:          info.Email,
		FirstName:      info.Name,
		AvatarURL:      info.Picture.Data.URL,
	}, nil
}
