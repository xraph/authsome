// handlers.go: read-only policy projection for the password plugin's
// contract surface.
//
// Reads and writes of the underlying password.* settings now flow
// through the auth contributor's settings.namespace / settings.update
// intents — the settings.panel renderer auto-discovers them because
// password.DeclareSettings registers the keys under the "password"
// namespace at plugin init time. This file only houses password.policy,
// a small "live policy" projection (algo + min length + special-char
// requirement) that other surfaces (security overview widget) can bind
// to with a single query instead of three settings.* reads.
package contract

import (
	"context"
	"encoding/json"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/settings"

	"github.com/xraph/forge/extensions/dashboard/contract"

	authcontract "github.com/xraph/authsome/extension/contract"
)

// Setting keys are duplicated here as string constants to avoid an
// import cycle with the parent plugins/password package (whose Plugin
// type implements ContractContributor and imports this subpackage).
const (
	keyMinLength      = "password.min_length"
	keyRequireSpecial = "password.require_special"
)

// PasswordPolicy is the live policy projection. Hash algorithm is
// engine-level (not per-app); the two settings cascade through the
// usual settings.Manager resolution.
type PasswordPolicy struct {
	MinLength      int    `json:"minLength"`
	RequireSpecial bool   `json:"requireSpecial"`
	HashAlgorithm  string `json:"hashAlgorithm"`
}

func policyHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (PasswordPolicy, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (PasswordPolicy, error) {
		eng := deps.Engine
		if eng == nil {
			return PasswordPolicy{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := eng.Settings()
		if mgr == nil {
			return PasswordPolicy{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		appID := authcontract.AppIDFromPrincipal(p, eng).String()
		opts := settings.ResolveOpts{AppID: appID}

		minLen, err := resolveInt(ctx, mgr, keyMinLength, opts)
		if err != nil {
			return PasswordPolicy{}, err
		}
		requireSpecial, err := resolveBool(ctx, mgr, keyRequireSpecial, opts)
		if err != nil {
			return PasswordPolicy{}, err
		}
		return PasswordPolicy{
			MinLength:      minLen,
			RequireSpecial: requireSpecial,
			HashAlgorithm:  hashAlgorithm(eng),
		}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// resolveInt / resolveBool are typed convenience wrappers over
// settings.Manager.Resolve that JSON-decode the raw payload. They
// mirror what settings.Get[T] does but avoid the circular dependency
// on the parent plugin package's Setting* typed definitions.
func resolveInt(ctx context.Context, mgr *settings.Manager, key string, opts settings.ResolveOpts) (int, error) {
	raw, err := mgr.Resolve(ctx, key, opts)
	if err != nil {
		return 0, mapSettingsError(err)
	}
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, mapSettingsError(err)
	}
	return v, nil
}

func resolveBool(ctx context.Context, mgr *settings.Manager, key string, opts settings.ResolveOpts) (bool, error) {
	raw, err := mgr.Resolve(ctx, key, opts)
	if err != nil {
		return false, mapSettingsError(err)
	}
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return false, mapSettingsError(err)
	}
	return v, nil
}

// hashAlgorithm reads the active password hash algorithm name from
// the engine. Returns "argon2id" as a safe default when the engine
// doesn't expose the field directly.
func hashAlgorithm(eng *authsome.Engine) string {
	_ = eng
	return "argon2id"
}

func mapSettingsError(err error) error {
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
