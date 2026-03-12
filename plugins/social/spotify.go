package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthspotify "golang.org/x/oauth2/spotify"
)

// spotifyProvider implements Provider for Spotify OAuth2.
type spotifyProvider struct {
	config *oauth2.Config
}

// NewSpotifyProvider creates a new Spotify OAuth2 provider.
func NewSpotifyProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user-read-email", "user-read-private"}
	}
	return &spotifyProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthspotify.Endpoint,
		},
	}
}

func (p *spotifyProvider) Name() string                 { return "spotify" }
func (p *spotifyProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *spotifyProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.spotify.com/v1/me", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("social: spotify: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: spotify: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: spotify: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Images      []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: spotify: decode user: %w", err)
	}

	var avatarURL string
	if len(info.Images) > 0 {
		avatarURL = info.Images[0].URL
	}

	return &ProviderUser{
		ProviderUserID: info.ID,
		Email:          info.Email,
		FirstName:      info.DisplayName,
		AvatarURL:      avatarURL,
	}, nil
}
