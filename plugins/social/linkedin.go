package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

// linkedinProvider implements Provider for LinkedIn OAuth2.
type linkedinProvider struct {
	config *oauth2.Config
}

// NewLinkedInProvider creates a new LinkedIn OAuth2 provider.
func NewLinkedInProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return &linkedinProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     linkedin.Endpoint,
		},
	}
}

func (p *linkedinProvider) Name() string                 { return "linkedin" }
func (p *linkedinProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *linkedinProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.linkedin.com/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("social: linkedin: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: linkedin: fetch user: status %d: %s", resp.StatusCode, body)
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
		return nil, fmt.Errorf("social: linkedin: decode user: %w", err)
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
