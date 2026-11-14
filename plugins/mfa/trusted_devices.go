package mfa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
)

// DefaultTrustedDeviceTTL is how long a trusted device skips the second
// factor when Config.TrustedDeviceTTL is left at zero.
const DefaultTrustedDeviceTTL = 30 * 24 * time.Hour

// SettingTrustedDevicesEnabled lets an app turn "trust this device" off
// without redeploying. Off, the challenge ignores trust_device and no
// existing token skips the second factor.
var SettingTrustedDevicesEnabled = settings.Define("mfa.trusted_devices_enabled", true,
	settings.WithDisplayName("Trusted Devices"),
	settings.WithDescription("Let users skip the second factor on a device they chose to trust"),
	settings.WithCategory("Multi-Factor Auth"),
	settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
	settings.WithHelpText("When off, every sign-in asks for the second factor, including on devices trusted earlier"),
	settings.WithOrder(20),
)

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// TrustedDevicesRequest binds GET and DELETE /mfa/trusted-devices. It takes
// nothing: the devices are always the caller's own.
type TrustedDevicesRequest struct{}

// TrustedDeviceRequest binds DELETE /mfa/trusted-devices/:id.
type TrustedDeviceRequest struct {
	ID string `path:"id" description:"Trusted device identifier"`
}

// TrustedDeviceInfo is one device that currently skips the second factor.
type TrustedDeviceInfo struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"        description:"Browser and OS, e.g. Chrome on macOS"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"   description:"Address the device was trusted from"`
	CreatedAt  time.Time `json:"created_at"   description:"When the device was trusted"`
	LastUsedAt time.Time `json:"last_used_at" description:"When the device last skipped the second factor"`
	ExpiresAt  time.Time `json:"expires_at"   description:"When the trust runs out"`
	Current    bool      `json:"current"      description:"Whether this is the device making the request"`
}

// TrustedDevicesResponse lists the caller's live trusted devices, newest
// first.
type TrustedDevicesResponse struct {
	Devices []TrustedDeviceInfo `json:"devices"`
}

// TrustedDevicesRemovedResponse reports how many devices stopped being
// trusted.
type TrustedDevicesRemovedResponse struct {
	Status  string `json:"status"`
	Removed int    `json:"removed"`
}

// ──────────────────────────────────────────────────
// Routes
// ──────────────────────────────────────────────────

func (p *Plugin) registerTrustedDeviceRoutes(g forge.Router) error {
	if err := g.GET("/trusted-devices", p.handleListTrustedDevices,
		forge.WithSummary("List trusted devices"),
		forge.WithDescription("Lists the devices that currently skip the second factor for the signed-in user. The device making the request is marked current."),
		forge.WithOperationID("listMFATrustedDevices"),
		forge.WithResponseSchema(http.StatusOK, "Trusted devices", TrustedDevicesResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := g.DELETE("/trusted-devices/:id", p.handleRemoveTrustedDevice,
		forge.WithSummary("Remove a trusted device"),
		forge.WithDescription("The device asks for the second factor again at its next sign-in."),
		forge.WithOperationID("removeMFATrustedDevice"),
		forge.WithResponseSchema(http.StatusOK, "Device removed", TrustedDevicesRemovedResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	return g.DELETE("/trusted-devices", p.handleRemoveAllTrustedDevices,
		forge.WithSummary("Remove all trusted devices"),
		forge.WithDescription("Every device asks for the second factor again at its next sign-in."),
		forge.WithOperationID("removeAllMFATrustedDevices"),
		forge.WithResponseSchema(http.StatusOK, "Devices removed", TrustedDevicesRemovedResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleListTrustedDevices(ctx forge.Context, _ *TrustedDevicesRequest) (*TrustedDevicesResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}
	devices, err := p.liveTrustedDevices(ctx.Context(), userID)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to list trusted devices: %w", err))
	}
	current := ""
	if tok := authsome.TrustedDeviceToken(ctx.Request()); tok != "" {
		current = hashTrustedDeviceToken(tok)
	}
	resp := &TrustedDevicesResponse{Devices: make([]TrustedDeviceInfo, 0, len(devices))}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, TrustedDeviceInfo{
			ID:         d.ID.String(),
			Label:      d.Label,
			UserAgent:  d.UserAgent,
			IPAddress:  d.IPAddress,
			CreatedAt:  d.CreatedAt,
			LastUsedAt: d.LastUsedAt,
			ExpiresAt:  d.ExpiresAt,
			Current:    current != "" && d.TokenHash == current,
		})
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleRemoveTrustedDevice(ctx forge.Context, req *TrustedDeviceRequest) (*TrustedDevicesRemovedResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}
	deviceID, err := id.ParseTrustedDeviceID(req.ID)
	if err != nil {
		return nil, forge.BadRequest("invalid trusted device id")
	}
	devices, err := p.store.ListTrustedDevices(ctx.Context(), userID)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to list trusted devices: %w", err))
	}
	var target *TrustedDevice
	for _, d := range devices {
		if d.ID == deviceID {
			target = d
			break
		}
	}
	// Another user's device answers the same as one that does not exist.
	if target == nil {
		return nil, forge.NotFound("trusted device not found")
	}
	if err := p.store.DeleteTrustedDevice(ctx.Context(), userID, deviceID); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to remove trusted device: %w", err))
	}
	if tok := authsome.TrustedDeviceToken(ctx.Request()); tok != "" && hashTrustedDeviceToken(tok) == target.TokenHash {
		authsome.ClearTrustedDeviceCookie(ctx.Response(), ctx.Request())
	}
	p.audit(ctx.Context(), hook.ActionMFATrustedDeviceRemove, "mfa", deviceID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFATrustedDeviceRemove, "mfa", deviceID.String(), userID.String(), "")
	return nil, ctx.JSON(http.StatusOK, &TrustedDevicesRemovedResponse{Status: "removed", Removed: 1})
}

func (p *Plugin) handleRemoveAllTrustedDevices(ctx forge.Context, _ *TrustedDevicesRequest) (*TrustedDevicesRemovedResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return nil, forge.Unauthorized("authentication required")
	}
	devices, err := p.liveTrustedDevices(ctx.Context(), userID)
	if err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to list trusted devices: %w", err))
	}
	if err := p.store.DeleteTrustedDevices(ctx.Context(), userID); err != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to remove trusted devices: %w", err))
	}
	authsome.ClearTrustedDeviceCookie(ctx.Response(), ctx.Request())
	p.audit(ctx.Context(), hook.ActionMFATrustedDeviceRemove, "mfa", "", userID.String(), "", bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFATrustedDeviceRemove, "mfa", "", userID.String(), "")
	return nil, ctx.JSON(http.StatusOK, &TrustedDevicesRemovedResponse{Status: "removed", Removed: len(devices)})
}

// ──────────────────────────────────────────────────
// Issuing and checking
// ──────────────────────────────────────────────────

// trustChallengedDevice trusts the device that just passed the sign-in
// challenge and puts its token on the response. The session is already
// issued, so a failure here is logged and the sign-in still succeeds; the
// device simply is not trusted.
func (p *Plugin) trustChallengedDevice(ctx forge.Context, eng *authsome.Engine, appID id.AppID, userID id.UserID, resp *ChallengeResponse) {
	if !p.trustedDevicesEnabled(ctx.Context(), appID) {
		return
	}
	r := ctx.Request()
	token, d, err := p.trustDevice(ctx.Context(), userID, middleware.ClientIP(r), r.UserAgent())
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("mfa: trust device after challenge",
				log.String("user_id", userID.String()),
				log.String("error", err.Error()),
			)
		}
		return
	}
	resp.TrustedDeviceToken = token
	resp.TrustedDeviceExpiresAt = d.ExpiresAt.UTC().Format(time.RFC3339)
	authsome.SetTrustedDeviceCookie(ctx.Response(), r, eng.Settings(), appID.String(), token, d.ExpiresAt)

	p.audit(ctx.Context(), hook.ActionMFATrustedDeviceAdd, "mfa", d.ID.String(), userID.String(), appID.String(), bridge.OutcomeSuccess)
	p.emitHook(ctx.Context(), hook.ActionMFATrustedDeviceAdd, "mfa", d.ID.String(), userID.String(), appID.String())
}

// trustDevice mints a token for a device and stores its hash. The plaintext
// is returned once, here, and never stored.
func (p *Plugin) trustDevice(ctx context.Context, userID id.UserID, ip, userAgent string) (string, *TrustedDevice, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	browser, osName, _ := device.ParseUserAgent(userAgent)
	now := time.Now().UTC()
	d := &TrustedDevice{
		ID:         id.NewTrustedDeviceID(),
		UserID:     userID,
		TokenHash:  hashTrustedDeviceToken(token),
		Label:      browser + " on " + osName,
		UserAgent:  userAgent,
		IPAddress:  ip,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(p.config.TrustedDeviceTTL),
	}
	if err := p.store.CreateTrustedDevice(ctx, d); err != nil {
		return "", nil, fmt.Errorf("store trusted device: %w", err)
	}
	// Expired rows have nothing left to do; clear them while we are here.
	_, _ = p.liveTrustedDevices(ctx, userID) //nolint:errcheck // best-effort prune
	return token, d, nil
}

// VerifyTrustedDevice reports whether token belongs to a live trusted device
// of userID, and when that device passed the second factor that earned the
// trust. The engine's sign-in gate calls it; a lookup error is returned so
// the gate can refuse to skip the factor.
func (p *Plugin) VerifyTrustedDevice(ctx context.Context, appID id.AppID, userID id.UserID, token string) (time.Time, bool, error) {
	if p.store == nil || token == "" || !p.trustedDevicesEnabled(ctx, appID) {
		return time.Time{}, false, nil
	}
	d, err := p.store.GetTrustedDeviceByTokenHash(ctx, hashTrustedDeviceToken(token))
	if errors.Is(err, ErrTrustedDeviceNotFound) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("mfa: get trusted device: %w", err)
	}
	now := time.Now()
	if d.UserID != userID || !now.Before(d.ExpiresAt) {
		return time.Time{}, false, nil
	}
	if err := p.store.TouchTrustedDevice(ctx, d.ID, now.UTC()); err != nil && p.logger != nil {
		p.logger.Warn("mfa: record trusted device use",
			log.String("device_id", d.ID.String()),
			log.String("error", err.Error()),
		)
	}
	return d.CreatedAt, true, nil
}

// liveTrustedDevices returns the user's unexpired trusted devices and drops
// the expired ones on the way.
func (p *Plugin) liveTrustedDevices(ctx context.Context, userID id.UserID) ([]*TrustedDevice, error) {
	all, err := p.store.ListTrustedDevices(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	live := make([]*TrustedDevice, 0, len(all))
	for _, d := range all {
		if now.Before(d.ExpiresAt) {
			live = append(live, d)
			continue
		}
		_ = p.store.DeleteTrustedDevice(ctx, userID, d.ID) //nolint:errcheck // best-effort prune
	}
	return live, nil
}

// trustedDevicesEnabled reports whether devices may be trusted for the app:
// on in config and not switched off by the app's setting.
func (p *Plugin) trustedDevicesEnabled(ctx context.Context, appID id.AppID) bool {
	if p.config.DisableTrustedDevices || p.config.TrustedDeviceTTL <= 0 {
		return false
	}
	if p.engine == nil {
		return true
	}
	mgr := p.engine.Settings()
	if mgr == nil {
		return true
	}
	opts := settings.ResolveOpts{}
	if !appID.IsNil() {
		opts.AppID = appID.String()
	}
	enabled, _ := settings.Get(ctx, mgr, SettingTrustedDevicesEnabled, opts) //nolint:errcheck // falls back to the default
	return enabled
}

// subscribePasswordChanges forgets a user's trusted devices when their
// password changes or is reset. A password change is usually a response to a
// suspected compromise, and a device trusted by whoever had the password
// should not outlive it.
func (p *Plugin) subscribePasswordChanges() {
	if p.hooks == nil {
		return
	}
	p.hooks.On("mfa-trusted-devices", func(ctx context.Context, event *hook.Event) error {
		if event == nil || event.Action != hook.ActionPasswordChange || event.Resource != hook.ResourceUser {
			return nil
		}
		userID, err := id.ParseUserID(event.ResourceID)
		if err != nil || p.store == nil {
			return nil //nolint:nilerr // not a user this plugin can act on
		}
		if err := p.store.DeleteTrustedDevices(ctx, userID); err != nil {
			return fmt.Errorf("mfa: forget trusted devices after password change: %w", err)
		}
		return nil
	})
}

// hashTrustedDeviceToken is the stored form of a token. The token carries 256
// random bits, so a plain SHA-256 is enough: there is nothing to brute-force.
func hashTrustedDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CountTrustedDevices is how many live trusted devices the user has, for a
// host's own account page.
func (p *Plugin) CountTrustedDevices(ctx context.Context, userID id.UserID) (int, error) {
	devices, err := p.liveTrustedDevices(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(devices), nil
}
