package authsome

import (
	"encoding/json"
	"fmt"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"
)

// Category: DPoP

var (
	// SettingDPoPMode controls whether tokens are bound to a client key.
	//
	// Defaults to off so existing deployments are unaffected. A per-client
	// value on OAuth2Client can raise this but never lower it.
	SettingDPoPMode = settings.Define("dpop.mode", string(dpop.ModeOff),
		settings.WithDisplayName("DPoP Mode"),
		settings.WithDescription("Bind issued tokens to a client-held key (RFC 9449)"),
		settings.WithCategory("DPoP"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Value: string(dpop.ModeOff), Label: "Off"},
			formconfig.SelectOption{Value: string(dpop.ModeOptional), Label: "Optional"},
			formconfig.SelectOption{Value: string(dpop.ModeRequired), Label: "Required"},
		),
		settings.WithHelpText("Off ignores proofs. Optional binds when a client proves and issues a bearer token otherwise. Required refuses to issue unbound tokens and will break clients that do not send proofs."),
		settings.WithOrder(200),
		settings.WithValidation(validateDPoPMode),
	)

	// SettingDPoPNonceRequired demands a server-issued nonce in every proof.
	//
	// Off by default because it costs every client one extra round trip: the
	// first request is answered with a challenge and has to be retried.
	SettingDPoPNonceRequired = settings.Define("dpop.nonce_required", false,
		settings.WithDisplayName("Require DPoP Nonce"),
		settings.WithDescription("Demand a server-issued nonce in every DPoP proof"),
		settings.WithCategory("DPoP"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Adds replay resistance at the cost of one extra round trip per client when its nonce expires."),
		settings.WithOrder(210),
	)

	// SettingDPoPIatLeewayPast is how far in the past a proof iat may sit.
	SettingDPoPIatLeewayPast = settings.Define("dpop.iat_leeway_past_seconds", 60,
		settings.WithDisplayName("DPoP Proof Age Tolerance (seconds)"),
		settings.WithDescription("How old a DPoP proof may be"),
		settings.WithCategory("DPoP"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: new(1), Max: new(300)}),
		settings.WithHelpText("Default: 60. Most proofs arriving late are clients with drifting clocks."),
		settings.WithOrder(220),
		settings.WithValidation(validateDPoPLeeway(1, 300)),
	)

	// SettingDPoPIatLeewayFuture is how far ahead a proof iat may sit.
	//
	// Tighter than the past window on purpose: a future-dated proof means
	// somebody minted it ahead of time, which has no innocent explanation.
	SettingDPoPIatLeewayFuture = settings.Define("dpop.iat_leeway_future_seconds", 30,
		settings.WithDisplayName("DPoP Clock Skew Tolerance (seconds)"),
		settings.WithDescription("How far ahead of server time a DPoP proof may be dated"),
		settings.WithCategory("DPoP"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: new(0), Max: new(120)}),
		settings.WithHelpText("Default: 30. Keep this small; a future-dated proof is pre-minting."),
		settings.WithOrder(230),
		settings.WithValidation(validateDPoPLeeway(0, 120)),
	)
)

// registerDPoPSettings registers everything under the "dpop" namespace.
func registerDPoPSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "dpop", SettingDPoPMode); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "dpop", SettingDPoPNonceRequired); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "dpop", SettingDPoPIatLeewayPast); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "dpop", SettingDPoPIatLeewayFuture)
}

func validateDPoPMode(val json.RawMessage) error {
	var v string
	if err := json.Unmarshal(val, &v); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	switch v {
	case string(dpop.ModeOff), string(dpop.ModeOptional), string(dpop.ModeRequired):
		return nil
	default:
		return fmt.Errorf("mode must be one of off, optional, required")
	}
}

func validateDPoPLeeway(minSec, maxSec int) func(json.RawMessage) error {
	return func(val json.RawMessage) error {
		var v int
		if err := json.Unmarshal(val, &v); err != nil {
			return fmt.Errorf("invalid value: %w", err)
		}
		if v < minSec || v > maxSec {
			return fmt.Errorf("value must be between %d and %d seconds", minSec, maxSec)
		}
		return nil
	}
}
