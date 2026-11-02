package authsome

import (
	"context"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// EmailVerificationRequired reports whether an account in appID must verify
// its email before it gets a session. Sign-up, sign-in, IssueSession and the
// client config all ask this, so the UI and the server never disagree.
//
// Resolution order (first match wins):
//  1. Dynamic setting auth.require_email_verification, when it says true
//  2. Per-app client config override (RequireEmailVerification)
//  3. Environment setting (SkipEmailVerification, inverted, with type defaults merged)
//  4. Default: no enforcement
func (e *Engine) EmailVerificationRequired(ctx context.Context, appID id.AppID) bool {
	if mgr := e.Settings(); mgr != nil {
		opts := settings.ResolveOpts{}
		if appID.Prefix() != "" {
			opts.AppID = appID.String()
		}
		if on, err := settings.Get(ctx, mgr, SettingRequireEmailVerification, opts); err == nil && on {
			return true
		}
	}
	if cfg, err := e.store.GetAppClientConfig(ctx, appID); err == nil && cfg.RequireEmailVerification != nil {
		return *cfg.RequireEmailVerification
	}
	if env, _ := e.GetDefaultEnvironment(ctx, appID); env != nil { //nolint:errcheck // best-effort env lookup
		effective := environment.MergeSettings(environment.DefaultSettingsForType(env.Type), env.Settings)
		return effective != nil && !effective.SkipEmailVerificationEnabled()
	}
	return false
}
