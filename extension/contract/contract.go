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
	return nil
}
