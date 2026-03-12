package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// twitterEndpoint is the OAuth2 endpoint for Twitter/X.
var twitterEndpoint = oauth2.Endpoint{ //nolint:gosec // G101: not credentials, OAuth endpoint
	AuthURL:  "https://twitter.com/i/oauth2/authorize",
	TokenURL: "https://api.x.com/2/oauth2/token",
}

// twitterProvider implements Provider for Twitter/X OAuth2.
type twitterProvider struct {
	config *oauth2.Config
}

// NewTwitterProvider creates a new Twitter/X OAuth2 provider.
func NewTwitterProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"users.read", "tweet.read"}
	}
	return &twitterProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     twitterEndpoint,
		},
	}
}

func (p *twitterProvider) Name() string                 { return "twitter" }
func (p *twitterProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *twitterProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.x.com/2/users/me?user.fields=profile_image_url,name,username", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: twitter: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: twitter: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: twitter: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Data struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Username        string `json:"username"`
			ProfileImageURL string `json:"profile_image_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("social: twitter: decode user: %w", err)
	}

	name := result.Data.Name
	if name == "" {
		name = result.Data.Username
	}

	return &ProviderUser{
		ProviderUserID: result.Data.ID,
		FirstName:      name,
		AvatarURL:      result.Data.ProfileImageURL,
	}, nil
}
