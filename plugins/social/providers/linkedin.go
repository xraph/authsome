package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/linkedin"
)

// LinkedInProvider implements OAuth for LinkedIn.
type LinkedInProvider struct {
	*BaseProvider
}

// NewLinkedInProvider creates a new LinkedIn OAuth provider.
func NewLinkedInProvider(config ProviderConfig) *LinkedInProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"r_liteprofile", "r_emailaddress"}
	}

	bp := NewBaseProvider(
		"linkedin",
		"LinkedIn",
		linkedin.Endpoint.AuthURL,
		linkedin.Endpoint.TokenURL,
		"https://api.linkedin.com/v2/me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &LinkedInProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from LinkedIn API.
func (l *LinkedInProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := l.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, l.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch LinkedIn user info: %w", err)
	}

	// Fetch email separately
	var emailData struct {
		Elements []struct {
			Handle      map[string]string `json:"handle~"`
			HandleValue string            `json:"handle"`
		} `json:"elements"`
	}

	if err := FetchJSON(ctx, client, "https://api.linkedin.com/v2/emailAddress?q=members&projection=(elements*(handle~))", &emailData); err == nil {
		if len(emailData.Elements) > 0 {
			if email, ok := emailData.Elements[0].Handle["emailAddress"]; ok {
				raw["email"] = email
			}
		}
	}

	userInfo := &UserInfo{
		Raw:           raw,
		EmailVerified: true, // LinkedIn emails are verified
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	// LinkedIn uses localized name format
	if localizedFirstName, ok := raw["localizedFirstName"].(string); ok {
		userInfo.FirstName = localizedFirstName
	}

	if localizedLastName, ok := raw["localizedLastName"].(string); ok {
		userInfo.LastName = localizedLastName
	}

	userInfo.Name = fmt.Sprintf("%s %s", userInfo.FirstName, userInfo.LastName)

	// Profile picture is in a complex nested structure
	if profilePicture, ok := raw["profilePicture"].(map[string]any); ok {
		if displayImage, ok := profilePicture["displayImage~"].(map[string]any); ok {
			if elements, ok := displayImage["elements"].([]any); ok && len(elements) > 0 {
				if elem, ok := elements[0].(map[string]any); ok {
					if identifiers, ok := elem["identifiers"].([]any); ok && len(identifiers) > 0 {
						if identifier, ok := identifiers[0].(map[string]any); ok {
							if url, ok := identifier["identifier"].(string); ok {
								userInfo.Avatar = url
							}
						}
					}
				}
			}
		}
	}

	return userInfo, nil
}
