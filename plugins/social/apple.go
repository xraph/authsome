package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

// Apple OAuth2 endpoints.
var appleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://appleid.apple.com/auth/authorize",
	TokenURL: "https://appleid.apple.com/auth/token",
}

type appleProvider struct {
	config *oauth2.Config
}

// NewAppleProvider creates an Apple Sign In OAuth2 provider.
// Apple returns user info in the ID token (JWT) rather than via a separate
// userinfo endpoint. The provider decodes the ID token claims directly.
func NewAppleProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"name", "email"}
	}

	return &appleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     appleEndpoint,
		},
	}
}

func (p *appleProvider) Name() string { return "apple" }

func (p *appleProvider) OAuth2Config() *oauth2.Config { return p.config }

// FetchUser extracts user information from the Apple ID token.
// Apple does not expose a userinfo endpoint; instead, user claims are
// embedded in the id_token JWT returned during the token exchange.
func (p *appleProvider) FetchUser(_ context.Context, token *oauth2.Token) (*ProviderUser, error) {
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, fmt.Errorf("apple: no id_token in token response")
	}

	claims, err := decodeJWTClaims(idToken)
	if err != nil {
		return nil, fmt.Errorf("apple: decode id_token: %w", err)
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	if sub == "" {
		return nil, fmt.Errorf("apple: missing sub claim in id_token")
	}

	return &ProviderUser{
		ProviderUserID: sub,
		Email:          email,
	}, nil
}

// decodeJWTClaims extracts the claims payload from a JWT without verifying the
// signature. Signature verification should be done at the transport/middleware
// layer using Apple's public keys. This function only parses the payload for
// user info extraction.
func decodeJWTClaims(tokenStr string) (map[string]any, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second segment)
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return claims, nil
}
