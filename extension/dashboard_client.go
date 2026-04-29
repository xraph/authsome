package extension

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	authdash "github.com/xraph/authsome/dashboard"
)

// proxyContributor is a LocalContributor used when authsome runs as a client
// of a remote authsome service. Its Manifest mirrors the engine-mode authsome
// contributor (so the icon and nav appear in the host dashboard's app grid),
// but its Render* methods proxy HTTP fragments from the remote authsome
// service so the host stays on its own URL.
type proxyContributor struct {
	manifest   *contributor.Manifest
	remoteBase string // e.g. "http://identity:7902/authsome"
	apiKey     string
	client     *http.Client
}

var (
	_ contributor.LocalContributor = (*proxyContributor)(nil)
	_ contributor.ContextPreparer  = (*proxyContributor)(nil)
)

// newProxyContributor constructs a proxy contributor from the authsome client
// portal URL and an optional service API key. portalURL is expected to be the
// authsome API base URL (e.g. "http://identity:7902/authsome") — that is the
// same prefix where the engine-mode extension publishes its contributor
// protocol routes.
func newProxyContributor(portalURL, apiKey string) *proxyContributor {
	return &proxyContributor{
		manifest:   buildProxyManifest(),
		remoteBase: strings.TrimRight(portalURL, "/"),
		apiKey:     apiKey,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// buildProxyManifest produces a manifest that matches the engine-mode
// authsome dashboard at the structural level (so the host dashboard renders
// the correct app grid entry, sidebar nav, and widget descriptors). It
// deliberately omits engine-bound switcher templ components — those would
// dereference a missing engine when rendered in client mode.
func buildProxyManifest() *contributor.Manifest {
	m := authdash.NewManifest(nil, nil)
	m.SidebarHeaderContent = nil
	m.TopbarExtraContent = nil
	return m
}

func (p *proxyContributor) Manifest() *contributor.Manifest { return p.manifest }

// PrepareContext is a no-op — context enrichment lives on the remote side.
func (p *proxyContributor) PrepareContext(ctx context.Context, _ string) context.Context {
	return ctx
}

func (p *proxyContributor) RenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	target := p.remoteBase + contributorProtocolPrefix + "/pages"
	if route != "" && route != "/" {
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		target += route
	}
	return p.fetchFragment(ctx, target, params.BasePath, params.PageBase)
}

func (p *proxyContributor) RenderWidget(ctx context.Context, widgetID string) (templ.Component, error) {
	target := p.remoteBase + contributorProtocolPrefix + "/widgets/" + url.PathEscape(widgetID)
	return p.fetchFragment(ctx, target, "", "")
}

func (p *proxyContributor) RenderSettings(ctx context.Context, settingID string) (templ.Component, error) {
	target := p.remoteBase + contributorProtocolPrefix + "/settings/" + url.PathEscape(settingID)
	return p.fetchFragment(ctx, target, "", "")
}

// fetchFragment performs an authenticated GET against the remote authsome
// contributor protocol and returns the response body wrapped as a templ.Raw
// component. basePath and pageBase are forwarded as ?bp= and ?pb= query params
// so the remote renders links scoped to the host dashboard's URL space.
func (p *proxyContributor) fetchFragment(ctx context.Context, target, basePath, pageBase string) (templ.Component, error) {
	target = appendQuery(target, basePathQueryParam, basePath)
	target = appendQuery(target, pageBaseQueryParam, pageBase)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("authsome proxy: build request: %w", err)
	}
	req.Header.Set("X-Forge-Dashboard", "true")
	req.Header.Set("Accept", "text/html")
	if p.apiKey != "" {
		req.Header.Set("X-Forge-Api-Key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authsome proxy: fetch %s: %w", target, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return templ.Raw(""), nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authsome proxy: %s returned %d", target, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("authsome proxy: read response: %w", err)
	}
	return templ.Raw(string(body)), nil
}

// appendQuery appends a query parameter to target, returning target unchanged
// when value is empty.
func appendQuery(target, key, value string) string {
	if value == "" {
		return target
	}
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	return target + sep + key + "=" + url.QueryEscape(value)
}
