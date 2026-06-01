// handlers_features.go: app-level feature toggles backed by
// appclientconfig.Config.
//
// The templui dashboard surfaced a "Sign-in Methods" panel with
// toggles for password / social / passkey / magiclink / mfa / sso /
// signup / waitlist / require-email-verification. This is the
// contract equivalent: a single query + command pair that lets the
// dashboard render switches for every feature the engine supports
// without having to special-case each one.
//
// Toggles are app-scoped — they write the per-app
// appclientconfig.Config overrides, falling back to "enabled if the
// underlying plugin is installed" (password is always available; the
// rest depend on a plugin being registered).
package contract

import (
	"context"
	"errors"
	"strings"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/appclientconfig"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// FeatureToggle is the wire shape for a single row in the
// auth.featureToggles response.
type FeatureToggle struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	// Available reports whether the underlying plugin is registered
	// (password is always available regardless). The renderer should
	// disable / dim the row when Available is false so admins
	// understand why the toggle won't take effect.
	Available bool `json:"available"`
}

type FeatureTogglesResponse struct {
	Toggles []FeatureToggle `json:"toggles"`
}

// ToggleFeatureInput is the wire shape for auth.toggleFeature. Each
// command flips a single key — the renderer dispatches one command per
// changed toggle rather than sending the whole bag.
type ToggleFeatureInput struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

// featureSpec is the catalog of toggleable features. Mirrors the
// templui dashboard's `methods` list so admins see the same affordances.
type featureSpec struct {
	Key    string
	Label  string
	Desc   string
	Plugin string // plugin name that must be loaded; "" means always-available
}

var featureCatalog = []featureSpec{
	{Key: "signup_enabled", Label: "Allow new sign-ups", Desc: "Lets unauthenticated visitors create accounts."},
	// Password is part of authsome core, not a plugin — always
	// available regardless of plugin registry contents.
	{Key: "password_enabled", Label: "Password", Desc: "Email + password sign-in."},
	{Key: "social_enabled", Label: "Social providers", Desc: "Google, GitHub, etc.", Plugin: "social"},
	{Key: "passkey_enabled", Label: "Passkeys", Desc: "Passwordless biometric sign-in (WebAuthn).", Plugin: "passkey"},
	{Key: "magic_link_enabled", Label: "Magic Link", Desc: "Passwordless email-link sign-in.", Plugin: "magiclink"},
	{Key: "mfa_enabled", Label: "Multi-Factor Authentication", Desc: "Require a second factor after password.", Plugin: "mfa"},
	{Key: "sso_enabled", Label: "SSO / SAML", Desc: "Enterprise single sign-on.", Plugin: "sso"},
	{Key: "require_email_verification", Label: "Require email verification", Desc: "Block sign-in until the email is verified."},
	{Key: "waitlist_enabled", Label: "Waitlist", Desc: "Require admin approval for new sign-ups.", Plugin: "waitlist"},
}

func featureTogglesHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (FeatureTogglesResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (FeatureTogglesResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return FeatureTogglesResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		// Plugin-installed set drives the Available flag for each row.
		installed := installedPluginSet(eng)
		appID := AppIDFromPrincipal(p, eng)

		var cfg *appclientconfig.Config
		if store := eng.Store(); store != nil {
			c, err := store.GetAppClientConfig(ctx, appID)
			if err != nil && !errors.Is(err, appclientconfig.ErrNotFound) {
				return FeatureTogglesResponse{}, mapEngineError(err)
			}
			cfg = c
		}

		out := FeatureTogglesResponse{Toggles: make([]FeatureToggle, 0, len(featureCatalog))}
		for _, f := range featureCatalog {
			available := f.Plugin == "" || installed[f.Plugin]
			enabled := resolveFeatureDefault(f, available)
			if cfg != nil {
				if override := readOverride(cfg, f.Key); override != nil {
					enabled = *override
				}
			}
			out.Toggles = append(out.Toggles, FeatureToggle{
				Key:         f.Key,
				Label:       f.Label,
				Description: f.Desc,
				Enabled:     enabled,
				Available:   available,
			})
		}
		return out, nil
	}
}

func toggleFeatureHandler(deps Deps) func(ctx context.Context, in ToggleFeatureInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in ToggleFeatureInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "key is required"}
		}
		if !isKnownFeature(key) {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "unknown feature key: " + key}
		}
		store := eng.Store()
		if store == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "client config store not configured"}
		}
		appID := AppIDFromPrincipal(p, eng)
		cfg, err := store.GetAppClientConfig(ctx, appID)
		if err != nil {
			if !errors.Is(err, appclientconfig.ErrNotFound) {
				return AckResponse{}, mapEngineError(err)
			}
			// First-touch: synthesise a fresh config for this app.
			cfg = &appclientconfig.Config{AppID: appID}
		}
		setOverride(cfg, key, in.Enabled)
		if err := store.SetAppClientConfig(ctx, cfg); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: key}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// installedPluginSet projects the engine's plugin registry into a
// name-keyed set. Cheap on every call — the registry is in-memory.
func installedPluginSet(eng *authsome.Engine) map[string]bool {
	out := map[string]bool{}
	if eng == nil {
		return out
	}
	reg := eng.Plugins()
	if reg == nil {
		return out
	}
	for _, p := range reg.Plugins() {
		out[p.Name()] = true
	}
	return out
}

func resolveFeatureDefault(_ featureSpec, available bool) bool {
	if !available {
		return false
	}
	// Defaults: all features default on when available. Password is
	// always available (core), so it always defaults on; the rest
	// default on once their backing plugin is installed.
	return true
}

func readOverride(cfg *appclientconfig.Config, key string) *bool {
	switch key {
	case "signup_enabled":
		return cfg.SignupEnabled
	case "password_enabled":
		return cfg.PasswordEnabled
	case "social_enabled":
		return cfg.SocialEnabled
	case "passkey_enabled":
		return cfg.PasskeyEnabled
	case "magic_link_enabled":
		return cfg.MagicLinkEnabled
	case "mfa_enabled":
		return cfg.MFAEnabled
	case "sso_enabled":
		return cfg.SSOEnabled
	case "require_email_verification":
		return cfg.RequireEmailVerification
	case "waitlist_enabled":
		return cfg.WaitlistEnabled
	default:
		return nil
	}
}

func setOverride(cfg *appclientconfig.Config, key string, enabled bool) {
	v := enabled
	switch key {
	case "signup_enabled":
		cfg.SignupEnabled = &v
	case "password_enabled":
		cfg.PasswordEnabled = &v
	case "social_enabled":
		cfg.SocialEnabled = &v
	case "passkey_enabled":
		cfg.PasskeyEnabled = &v
	case "magic_link_enabled":
		cfg.MagicLinkEnabled = &v
	case "mfa_enabled":
		cfg.MFAEnabled = &v
	case "sso_enabled":
		cfg.SSOEnabled = &v
	case "require_email_verification":
		cfg.RequireEmailVerification = &v
	case "waitlist_enabled":
		cfg.WaitlistEnabled = &v
	}
}

func isKnownFeature(key string) bool {
	for _, f := range featureCatalog {
		if f.Key == key {
			return true
		}
	}
	return false
}
