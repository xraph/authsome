package extension

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a-h/templ"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/dashboard/contributor"

	authdash "github.com/xraph/authsome/dashboard"
)

// contributorProtocolPrefix is appended to authsome's basePath to form the
// public contributor protocol routes. The full path is shared with the URL
// pattern that github.com/xraph/forge/extensions/dashboard/contributor.RemoteContributor
// expects, so a remote consumer's baseURL is just authsome's basePath
// (e.g. "http://identity:7902/authsome").
const contributorProtocolPrefix = "/_forge/dashboard"

// pageBaseQueryParam is an optional query parameter that consumers send to
// override the contributor's default PageBase, so links rendered by authsome
// resolve under the consumer's dashboard URL space rather than this service's.
const pageBaseQueryParam = "pb"

// basePathQueryParam is the consumer's dashboard base path (e.g. "/dashboard").
// Forwarded so authsome's HTMX redirects and absolute links land back on the
// consumer's dashboard, not on this service's local dashboard URL.
const basePathQueryParam = "bp"

// registerContributorProtocol exposes the local authsome dashboard contributor
// over HTTP so other Forge apps can consume it as a remote contributor. Routes:
//
//	GET <basePath>/_forge/dashboard/manifest
//	GET <basePath>/_forge/dashboard/pages/*filepath
//	GET <basePath>/_forge/dashboard/widgets/:id
//	GET <basePath>/_forge/dashboard/settings/:id
//
// Returns nil silently when the engine is not initialised (e.g. client mode).
func (e *Extension) registerContributorProtocol(r forge.Router) error {
	if e.engine == nil || r == nil {
		return nil
	}

	contrib := authdash.New(
		authdash.NewManifest(e.engine, e.plugins),
		e.engine,
		e.plugins,
	)

	if err := r.GET(contributorProtocolPrefix+"/manifest", manifestHandler(contrib),
		forge.WithSchemaExclude(),
	); err != nil {
		return fmt.Errorf("authsome: register contributor manifest route: %w", err)
	}

	pages := pageHandler(contrib)
	if err := r.GET(contributorProtocolPrefix+"/pages", pages,
		forge.WithSchemaExclude(),
	); err != nil {
		return fmt.Errorf("authsome: register contributor pages route: %w", err)
	}
	if err := r.GET(contributorProtocolPrefix+"/pages/*filepath", pages,
		forge.WithSchemaExclude(),
	); err != nil {
		return fmt.Errorf("authsome: register contributor pages wildcard route: %w", err)
	}

	if err := r.GET(contributorProtocolPrefix+"/widgets/:id", widgetHandler(contrib),
		forge.WithSchemaExclude(),
	); err != nil {
		return fmt.Errorf("authsome: register contributor widgets route: %w", err)
	}

	if err := r.GET(contributorProtocolPrefix+"/settings/:id", settingsHandler(contrib),
		forge.WithSchemaExclude(),
	); err != nil {
		return fmt.Errorf("authsome: register contributor settings route: %w", err)
	}

	e.Logger().Info("authsome: registered dashboard contributor protocol routes")
	return nil
}

func manifestHandler(c *authdash.Contributor) forge.Handler {
	return func(ctx forge.Context) error {
		ctx.SetHeader("Content-Type", "application/json")
		return json.NewEncoder(ctx.Response()).Encode(c.Manifest())
	}
}

func pageHandler(c *authdash.Contributor) forge.Handler {
	return func(ctx forge.Context) error {
		filepath := ctx.Param("filepath")
		route := "/" + filepath
		if filepath == "" {
			route = "/"
		}

		params := contributor.Params{
			Route:       route,
			BasePath:    ctx.Request().URL.Query().Get(basePathQueryParam),
			PageBase:    pageBaseFromContext(ctx),
			QueryParams: queryParamMap(ctx.Request()),
		}

		comp, err := c.RenderPage(ctx.Request().Context(), route, params)
		return writeFragment(ctx, comp, err)
	}
}

func widgetHandler(c *authdash.Contributor) forge.Handler {
	return func(ctx forge.Context) error {
		comp, err := c.RenderWidget(ctx.Request().Context(), ctx.Param("id"))
		return writeFragment(ctx, comp, err)
	}
}

func settingsHandler(c *authdash.Contributor) forge.Handler {
	return func(ctx forge.Context) error {
		comp, err := c.RenderSettings(ctx.Request().Context(), ctx.Param("id"))
		return writeFragment(ctx, comp, err)
	}
}

// pageBaseFromContext returns the consumer-provided PageBase (?pb=) if set,
// or empty string to fall back to the contributor's defaults.
func pageBaseFromContext(ctx forge.Context) string {
	return ctx.Request().URL.Query().Get(pageBaseQueryParam)
}

func queryParamMap(r *http.Request) map[string]string {
	q := r.URL.Query()
	out := make(map[string]string, len(q))
	for k, v := range q {
		if k == pageBaseQueryParam || k == basePathQueryParam {
			continue
		}
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// writeFragment renders a templ.Component to the response as text/html. On a
// nil component or error, returns 500 with a short message — the consumer
// can display its own error fragment.
func writeFragment(ctx forge.Context, comp templ.Component, renderErr error) error {
	if renderErr != nil {
		ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
		ctx.Status(http.StatusInternalServerError)
		_, _ = ctx.Response().Write([]byte("authsome: render failed: " + renderErr.Error())) //nolint:errcheck // best-effort
		return nil
	}
	if comp == nil {
		ctx.Status(http.StatusNoContent)
		return nil
	}

	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := comp.Render(context.Background(), &buf); err != nil {
		ctx.Status(http.StatusInternalServerError)
		_, _ = ctx.Response().Write([]byte("authsome: render failed: " + err.Error())) //nolint:errcheck // best-effort
		return nil
	}
	_, _ = ctx.Response().Write(buf.Bytes()) //nolint:errcheck // best-effort
	return nil
}
