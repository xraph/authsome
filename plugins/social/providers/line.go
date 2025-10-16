package providers

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// LINEProvider implements OAuth for LINE
type LINEProvider struct {
	*BaseProvider
}

// NewLINEProvider creates a new LINE OAuth provider
func NewLINEProvider(config ProviderConfig) *LINEProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"profile", "openid", "email"}
	}

	bp := NewBaseProvider(
		"line",
		"LINE",
		"https://access.line.me/oauth2/v2.1/authorize",
		"https://api.line.me/oauth2/v2.1/token",
		"https://api.line.me/v2/profile",
		config.ClientID,
		config.ClientSecret,
		config.RedirectURL,
		scopes,
	)

	return &LINEProvider{BaseProvider: bp}
}

// GetUserInfo fetches user information from LINE API
func (l *LINEProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := l.oauth2Config.Client(ctx, token)

	var raw map[string]interface{}
	if err := FetchJSON(ctx, client, l.userInfoURL, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch LINE user info: %w", err)
	}

	userInfo := &UserInfo{
		Raw: raw,
	}

	if userID, ok := raw["userId"].(string); ok {
		userInfo.ID = userID
	}
	if displayName, ok := raw["displayName"].(string); ok {
		userInfo.Name = displayName
	}
	if picture, ok := raw["pictureUrl"].(string); ok {
		userInfo.Avatar = picture
	}

	// Email requires ID token verification for LINE
	if idToken, ok := token.Extra("id_token").(string); ok && idToken != "" {
		// Would need to decode ID token for email
		// LINE provides email in ID token claims
	}

	return userInfo, nil
}
