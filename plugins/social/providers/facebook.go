package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

// FacebookProvider implements OAuth for Facebook
type FacebookProvider struct {
	*BaseProvider
}

// NewFacebookProvider creates a new Facebook OAuth provider
func NewFacebookProvider(config ProviderConfig) *FacebookProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"email", "public_profile"}
	}

	bp := NewBaseProvider(
		"facebook",
		"Facebook",
		facebook.Endpoint.AuthURL,
		facebook.Endpoint.TokenURL,
		"https://graph.facebook.com/me?fields=id,name,email,first_name,last_name,picture",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &FacebookProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Facebook Graph API
func (f *FacebookProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := f.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	if err := FetchJSON(ctx, client, f.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Facebook user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}
	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
		userInfo.EmailVerified = true // Facebook emails are verified
	}
	if name, ok := raw["name"].(string); ok {
		userInfo.Name = name
	}
	if firstName, ok := raw["first_name"].(string); ok {
		userInfo.FirstName = firstName
	}
	if lastName, ok := raw["last_name"].(string); ok {
		userInfo.LastName = lastName
	}
	if picture, ok := raw["picture"].(map[string]interface{}); ok {
		if data, ok := picture["data"].(map[string]interface{}); ok {
			if url, ok := data["url"].(string); ok {
				userInfo.Avatar = url
			}
		}
	}

	return userInfo, nil
}
