package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// DropboxProvider implements OAuth for Dropbox
type DropboxProvider struct {
	*BaseProvider
}

// NewDropboxProvider creates a new Dropbox OAuth provider
func NewDropboxProvider(config ProviderConfig) *DropboxProvider {
	scopes := config.Scopes
	// Dropbox doesn't use traditional scopes in OAuth 2.0

	bp := NewBaseProvider(
		"dropbox",
		"Dropbox",
		"https://www.dropbox.com/oauth2/authorize",
		"https://api.dropboxapi.com/oauth2/token",
		"https://api.dropboxapi.com/2/users/get_current_account",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &DropboxProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Dropbox API
func (d *DropboxProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := d.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	// Dropbox requires POST for user info
	if err := PostForm(ctx, client, d.userInfoURL, nil, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Dropbox user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw:           raw,
		EmailVerified: true, // Dropbox emails are verified
	}

	if accountID, ok := raw["account_id"].(string); ok {
		userInfo.ID = accountID
	}
	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	// Dropbox uses name structure
	if name, ok := raw["name"].(map[string]interface{}); ok {
		if displayName, ok := name["display_name"].(string); ok {
			userInfo.Name = displayName
		}
		if givenName, ok := name["given_name"].(string); ok {
			userInfo.FirstName = givenName
		}
		if surname, ok := name["surname"].(string); ok {
			userInfo.LastName = surname
		}
	}

	if profilePhotoURL, ok := raw["profile_photo_url"].(string); ok {
		userInfo.Avatar = profilePhotoURL
	}

	return userInfo, nil
}
