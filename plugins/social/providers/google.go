package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements OAuth for Google
type GoogleProvider struct {
	*BaseProvider
	accessType string
	prompt     string
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(config ProviderConfig) *GoogleProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		}
	}

	bp := NewBaseProvider(
		"google",
		"Google",
		google.Endpoint.AuthURL,
		google.Endpoint.TokenURL,
		"https://www.googleapis.com/oauth2/v2/userinfo",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &GoogleProvider{
		BaseProvider: bp,
		accessType:   config.AccessType,
		prompt:       config.Prompt,
	}
}

// GetOAuth2Config overrides to add Google-specific options
func (g *GoogleProvider) GetOAuth2Config() *oauth2.Config {
	return g.oauth2Config
}

// GetAuthURL returns the authorization URL with Google-specific parameters
func (g *GoogleProvider) GetAuthURL(state string) string {
	opts := []oauth2.AuthCodeOption{}

	if g.accessType != "" {
		opts = append(opts, oauth2.AccessTypeOffline)
	}

	if g.prompt != "" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", g.prompt))
	}

	return g.oauth2Config.AuthCodeURL(state, opts...)
}

// GetUserInfo fetches user information from Google
func (g *GoogleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := g.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	if err := FetchJSON(ctx, client, g.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Google user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}
	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}
	if verified, ok := raw["verified_email"].(bool); ok {
		userInfo.EmailVerified = verified
	}
	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}
	if givenName, ok := raw["given_name"].(string); ok {
		userInfo.FirstName = givenName
	}
	if familyName, ok := raw["family_name"].(string); ok {
		userInfo.LastName = familyName
	}
	if picture, ok := raw["picture"].(string); ok {
		userInfo.Avatar = picture
	}

	return userInfo, nil
}
