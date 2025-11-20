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

	"github.com/xraph/authsome/core/audit"
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
	stateStore  StateStore
	audit       *audit.Service
}

// NewService creates a new social auth service
func NewService(config Config, socialRepo repository.SocialAccountRepository, userSvc *user.Service, stateStore StateStore, auditSvc *audit.Service) *Service {
	s := &Service{
		config:      config,
		providers:   make(map[string]providers.Provider),
		socialRepo:  socialRepo,
		userService: userSvc,
		stateStore:  stateStore,
		audit:       auditSvc,
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
		// Audit: provider not found
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_provider_not_found",
				fmt.Sprintf("provider:%s app_id:%s", providerName, appID.String()),
				"", "",
				fmt.Sprintf(`{"provider":"%s","app_id":"%s"}`, providerName, appID.String()))
		}
		return "", fmt.Errorf("provider %s not configured", providerName)
	}

	// Generate secure state token
	state, err := s.generateState(ctx, providerName, appID, userOrganizationID, extraScopes, nil)
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
	
	// Audit: OAuth flow initiated
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "social_signin_initiated",
			fmt.Sprintf("provider:%s app_id:%s", providerName, appID.String()),
			"", "",
			fmt.Sprintf(`{"provider":"%s","app_id":"%s","scopes":%s}`,
				providerName, appID.String(), func() string {
					if len(extraScopes) > 0 {
						b, _ := json.Marshal(extraScopes)
						return string(b)
					}
					return "[]"
				}()))
	}
	
	return authURL, nil
}

// GetLinkAccountURL generates a URL to link an additional provider to an existing user
func (s *Service) GetLinkAccountURL(ctx context.Context, providerName string, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, extraScopes []string) (string, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		if s.audit != nil {
			_ = s.audit.Log(ctx, &userID, "social_provider_not_found",
				fmt.Sprintf("provider:%s user_id:%s", providerName, userID.String()),
				"", "",
				fmt.Sprintf(`{"provider":"%s","user_id":"%s","action":"link"}`, providerName, userID.String()))
		}
		return "", fmt.Errorf("provider %s not configured", providerName)
	}

	// Generate state with user linking
	state, err := s.generateState(ctx, providerName, appID, userOrganizationID, extraScopes, &userID)
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
	
	// Audit: link flow initiated
	if s.audit != nil {
		_ = s.audit.Log(ctx, &userID, "social_link_initiated",
			fmt.Sprintf("provider:%s user_id:%s", providerName, userID.String()),
			"", "",
			fmt.Sprintf(`{"provider":"%s","user_id":"%s","app_id":"%s"}`, providerName, userID.String(), appID.String()))
	}
	
	return authURL, nil
}

// HandleCallback processes the OAuth callback
func (s *Service) HandleCallback(ctx context.Context, providerName, stateToken, code string) (*CallbackResult, error) {
	// Audit: callback received
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "social_callback_received",
			fmt.Sprintf("provider:%s", providerName),
			"", "",
			fmt.Sprintf(`{"provider":"%s"}`, providerName))
	}

	// Verify state
	state, err := s.verifyState(ctx, stateToken)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	if state.Provider != providerName {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_state_provider_mismatch",
				fmt.Sprintf("expected:%s got:%s", state.Provider, providerName),
				"", "",
				fmt.Sprintf(`{"expected":"%s","got":"%s"}`, state.Provider, providerName))
		}
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
		// Audit: token exchange failed
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_token_exchange_failed",
				fmt.Sprintf("provider:%s error:%s", providerName, err.Error()),
				"", "",
				fmt.Sprintf(`{"provider":"%s","error":"%s"}`, providerName, err.Error()))
		}
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Audit: token exchange success
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "social_token_exchange_success",
			fmt.Sprintf("provider:%s", providerName),
			"", "",
			fmt.Sprintf(`{"provider":"%s","app_id":"%s"}`, providerName, state.AppID.String()))
	}

	// Get user info from provider
	userInfo, err := provider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Audit: user info fetched
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "social_userinfo_fetched",
			fmt.Sprintf("provider:%s email:%s", providerName, userInfo.Email),
			"", "",
			fmt.Sprintf(`{"provider":"%s","email":"%s","verified":%t}`, providerName, userInfo.Email, userInfo.EmailVerified))
	}

	// Check email verification requirement
	if s.config.RequireEmailVerified && !userInfo.EmailVerified {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_email_not_verified",
				fmt.Sprintf("provider:%s email:%s", providerName, userInfo.Email),
				"", "",
				fmt.Sprintf(`{"provider":"%s","email":"%s"}`, providerName, userInfo.Email))
		}
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
func (s *Service) generateState(ctx context.Context, provider string, appID xid.ID, userOrganizationID *xid.ID, extraScopes []string, linkUserID *xid.ID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	stateToken := base64.URLEncoding.EncodeToString(b)

	state := &OAuthState{
		Provider:           provider,
		AppID:              appID,
		UserOrganizationID: userOrganizationID,
		CreatedAt:          time.Now(),
		ExtraScopes:        extraScopes,
		LinkUserID:         linkUserID,
	}

	// Store state with TTL
	ttl := s.config.StateStorage.StateTTL
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	
	if err := s.stateStore.Set(ctx, stateToken, state, ttl); err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	return stateToken, nil
}

// verifyState validates and retrieves the state
func (s *Service) verifyState(ctx context.Context, stateToken string) (*OAuthState, error) {
	state, err := s.stateStore.Get(ctx, stateToken)
	if err != nil {
		// Audit: invalid state
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_state_invalid",
				fmt.Sprintf("state_token:%s error:%s", stateToken[:10]+"...", err.Error()),
				"", "",
				fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		}
		return nil, err
	}

	// Delete state after use (one-time use)
	_ = s.stateStore.Delete(ctx, stateToken)

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
