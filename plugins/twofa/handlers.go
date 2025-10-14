package twofa

import (
    "encoding/json"
    "github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for 2FA operations
type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

func (h *Handler) Enable(c *forge.Context) error {
    var body struct{
        UserID string `json:"user_id"`
        Method string `json:"method"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.UserID == "" {
        return c.JSON(400, map[string]string{"error": "missing user_id"})
    }
    bundle, err := h.svc.Enable(c.Request().Context(), body.UserID, &EnableRequest{Method: body.Method})
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    resp := map[string]interface{}{"status": "2fa_enabled"}
    if bundle != nil {
        resp["totp_uri"] = bundle.URI
    }
    return c.JSON(200, resp)
}

func (h *Handler) Verify(c *forge.Context) error {
    var body struct{
        UserID         string `json:"user_id"`
        Code           string `json:"code"`
        RememberDevice bool   `json:"remember_device"`
        DeviceID       string `json:"device_id"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.UserID == "" {
        return c.JSON(400, map[string]string{"error": "missing user_id"})
    }
    ok, err := h.svc.Verify(c.Request().Context(), body.UserID, &VerifyRequest{Code: body.Code})
    if err != nil || !ok {
        return c.JSON(401, map[string]string{"error": "invalid code"})
    }
    // Optionally mark device as trusted
    if body.RememberDevice && body.DeviceID != "" {
        _ = h.svc.MarkTrusted(c.Request().Context(), body.UserID, body.DeviceID, 30)
    }
    return c.JSON(200, map[string]string{"status": "verified"})
}

func (h *Handler) Disable(c *forge.Context) error {
    var body struct{ UserID string `json:"user_id"` }
    _ = json.NewDecoder(c.Request().Body).Decode(&body)
    if body.UserID == "" {
        return c.JSON(400, map[string]string{"error": "missing user_id"})
    }
    if err := h.svc.Disable(c.Request().Context(), body.UserID); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]string{"status": "2fa_disabled"})
}

func (h *Handler) GenerateBackupCodes(c *forge.Context) error {
    var body struct{ UserID string `json:"user_id"`; Count int `json:"count"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        body.Count = 10
    }
    if body.UserID == "" {
        return c.JSON(400, map[string]string{"error": "missing user_id"})
    }
    codes, err := h.svc.GenerateBackupCodes(c.Request().Context(), body.UserID, body.Count)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]interface{}{"codes": codes})
}

// SendOTP triggers generation of an OTP code for a user (returns code in response for dev/testing)
func (h *Handler) SendOTP(c *forge.Context) error {
    var body struct{ UserID string `json:"user_id"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.UserID == "" {
        return c.JSON(400, map[string]string{"error": "missing user_id"})
    }
    code, err := h.svc.SendOTP(c.Request().Context(), body.UserID)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    // In production, deliver via email/SMS; here we return for testing
    return c.JSON(200, map[string]interface{}{"status": "otp_sent", "code": code})
}

// Status returns whether 2FA is enabled and whether the device is trusted
func (h *Handler) Status(c *forge.Context) error {
    var body struct{
        UserID   string `json:"user_id"`
        DeviceID string `json:"device_id"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.UserID == "" { return c.JSON(400, map[string]string{"error": "missing user_id"}) }
    st, err := h.svc.GetStatus(c.Request().Context(), body.UserID, body.DeviceID)
    if err != nil {
        // Provide a friendlier message when the user_id is not a valid xid
        if err.Error() == "xid: invalid ID" {
            return c.JSON(400, map[string]string{"error": "invalid user_id"})
        }
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]interface{}{"enabled": st.Enabled, "method": st.Method, "trusted": st.Trusted})
}