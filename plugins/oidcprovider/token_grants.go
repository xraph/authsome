package oidcprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// RefreshAccessToken refreshes an access token using a refresh token
// Implements OAuth2 Refresh Token Grant (RFC 6749 Section 6)
// Optionally rotates the refresh token for improved security.
func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken, clientID, requestedScope string) (*TokenResponse, error) {
	// Find the token by refresh token
	token, err := s.tokenRepo.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errs.UnauthorizedWithMessage("invalid refresh token")
	}

	// Validate token not expired
	if token.RefreshExpiresAt != nil && time.Now().After(*token.RefreshExpiresAt) {
		return nil, errs.UnauthorizedWithMessage("refresh token expired")
	}

	// Validate token not revoked
	if token.Revoked {
		return nil, errs.UnauthorizedWithMessage("refresh token has been revoked")
	}

	// Validate client ID matches
	if token.ClientID != clientID {
		return nil, errs.UnauthorizedWithMessage("client ID mismatch")
	}

	// Validate requested scope (must be subset of original scope)
	originalScopes := strings.Split(token.Scope, " ")
	effectiveScope := token.Scope

	if requestedScope != "" {
		requestedScopes := strings.Split(requestedScope, " ")
		if !isScopeSubset(requestedScopes, originalScopes) {
			return nil, errs.BadRequest("requested scope exceeds original grant")
		}

		effectiveScope = requestedScope
	}

	// Get user for ID token
	user, err := s.userSvc.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, errs.DatabaseError("find user", err)
	}

	userInfo := map[string]any{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
	}

	// Generate new access token
	newAccessToken, err := s.jwtService.GenerateAccessToken(
		token.UserID.String(),
		token.ClientID,
		effectiveScope,
	)
	if err != nil {
		return nil, errs.InternalError(fmt.Errorf("failed to generate access token: %w", err))
	}

	// Generate new ID token if openid scope is present
	var newIDToken string
	if containsScope(effectiveScope, "openid") {
		newIDToken, err = s.jwtService.GenerateIDToken(
			token.UserID.String(),
			token.ClientID,
			"", // No nonce for refresh
			time.Now(),
			userInfo,
		)
		if err != nil {
			return nil, errs.InternalError(fmt.Errorf("failed to generate ID token: %w", err))
		}
	}

	// Token rotation: generate new refresh token and revoke old one
	newRefreshToken := "refresh_" + xid.New().String()

	// Create new token record
	newTokenRecord := &schema.OAuthToken{
		AppID:            token.AppID,
		EnvironmentID:    token.EnvironmentID,
		OrganizationID:   token.OrganizationID,
		SessionID:        token.SessionID,
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		IDToken:          newIDToken,
		TokenType:        "Bearer",
		TokenClass:       "access_token",
		ClientID:         token.ClientID,
		UserID:           token.UserID,
		Scope:            effectiveScope,
		JTI:              "jti_" + xid.New().String(),
		Issuer:           s.config.Issuer,
		AuthTime:         token.AuthTime,
		ACR:              token.ACR,
		AMR:              token.AMR,
		ExpiresAt:        time.Now().Add(time.Hour), // 1 hour
		RefreshExpiresAt: token.RefreshExpiresAt,    // Keep same refresh expiry
	}

	// Store new token
	if err := s.tokenRepo.Create(ctx, newTokenRecord); err != nil {
		return nil, errs.DatabaseError("create token", err)
	}

	// Revoke old refresh token (token rotation)
	if err := s.tokenRepo.RevokeByRefreshToken(ctx, refreshToken); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging when logger is available
		_ = err
	}

	// Build response
	response := &TokenResponse{
		AccessToken:  newAccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		RefreshToken: newRefreshToken,
		Scope:        effectiveScope,
	}

	if newIDToken != "" {
		response.IDToken = newIDToken
	}

	return response, nil
}

// GenerateClientCredentialsToken generates a token for client credentials grant (M2M)
// Implements OAuth2 Client Credentials Grant (RFC 6749 Section 4.4).
func (s *Service) GenerateClientCredentialsToken(ctx context.Context, client *schema.OAuthClient, scope string) (*TokenResponse, error) {
	if s.jwtService == nil {
		return nil, errs.InternalServerErrorWithMessage("JWT service not initialized")
	}

	// Validate scope against client's allowed scopes
	// TODO: Implement proper scope validation
	// For now, we'll allow any scope but this should be restricted

	effectiveScope := scope
	if effectiveScope == "" {
		// Default M2M scopes
		effectiveScope = "api:read api:write"
	}

	// Generate access token (no user context, use client ID as subject)
	accessToken, err := s.jwtService.GenerateAccessToken(
		client.ClientID, // Use client ID as subject for M2M
		client.ClientID,
		effectiveScope,
	)
	if err != nil {
		return nil, errs.InternalError(fmt.Errorf("failed to generate access token: %w", err))
	}

	// Client credentials grant does not issue refresh tokens or ID tokens
	// Store token in database for tracking
	tokenRecord := &schema.OAuthToken{
		AppID:          client.AppID,
		EnvironmentID:  client.EnvironmentID,
		OrganizationID: client.OrganizationID,
		AccessToken:    accessToken,
		TokenType:      "Bearer",
		TokenClass:     "access_token",
		ClientID:       client.ClientID,
		UserID:         xid.NilID(), // No user for M2M
		Scope:          effectiveScope,
		JTI:            "jti_" + xid.New().String(),
		Issuer:         s.config.Issuer,
		ExpiresAt:      time.Now().Add(time.Hour), // 1 hour
	}

	if err := s.tokenRepo.Create(ctx, tokenRecord); err != nil {
		return nil, errs.DatabaseError("create token", err)
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
		Scope:       effectiveScope,
		// No refresh_token or id_token for client_credentials grant
	}, nil
}

// isScopeSubset checks if requested scopes are a subset of original scopes.
func isScopeSubset(requested, original []string) bool {
	originalSet := make(map[string]bool)
	for _, scope := range original {
		originalSet[scope] = true
	}

	for _, scope := range requested {
		if !originalSet[scope] {
			return false
		}
	}

	return true
}
