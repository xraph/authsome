package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/spotify"
)

// SpotifyProvider implements OAuth for Spotify.
type SpotifyProvider struct {
	*BaseProvider
}

// NewSpotifyProvider creates a new Spotify OAuth provider.
func NewSpotifyProvider(config ProviderConfig) *SpotifyProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user-read-email", "user-read-private"}
	}

	bp := NewBaseProvider(
		"spotify",
		"Spotify",
		spotify.Endpoint.AuthURL,
		spotify.Endpoint.TokenURL,
		"https://api.spotify.com/v1/me",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &SpotifyProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from Spotify API.
func (s *SpotifyProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := s.oauth2Config.Client(ctx, token)

	var raw map[string]any
	if err := FetchJSON(ctx, client, s.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch Spotify user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw:           raw,
		EmailVerified: true, // Spotify emails are verified
	}

	if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	if email, ok := raw["email"].(string); ok {
		userInfo.Email = email
	}

	if displayName, ok := raw["display_name"].(string); ok {
		userInfo.Name = displayName
	}

	// Spotify avatar
	if images, ok := raw["images"].([]any); ok && len(images) > 0 {
		if img, ok := images[0].(map[string]any); ok {
			if url, ok := img["url"].(string); ok {
				userInfo.Avatar = url
			}
		}
	}

	return userInfo, nil
}
