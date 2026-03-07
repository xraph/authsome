package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// lineEndpoint is the OAuth2 endpoint for LINE.
var lineEndpoint = oauth2.Endpoint{
	AuthURL:  "https://access.line.me/oauth2/v2.1/authorize",
	TokenURL: "https://api.line.me/oauth2/v2.1/token",
}

// lineProvider implements Provider for LINE OAuth2.
type lineProvider struct {
	config *oauth2.Config
}

// NewLineProvider creates a new LINE OAuth2 provider.
func NewLineProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"profile", "openid", "email"}
	}
	return &lineProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     lineEndpoint,
		},
	}
}

func (p *lineProvider) Name() string                 { return "line" }
func (p *lineProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *lineProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.line.me/v2/profile")
	if err != nil {
		return nil, fmt.Errorf("social: line: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: line: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
		PictureURL  string `json:"pictureUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: line: decode user: %w", err)
	}

	return &ProviderUser{
		ProviderUserID: info.UserID,
		FirstName:      info.DisplayName,
		AvatarURL:      info.PictureURL,
	}, nil
}
