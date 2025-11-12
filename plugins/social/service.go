package social

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"golang.org/x/oauth2"

	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/social/providers"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Service handles social OAuth flows
type Service struct {
	config      Config
	providers   map[string]providers.Provider
	socialRepo  repository.SocialAccountRepository
	userService *user.Service
	stateStore  map[string]*OAuthState // In-memory state storage (use Redis in production)
}

// OAuthState stores temporary OAuth state data
type OAuthState struct {
	Provider           string
	AppID              xid.ID  // Platform app (required)
	UserOrganizationID *xid.ID // User-created org (optional)
	RedirectURL        string
	CreatedAt          time.Time
	ExtraScopes        []string // Additional scopes requested
	LinkUserID         *xid.ID  // If linking to existing user
}

// NewService creates a new social auth service
func NewService(config Config, socialRepo repository.SocialAccountRepository, userSvc *user.Service) *Service {
	s := &Service{
		config:      config,
		providers:   make(map[string]providers.Provider),
		socialRepo:  socialRepo,
		userService: userSvc,
		stateStore:  make(map[string]*OAuthState),
	}

	// Initialize configured providers
	s.initializeProviders()

	return s
}

// initializeProviders creates provider instances based on configuration
func (s *Service) initializeProviders() {
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	pc := s.config.Providers

	if pc.Google != nil && pc.Google.Enabled {
		pc.Google.RedirectURL = baseURL + "/api/auth/callback/google"
		s.providers["google"] = providers.NewGoogleProvider(*pc.Google)
	}
	if pc.GitHub != nil && pc.GitHub.Enabled {
		pc.GitHub.RedirectURL = baseURL + "/api/auth/callback/github"
		s.providers["github"] = providers.NewGitHubProvider(*pc.GitHub)
	}
	if pc.Microsoft != nil && pc.Microsoft.Enabled {
		pc.Microsoft.RedirectURL = baseURL + "/api/auth/callback/microsoft"
		s.providers["microsoft"] = providers.NewMicrosoftProvider(*pc.Microsoft)
	}
	if pc.Apple != nil && pc.Apple.Enabled {
		pc.Apple.RedirectURL = baseURL + "/api/auth/callback/apple"
		s.providers["apple"] = providers.NewAppleProvider(*pc.Apple)
	}
	if pc.Facebook != nil && pc.Facebook.Enabled {
		pc.Facebook.RedirectURL = baseURL + "/api/auth/callback/facebook"
		s.providers["facebook"] = providers.NewFacebookProvider(*pc.Facebook)
	}
	if pc.Discord != nil && pc.Discord.Enabled {
		pc.Discord.RedirectURL = baseURL + "/api/auth/callback/discord"
		s.providers["discord"] = providers.NewDiscordProvider(*pc.Discord)
	}
	if pc.Twitter != nil && pc.Twitter.Enabled {
		pc.Twitter.RedirectURL = baseURL + "/api/auth/callback/twitter"
		s.providers["twitter"] = providers.NewTwitterProvider(*pc.Twitter)
	}
	if pc.LinkedIn != nil && pc.LinkedIn.Enabled {
		pc.LinkedIn.RedirectURL = baseURL + "/api/auth/callback/linkedin"
		s.providers["linkedin"] = providers.NewLinkedInProvider(*pc.LinkedIn)
	}
	if pc.Spotify != nil && pc.Spotify.Enabled {
		pc.Spotify.RedirectURL = baseURL + "/api/auth/callback/spotify"
		s.providers["spotify"] = providers.NewSpotifyProvider(*pc.Spotify)
	}
	if pc.Twitch != nil && pc.Twitch.Enabled {
		pc.Twitch.RedirectURL = baseURL + "/api/auth/callback/twitch"
		s.providers["twitch"] = providers.NewTwitchProvider(*pc.Twitch)
	}
	if pc.Dropbox != nil && pc.Dropbox.Enabled {
		pc.Dropbox.RedirectURL = baseURL + "/api/auth/callback/dropbox"
		s.providers["dropbox"] = providers.NewDropboxProvider(*pc.Dropbox)
	}
	if pc.GitLab != nil && pc.GitLab.Enabled {
		pc.GitLab.RedirectURL = baseURL + "/api/auth/callback/gitlab"
		s.providers["gitlab"] = providers.NewGitLabProvider(*pc.GitLab)
	}
	if pc.LINE != nil && pc.LINE.Enabled {
		pc.LINE.RedirectURL = baseURL + "/api/auth/callback/line"
		s.providers["line"] = providers.NewLINEProvider(*pc.LINE)
	}
	if pc.Reddit != nil && pc.Reddit.Enabled {
		pc.Reddit.RedirectURL = baseURL + "/api/auth/callback/reddit"
		s.providers["reddit"] = providers.NewRedditProvider(*pc.Reddit)
	}
	if pc.Slack != nil && pc.Slack.Enabled {
		pc.Slack.RedirectURL = baseURL + "/api/auth/callback/slack"
		s.providers["slack"] = providers.NewSlackProvider(*pc.Slack)
	}
	if pc.Bitbucket != nil && pc.Bitbucket.Enabled {
		pc.Bitbucket.RedirectURL = baseURL + "/api/auth/callback/bitbucket"
		s.providers["bitbucket"] = providers.NewBitbucketProvider(*pc.Bitbucket)
	}
	if pc.Notion != nil && pc.Notion.Enabled {
		pc.Notion.RedirectURL = baseURL + "/api/auth/callback/notion"
		s.providers["notion"] = providers.NewNotionProvider(*pc.Notion)
	}
}

// GetAuthorizationURL generates an OAuth authorization URL
func (s *Service) GetAuthorizationURL(ctx context.Context, providerName string, appID xid.ID, userOrganizationID *xid.ID, extraScopes []string) (string, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return "", fmt.Errorf("provider %s not configured", providerName)
	}

	// Generate secure state token
	state, err := s.generateState(providerName, appID, userOrganizationID, extraScopes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Build OAuth config with extra scopes if needed
	oauth2Config := provider.GetOAuth2Config()
	if len(extraScopes) > 0 {
		// Clone config with additional scopes
		newConfig := *oauth2Config
		newConfig.Scopes = append(oauth2Config.Scopes, extraScopes...)
		oauth2Config = &newConfig
	}

	authURL := oauth2Config.AuthCodeURL(state)
	return authURL, nil
}

// GetLinkAccountURL generates a URL to link an additional provider to an existing user
func (s *Service) GetLinkAccountURL(ctx context.Context, providerName string, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, extraScopes []string) (string, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return "", fmt.Errorf("provider %s not configured", providerName)
	}

	// Generate state with user linking
	state, err := s.generateState(providerName, appID, userOrganizationID, extraScopes, &userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	oauth2Config := provider.GetOAuth2Config()
	if len(extraScopes) > 0 {
		newConfig := *oauth2Config
		newConfig.Scopes = append(oauth2Config.Scopes, extraScopes...)
		oauth2Config = &newConfig
	}

	authURL := oauth2Config.AuthCodeURL(state)
	return authURL, nil
}

// HandleCallback processes the OAuth callback
func (s *Service) HandleCallback(ctx context.Context, providerName, stateToken, code string) (*CallbackResult, error) {
	// Verify state
	state, err := s.verifyState(stateToken)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	if state.Provider != providerName {
		return nil, fmt.Errorf("state provider mismatch")
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Exchange code for token
	oauth2Config := provider.GetOAuth2Config()
	token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from provider
	userInfo, err := provider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check email verification requirement
	if s.config.RequireEmailVerified && !userInfo.EmailVerified {
		return nil, fmt.Errorf("email not verified by provider")
	}

	// Check if social account already exists
	existingAccount, err := s.socialRepo.FindByProviderAndProviderID(ctx, providerName, userInfo.ID, state.AppID, state.UserOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	var targetUser *user.User

	// If linking to existing user
	if state.LinkUserID != nil {
		targetUser, err = s.userService.FindByID(ctx, *state.LinkUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to find user for linking: %w", err)
		}

		if existingAccount != nil {
			// Update existing account with new scopes/tokens
			existingAccount.AccessToken = token.AccessToken
			existingAccount.RefreshToken = token.RefreshToken
			if token.Expiry.IsZero() {
				existingAccount.ExpiresAt = nil
			} else {
				existingAccount.ExpiresAt = &token.Expiry
			}
			existingAccount.Scope = provider.GetOAuth2Config().Scopes[0] // Join scopes
			if err := s.socialRepo.Update(ctx, existingAccount); err != nil {
				return nil, fmt.Errorf("failed to update social account: %w", err)
			}
		} else {
			// Create new social account link
			if err := s.createSocialAccount(ctx, targetUser.ID, state.AppID, state.UserOrganizationID, providerName, userInfo, token); err != nil {
				return nil, fmt.Errorf("failed to link social account: %w", err)
			}
		}

		return &CallbackResult{
			User:          &schema.User{ID: targetUser.ID, Email: targetUser.Email, Name: targetUser.Name, Image: targetUser.Image, EmailVerified: targetUser.EmailVerified},
			IsNewUser:     false,
			SocialAccount: existingAccount,
			Action:        "linked",
		}, nil
	}

	// If account exists, return existing user
	if existingAccount != nil {
		targetUser, err = s.userService.FindByID(ctx, existingAccount.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to find user: %w", err)
		}

		return &CallbackResult{
			User:          &schema.User{ID: targetUser.ID, Email: targetUser.Email, Name: targetUser.Name, Image: targetUser.Image, EmailVerified: targetUser.EmailVerified},
			IsNewUser:     false,
			SocialAccount: existingAccount,
			Action:        "signin",
		}, nil
	}

	// Auto-create user if configured
	if s.config.AutoCreateUser {
		// Check if user with email exists
		if userInfo.Email != "" {
			existingUser, _ := s.userService.FindByEmail(ctx, userInfo.Email)
			if existingUser != nil {
				// Link to existing user if account linking is allowed
				if s.config.AllowAccountLinking {
					if err := s.createSocialAccount(ctx, existingUser.ID, state.AppID, state.UserOrganizationID, providerName, userInfo, token); err != nil {
						return nil, fmt.Errorf("failed to link social account: %w", err)
					}

					return &CallbackResult{
						User:      &schema.User{ID: existingUser.ID, Email: existingUser.Email, Name: existingUser.Name, Image: existingUser.Image, EmailVerified: existingUser.EmailVerified},
						IsNewUser: false,
						Action:    "linked",
					}, nil
				}
				return nil, fmt.Errorf("user with email already exists")
			}
		}

		// Create new user
		createReq := &user.CreateUserRequest{
			Email:    userInfo.Email,
			Name:     userInfo.Name,
			Password: "", // No password for OAuth users
		}

		newUser, err := s.userService.Create(ctx, createReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Update additional fields after creation
		if userInfo.Avatar != "" {
			updateReq := &user.UpdateUserRequest{
				Image: &userInfo.Avatar,
			}
			newUser, err = s.userService.Update(ctx, newUser, updateReq)
			if err != nil {
				// Log but don't fail - user is already created
				fmt.Printf("warning: failed to update user profile: %v\n", err)
			}
		}

		// Create social account
		if err := s.createSocialAccount(ctx, newUser.ID, state.AppID, state.UserOrganizationID, providerName, userInfo, token); err != nil {
			return nil, fmt.Errorf("failed to create social account: %w", err)
		}

		return &CallbackResult{
			User:      &schema.User{ID: newUser.ID, Email: newUser.Email, Name: newUser.Name, Image: newUser.Image, EmailVerified: newUser.EmailVerified},
			IsNewUser: true,
			Action:    "signup",
		}, nil
	}

	return nil, fmt.Errorf("user does not exist and auto-creation is disabled")
}

// createSocialAccount creates a new social account record
func (s *Service) createSocialAccount(ctx context.Context, userID, appID xid.ID, userOrganizationID *xid.ID, provider string, userInfo *providers.UserInfo, token *oauth2.Token) error {
	rawJSON, _ := json.Marshal(userInfo.Raw)

	account := &schema.SocialAccount{
		ID:                 xid.New(),
		UserID:             userID,
		AppID:              appID,
		UserOrganizationID: userOrganizationID,
		Provider:           provider,
		ProviderID:         userInfo.ID,
		Email:              userInfo.Email,
		Name:               userInfo.Name,
		Avatar:             userInfo.Avatar,
		AccessToken:        token.AccessToken,
		RefreshToken:       token.RefreshToken,
		TokenType:          token.TokenType,
		RawUserInfo:        string(rawJSON),
	}

	if !token.Expiry.IsZero() {
		account.ExpiresAt = &token.Expiry
	}

	if idToken, ok := token.Extra("id_token").(string); ok {
		account.IDToken = idToken
	}

	return s.socialRepo.Create(ctx, account)
}

// generateState creates a secure state token
func (s *Service) generateState(provider string, appID xid.ID, userOrganizationID *xid.ID, extraScopes []string, linkUserID *xid.ID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)

	s.stateStore[state] = &OAuthState{
		Provider:           provider,
		AppID:              appID,
		UserOrganizationID: userOrganizationID,
		CreatedAt:          time.Now(),
		ExtraScopes:        extraScopes,
		LinkUserID:         linkUserID,
	}

	// TODO: Implement cleanup for old states (use Redis with TTL in production)

	return state, nil
}

// verifyState validates and retrieves the state
func (s *Service) verifyState(stateToken string) (*OAuthState, error) {
	state, ok := s.stateStore[stateToken]
	if !ok {
		return nil, fmt.Errorf("state not found")
	}

	// Check if state is expired (15 minutes)
	if time.Since(state.CreatedAt) > 15*time.Minute {
		delete(s.stateStore, stateToken)
		return nil, fmt.Errorf("state expired")
	}

	// Delete state after use
	delete(s.stateStore, stateToken)

	return state, nil
}

// CallbackResult holds the result of OAuth callback processing
type CallbackResult struct {
	User          *schema.User
	SocialAccount *schema.SocialAccount
	IsNewUser     bool
	Action        string // "signin", "signup", "linked"
}

// ListProviders returns available providers
func (s *Service) ListProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// UnlinkAccount removes a social account link
func (s *Service) UnlinkAccount(ctx context.Context, userID xid.ID, provider string) error {
	return s.socialRepo.Unlink(ctx, userID, provider)
}
