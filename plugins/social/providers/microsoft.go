package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// MicrosoftProvider implements OAuth for Microsoft
type MicrosoftProvider struct {
	*BaseProvider
}

// NewMicrosoftProvider creates a new Microsoft OAuth provider
func NewMicrosoftProvider(config ProviderConfig) *MicrosoftProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"User.Read", "openid", "profile", "email"}
	}

	bp := NewBaseProvider(
		"microsoft",
		"Microsoft",
		microsoft.AzureADEndpoint("common").AuthURL,
		microsoft.AzureADEndpoint("common").TokenURL,
		"https://graph.microsoft.com/v1.0/me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &MicrosoftProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Microsoft Graph API
func (m *MicrosoftProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := m.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	if err := FetchJSON(ctx, client, m.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Microsoft user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}
	if email, ok := raw["mail"].(string); ok {
		userInfo.Email = email
	} else if upn, ok := raw["userPrincipalName"].(string); ok {
		userInfo.Email = upn
	}
	if name, ok := raw["displayName"].(string); ok {
		userInfo.Name = name
	}
	if givenName, ok := raw["givenName"].(string); ok {
		userInfo.FirstName = givenName
	}
	if surname, ok := raw["surname"].(string); ok {
		userInfo.LastName = surname
	}

	// Microsoft doesn't always provide avatar in user info
	// May need separate Graph API call for photo
	userInfo.EmailVerified = true // Microsoft accounts are generally verified

	return userInfo, nil
}
