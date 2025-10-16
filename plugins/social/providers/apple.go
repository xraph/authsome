package providers

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// AppleProvider implements Sign in with Apple
type AppleProvider struct {
	*BaseProvider
	teamID string
	keyID  string
}

// NewAppleProvider creates a new Apple OAuth provider
func NewAppleProvider(config ProviderConfig) *AppleProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"name", "email"}
	}

	bp := NewBaseProvider(
		"apple",
		"Apple",
		"https://appleid.apple.com/auth/authorize",
		"https://appleid.apple.com/auth/token",
		"", // Apple uses ID token, not a separate userinfo endpoint
		config.ClientID,
		config.ClientSecret, // For Apple, this is a JWT you generate
		config.RedirectURL,
		scopes,
	)

	return &AppleProvider{
		BaseProvider: bp,
	}
}

// GetUserInfo extracts user information from Apple's ID token
func (a *AppleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	// Apple provides user info in the ID token
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, fmt.Errorf("no ID token provided by Apple")
	}

	// Parse the ID token (without verification for now - in production, verify!)
	claims := jwt.MapClaims{}
	parser := jwt.NewParser()
	_, _, err := parser.ParseUnverified(idToken, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Apple ID token: %w", err)
	}

	userInfo := &UserInfo{
		Raw:           claims,
		EmailVerified: true, // Apple emails are always verified
	}

	if sub, ok := claims["sub"].(string); ok {
		userInfo.ID = sub
	}
	if email, ok := claims["email"].(string); ok {
		userInfo.Email = email
	}

	// Apple may provide name in the initial authorization response
	// This is typically passed separately in the POST body

	return userInfo, nil
}

// GenerateClientSecret generates a client secret JWT for Apple
// Required for Apple's OAuth flow
func GenerateAppleClientSecret(teamID, clientID, keyID string, privateKey *rsa.PrivateKey) (string, error) {
	now := jwt.NewNumericDate(time.Now())
	expiresAt := jwt.NewNumericDate(time.Now().Add(6 * 30 * 24 * time.Hour)) // 6 months

	claims := jwt.RegisteredClaims{
		Issuer:    teamID,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		Audience:  jwt.ClaimStrings{"https://appleid.apple.com"},
		Subject:   clientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(privateKey)
}
