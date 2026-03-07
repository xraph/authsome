package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// stravaEndpoint is the OAuth2 endpoint for Strava.
var stravaEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.strava.com/oauth/authorize",
	TokenURL: "https://www.strava.com/oauth/token",
}

// stravaProvider implements Provider for Strava OAuth2.
type stravaProvider struct {
	config *oauth2.Config
}

// NewStravaProvider creates a new Strava OAuth2 provider.
func NewStravaProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read"}
	}
	return &stravaProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     stravaEndpoint,
		},
	}
}

func (p *stravaProvider) Name() string                 { return "strava" }
func (p *stravaProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *stravaProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://www.strava.com/api/v3/athlete")
	if err != nil {
		return nil, fmt.Errorf("social: strava: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: strava: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID        int64  `json:"id"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
		Profile   string `json:"profile"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: strava: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: fmt.Sprintf("%d", info.ID),
		FirstName:      info.FirstName,
		LastName:       info.LastName,
		AvatarURL:      info.Profile,
	}, nil
}
