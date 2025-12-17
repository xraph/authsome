package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	aud "github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/device"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/pagination"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/responses"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	coreuser "github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

type AuthHandler struct {
	auth              auth.ServiceInterface
	rl                *rl.Service
	dev               *dev.Service
	sec               *sec.Service
	aud               *aud.Service
	twofaRepo         *repo.TwoFARepository
	sessionCookieName string
	appService        *app.ServiceImpl
	cookieConfig      *session.CookieConfig
}

// Use shared response types
type TwoFARequiredResponse = responses.TwoFARequiredResponse
type SessionResponse = responses.SessionResponse

func NewAuthHandler(a auth.ServiceInterface, rlsvc *rl.Service, dsvc *dev.Service, ssvc *sec.Service, asvc *aud.Service, tfrepo *repo.TwoFARepository, appSvc *app.ServiceImpl, cookieCfg *session.CookieConfig) *AuthHandler {
	// Set default cookie name if not provided (for backward compatibility)
	sessionCookieName := "authsome_session"
	if cookieCfg != nil && cookieCfg.Name != "" {
		sessionCookieName = cookieCfg.Name
	}

	return &AuthHandler{
		auth:              a,
		rl:                rlsvc,
		dev:               dsvc,
		sec:               ssvc,
		aud:               asvc,
		twofaRepo:         tfrepo,
		sessionCookieName: sessionCookieName,
		appService:        appSvc,
		cookieConfig:      cookieCfg,
	}
}

func (h *AuthHandler) SignUp(c forge.Context) error {
	// Verify AppID is in context (populated by middleware from API key)
	if _, ok := contexts.GetAppID(c.Request().Context()); !ok {
		return c.JSON(http.StatusUnauthorized, errs.New("API_KEY_REQUIRED", "Valid API key required for app identification", http.StatusUnauthorized))
	}

	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	// Parse IP without port and enforce security rules
	ip := clientIPFromRequest(c.Request(), h.sec)
	if h.sec != nil {
		if allowed := h.sec.CheckIPAllowed(c.Request().Context(), ip); !allowed {
			_ = h.sec.LogEvent(c.Request().Context(), "ip_blocked_signup", nil, ip, c.Request().UserAgent(), "")
			return c.JSON(http.StatusForbidden, errs.New("IP_NOT_ALLOWED", "IP address not allowed", http.StatusForbidden))
		}
		// Geo-based restrictions if configured
		if ok := h.sec.CheckCountryAllowed(c.Request().Context(), ip); !ok {
			_ = h.sec.LogEvent(c.Request().Context(), "country_blocked_signup", nil, ip, c.Request().UserAgent(), "")
			return c.JSON(http.StatusForbidden, errs.New("GEO_RESTRICTED", "Geographic restriction", http.StatusForbidden))
		}
	}
	var req auth.SignUpRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}
	req.IPAddress = ip
	req.UserAgent = c.Request().UserAgent()
	res, err := h.auth.SignUp(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}
	if h.sec != nil && res.User != nil {
		uid := res.User.ID
		_ = h.sec.LogEvent(c.Request().Context(), "signup_success", &uid, ip, req.UserAgent, "")
	}
	if h.aud != nil && res.User != nil {
		uid := res.User.ID
		if err := h.aud.Log(c.Request().Context(), &uid, "signup", "user:"+uid.String(), ip, req.UserAgent, ""); err != nil {
			log.Printf("audit log signup error: %v", err)
		}
	}

	// Set session cookie if enabled
	if h.cookieConfig != nil && h.cookieConfig.Enabled && res.Session != nil && res.Token != "" {
		appID, _ := contexts.GetAppID(c.Request().Context())
		if h.appService != nil {
			appCookieCfg, err := h.appService.App.GetCookieConfig(c.Request().Context(), appID)
			if err == nil && appCookieCfg != nil && appCookieCfg.Enabled {
				_ = session.SetCookie(c, res.Token, res.Session.ExpiresAt, appCookieCfg)
			}
		}
	}

	return c.JSON(http.StatusOK, res)
}

func (h *AuthHandler) SignIn(c forge.Context) error {
	// Verify AppID is in context (populated by middleware from API key)
	if _, ok := contexts.GetAppID(c.Request().Context()); !ok {
		return c.JSON(http.StatusUnauthorized, errs.New("API_KEY_REQUIRED", "Valid API key required for app identification", http.StatusUnauthorized))
	}

	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	ip := clientIPFromRequest(c.Request(), h.sec)
	if h.sec != nil {
		if allowed := h.sec.CheckIPAllowed(c.Request().Context(), ip); !allowed {
			_ = h.sec.LogEvent(c.Request().Context(), "ip_blocked_signin", nil, ip, c.Request().UserAgent(), "")
			return c.JSON(http.StatusForbidden, errs.New("IP_NOT_ALLOWED", "IP address not allowed", http.StatusForbidden))
		}
		if ok := h.sec.CheckCountryAllowed(c.Request().Context(), ip); !ok {
			_ = h.sec.LogEvent(c.Request().Context(), "country_blocked_signin", nil, ip, c.Request().UserAgent(), "")
			return c.JSON(http.StatusForbidden, errs.New("GEO_RESTRICTED", "Geographic restriction", http.StatusForbidden))
		}
	}
	var req auth.SignInRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}
	// Lockout check now that we have email
	if h.sec != nil {
		lockKey := req.Email
		if lockKey == "" {
			lockKey = ip
		}
		if h.sec.IsLockedOut(c.Request().Context(), lockKey) {
			_ = h.sec.LogEvent(c.Request().Context(), "lockout_active", nil, ip, c.Request().UserAgent(), "")
			return c.JSON(423, errs.AccountLocked("too many failed attempts"))
		}
	}
	req.IPAddress = ip
	req.UserAgent = c.Request().UserAgent()
	// Separate credentials check from session creation to allow 2FA gating
	u, err := h.auth.CheckCredentials(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if h.sec != nil {
			lockKey := req.Email
			if lockKey == "" {
				lockKey = ip
			}
			h.sec.RecordFailedAttempt(c.Request().Context(), lockKey)
			_ = h.sec.LogEvent(c.Request().Context(), "signin_failed", nil, ip, req.UserAgent, "")
		}
		if h.aud != nil {
			_ = h.aud.Log(c.Request().Context(), nil, "signin_failed", "auth:signin", ip, req.UserAgent, "")
		}
		return c.JSON(http.StatusUnauthorized, errs.Wrap(err, "UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized))
	}
	// Determine device fingerprint
	fp := req.UserAgent + "|" + req.IPAddress
	// If 2FA is enabled and device is not trusted, return challenge requirement
	require2FA := false
	if h.twofaRepo != nil && u != nil {
		if sec, _ := h.twofaRepo.GetSecret(c.Request().Context(), u.ID); sec != nil && sec.Enabled {
			trusted, _ := h.twofaRepo.IsTrustedDevice(c.Request().Context(), u.ID, fp, time.Now())
			if !trusted {
				require2FA = true
			}
		}
	}
	if require2FA {
		// Track device even when requiring 2FA
		if h.dev != nil {
			appID, _ := contexts.GetAppID(c.Request().Context())
			_, _ = h.dev.TrackDevice(c.Request().Context(), appID, u.ID, fp, req.UserAgent, req.IPAddress)
		}
		if h.sec != nil {
			lockKey := req.Email
			if lockKey == "" {
				lockKey = ip
			}
			h.sec.ResetFailedAttempts(c.Request().Context(), lockKey)
			uid := u.ID
			_ = h.sec.LogEvent(c.Request().Context(), "signin_twofa_required", &uid, ip, req.UserAgent, "")
		}
		if h.aud != nil {
			uid := u.ID
			_ = h.aud.Log(c.Request().Context(), &uid, "signin_twofa_required", "user:"+uid.String(), ip, req.UserAgent, "")
		}
		return c.JSON(http.StatusOK, &TwoFARequiredResponse{User: u, RequireTwoFA: true, DeviceID: fp})
	}
	// Otherwise, create session and return normal auth response
	res, err := h.auth.CreateSessionForUser(c.Request().Context(), u, req.RememberMe, req.IPAddress, req.UserAgent)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}
	// Track device on successful login
	if h.dev != nil && res.User != nil {
		appID, _ := contexts.GetAppID(c.Request().Context())
		_, _ = h.dev.TrackDevice(c.Request().Context(), appID, res.User.ID, fp, req.UserAgent, req.IPAddress)
	}
	if h.sec != nil && res.User != nil {
		// Reset failed attempts on success
		lockKey := req.Email
		if lockKey == "" {
			lockKey = ip
		}
		h.sec.ResetFailedAttempts(c.Request().Context(), lockKey)
		uid := res.User.ID
		_ = h.sec.LogEvent(c.Request().Context(), "signin_success", &uid, ip, req.UserAgent, "")
	}
	if h.aud != nil && res.User != nil {
		uid := res.User.ID
		if err := h.aud.Log(c.Request().Context(), &uid, "signin", "user:"+uid.String(), ip, req.UserAgent, ""); err != nil {
			log.Printf("audit log signin error: %v", err)
		}
	}

	// Set session cookie if enabled
	if h.cookieConfig != nil && h.cookieConfig.Enabled && res.Session != nil && res.Token != "" {
		appID, _ := contexts.GetAppID(c.Request().Context())
		if h.appService != nil {
			appCookieCfg, err := h.appService.App.GetCookieConfig(c.Request().Context(), appID)
			if err == nil && appCookieCfg != nil && appCookieCfg.Enabled {
				_ = session.SetCookie(c, res.Token, res.Session.ExpiresAt, appCookieCfg)
			}
		}
	}

	return c.JSON(http.StatusOK, res)
}

// UpdateUser updates the authenticated user's profile (name, image, username)
func (h *AuthHandler) UpdateUser(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	// Require authentication via session cookie
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}
	token := cookie.Value
	res, err := h.auth.GetSession(c.Request().Context(), token)
	if err != nil || res.User == nil {
		return c.JSON(http.StatusUnauthorized, errs.SessionInvalid())
	}
	// Parse request
	var body struct {
		Name            *string `json:"name"`
		Image           *string `json:"image"`
		Username        *string `json:"username"`
		DisplayUsername *string `json:"display_username"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// Update via auth service
	updated, err := h.auth.UpdateUser(c.Request().Context(), res.User.ID, &coreuser.UpdateUserRequest{
		Name:            body.Name,
		Image:           body.Image,
		Username:        body.Username,
		DisplayUsername: body.DisplayUsername,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}
	// Audit event
	if h.aud != nil {
		uid := updated.ID
		ip := clientIPFromRequest(c.Request(), h.sec)
		_ = h.aud.Log(c.Request().Context(), &uid, "user_updated", "user:"+uid.String(), ip, c.Request().UserAgent(), "")
	}
	return c.JSON(http.StatusOK, updated)
}

// clientIPFromRequest attempts to extract the original client IP.
// Honors forwarded headers only if the security service is configured to trust them.
func clientIPFromRequest(r *http.Request, ssvc *sec.Service) string {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	trust := false
	if ssvc != nil {
		trust = ssvc.ShouldTrustForwardedHeaders(remote)
	}
	if trust {
		// X-Forwarded-For: may be a comma-separated list of IPs
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				ip := strings.TrimSpace(parts[0])
				if ip != "" {
					if host, _, err := net.SplitHostPort(ip); err == nil {
						return host
					}
					return ip
				}
			}
		}
		// X-Real-IP: direct IP
		if xr := r.Header.Get("X-Real-IP"); xr != "" {
			if host, _, err := net.SplitHostPort(xr); err == nil {
				return host
			}
			return xr
		}
		// Forwarded: e.g., for=192.0.2.60;proto=http;by=203.0.113.43
		if fwd := r.Header.Get("Forwarded"); fwd != "" {
			lower := strings.ToLower(fwd)
			if idx := strings.Index(lower, "for="); idx >= 0 {
				rest := fwd[idx+4:]
				// value may be quoted or bracketed for IPv6
				rest = strings.Trim(rest, "\"[]")
				// cut at delimiter
				if cut := strings.IndexAny(rest, ";, "); cut > 0 {
					rest = rest[:cut]
				}
				if rest != "" {
					if host, _, err := net.SplitHostPort(rest); err == nil {
						return host
					}
					return rest
				}
			}
		}
	}
	return remote
}

func (h *AuthHandler) SignOut(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}

	// Try to get token from multiple sources:
	// 1. Session from context (set by auth middleware)
	// 2. Token from request body
	// 3. Token from cookie
	var token string
	var userID *xid.ID

	// Try to get session from auth context (middleware-based auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if ok && authCtx.Session != nil {
		token = authCtx.Session.Token
		if authCtx.User != nil {
			userID = &authCtx.User.ID
		}
	}

	// If no session in context, try request body
	if token == "" {
		var body struct {
			Token string `json:"token,omitempty"`
		}
		// Ignore decode errors since token is optional when using cookies
		_ = json.NewDecoder(c.Request().Body).Decode(&body)
		if body.Token != "" {
			token = body.Token
		}
	}

	// If still no token, try cookie
	if token == "" {
		cookie, err := c.Request().Cookie(h.sessionCookieName)
		if err == nil && cookie != nil && cookie.Value != "" {
			token = cookie.Value
		}
	}

	// If we still don't have a token, return error
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Token required: provide via cookie, request body, or authentication middleware", http.StatusBadRequest))
	}

	// Sign out the session
	if err := h.auth.SignOut(c.Request().Context(), &auth.SignOutRequest{Token: token}); err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}

	// Clear the session cookie if cookie config is available
	if h.cookieConfig != nil && h.cookieConfig.Enabled {
		appID, _ := contexts.GetAppID(c.Request().Context())
		if h.appService != nil {
			appCookieCfg, err := h.appService.App.GetCookieConfig(c.Request().Context(), appID)
			if err == nil && appCookieCfg != nil && appCookieCfg.Enabled {
				_ = session.ClearCookie(c, appCookieCfg)
			}
		} else {
			// Fall back to global cookie config
			_ = session.ClearCookie(c, h.cookieConfig)
		}
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		_ = h.aud.Log(c.Request().Context(), userID, "signout", "auth:session", ip, c.Request().UserAgent(), "")
	}

	return c.JSON(http.StatusOK, &StatusResponse{Status: "signed_out"})
}

// RefreshSession refreshes an access token using a refresh token
func (h *AuthHandler) RefreshSession(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}

	// Get refresh token from request body
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_REFRESH_TOKEN", "Refresh token is required", http.StatusBadRequest))
	}

	// Refresh the session via auth service
	refreshResp, err := h.auth.RefreshSession(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Wrap(err, "REFRESH_FAILED", "Failed to refresh session", http.StatusUnauthorized))
	}

	// Log audit event
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		userID := refreshResp.Session.UserID
		_ = h.aud.Log(c.Request().Context(), &userID, "session_refreshed", "auth:session", ip, c.Request().UserAgent(), "")
	}

	// Return response with new tokens
	return c.JSON(http.StatusOK, map[string]interface{}{
		"session":          refreshResp.Session,
		"accessToken":      refreshResp.AccessToken,
		"refreshToken":     refreshResp.RefreshToken,
		"expiresAt":        refreshResp.ExpiresAt,
		"refreshExpiresAt": refreshResp.RefreshExpiresAt,
	})
}

func (h *AuthHandler) GetSession(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	cookie, err := c.Request().Cookie(h.sessionCookieName)
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}
	token := cookie.Value
	res, err := h.auth.GetSession(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errs.Wrap(err, "UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized))
	}
	if h.aud != nil && res.User != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		uid := res.User.ID
		_ = h.aud.Log(c.Request().Context(), &uid, "session_checked", "session:"+res.Session.ID.String(), ip, c.Request().UserAgent(), "")
	}
	return c.JSON(http.StatusOK, &SessionResponse{
		User:    res.User,
		Session: res.Session,
	})
}

// ListDevices lists devices for the authenticated user
func (h *AuthHandler) ListDevices(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}
	token := cookie.Value
	res, err := h.auth.GetSession(c.Request().Context(), token)
	if err != nil || res.User == nil {
		return c.JSON(http.StatusUnauthorized, errs.SessionInvalid())
	}
	list, err := h.dev.ListDevices(c.Request().Context(), &device.ListDevicesFilter{
		UserID: res.User.ID,
		PaginationParams: pagination.PaginationParams{
			Limit:  50,
			Offset: 0,
		},
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		uid := res.User.ID
		_ = h.aud.Log(c.Request().Context(), &uid, "devices_listed", "user:"+uid.String(), ip, c.Request().UserAgent(), "")
	}
	return c.JSON(http.StatusOK, list)
}

// RevokeDevice deletes a device by fingerprint for the authenticated user
func (h *AuthHandler) RevokeDevice(c forge.Context) error {
	if h.rl != nil {
		key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
		if err != nil || !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", http.StatusTooManyRequests))
		}
	}
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}
	token := cookie.Value
	res, err := h.auth.GetSession(c.Request().Context(), token)
	if err != nil || res.User == nil {
		return c.JSON(http.StatusUnauthorized, errs.SessionInvalid())
	}
	var body struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Fingerprint == "" {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if err := h.dev.RevokeDevice(c.Request().Context(), res.User.ID, body.Fingerprint); err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "BAD_REQUEST", "Bad request", http.StatusBadRequest))
	}
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		uid := res.User.ID
		_ = h.aud.Log(c.Request().Context(), &uid, "device_revoked", "user:"+uid.String(), ip, c.Request().UserAgent(), "fingerprint="+body.Fingerprint)
	}
	return c.JSON(http.StatusOK, &StatusResponse{Status: "device_revoked"})
}

// RequestPasswordReset handles password reset requests
func (h *AuthHandler) RequestPasswordReset(c forge.Context) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	// Rate limiting
	if h.rl != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		key := "password_reset:" + ip
		allowed, err := h.rl.CheckLimit(c.Request().Context(), key, rl.Rule{Max: 3, Window: time.Hour})
		if err != nil || !allowed {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many password reset requests", http.StatusTooManyRequests))
		}
	}

	// Request password reset
	token, err := h.auth.RequestPasswordReset(c.Request().Context(), req.Email)
	if err != nil {
		// Log error but still return success to prevent email enumeration
		log.Printf("Password reset error: %v", err)
	}

	// Get base URL from app (default to empty, can be configured per app)
	baseURL := ""
	if h.appService != nil {
		if app, err := h.appService.FindAppByID(c.Request().Context(), appID); err == nil && app != nil {
			// Try to get baseURL from metadata if configured
			if app.Metadata != nil {
				if url, ok := app.Metadata["baseURL"].(string); ok && url != "" {
					baseURL = url
				}
			}
		}
	}

	// If we have a token, send notification
	if token != "" && h.auth != nil {
		// Get notification adapter from service registry if available
		if authService, ok := h.auth.(interface {
			GetServiceRegistry() interface {
				Get(string) (interface{}, error)
			}
		}); ok {
			if registry := authService.GetServiceRegistry(); registry != nil {
				if adapterIntf, err := registry.Get("notification.adapter"); err == nil && adapterIntf != nil {
					if adapter, ok := adapterIntf.(interface {
						SendPasswordReset(ctx context.Context, appID xid.ID, email, userName, resetURL, resetCode string, expiryMinutes int) error
					}); ok {
						// Construct reset URL
						resetURL := baseURL + "/auth/reset-password?token=" + token

						// Try to get user name
						userName := req.Email
						if userService, ok := h.auth.(interface {
							GetUserService() interface {
								FindByEmail(context.Context, string) (interface {
									GetName() string
									GetEmail() string
								}, error)
							}
						}); ok {
							if svc := userService.GetUserService(); svc != nil {
								if user, err := svc.FindByEmail(c.Request().Context(), req.Email); err == nil && user != nil {
									if name := user.GetName(); name != "" {
										userName = name
									}
								}
							}
						}

						// Send password reset email
						_ = adapter.SendPasswordReset(
							c.Request().Context(),
							appID,
							req.Email,
							userName,
							resetURL,
							token,
							60, // 60 minutes expiry
						)
					}
				}
			}
		}
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		_ = h.aud.Log(c.Request().Context(), nil, "password_reset_requested", "email:"+req.Email, ip, c.Request().UserAgent(), "")
	}

	// Always return success to prevent email enumeration
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset confirmation
func (h *AuthHandler) ResetPassword(c forge.Context) error {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"newPassword" validate:"required,min=8"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Reset password
	err := h.auth.ResetPassword(c.Request().Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidResetToken) {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_TOKEN", "Invalid or expired reset token", http.StatusBadRequest))
		}
		if errors.Is(err, auth.ErrResetTokenExpired) {
			return c.JSON(http.StatusBadRequest, errs.New("TOKEN_EXPIRED", "Reset token has expired", http.StatusBadRequest))
		}
		if errors.Is(err, auth.ErrResetTokenAlreadyUsed) {
			return c.JSON(http.StatusBadRequest, errs.New("TOKEN_USED", "Reset token has already been used", http.StatusBadRequest))
		}
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "RESET_FAILED", "Failed to reset password", http.StatusInternalServerError))
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		_ = h.aud.Log(c.Request().Context(), nil, "password_reset_completed", "token", ip, c.Request().UserAgent(), "")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Password has been reset successfully",
	})
}

// ValidateResetToken validates a password reset token
func (h *AuthHandler) ValidateResetToken(c forge.Context) error {
	token := c.Request().URL.Query().Get("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Token parameter required", http.StatusBadRequest))
	}

	valid, err := h.auth.ValidateResetToken(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "VALIDATION_FAILED", "Failed to validate token", http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid": valid,
	})
}

// ChangePassword handles password change requests
func (h *AuthHandler) ChangePassword(c forge.Context) error {
	res, err := h.getAuthenticatedUser(c)
	if err != nil || res == nil || res.User == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	var req struct {
		OldPassword string `json:"oldPassword" validate:"required"`
		NewPassword string `json:"newPassword" validate:"required,min=8"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Change password
	err = h.auth.ChangePassword(c.Request().Context(), res.User.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, types.ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, errs.New("INVALID_OLD_PASSWORD", "Current password is incorrect", http.StatusUnauthorized))
		}
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "PASSWORD_CHANGE_FAILED", "Failed to change password", http.StatusInternalServerError))
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		uid := res.User.ID
		_ = h.aud.Log(c.Request().Context(), &uid, "password_changed", "user:"+uid.String(), ip, c.Request().UserAgent(), "")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Password changed successfully",
	})
}

// RequestEmailChange handles email change requests
func (h *AuthHandler) RequestEmailChange(c forge.Context) error {
	res, err := h.getAuthenticatedUser(c)
	if err != nil || res == nil || res.User == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	var req struct {
		NewEmail string `json:"newEmail" validate:"required,email"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Request email change
	changeToken, err := h.auth.RequestEmailChange(c.Request().Context(), res.User.ID, req.NewEmail)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.Wrap(err, "EMAIL_CHANGE_FAILED", "Failed to request email change", http.StatusBadRequest))
	}

	// Get base URL from app
	appID, _ := contexts.GetAppID(c.Request().Context())
	baseURL := ""
	if h.appService != nil && !appID.IsNil() {
		if app, err := h.appService.FindAppByID(c.Request().Context(), appID); err == nil && app != nil {
			// Try to get baseURL from metadata if configured
			if app.Metadata != nil {
				if url, ok := app.Metadata["baseURL"].(string); ok && url != "" {
					baseURL = url
				}
			}
		}
	}

	// Send notification with confirmation URL
	if changeToken != "" && h.auth != nil {
		// Get notification adapter from service registry if available
		if authService, ok := h.auth.(interface {
			GetServiceRegistry() interface {
				Get(string) (interface{}, error)
			}
		}); ok {
			if registry := authService.GetServiceRegistry(); registry != nil {
				if adapterIntf, err := registry.Get("notification.adapter"); err == nil && adapterIntf != nil {
					if adapter, ok := adapterIntf.(interface {
						SendEmailChangeRequest(ctx context.Context, appID xid.ID, recipientEmail, userName, newEmail, confirmationUrl, timestamp string) error
					}); ok {
						// Construct confirmation URL
						confirmationURL := baseURL + "/auth/email/change/confirm?token=" + changeToken

						userName := res.User.Name
						if userName == "" {
							userName = res.User.Email
						}

						timestamp := time.Now().Format(time.RFC3339)

						// Send to OLD email for security
						_ = adapter.SendEmailChangeRequest(
							c.Request().Context(),
							appID,
							res.User.Email,
							userName,
							req.NewEmail,
							confirmationURL,
							timestamp,
						)
					}
				}
			}
		}
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		uid := res.User.ID
		_ = h.aud.Log(c.Request().Context(), &uid, "email_change_requested", "user:"+uid.String(), ip, c.Request().UserAgent(), "new_email="+req.NewEmail)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Email change confirmation sent to your current email address",
	})
}

// ConfirmEmailChange handles email change confirmation
func (h *AuthHandler) ConfirmEmailChange(c forge.Context) error {
	var req struct {
		Token string `json:"token" validate:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Confirm email change
	err := h.auth.ConfirmEmailChange(c.Request().Context(), req.Token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidChangeToken) {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_TOKEN", "Invalid or expired email change token", http.StatusBadRequest))
		}
		if errors.Is(err, auth.ErrChangeTokenExpired) {
			return c.JSON(http.StatusBadRequest, errs.New("TOKEN_EXPIRED", "Email change token has expired", http.StatusBadRequest))
		}
		if errors.Is(err, auth.ErrChangeTokenAlreadyUsed) {
			return c.JSON(http.StatusBadRequest, errs.New("TOKEN_USED", "Email change token has already been used", http.StatusBadRequest))
		}
		return c.JSON(http.StatusInternalServerError, errs.Wrap(err, "EMAIL_CHANGE_FAILED", "Failed to change email", http.StatusInternalServerError))
	}

	// Audit log
	if h.aud != nil {
		ip := clientIPFromRequest(c.Request(), h.sec)
		_ = h.aud.Log(c.Request().Context(), nil, "email_change_confirmed", "token", ip, c.Request().UserAgent(), "")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Email address has been changed successfully",
	})
}

// getAuthenticatedUser retrieves the authenticated user from the session cookie
func (h *AuthHandler) getAuthenticatedUser(c forge.Context) (*responses.AuthResponse, error) {
	// Get session token from cookie
	cookie, err := c.Request().Cookie(h.sessionCookieName)
	if err != nil {
		return nil, err
	}

	// Get session from auth service
	authResp, err := h.auth.GetSession(c.Request().Context(), cookie.Value)
	if err != nil {
		return nil, err
	}

	return authResp, nil
}
