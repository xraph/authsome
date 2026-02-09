package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

// BitbucketProvider implements OAuth for Bitbucket.
type BitbucketProvider struct {
	*BaseProvider
}

// NewBitbucketProvider creates a new Bitbucket OAuth provider.
func NewBitbucketProvider(config ProviderConfig) *BitbucketProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"account", "email"}
	}

	bp := NewBaseProvider(
		"bitbucket",
		"Bitbucket",
		bitbucket.Endpoint.AuthURL,
		bitbucket.Endpoint.TokenURL,
		"https://api.bitbucket.org/2.0/user",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &BitbucketProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Bitbucket API.
func (b *BitbucketProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := b.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, b.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Bitbucket user info: %w", err)
	}

	// Fetch email separately
	var emailResponse struct {
		Values []map[string]any `json:"values"`
	}

	if err := FetchJSON(ctx, client, "https://api.bitbucket.org/2.0/user/emails", &emailResponse); err == nil {
		for _, emailData := range emailResponse.Values {
			if isPrimary, ok := emailData["is_primary"].(bool); ok && isPrimary {
				if email, ok := emailData["email"].(string); ok {
					raw["email"] = email
					if isConfirmed, ok := emailData["is_confirmed"].(bool); ok {
						raw["email_verified"] = isConfirmed
					}

					break
				}
			}
		}
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if uuid, ok := raw["uuid"].(string); ok {
		userInfo.ID = uuid
	} else if accountID, ok := raw["account_id"].(string); ok {
		userInfo.ID = accountID
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	if verified, ok := raw["email_verified"].(bool); ok {
		userInfo.EmailVerified = verified
	}

	if displayName, ok := raw["display_name"].(string); ok {
		userInfo.Name = displayName
	}

	if username, ok := raw["username"].(string); ok {
		userInfo.Username = username
	}

	// Bitbucket avatar in links
	if links, ok := raw["links"].(map[string]any); ok {
		if avatar, ok := links["avatar"].(map[string]any); ok {
			if href, ok := avatar["href"].(string); ok {
				userInfo.Avatar = href
			}
		}
	}

	return userInfo, nil
}
