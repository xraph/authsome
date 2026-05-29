// Package contract wires authsome into the Forge dashboard's contract path.
// It registers the `auth` contributor with the dashboard's contract registry,
// declares the auth.login + auth.logout command intents, and ships the
// /login graph route the React shell renders inside its AuthGate.
//
// Authsome continues to expose its templ-based pages and AuthChecker via
// RegisterDashboardAuth; this package is the parallel contract surface so
// the slice (l) React shell can sign in without falling back to its
// built-in LoginScreen. The two paths share the engine: both call
// engine.SignIn and the same dashboard auth_token cookie scheme.
package contract

import (
	"bytes"
	_ "embed"
	"fmt"

	authsome "github.com/xraph/authsome"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// Deps bundles what the contract handlers need at registration time.
// Engine is required; everything else is optional UI/operational tuning.
type Deps struct {
	// Engine is the live authsome engine. Required.
	Engine *authsome.Engine

	// CookieSecure overrides the auto-detected request scheme when an
	// upstream proxy strips it (rare — leave zero in production behind a
	// TLS-aware reverse proxy).
	CookieSecure *bool

	// SocialBasePath overrides the URL prefix for social OAuth start
	// endpoints. Defaults to "/v1/social" matching the social plugin's
	// default route registration.
	SocialBasePath string

	// Brand / BrandLogoURL override the platform app's name + logo on the
	// /login page. When empty the values fall back to App.Name / App.Logo
	// resolved at handler time.
	Brand        string
	BrandLogoURL string

	// SignupURL is the public-facing /signup link rendered above the login
	// form. Empty hides the "Don't have an account? Sign up" line.
	SignupURL   string
	SignupLabel string

	// TermsURL / PrivacyURL feed the legal footer beneath the form.
	// Both empty hides the footer entirely.
	TermsURL   string
	PrivacyURL string

	// RequiredRoles, if non-empty, gates dashboard access to users carrying
	// at least one matching role. Authsome calls Extension.SetRequiredRoles
	// with these values in RegisterDashboardAuth so the principal endpoint
	// returns 403 PERMISSION_DENIED to anyone without them.
	RequiredRoles []string
}

// Register loads the embedded manifest, validates it, registers the `auth`
// contributor with reg, and binds the contract handlers against d. The
// dashboard's auto-discovery calls this via Extension.RegisterContractContributor.
func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("authsome/contract: Engine is required")
	}

	m, err := loader.Load(bytes.NewReader(manifestYAML), "authsome/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("authsome/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("authsome/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("authsome/contract: register manifest: %w", err)
	}

	const c = "auth"
	if err := dispatcher.RegisterCommand(d, c, "auth.login", 1, loginHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.login: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.logout", 1, logoutHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.logout: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "auth.config", 1, configHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.config: %w", err)
	}
	// Phase C.1 — Users dashboard. All seven intents route through
	// handlers_users.go to the engine's Admin* methods. Each registration
	// fails loudly so missing or shadowed handlers don't slip past.
	if err := dispatcher.RegisterQuery(d, c, "users.list", 1, usersListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "users.detail", 1, usersDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "users.create", 1, usersCreateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "users.update", 1, usersUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "users.ban", 1, usersBanHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.ban: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "users.unban", 1, usersUnbanHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.unban: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "users.delete", 1, usersDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register users.delete: %w", err)
	}

	// Phase C.2 — Apps
	if err := dispatcher.RegisterQuery(d, c, "apps.list", 1, appsListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "apps.detail", 1, appsDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apps.create", 1, appsCreateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apps.update", 1, appsUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apps.delete", 1, appsDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.delete: %w", err)
	}

	// Phase C.3 — Sessions
	if err := dispatcher.RegisterQuery(d, c, "sessions.list", 1, sessionsListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register sessions.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "sessions.detail", 1, sessionsDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register sessions.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "sessions.revoke", 1, sessionsRevokeHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register sessions.revoke: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "sessions.bulkRevoke", 1, sessionsBulkRevokeHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register sessions.bulkRevoke: %w", err)
	}

	// Phase C.4 — Roles & RBAC
	if err := dispatcher.RegisterQuery(d, c, "roles.list", 1, rolesListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "roles.detail", 1, rolesDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "roles.create", 1, rolesCreateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "roles.update", 1, rolesUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "roles.delete", 1, rolesDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.delete: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "roles.assign", 1, rolesAssignHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.assign: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "roles.unassign", 1, rolesUnassignHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register roles.unassign: %w", err)
	}

	// Phase C.5 — API Keys: moved to the apikey plugin (plugins/apikey/contract).
	// The /apikeys page graph node stays declared here because it's a
	// platform-level navigation entry; the intent handlers register
	// against the plugin's contributor name. Cross-contributor lookups
	// are name-only on the wire so the auth manifest's binding to
	// `apikeys.list` resolves through the plugin's dispatch
	// registration.

	// Phase C.6 — Devices
	if err := dispatcher.RegisterQuery(d, c, "devices.list", 1, devicesListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register devices.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "devices.detail", 1, devicesDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register devices.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "devices.trust", 1, devicesTrustHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register devices.trust: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "devices.delete", 1, devicesDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register devices.delete: %w", err)
	}

	// Phase C.7 — Environments
	if err := dispatcher.RegisterQuery(d, c, "environments.list", 1, envsListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "environments.detail", 1, envsDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.create", 1, envsCreateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.update", 1, envsUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.delete", 1, envsDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.delete: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.clone", 1, envsCloneHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.clone: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.setDefault", 1, envsSetDefaultHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.setDefault: %w", err)
	}

	// Phase C.8 — Webhooks
	if err := dispatcher.RegisterQuery(d, c, "webhooks.list", 1, webhooksListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register webhooks.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "webhooks.detail", 1, webhooksDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register webhooks.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "webhooks.create", 1, webhooksCreateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register webhooks.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "webhooks.update", 1, webhooksUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register webhooks.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "webhooks.delete", 1, webhooksDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register webhooks.delete: %w", err)
	}

	// Phase C.9 — Form Configs
	if err := dispatcher.RegisterQuery(d, c, "formConfigs.list", 1, formConfigsListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register formConfigs.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "formConfigs.signup", 1, formConfigsSignupDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register formConfigs.signup: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "formConfigs.saveSignup", 1, formConfigsSignupSaveHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register formConfigs.saveSignup: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "formConfigs.deleteSignup", 1, formConfigsSignupDeleteHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register formConfigs.deleteSignup: %w", err)
	}

	// Phase C.10 + D.2.5 — Settings.
	//
	// D.2.5 added the namespaces / namespace queries (drive settings.tabs
	// + settings.panel) plus the enforce / unenforce commands. The legacy
	// list / detail queries stay registered for one release for any callers
	// still bound to them; remove in the next minor.
	if err := dispatcher.RegisterQuery(d, c, "settings.namespaces", 1, settingsNamespacesHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.namespaces: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "settings.namespace", 1, settingsNamespaceHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.namespace: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "settings.list", 1, settingsListHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "settings.detail", 1, settingsDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "settings.update", 1, settingsUpdateHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "settings.enforce", 1, settingsEnforceHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.enforce: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "settings.unenforce", 1, settingsUnenforceHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register settings.unenforce: %w", err)
	}

	// Phase C.11 — Organizations: moved to the organization plugin
	// (plugins/organization/contract). The /organizations page graph
	// node stays declared in the auth manifest; intent handlers
	// register against the plugin's contributor name.

	// Phase C.12 — Overview dashboard. Two queries powering the
	// dashboard.grid widgets and the recent-signups list on `/`.
	if err := dispatcher.RegisterQuery(d, c, "overview.stats", 1, overviewStatsHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register overview.stats: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "overview.recentSignups", 1, overviewRecentSignupsHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register overview.recentSignups: %w", err)
	}

	// Phase C.13 — Credentials page. Single query surfacing the platform
	// app's publishable key and environment metadata for SDK setup.
	if err := dispatcher.RegisterQuery(d, c, "credentials.detail", 1, credentialsDetailHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register credentials.detail: %w", err)
	}

	// Phase C.14 — Anonymous auth pages. Three commands dispatched by
	// the dashboard's AuthGate before a session exists: signup,
	// forgot-password (request reset link), and reset-password (consume
	// reset token + set new password). All three are publicly callable
	// — the engine's password policy + rate limits gate misuse.
	if err := dispatcher.RegisterCommand(d, c, "auth.signup", 1, signupHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.signup: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.forgotPassword", 1, forgotPasswordHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.forgotPassword: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.resetPassword", 1, resetPasswordHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.resetPassword: %w", err)
	}
	// First-run bootstrap + dynamic signup (form-config-driven).
	if err := dispatcher.RegisterQuery(d, c, "auth.setupStatus", 1, setupStatusHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.setupStatus: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.setup", 1, setupHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.setup: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "auth.dynamicConfig", 1, dynamicConfigHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.dynamicConfig: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.dynamicRegister", 1, dynamicRegisterHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.dynamicRegister: %w", err)
	}
	// Per-app feature toggles backed by appclientconfig (Sign-in
	// Methods panel from the templui dashboard).
	if err := dispatcher.RegisterQuery(d, c, "auth.featureToggles", 1, featureTogglesHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.featureToggles: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "auth.toggleFeature", 1, toggleFeatureHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register auth.toggleFeature: %w", err)
	}

	// Phase C.17 — App + environment switcher. Three intents back the
	// dashboard chrome's switcher: apps.context (read the current
	// app+env + the available lists), apps.switch / environments.switch
	// (set the active selection via cookies the auth resolver projects
	// onto principal claims).
	if err := dispatcher.RegisterQuery(d, c, "apps.context", 1, appsContextHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.context: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "apps.switch", 1, appsSwitchHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register apps.switch: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "environments.switch", 1, environmentsSwitchHandler(deps)); err != nil {
		return fmt.Errorf("authsome/contract: register environments.switch: %w", err)
	}

	// Phase C.14 — Subscriptions: moved to the subscription plugin
	// (plugins/subscription/contract). The /plans pages stay declared
	// on the auth manifest; intent handlers register against the
	// plugin's contributor name.

	return nil
}
