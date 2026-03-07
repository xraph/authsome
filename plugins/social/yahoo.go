package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// yahooEndpoint is the OAuth2 endpoint for Yahoo.
var yahooEndpoint = oauth2.Endpoint{
	AuthURL:  "https://api.login.yahoo.com/oauth2/request_auth",
	TokenURL: "https://api.login.yahoo.com/oauth2/get_token",
}

// yahooProvider implements Provider for Yahoo OAuth2.
type yahooProvider struct {
	config *oauth2.Config
}

// NewYahooProvider creates a new Yahoo OAuth2 provider.
func NewYahooProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return &yahooProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     yahooEndpoint,
		},
	}
}

func (p *yahooProvider) Name() string                 { return "yahoo" }
func (p *yahooProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *yahooProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.login.yahoo.com/openid/v1/userinfo")
	if err != nil {
		return nil, fmt.Errorf("social: yahoo: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: yahoo: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		Sub        string `json:"sub"`
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		Email      string `json:"email"`
		Picture    string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: yahoo: decode user: %w", err)
	}

	firstName := info.GivenName
	if firstName == "" {
		firstName = info.Name
	}

	return &ProviderUser{
		ProviderUserID: info.Sub,
		Email:          info.Email,
		FirstName:      firstName,
		LastName:       info.FamilyName,
		AvatarURL:      info.Picture,
	}, nil
}
