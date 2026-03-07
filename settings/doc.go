// Package settings provides a dynamic, plugin-extensible configuration system
// for AuthSome. It supports typed setting definitions, multi-scope overrides
// (global, app, org, user), enforcement (locking settings at higher scopes),
// and change notifications.
//
// Settings are declared by plugins via [SettingsProvider] and resolved through
// a cascade: code default → global DB → app DB → org DB → user DB.
// Enforcement flags prevent lower scopes from overriding locked values.
//
// Use [Define] to create typed setting definitions and [Get] to resolve them:
//
//	var MinLength = settings.Define("password.min_length", 8,
//	    settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
//	    settings.WithEnforceable(),
//	)
//
//	val, err := settings.Get(ctx, mgr, MinLength, settings.ResolveOpts{AppID: appID})
package settings
