package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// dropboxEndpoint is the OAuth2 endpoint for Dropbox.
var dropboxEndpoint = oauth2.Endpoint{ //nolint:gosec // G101: not credentials, OAuth endpoint
	AuthURL:  "https://www.dropbox.com/oauth2/authorize",
	TokenURL: "https://api.dropboxapi.com/oauth2/token",
}

// dropboxProvider implements Provider for Dropbox OAuth2.
type dropboxProvider struct {
	config *oauth2.Config
}

// NewDropboxProvider creates a new Dropbox OAuth2 provider.
func NewDropboxProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"account_info.read"}
	}
	return &dropboxProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     dropboxEndpoint,
		},
	}
}

func (p *dropboxProvider) Name() string                 { return "dropbox" }
func (p *dropboxProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *dropboxProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	// Dropbox uses POST with null body for get_current_account.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.dropboxapi.com/2/users/get_current_account", bytes.NewReader([]byte("null")))
	if err != nil {
		return nil, fmt.Errorf("social: dropbox: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := p.config.Client(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("social: dropbox: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return nil, fmt.Errorf("social: dropbox: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		AccountID string `json:"account_id"`
		Email     string `json:"email"`
		Name      struct {
			GivenName   string `json:"given_name"`
			Surname     string `json:"surname"`
			DisplayName string `json:"display_name"`
		} `json:"name"`
		ProfilePhotoURL string `json:"profile_photo_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: dropbox: decode user: %w", err)
	}

	firstName := info.Name.GivenName
	if firstName == "" {
		firstName = info.Name.DisplayName
	}

	return &ProviderUser{
		ProviderUserID: info.AccountID,
		Email:          info.Email,
		FirstName:      firstName,
		LastName:       info.Name.Surname,
		AvatarURL:      info.ProfilePhotoURL,
	}, nil
}
