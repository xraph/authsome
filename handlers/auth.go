package handlers

import (
    "encoding/json"
    "net"
    "net/http"
    "strings"
    "time"
    "log"

    "github.com/xraph/authsome/core/auth"
    coreuser "github.com/xraph/authsome/core/user"
    aud "github.com/xraph/authsome/core/audit"
    rl "github.com/xraph/authsome/core/ratelimit"
    dev "github.com/xraph/authsome/core/device"
    sec "github.com/xraph/authsome/core/security"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/forge"
)

type AuthHandler struct {
    auth auth.ServiceInterface
    rl   *rl.Service
    dev  *dev.Service
    sec  *sec.Service
    aud  *aud.Service
    twofaRepo *repo.TwoFARepository
}

func NewAuthHandler(a auth.ServiceInterface, rlsvc *rl.Service, dsvc *dev.Service, ssvc *sec.Service, asvc *aud.Service, tfrepo *repo.TwoFARepository) *AuthHandler {
    return &AuthHandler{auth: a, rl: rlsvc, dev: dsvc, sec: ssvc, aud: asvc, twofaRepo: tfrepo}
}

func (h *AuthHandler) SignUp(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    // Parse IP without port and enforce security rules
    ip := clientIPFromRequest(c.Request(), h.sec)
    if h.sec != nil {
        if allowed := h.sec.CheckIPAllowed(c.Request().Context(), ip); !allowed {
            _ = h.sec.LogEvent(c.Request().Context(), "ip_blocked_signup", nil, ip, c.Request().UserAgent(), "")
            return c.JSON(403, map[string]string{"error": "ip not allowed"})
        }
        // Geo-based restrictions if configured
        if ok := h.sec.CheckCountryAllowed(c.Request().Context(), ip); !ok {
            _ = h.sec.LogEvent(c.Request().Context(), "country_blocked_signup", nil, ip, c.Request().UserAgent(), "")
            return c.JSON(403, map[string]string{"error": "geo restriction"})
        }
    }
    var req auth.SignUpRequest
    if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    req.IPAddress = ip
    req.UserAgent = c.Request().UserAgent()
    res, err := h.auth.SignUp(c.Request().Context(), &req)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
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
    return c.JSON(200, res)
}

func (h *AuthHandler) SignIn(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    ip := clientIPFromRequest(c.Request(), h.sec)
    if h.sec != nil {
        if allowed := h.sec.CheckIPAllowed(c.Request().Context(), ip); !allowed {
            _ = h.sec.LogEvent(c.Request().Context(), "ip_blocked_signin", nil, ip, c.Request().UserAgent(), "")
            return c.JSON(403, map[string]string{"error": "ip not allowed"})
        }
        if ok := h.sec.CheckCountryAllowed(c.Request().Context(), ip); !ok {
            _ = h.sec.LogEvent(c.Request().Context(), "country_blocked_signin", nil, ip, c.Request().UserAgent(), "")
            return c.JSON(403, map[string]string{"error": "geo restriction"})
        }
    }
    var req auth.SignInRequest
    if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    // Lockout check now that we have email
    if h.sec != nil {
        lockKey := req.Email
        if lockKey == "" { lockKey = ip }
        if h.sec.IsLockedOut(c.Request().Context(), lockKey) {
            _ = h.sec.LogEvent(c.Request().Context(), "lockout_active", nil, ip, c.Request().UserAgent(), "")
            return c.JSON(423, map[string]string{"error": "account temporarily locked"})
        }
    }
    req.IPAddress = ip
    req.UserAgent = c.Request().UserAgent()
    // Separate credentials check from session creation to allow 2FA gating
    u, err := h.auth.CheckCredentials(c.Request().Context(), req.Email, req.Password)
    if err != nil {
        if h.sec != nil {
            lockKey := req.Email
            if lockKey == "" { lockKey = ip }
            h.sec.RecordFailedAttempt(c.Request().Context(), lockKey)
            _ = h.sec.LogEvent(c.Request().Context(), "signin_failed", nil, ip, req.UserAgent, "")
        }
        if h.aud != nil {
            _ = h.aud.Log(c.Request().Context(), nil, "signin_failed", "auth:signin", ip, req.UserAgent, "")
        }
        return c.JSON(401, map[string]string{"error": err.Error()})
    }
    // Determine device fingerprint
    fp := req.UserAgent + "|" + req.IPAddress
    // If 2FA is enabled and device is not trusted, return challenge requirement
    require2FA := false
    if h.twofaRepo != nil && u != nil {
        if sec, _ := h.twofaRepo.GetSecret(c.Request().Context(), u.ID); sec != nil && sec.Enabled {
            trusted, _ := h.twofaRepo.IsTrustedDevice(c.Request().Context(), u.ID, fp, time.Now())
            if !trusted { require2FA = true }
        }
    }
    if require2FA {
        // Track device even when requiring 2FA
        if h.dev != nil {
            _, _ = h.dev.TrackDevice(c.Request().Context(), u.ID, fp, req.UserAgent, req.IPAddress)
        }
        if h.sec != nil {
            lockKey := req.Email
            if lockKey == "" { lockKey = ip }
            h.sec.ResetFailedAttempts(c.Request().Context(), lockKey)
            uid := u.ID
            _ = h.sec.LogEvent(c.Request().Context(), "signin_twofa_required", &uid, ip, req.UserAgent, "")
        }
        if h.aud != nil {
            uid := u.ID
            _ = h.aud.Log(c.Request().Context(), &uid, "signin_twofa_required", "user:"+uid.String(), ip, req.UserAgent, "")
        }
        return c.JSON(200, map[string]interface{}{"user": u, "require_twofa": true, "device_id": fp})
    }
    // Otherwise, create session and return normal auth response
    res, err := h.auth.CreateSessionForUser(c.Request().Context(), u, req.Remember || req.RememberMe, req.IPAddress, req.UserAgent)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    // Track device on successful login
    if h.dev != nil && res.User != nil {
        _, _ = h.dev.TrackDevice(c.Request().Context(), res.User.ID, fp, req.UserAgent, req.IPAddress)
    }
    if h.sec != nil && res.User != nil {
        // Reset failed attempts on success
        lockKey := req.Email
        if lockKey == "" { lockKey = ip }
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
    return c.JSON(200, res)
}

// UpdateUser updates the authenticated user's profile (name, image, username)
func (h *AuthHandler) UpdateUser(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    // Require authentication via session cookie
    cookie, err := c.Request().Cookie("session_token")
    if err != nil || cookie == nil {
        return c.JSON(401, map[string]string{"error": "not authenticated"})
    }
    token := cookie.Value
    res, err := h.auth.GetSession(c.Request().Context(), token)
    if err != nil || res.User == nil {
        return c.JSON(401, map[string]string{"error": "invalid session"})
    }
    // Parse request
    var body struct {
        Name            *string `json:"name"`
        Image           *string `json:"image"`
        Username        *string `json:"username"`
        DisplayUsername *string `json:"display_username"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    // Update via auth service
    updated, err := h.auth.UpdateUser(c.Request().Context(), res.User.ID, &coreuser.UpdateUserRequest{
        Name:            body.Name,
        Image:           body.Image,
        Username:        body.Username,
        DisplayUsername: body.DisplayUsername,
    })
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    // Audit event
    if h.aud != nil {
        uid := updated.ID
        ip := clientIPFromRequest(c.Request(), h.sec)
        _ = h.aud.Log(c.Request().Context(), &uid, "user_updated", "user:"+uid.String(), ip, c.Request().UserAgent(), "")
    }
    return c.JSON(200, updated)
}

// clientIPFromRequest attempts to extract the original client IP.
// Honors forwarded headers only if the security service is configured to trust them.
func clientIPFromRequest(r *http.Request, ssvc *sec.Service) string {
    remote := r.RemoteAddr
    if host, _, err := net.SplitHostPort(remote); err == nil { remote = host }
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
                    if host, _, err := net.SplitHostPort(ip); err == nil { return host }
                    return ip
                }
            }
        }
        // X-Real-IP: direct IP
        if xr := r.Header.Get("X-Real-IP"); xr != "" {
            if host, _, err := net.SplitHostPort(xr); err == nil { return host }
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
                    if host, _, err := net.SplitHostPort(rest); err == nil { return host }
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
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    var body struct{ Token string `json:"token"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Token == "" {
        return c.JSON(400, map[string]string{"error": "missing token"})
    }
    if err := h.auth.SignOut(c.Request().Context(), &auth.SignOutRequest{Token: body.Token}); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    if h.aud != nil {
        ip := clientIPFromRequest(c.Request(), h.sec)
        _ = h.aud.Log(c.Request().Context(), nil, "signout", "auth:session", ip, c.Request().UserAgent(), "")
    }
    return c.JSON(200, map[string]string{"status": "signed_out"})
}

func (h *AuthHandler) GetSession(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    cookie, err := c.Request().Cookie("session_token")
    if err != nil || cookie == nil {
        return c.JSON(401, map[string]string{"error": "not authenticated"})
    }
    token := cookie.Value
    res, err := h.auth.GetSession(c.Request().Context(), token)
    if err != nil {
        return c.JSON(401, map[string]string{"error": err.Error()})
    }
    if h.aud != nil && res.User != nil {
        ip := clientIPFromRequest(c.Request(), h.sec)
        uid := res.User.ID
        _ = h.aud.Log(c.Request().Context(), &uid, "session_checked", "session:"+res.Session.ID.String(), ip, c.Request().UserAgent(), "")
    }
    return c.JSON(200, map[string]interface{}{
        "user":    res.User,
        "session": res.Session,
    })
}

// ListDevices lists devices for the authenticated user
func (h *AuthHandler) ListDevices(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    cookie, err := c.Request().Cookie("session_token")
    if err != nil || cookie == nil {
        return c.JSON(401, map[string]string{"error": "not authenticated"})
    }
    token := cookie.Value
    res, err := h.auth.GetSession(c.Request().Context(), token)
    if err != nil || res.User == nil {
        return c.JSON(401, map[string]string{"error": "invalid session"})
    }
    list, err := h.dev.ListDevices(c.Request().Context(), res.User.ID, 50, 0)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    if h.aud != nil {
        ip := clientIPFromRequest(c.Request(), h.sec)
        uid := res.User.ID
        _ = h.aud.Log(c.Request().Context(), &uid, "devices_listed", "user:"+uid.String(), ip, c.Request().UserAgent(), "")
    }
    return c.JSON(200, list)
}

// RevokeDevice deletes a device by fingerprint for the authenticated user
func (h *AuthHandler) RevokeDevice(c forge.Context) error {
    if h.rl != nil {
        key := c.Request().RemoteAddr + ":" + c.Request().URL.Path
        ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, c.Request().URL.Path)
        if err != nil || !ok {
            return c.JSON(429, map[string]string{"error": "rate limit exceeded"})
        }
    }
    cookie, err := c.Request().Cookie("session_token")
    if err != nil || cookie == nil {
        return c.JSON(401, map[string]string{"error": "not authenticated"})
    }
    token := cookie.Value
    res, err := h.auth.GetSession(c.Request().Context(), token)
    if err != nil || res.User == nil {
        return c.JSON(401, map[string]string{"error": "invalid session"})
    }
    var body struct{ Fingerprint string `json:"fingerprint"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Fingerprint == "" {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if err := h.dev.RevokeDevice(c.Request().Context(), res.User.ID, body.Fingerprint); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    if h.aud != nil {
        ip := clientIPFromRequest(c.Request(), h.sec)
        uid := res.User.ID
        _ = h.aud.Log(c.Request().Context(), &uid, "device_revoked", "user:"+uid.String(), ip, c.Request().UserAgent(), "fingerprint="+body.Fingerprint)
    }
    return c.JSON(200, map[string]string{"status": "device_revoked"})
}