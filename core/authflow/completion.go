package authflow

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// AuthServiceInterface defines methods needed from auth service
// This interface is implemented by core/auth.Service
type AuthServiceInterface interface {
	SignUp(ctx context.Context, req *auth.SignUpRequest) (*responses.AuthResponse, error)
	SignIn(ctx context.Context, req *auth.SignInRequest) (*responses.AuthResponse, error)
	CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*responses.AuthResponse, error)
}

// DeviceServiceInterface defines methods needed from device service
type DeviceServiceInterface interface {
	TrackDevice(ctx context.Context, appID, userID xid.ID, fingerprint, userAgent, ipAddress string) (*device.Device, error)
}

// AuditServiceInterface defines methods needed from audit service
type AuditServiceInterface interface {
	Log(ctx context.Context, userID *xid.ID, action, target, ipAddress, userAgent, metadata string) error
}

// AppServiceInterface defines methods needed from app service
// AppService is accessed via ServiceImpl.App
type AppServiceInterface interface {
	GetCookieConfig(ctx context.Context, appID xid.ID) (*session.CookieConfig, error)
}

// CompletionService handles the final steps of authentication
type CompletionService struct {
	authService   AuthServiceInterface
	deviceService DeviceServiceInterface
	auditService  AuditServiceInterface
	appService    AppServiceInterface
	cookieConfig  *session.CookieConfig
}

// NewCompletionService creates a new authentication completion service
func NewCompletionService(
	authService AuthServiceInterface,
	deviceService DeviceServiceInterface,
	auditService AuditServiceInterface,
	appService AppServiceInterface,
	cookieConfig *session.CookieConfig,
) *CompletionService {
	return &CompletionService{
		authService:   authService,
		deviceService: deviceService,
		auditService:  auditService,
		appService:    appService,
		cookieConfig:  cookieConfig,
	}
}

// CompleteAuthenticationRequest contains all data needed to complete authentication
type CompleteAuthenticationRequest struct {
	User         *user.User
	RememberMe   bool
	IPAddress    string
	UserAgent    string
	Context      context.Context
	ForgeContext forge.Context
	AuthMethod   string // "email", "social", "passkey", "mfa", etc.
	AuthProvider string // "google", "github", etc. for social
}

// CompleteSignUpOrSignInRequest contains all data needed for signup or signin completion
type CompleteSignUpOrSignInRequest struct {
	Email        string     // Email for new user signup
	Password     string     // Password for new user (may be empty for OAuth/magic link)
	Name         string     // Name for new user
	User         *user.User // Existing user (for signin flow)
	IsNewUser    bool       // True if this is a signup, false if signin
	RememberMe   bool
	IPAddress    string
	UserAgent    string
	Context      context.Context
	ForgeContext forge.Context
	AuthMethod   string // "social", "magiclink", "email", etc.
	AuthProvider string // e.g., "github", "google"
}

// CompleteAuthentication handles all post-authentication steps
func (s *CompletionService) CompleteAuthentication(req *CompleteAuthenticationRequest) (*responses.AuthResponse, error) {
	ctx := req.Context

	// 1. Create session
	authResp, err := s.authService.CreateSessionForUser(ctx, req.User, req.RememberMe, req.IPAddress, req.UserAgent)
	if err != nil {
		return nil, errs.Wrap(err, "SESSION_CREATION_FAILED", "Failed to create session", 500)
	}

	// 2. Get app ID from context
	appID, _ := contexts.GetAppID(ctx)

	// 3. Track device if enabled
	if s.deviceService != nil {
		fingerprint := req.UserAgent + "|" + req.IPAddress
		fmt.Printf("[CompletionService] Tracking device for user %s in app %s\n", req.User.ID.String(), appID.String())
		_, _ = s.deviceService.TrackDevice(ctx, appID, req.User.ID, fingerprint, req.UserAgent, req.IPAddress)
	}

	// 4. Set session cookie if enabled
	if req.ForgeContext != nil && s.cookieConfig != nil && s.cookieConfig.Enabled {
		s.setSessionCookie(ctx, req.ForgeContext, authResp, appID)
	}

	// 5. Audit log the authentication
	if s.auditService != nil {
		action := "signin_" + req.AuthMethod
		if req.AuthProvider != "" {
			action = action + "_" + req.AuthProvider
		}
		userID := req.User.ID
		_ = s.auditService.Log(ctx, &userID, action, "user:"+userID.String(), req.IPAddress, req.UserAgent, "")
	}

	return authResp, nil
}

// CompleteSignUpOrSignIn handles signup for new users or signin for existing users
// This method ensures proper membership creation through decorated auth services
func (s *CompletionService) CompleteSignUpOrSignIn(req *CompleteSignUpOrSignInRequest) (*responses.AuthResponse, error) {
	ctx := req.Context
	var authResp *responses.AuthResponse
	var err error

	if req.IsNewUser {
		// For new users, call SignUp which handles membership via decorator
		signupReq := &auth.SignUpRequest{
			Email:      req.Email,
			Password:   req.Password,
			Name:       req.Name,
			RememberMe: req.RememberMe,
			IPAddress:  req.IPAddress,
			UserAgent:  req.UserAgent,
		}
		authResp, err = s.authService.SignUp(ctx, signupReq)
		if err != nil {
			return nil, errs.Wrap(err, "SIGNUP_FAILED", "Failed to sign up user", 500)
		}
	} else {
		// For existing users, create session (membership check via decorator)
		if req.User == nil {
			return nil, errs.New("USER_REQUIRED", "User is required for signin", 400)
		}
		authResp, err = s.authService.CreateSessionForUser(ctx, req.User, req.RememberMe, req.IPAddress, req.UserAgent)
		if err != nil {
			return nil, errs.Wrap(err, "SESSION_CREATION_FAILED", "Failed to create session", 500)
		}
	}

	// 2. Get app ID from context
	appID, _ := contexts.GetAppID(ctx)

	// 3. Track device if enabled (only if we have a user and session was created)
	if s.deviceService != nil && authResp != nil && authResp.User != nil {
		fingerprint := req.UserAgent + "|" + req.IPAddress
		fmt.Printf("[CompletionService] Tracking device for user %s in app %s\n", authResp.User.ID.String(), appID.String())
		_, _ = s.deviceService.TrackDevice(ctx, appID, authResp.User.ID, fingerprint, req.UserAgent, req.IPAddress)
	}

	// 4. Set session cookie if enabled
	// Check if we should attempt to set cookies - appService.GetCookieConfig will get
	// the proper config (app-specific or global default)
	if req.ForgeContext != nil && authResp != nil {
		s.setSessionCookie(ctx, req.ForgeContext, authResp, appID)
	}

	// 5. Audit log the authentication
	if s.auditService != nil && authResp != nil && authResp.User != nil {
		action := req.AuthMethod
		if req.IsNewUser {
			action = "signup_" + req.AuthMethod
		} else {
			action = "signin_" + req.AuthMethod
		}
		if req.AuthProvider != "" {
			action = action + "_" + req.AuthProvider
		}
		userID := authResp.User.ID
		_ = s.auditService.Log(ctx, &userID, action, "user:"+userID.String(), req.IPAddress, req.UserAgent, "")
	}

	return authResp, nil
}

// setSessionCookie sets the session cookie in the response
func (s *CompletionService) setSessionCookie(ctx context.Context, c forge.Context, authResp *responses.AuthResponse, appID xid.ID) {
	if authResp.Session == nil || authResp.Token == "" {
		fmt.Printf("[CompletionService] Skipping cookie - no session or token\n")
		return
	}

	// Get app-specific cookie config first (includes global config as fallback)
	var cookieConfig *session.CookieConfig
	if s.appService != nil {
		appCookieCfg, err := s.appService.GetCookieConfig(ctx, appID)
		if err == nil && appCookieCfg != nil {
			cookieConfig = appCookieCfg
			fmt.Printf("[CompletionService] Got cookie config from appService: enabled=%v, name=%s\n", cookieConfig.Enabled, cookieConfig.Name)
		} else if err != nil {
			fmt.Printf("[CompletionService] Error getting cookie config from appService: %v\n", err)
		}
	}

	// Fallback to completion service's cookie config
	if cookieConfig == nil && s.cookieConfig != nil {
		cookieConfig = s.cookieConfig
		fmt.Printf("[CompletionService] Using fallback cookie config: enabled=%v, name=%s\n", cookieConfig.Enabled, cookieConfig.Name)
	}

	if cookieConfig == nil {
		fmt.Printf("[CompletionService] No cookie config available - skipping cookie\n")
		return
	}

	if !cookieConfig.Enabled {
		fmt.Printf("[CompletionService] Cookie config disabled - skipping cookie\n")
		return
	}

	fmt.Printf("[CompletionService] Setting session cookie: name=%s, token=%s...\n", cookieConfig.Name, authResp.Token[:min(10, len(authResp.Token))])
	if err := session.SetCookie(c, authResp.Token, authResp.Session.ExpiresAt, cookieConfig); err != nil {
		fmt.Printf("[CompletionService] Error setting cookie: %v\n", err)
	}
}

// AppServiceAdapter adapts app.AppService to AppServiceInterface
type AppServiceAdapter struct {
	AppService interface {
		GetCookieConfig(ctx context.Context, appID xid.ID) (*session.CookieConfig, error)
	}
}

func (a *AppServiceAdapter) GetCookieConfig(ctx context.Context, appID xid.ID) (*session.CookieConfig, error) {
	return a.AppService.GetCookieConfig(ctx, appID)
}

// DeviceServiceAdapter adapts device.Service to DeviceServiceInterface
type DeviceServiceAdapter struct {
	DeviceService *device.Service
}

func (d *DeviceServiceAdapter) TrackDevice(ctx context.Context, appID, userID xid.ID, fingerprint, userAgent, ipAddress string) (*device.Device, error) {
	return d.DeviceService.TrackDevice(ctx, appID, userID, fingerprint, userAgent, ipAddress)
}

// ExtractClientIP extracts client IP from request
// This can be enhanced to check X-Forwarded-For headers
func ExtractClientIP(remoteAddr string) string {
	return remoteAddr
}
