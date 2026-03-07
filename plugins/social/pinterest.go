package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// pinterestEndpoint is the OAuth2 endpoint for Pinterest.
var pinterestEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.pinterest.com/oauth/",
	TokenURL: "https://api.pinterest.com/v5/oauth/token",
}

// pinterestProvider implements Provider for Pinterest OAuth2.
type pinterestProvider struct {
	config *oauth2.Config
}

// NewPinterestProvider creates a new Pinterest OAuth2 provider.
func NewPinterestProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user_accounts:read"}
	}
	return &pinterestProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     pinterestEndpoint,
		},
	}
}

func (p *pinterestProvider) Name() string                 { return "pinterest" }
func (p *pinterestProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *pinterestProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.pinterest.com/v5/user_account")
	if err != nil {
		return nil, fmt.Errorf("social: pinterest: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: pinterest: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		Username         string `json:"username"`
		ProfileImage     string `json:"profile_image"`
		BusinessName     string `json:"business_name"`
		WebsiteURL       string `json:"website_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: pinterest: decode user: %w", err)
	}

	name := info.BusinessName
	if name == "" {
		name = info.Username
	}

	return &ProviderUser{
		ProviderUserID: info.Username,
		FirstName:      name,
		AvatarURL:      info.ProfileImage,
	}, nil
}
