package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthbitbucket "golang.org/x/oauth2/bitbucket"
)

// bitbucketProvider implements Provider for Bitbucket OAuth2.
type bitbucketProvider struct {
	config *oauth2.Config
}

// NewBitbucketProvider creates a new Bitbucket OAuth2 provider.
func NewBitbucketProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"account", "email"}
	}
	return &bitbucketProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     oauthbitbucket.Endpoint,
		},
	}
}

func (p *bitbucketProvider) Name() string                 { return "bitbucket" }
func (p *bitbucketProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *bitbucketProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)

	// Fetch user profile.
	resp, err := client.Get("https://api.bitbucket.org/2.0/user")
	if err != nil {
		return nil, fmt.Errorf("social: bitbucket: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: bitbucket: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		UUID        string `json:"uuid"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: bitbucket: decode user: %w", err)
	}

	// Fetch primary email.
	email := p.fetchPrimaryEmail(ctx, client)

	name := info.DisplayName
	if name == "" {
		name = info.Username
	}

	return &ProviderUser{
		ProviderUserID: info.UUID,
		Email:          email,
		FirstName:      name,
		AvatarURL:      info.Links.Avatar.Href,
	}, nil
}

func (p *bitbucketProvider) fetchPrimaryEmail(ctx context.Context, client *http.Client) string {
	resp, err := client.Get("https://api.bitbucket.org/2.0/user/emails")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Values []struct {
			Email     string `json:"email"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	for _, v := range result.Values {
		if v.IsPrimary {
			return v.Email
		}
	}
	if len(result.Values) > 0 {
		return result.Values[0].Email
	}
	return ""
}
