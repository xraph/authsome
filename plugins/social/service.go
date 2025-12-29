package social

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"golang.org/x/oauth2"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
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
	configRepo  repository.SocialProviderConfigRepository // For DB-backed config
	userService *user.Service
	stateStore  StateStore
	audit       *audit.Service

	// Environment-specific provider caching
	envProviders map[string]map[string]providers.Provider // envKey -> provider map
	envMutex     sync.RWMutex                             // Protect envProviders map
}

// NewService creates a new social auth service
func NewService(config Config, socialRepo repository.SocialAccountRepository, userSvc *user.Service, stateStore StateStore, auditSvc *audit.Service) *Service {
	s := &Service{
		config:       config,
		providers:    make(map[string]providers.Provider),
		socialRepo:   socialRepo,
		userService:  userSvc,
		stateStore:   stateStore,
		audit:        auditSvc,
		envProviders: make(map[string]map[string]providers.Provider),
	}

	// Initialize configured providers from static config
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
	// Extract environment ID from context
	envID := getEnvironmentIDFromContext(ctx)

	// Ensure providers are loaded for this environment
	providers, err := s.ensureProvidersLoaded(ctx, appID, envID)
	if err != nil {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_provider_load_failed",
				fmt.Sprintf("provider:%s app_id:%s env_id:%s error:%s", providerName, appID.String(), envID.String(), err.Error()),
				"", "",
				fmt.Sprintf(`{"provider":"%s","app_id":"%s","env_id":"%s","error":"%s"}`, providerName, appID.String(), envID.String(), err.Error()))
		}
		return "", fmt.Errorf("failed to load provider configuration: %w", err)
	}

	provider, ok := providers[providerName]
	if !ok {
		// Audit: provider not found
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "social_provider_not_found",
				fmt.Sprintf("provider:%s app_id:%s env_id:%s", providerName, appID.String(), envID.String()),
				"", "",
				fmt.Sprintf(`{"provider":"%s","app_id":"%s","env_id":"%s"}`, providerName, appID.String(), envID.String()))
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
	// Extract environment ID from context
	envID := getEnvironmentIDFromContext(ctx)

	// Ensure providers are loaded for this environment
	providers, err := s.ensureProvidersLoaded(ctx, appID, envID)
	if err != nil {
		return "", fmt.Errorf("failed to load provider configuration: %w", err)
	}

	provider, ok := providers[providerName]
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

	// Extract environment ID from context
	envID := getEnvironmentIDFromContext(ctx)

	// Ensure providers are loaded for this environment
	providers, err := s.ensureProvidersLoaded(ctx, state.AppID, envID)
	if err != nil {
		return nil, fmt.Errorf("failed to load provider configuration: %w", err)
	}

	provider, ok := providers[providerName]
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
			if err := s.CreateSocialAccount(ctx, targetUser.ID, state.AppID, state.UserOrganizationID, providerName, userInfo, token); err != nil {
				return nil, fmt.Errorf("failed to link social account: %w", err)
			}
		}

		return &CallbackResult{
			User:          targetUser,
			OAuthUserInfo: userInfo,
			OAuthToken:    token,
			Provider:      providerName,
			IsNewUser:     false,
			SocialAccount: existingAccount,
			Action:        "linked",
			AppID:         state.AppID,
			UserOrgID:     state.UserOrganizationID,
		}, nil
	}

	// If account exists, return existing user
	if existingAccount != nil {
		targetUser, err = s.userService.FindByID(ctx, existingAccount.UserID)
		if err != nil {
			// Orphaned social account - user was deleted
			// Delete the orphaned social account and treat as new signup
			_ = s.socialRepo.Delete(ctx, existingAccount.ID)
			existingAccount = nil // Continue to new user flow
		} else {
			// User found, return for signin
			return &CallbackResult{
				User:          targetUser,
				OAuthUserInfo: userInfo,
				OAuthToken:    token,
				Provider:      providerName,
				IsNewUser:     false,
				SocialAccount: existingAccount,
				Action:        "signin",
				AppID:         state.AppID,
				UserOrgID:     state.UserOrganizationID,
			}, nil
		}
	}

	// Auto-create user if configured
	if s.config.AutoCreateUser {
		// Check if user with email exists
		if userInfo.Email != "" {
			existingUser, _ := s.userService.FindByEmail(ctx, userInfo.Email)
			if existingUser != nil {
				// Link to existing user if account linking is allowed
				if s.config.AllowAccountLinking {
					if err := s.CreateSocialAccount(ctx, existingUser.ID, state.AppID, state.UserOrganizationID, providerName, userInfo, token); err != nil {
						return nil, fmt.Errorf("failed to link social account: %w", err)
					}

					return &CallbackResult{
						User:          existingUser,
						OAuthUserInfo: userInfo,
						OAuthToken:    token,
						Provider:      providerName,
						IsNewUser:     false,
						Action:        "linked",
						AppID:         state.AppID,
						UserOrgID:     state.UserOrganizationID,
					}, nil
				}
				return nil, fmt.Errorf("user with email already exists")
			}
		}

		// Return user info for new users - handler will create user via authService.SignUp
		// This ensures proper membership handling through decorated auth service
		return &CallbackResult{
			User:          nil, // User not created yet
			OAuthUserInfo: userInfo,
			OAuthToken:    token,
			Provider:      providerName,
			IsNewUser:     true,
			Action:        "signup",
			AppID:         state.AppID,
			UserOrgID:     state.UserOrganizationID,
		}, nil
	}

	return nil, fmt.Errorf("user does not exist and auto-creation is disabled")
}

// CreateSocialAccount creates a new social account record
// This is called after user creation to link the OAuth provider
func (s *Service) CreateSocialAccount(ctx context.Context, userID, appID xid.ID, userOrganizationID *xid.ID, provider string, userInfo *providers.UserInfo, token *oauth2.Token) error {
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
	User          *user.User          // Nil for new users, populated for existing users
	OAuthUserInfo *providers.UserInfo // OAuth provider user info (always populated)
	OAuthToken    *oauth2.Token       // OAuth token for linking social account
	Provider      string              // OAuth provider name (e.g., "github", "google")
	SocialAccount *schema.SocialAccount
	IsNewUser     bool
	Action        string  // "signin", "signup", "linked"
	AppID         xid.ID  // App ID from state
	UserOrgID     *xid.ID // Optional user organization ID from state
}

// ListProviders returns available providers for a specific environment
func (s *Service) ListProviders(ctx context.Context, appID, envID xid.ID) []string {
	// Ensure providers are loaded for this environment
	providers, err := s.ensureProvidersLoaded(ctx, appID, envID)
	if err != nil {
		// Fallback to static providers if loading fails
		providers = s.providers
	}

	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}
	return providerNames
}

// UnlinkAccount removes a social account link
func (s *Service) UnlinkAccount(ctx context.Context, userID xid.ID, provider string) error {
	return s.socialRepo.Unlink(ctx, userID, provider)
}

// SetConfigRepository sets the config repository for DB-backed configuration
func (s *Service) SetConfigRepository(repo repository.SocialProviderConfigRepository) {
	s.configRepo = repo
}

// LoadConfigForEnvironment loads provider configurations from the database for a specific environment
// and merges them with the current configuration. DB configs take precedence over code-based configs.
func (s *Service) LoadConfigForEnvironment(ctx context.Context, appID, envID xid.ID) error {
	if s.configRepo == nil {
		return fmt.Errorf("config repository not set")
	}

	// Get enabled configurations from database
	configs, err := s.configRepo.ListEnabledByEnvironment(ctx, appID, envID)
	if err != nil {
		return fmt.Errorf("failed to load provider configs from database: %w", err)
	}

	// Keep the base URL from the existing config
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	// Clear existing providers and reinitialize with DB-backed configs
	s.providers = make(map[string]providers.Provider)

	for _, cfg := range configs {
		if !cfg.IsEnabled {
			continue
		}

		// Build provider config
		providerConfig := providers.ProviderConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Enabled:      true,
		}

		// Use custom redirect URL if set, otherwise build default
		if cfg.RedirectURL != "" {
			providerConfig.RedirectURL = cfg.RedirectURL
		} else {
			providerConfig.RedirectURL = baseURL + "/api/auth/callback/" + cfg.ProviderName
		}

		// Use custom scopes if set, otherwise use defaults
		if len(cfg.Scopes) > 0 {
			providerConfig.Scopes = cfg.Scopes
		} else {
			providerConfig.Scopes = schema.GetProviderDefaultScopes(cfg.ProviderName)
		}

		// Apply advanced config options if present
		if cfg.AdvancedConfig != nil {
			if accessType, ok := cfg.AdvancedConfig["accessType"].(string); ok {
				providerConfig.AccessType = accessType
			}
			if prompt, ok := cfg.AdvancedConfig["prompt"].(string); ok {
				providerConfig.Prompt = prompt
			}
		}

		// Create provider instance
		provider := s.createProviderInstance(cfg.ProviderName, providerConfig)
		if provider != nil {
			s.providers[cfg.ProviderName] = provider
		}
	}

	return nil
}

// getEnvironmentIDFromContext extracts environment ID from context using the contexts package
func getEnvironmentIDFromContext(ctx context.Context) xid.ID {
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return xid.NilID()
	}
	return envID
}

// ensureProvidersLoaded ensures that providers are loaded for the given environment
// It uses a cache to avoid reloading on every request
func (s *Service) ensureProvidersLoaded(ctx context.Context, appID, envID xid.ID) (map[string]providers.Provider, error) {
	// If no config repo is set, use static providers
	if s.configRepo == nil {
		fmt.Printf("[Social] No config repo set, using static providers (count: %d)\n", len(s.providers))
		return s.providers, nil
	}

	// If environment is not set, use static providers
	if envID.IsNil() {
		fmt.Printf("[Social] No environment ID provided, using static providers (count: %d)\n", len(s.providers))
		return s.providers, nil
	}

	// Create cache key
	envKey := fmt.Sprintf("%s:%s", appID.String(), envID.String())
	fmt.Printf("[Social] Loading providers for app:%s env:%s\n", appID.String(), envID.String())

	// Check cache first (read lock)
	s.envMutex.RLock()
	cached, exists := s.envProviders[envKey]
	s.envMutex.RUnlock()

	if exists {
		fmt.Printf("[Social] Using cached providers for %s (count: %d)\n", envKey, len(cached))
		return cached, nil
	}

	// Not in cache, load from database (write lock)
	s.envMutex.Lock()
	defer s.envMutex.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := s.envProviders[envKey]; exists {
		return cached, nil
	}

	// Get enabled configurations from database
	configs, err := s.configRepo.ListEnabledByEnvironment(ctx, appID, envID)
	if err != nil {
		fmt.Printf("[Social] Failed to load configs from DB: %v\n", err)
		return nil, fmt.Errorf("failed to load provider configs from database: %w", err)
	}

	fmt.Printf("[Social] Loaded %d provider configs from database for %s\n", len(configs), envKey)

	// If no configs found in DB, use static providers as fallback
	if len(configs) == 0 {
		fmt.Printf("[Social] No DB configs found, using static providers as fallback (count: %d)\n", len(s.providers))
		s.envProviders[envKey] = s.providers
		return s.providers, nil
	}

	// Keep the base URL from the existing config
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	// Build environment-specific providers
	envProviders := make(map[string]providers.Provider)

	for _, cfg := range configs {
		if !cfg.IsEnabled {
			continue
		}

		// Build provider config
		providerConfig := providers.ProviderConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Enabled:      true,
		}

		// Use custom redirect URL if set, otherwise build default
		if cfg.RedirectURL != "" {
			providerConfig.RedirectURL = cfg.RedirectURL
		} else {
			providerConfig.RedirectURL = baseURL + "/api/auth/callback/" + cfg.ProviderName
		}

		// Use custom scopes if set, otherwise use defaults
		if len(cfg.Scopes) > 0 {
			providerConfig.Scopes = cfg.Scopes
		} else {
			providerConfig.Scopes = schema.GetProviderDefaultScopes(cfg.ProviderName)
		}

		// Apply advanced config options if present
		if cfg.AdvancedConfig != nil {
			if accessType, ok := cfg.AdvancedConfig["accessType"].(string); ok {
				providerConfig.AccessType = accessType
			}
			if prompt, ok := cfg.AdvancedConfig["prompt"].(string); ok {
				providerConfig.Prompt = prompt
			}
		}

		// Create provider instance
		provider := s.createProviderInstance(cfg.ProviderName, providerConfig)
		if provider != nil {
			envProviders[cfg.ProviderName] = provider
			fmt.Printf("[Social] Created provider %s with clientID: %s...\n", cfg.ProviderName, cfg.ClientID[:10])
		} else {
			fmt.Printf("[Social] WARNING: Failed to create provider instance for %s\n", cfg.ProviderName)
		}
	}

	// Cache the result
	s.envProviders[envKey] = envProviders
	fmt.Printf("[Social] Cached %d providers for %s\n", len(envProviders), envKey)

	return envProviders, nil
}

// InvalidateEnvironmentCache clears the cache for a specific environment
// This should be called when provider configurations are updated
func (s *Service) InvalidateEnvironmentCache(appID, envID xid.ID) {
	envKey := fmt.Sprintf("%s:%s", appID.String(), envID.String())
	s.envMutex.Lock()
	defer s.envMutex.Unlock()
	delete(s.envProviders, envKey)
}

// createProviderInstance creates a provider instance for the given provider name and config
func (s *Service) createProviderInstance(providerName string, cfg providers.ProviderConfig) providers.Provider {
	switch providerName {
	case "google":
		return providers.NewGoogleProvider(cfg)
	case "github":
		return providers.NewGitHubProvider(cfg)
	case "microsoft":
		return providers.NewMicrosoftProvider(cfg)
	case "apple":
		return providers.NewAppleProvider(cfg)
	case "facebook":
		return providers.NewFacebookProvider(cfg)
	case "discord":
		return providers.NewDiscordProvider(cfg)
	case "twitter":
		return providers.NewTwitterProvider(cfg)
	case "linkedin":
		return providers.NewLinkedInProvider(cfg)
	case "spotify":
		return providers.NewSpotifyProvider(cfg)
	case "twitch":
		return providers.NewTwitchProvider(cfg)
	case "dropbox":
		return providers.NewDropboxProvider(cfg)
	case "gitlab":
		return providers.NewGitLabProvider(cfg)
	case "line":
		return providers.NewLINEProvider(cfg)
	case "reddit":
		return providers.NewRedditProvider(cfg)
	case "slack":
		return providers.NewSlackProvider(cfg)
	case "bitbucket":
		return providers.NewBitbucketProvider(cfg)
	case "notion":
		return providers.NewNotionProvider(cfg)
	default:
		return nil
	}
}

// GetProviderConfig returns the current provider configuration for a specific provider
// This can be used to inspect what's currently configured
func (s *Service) GetProviderConfig(providerName string) *providers.ProviderConfig {
	provider, ok := s.providers[providerName]
	if !ok {
		return nil
	}

	oauth2Config := provider.GetOAuth2Config()
	return &providers.ProviderConfig{
		ClientID:    oauth2Config.ClientID,
		RedirectURL: oauth2Config.RedirectURL,
		Scopes:      oauth2Config.Scopes,
		Enabled:     true,
	}
}

// IsProviderEnabled checks if a provider is currently enabled and configured
func (s *Service) IsProviderEnabled(providerName string) bool {
	_, ok := s.providers[providerName]
	return ok
}
