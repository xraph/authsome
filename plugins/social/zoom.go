package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// zoomEndpoint is the OAuth2 endpoint for Zoom.
var zoomEndpoint = oauth2.Endpoint{ //nolint:gosec // G101: not credentials, OAuth endpoint
	AuthURL:  "https://zoom.us/oauth/authorize",
	TokenURL: "https://zoom.us/oauth/token",
}

// zoomProvider implements Provider for Zoom OAuth2.
type zoomProvider struct {
	config *oauth2.Config
}

// NewZoomProvider creates a new Zoom OAuth2 provider.
func NewZoomProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:read:email", "user:read:user"}
	}
	return &zoomProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     zoomEndpoint,
		},
	}
}

func (p *zoomProvider) Name() string                 { return "zoom" }
func (p *zoomProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *zoomProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.zoom.us/v2/users/me", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: zoom: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: zoom: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: zoom: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		PicURL    string `json:"pic_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: zoom: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: info.ID,
		Email:          info.Email,
		FirstName:      info.FirstName,
		LastName:       info.LastName,
		AvatarURL:      info.PicURL,
	}, nil
}
