package username

import (
    "encoding/json"
    "net"
    "time"
    "strings"
    "github.com/xraph/authsome/internal/crypto"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for username auth
type Handler struct{
    svc *Service
    twofa *repo.TwoFARepository
}

func NewHandler(s *Service, tf *repo.TwoFARepository) *Handler { return &Handler{svc: s, twofa: tf} }

func (h *Handler) SignUp(c *forge.Context) error {
    var body struct{
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.Username == "" || body.Password == "" {
        return c.JSON(400, map[string]string{"error": "missing fields"})
    }
    if err := h.svc.SignUpWithUsername(c.Request().Context(), body.Username, body.Password); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(201, map[string]string{"status": "created"})
}

func (h *Handler) SignIn(c *forge.Context) error {
    var body struct{
        Username string `json:"username"`
        Password string `json:"password"`
        Remember  bool   `json:"remember"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    un := strings.ToLower(strings.TrimSpace(body.Username))
    if un == "" || body.Password == "" {
        return c.JSON(400, map[string]string{"error": "missing fields"})
    }
    // Lookup user by username and verify password
    u, err := h.svc.users.FindByUsername(c.Request().Context(), un)
    if err != nil || u == nil {
        return c.JSON(401, map[string]string{"error": "invalid credentials"})
    }
    if ok := crypto.CheckPassword(body.Password, u.PasswordHash); !ok {
        return c.JSON(401, map[string]string{"error": "invalid credentials"})
    }
    // Determine device fingerprint from IP and UA
    ip := c.Request().RemoteAddr
    if host, _, err := net.SplitHostPort(ip); err == nil { ip = host }
    ua := c.Request().UserAgent()
    fp := ua + "|" + ip
    // Check 2FA requirement and trusted device
    if h.twofa != nil {
        if sec, _ := h.twofa.GetSecret(c.Request().Context(), u.ID); sec != nil && sec.Enabled {
            trusted, _ := h.twofa.IsTrustedDevice(c.Request().Context(), u.ID, fp, time.Now())
            if !trusted {
                return c.JSON(200, map[string]interface{}{"user": u, "require_twofa": true, "device_id": fp})
            }
        }
    }
    // Create session via core auth service
    res, err := h.svc.auth.CreateSessionForUser(c.Request().Context(), u, body.Remember, ip, ua)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]interface{}{"user": res.User, "session": res.Session, "token": res.Token})
}