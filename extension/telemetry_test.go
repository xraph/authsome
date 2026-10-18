package extension_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/chronicle"
	chronstore "github.com/xraph/chronicle/store"
	chronmem "github.com/xraph/chronicle/store/memory"
	"github.com/xraph/forge"
	"github.com/xraph/vessel"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/extension"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store/memory"
)

// The extension hands the engine the app's registry, so authsome's
// families land on /_/metrics under forge's namespace. Route latency and
// status are forge's own http.server metrics now: authsome records no
// per-request event of its own, so no route path (and none of the IDs and
// secrets some paths carry) ever becomes a label.
func TestExtensionRecordsIntoTheAppRegistry(t *testing.T) {
	app := forge.New(forge.WithAppName("authsome-extension-telemetry"), forge.WithAppLogger(forge.NewNoopLogger()))

	chron, err := chronicle.New(chronicle.WithStore(chronstore.NewAdapter(chronmem.New())))
	require.NoError(t, err)
	require.NoError(t, vessel.Provide(app.Container(), func() (chronicle.Emitter, error) { return chron, nil }))
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	require.NoError(t, vessel.Provide(app.Container(), func() (*warden.Engine, error) { return w, nil }))

	ext := extension.New(extension.WithEngineOptions(authsome.WithStore(memory.New())))
	require.NoError(t, ext.Register(app))
	ctx := context.Background()
	require.NoError(t, ext.Start(ctx))
	t.Cleanup(func() { _ = ext.Stop(ctx) })
	require.NotNil(t, ext.Engine().Telemetry(), "the extension wires the app's registry into the engine")

	// One request through the mounted routes.
	rr := httptest.NewRecorder()
	app.Router().ServeHTTP(rr, httptest.NewRequestWithContext(ctx, http.MethodGet, "/authsome/v1/health", http.NoBody))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	// One sign-in nobody can complete.
	platform, err := id.ParseAppID(ext.Engine().PlatformAppID().String())
	require.NoError(t, err)
	_, _, err = ext.Engine().SignIn(ctx, &account.SignInRequest{AppID: platform, Email: "nobody@example.com", Password: "not-the-password"})
	require.ErrorIs(t, err, account.ErrInvalidCredentials)

	p, ok := app.Metrics().(interface{ PrometheusHandler() http.Handler })
	require.True(t, ok)
	scrape := httptest.NewRecorder()
	p.PrometheusHandler().ServeHTTP(scrape, httptest.NewRequestWithContext(ctx, http.MethodGet, "/_/metrics", http.NoBody))
	body := scrape.Body.String()

	assert.Contains(t, body, `forge_authsome_signin_attempts_total{method="password",result="invalid"} 1`)
	assert.Contains(t, body, `forge_authsome_password_hash_duration_seconds_count{algorithm="bcrypt",operation="verify"} 1`)
	assert.NotContains(t, body, `action="http.request"`, "the old per-request middleware is gone")
	assert.Contains(t, body, `http_route="/authsome/v1/health"`, "forge measures the route, by its template")
	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, "forge_authsome_") {
			continue
		}
		assert.NotContains(t, line, "/v1/", "no route path is an authsome label")
		assert.NotContains(t, line, "tenant=", "no authsome family is labelled by tenant")
		assert.NotContains(t, line, platform.String(), "no app ID in any authsome series")
	}
}
