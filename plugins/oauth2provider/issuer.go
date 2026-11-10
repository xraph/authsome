package oauth2provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// settingsNamespace is the namespace the plugin registers its dynamic
// settings under.
const settingsNamespace = "oauth2provider"

// SettingAppIssuer lets an app have its own OpenID Connect issuer.
var SettingAppIssuer = settings.Define("oauth2.app_issuer", false,
	settings.WithScopes(settings.ScopeApp),
	settings.WithDisplayName("Per-app OIDC issuer"),
	settings.WithDescription("Give this app its own issuer at {issuer}/apps/{slug}. Turning this on or off, or renaming the app's slug, changes the issuer for every OAuth2 client of this app, and they will reject tokens until they are reconfigured."),
	settings.WithCategory("OpenID Connect"),
)

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, settingsNamespace, SettingAppIssuer); err != nil {
		return err
	}
	return settings.RegisterTyped(m, settingsNamespace, SettingAppLoginURL)
}

// errUnknownIssuer is the one answer for a slug that names no app and an app
// that has not opted in. The per-app routes are unauthenticated, so the two
// must not be told apart.
func errUnknownIssuer() error { return forge.NotFound("no such issuer") }

// AppSlugRequest addresses one app's issuer by its slug.
type AppSlugRequest struct {
	Slug string `path:"slug"`
}

// appIssuerEnabled reports whether appID has opted into its own issuer. An
// unset value is false, and so is a plugin with no settings manager to ask.
// A read that fails is an error: the caller must not guess an issuer.
func (p *Plugin) appIssuerEnabled(ctx context.Context, appID id.AppID) (bool, error) {
	if p.engine == nil {
		return false, nil
	}
	mgr := p.engine.Settings()
	if mgr == nil {
		return false, nil
	}
	on, err := settings.Get(ctx, mgr, SettingAppIssuer, settings.ResolveOpts{AppID: appID.String()})
	if err != nil {
		// The plugin was not registered with the engine, so nothing can have
		// turned the setting on. On a real engine that is a wiring mistake
		// worth one warning: per-app issuers silently don't apply.
		if errors.Is(err, settings.ErrUnknownKey) {
			if _, isEngine := p.engine.(*authsome.Engine); isEngine {
				p.unregisteredSettingOnce.Do(func() {
					if p.logger != nil {
						p.logger.Warn("oauth2: the "+SettingAppIssuer.Def.Key+" setting is not declared, "+
							"so per-app issuers are ignored and every app uses the deployment issuer; "+
							"register the oauth2provider plugin with the engine to declare it",
							log.String("app_id", appID.String()))
					}
				})
			}
			return false, nil
		}
		return false, fmt.Errorf("oauth2: read %s: %w", SettingAppIssuer.Def.Key, err)
	}
	return on, nil
}

// appIssuer is the issuer an opted-in app is known by.
func (p *Plugin) appIssuer(slug string) string {
	return p.issuerURL() + "/apps/" + slug
}

// issuerFor returns the issuer an id_token for appID names and the signing
// key scope that goes with it: the deployment issuer and scope "" unless the
// app has opted into its own. Every mint goes through here, so the issuer is
// chosen at token time. An error is returned rather than answered with the
// deployment issuer, since a token under the wrong issuer breaks every
// client of the app.
func (p *Plugin) issuerFor(ctx context.Context, appID id.AppID) (issuer, scope string, err error) {
	on, err := p.appIssuerEnabled(ctx, appID)
	if err != nil {
		return "", "", err
	}
	if !on {
		return p.issuerURL(), "", nil
	}
	a, err := p.store.GetApp(ctx, appID)
	if err != nil {
		return "", "", fmt.Errorf("oauth2: load app %s for its issuer: %w", appID, err)
	}
	return p.appIssuer(a.Slug), appID.String(), nil
}

// appIssuerBySlug resolves a slug from an unauthenticated request. ok is
// false, with a nil error, for a slug that names no app and for an app that
// has not opted in; err is a lookup that failed. It touches no signing key.
func (p *Plugin) appIssuerBySlug(ctx context.Context, slug string) (issuer, scope string, ok bool, err error) {
	a, err := p.store.GetAppBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("oauth2: look up app %q: %w", slug, err)
	}
	on, err := p.appIssuerEnabled(ctx, a.ID)
	if err != nil || !on {
		return "", "", false, err
	}
	return p.appIssuer(a.Slug), a.ID.String(), true, nil
}

func (p *Plugin) handleAppAuthServerMetadata(ctx forge.Context, req *AppSlugRequest) (*AuthServerMetadata, error) {
	issuer, scope, ok, err := p.appIssuerBySlug(ctx.Context(), req.Slug)
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	if !ok {
		return nil, errUnknownIssuer()
	}
	return p.buildAuthServerMetadata(ctx.Context(), issuer, scope), nil
}

func (p *Plugin) handleAppDiscovery(ctx forge.Context, req *AppSlugRequest) (*DiscoveryResponse, error) {
	m, err := p.handleAppAuthServerMetadata(ctx, req)
	if err != nil {
		return nil, err
	}
	return discoveryFrom(m), nil
}

func (p *Plugin) handleAppJWKS(ctx forge.Context, req *AppSlugRequest) (*JWKSResponse, error) {
	_, scope, ok, err := p.appIssuerBySlug(ctx.Context(), req.Slug)
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	if !ok {
		return nil, errUnknownIssuer()
	}
	doc, degraded := p.jwksFor(ctx.Context(), scope)
	writeJWKSCacheControl(ctx, degraded)
	return doc, nil
}

// registerAppJWKSRoute mounts the per-app key set. It lives on the grouped
// router, not the root: jwks_uri is the app's issuer plus /v1/oauth/jwks, and
// that issuer carries the mount prefix.
func (p *Plugin) registerAppJWKSRoute(router forge.Router) error {
	if err := p.registerPreflight(router, "/apps/:slug/v1/oauth/jwks"); err != nil {
		return err
	}
	return router.GET("/apps/:slug/v1/oauth/jwks", p.handleAppJWKS, p.withCORS(
		forge.WithSummary("App JSON Web Key Set"),
		forge.WithDescription("Public keys that verify id_tokens issued under an app's own issuer. 404 unless the app has opted in."),
		forge.WithOperationID("oauth2AppJWKS"),
		forge.WithResponseSchema(http.StatusOK, "JWKS", JWKSResponse{}),
		forge.WithTags("OAuth2"),
		forge.WithErrorResponses(),
	)...)
}

// registerAppWellKnown mounts the per-app discovery documents.
func (p *Plugin) registerAppWellKnown(router forge.Router) error {
	if err := router.GET("/apps/:slug/.well-known/openid-configuration", p.handleAppDiscovery, p.withCORS(
		forge.WithSummary("App OpenID Connect Discovery"),
		forge.WithDescription("Discovery document for an app that has its own issuer. 404 unless the app has opted in."),
		forge.WithOperationID("oauth2AppDiscovery"),
		forge.WithTags("OAuth2"),
	)...); err != nil {
		return err
	}
	for _, path := range []string{"/apps/:slug/.well-known/openid-configuration", "/apps/:slug/.well-known/oauth-authorization-server"} {
		if err := p.registerPreflight(router, path); err != nil {
			return err
		}
	}
	return router.GET("/apps/:slug/.well-known/oauth-authorization-server", p.handleAppAuthServerMetadata, p.withCORS(
		forge.WithSummary("App OAuth2 Authorization Server Metadata"),
		forge.WithDescription("RFC 8414 metadata for an app that has its own issuer. 404 unless the app has opted in."),
		forge.WithOperationID("oauth2AppAuthServerMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", AuthServerMetadata{}),
		forge.WithTags("OAuth2"),
	)...)
}
