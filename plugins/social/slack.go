package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthslack "golang.org/x/oauth2/slack"
)

// slackProvider implements Provider for Slack OAuth2.
type slackProvider struct {
	config *oauth2.Config
}

// NewSlackProvider creates a new Slack OAuth2 provider.
func NewSlackProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	return &slackProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthslack.Endpoint,
		},
	}
}

func (p *slackProvider) Name() string                 { return "slack" }
func (p *slackProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *slackProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://slack.com/api/openid.connect.userInfo")
	if err != nil {
		return nil, fmt.Errorf("social: slack: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: slack: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		OK      bool   `json:"ok"`
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: slack: decode user: %w", err)
	}
	if !info.OK {
		return nil, fmt.Errorf("social: slack: API error: %s", info.Error)
	}

	return &ProviderUser{
		ProviderUserID: info.Sub,
		Email:          info.Email,
		FirstName:      info.Name,
		AvatarURL:      info.Picture,
	}, nil
}
