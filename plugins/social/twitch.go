package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthtwitch "golang.org/x/oauth2/twitch"
)

// twitchProvider implements Provider for Twitch OAuth2.
type twitchProvider struct {
	config *oauth2.Config
}

// NewTwitchProvider creates a new Twitch OAuth2 provider.
func NewTwitchProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:read:email"}
	}
	return &twitchProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthtwitch.Endpoint,
		},
	}
}

func (p *twitchProvider) Name() string                 { return "twitch" }
func (p *twitchProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *twitchProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		return nil, fmt.Errorf("social: twitch: create request: %w", err)
	}
	req.Header.Set("Client-ID", p.config.ClientID)

	client := p.config.Client(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: twitch: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: twitch: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Data []struct {
			ID              string `json:"id"`
			Login           string `json:"login"`
			DisplayName     string `json:"display_name"`
			Email           string `json:"email"`
			ProfileImageURL string `json:"profile_image_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("social: twitch: decode user: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("social: twitch: no user data returned")
	}

	user := result.Data[0]
	name := user.DisplayName
	if name == "" {
		name = user.Login
	}

	return &ProviderUser{
		ProviderUserID: user.ID,
		Email:          user.Email,
		FirstName:      name,
		AvatarURL:      user.ProfileImageURL,
	}, nil
}
