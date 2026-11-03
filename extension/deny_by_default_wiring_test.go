package extension_test

import (
	"context"
	"net/http"
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
	"github.com/xraph/authsome/extension"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/consent"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/scim"
	"github.com/xraph/authsome/plugins/sharedsignals"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/sso"
	"github.com/xraph/authsome/plugins/subscription"
	"github.com/xraph/authsome/plugins/waitlist"
	"github.com/xraph/authsome/store/memory"
)

// extensionAgentSafe is the whole allowlist, as the extension mounts it.
var extensionAgentSafe = map[string]bool{
	"POST /authsome/v1/signout":      true,
	"POST /authsome/v1/oauth/revoke": true,
	"POST /authsome/v1/introspect":   true,
}

// The table test in api builds its own wiring, so it would stay green if the
// extension mounted plugins on a raw router. This boots the real extension
// with every route-serving plugin and checks what it actually registered:
// every write route carries the deny-by-default interceptor, nothing skips
// it, and the agent-safe marker sits on exactly the allowlist.
func TestExtension_EveryWriteRouteCarriesTheDelegatedWriteGuard(t *testing.T) {
	app := forge.New(forge.WithAppName("authsome-deny-by-default"), forge.WithAppLogger(forge.NewNoopLogger()))

	chron, err := chronicle.New(chronicle.WithStore(chronstore.NewAdapter(chronmem.New())))
	require.NoError(t, err)
	require.NoError(t, vessel.Provide(app.Container(), func() (chronicle.Emitter, error) { return chron, nil }))
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	require.NoError(t, vessel.Provide(app.Container(), func() (*warden.Engine, error) { return w, nil }))

	wl := waitlist.New()
	wl.SetWaitlistStore(waitlist.NewMemoryStore())
	ssf := sharedsignals.New()
	ssf.SetStore(sharedsignals.NewMemoryStore())
	ext := extension.New(extension.WithEngineOptions(
		authsome.WithStore(memory.New()),
		authsome.WithPlugin(password.New()),
		authsome.WithPlugin(social.New(social.Config{})),
		authsome.WithPlugin(mfa.New(mfa.Config{})),
		authsome.WithPlugin(apikey.New()),
		authsome.WithPlugin(magiclink.New(magiclink.Config{})),
		authsome.WithPlugin(sso.New(sso.Config{})),
		authsome.WithPlugin(passkey.New(passkey.Config{RPID: "localhost"})),
		authsome.WithPlugin(organization.New()),
		authsome.WithPlugin(scim.New()),
		authsome.WithPlugin(subscription.New()),
		authsome.WithPlugin(phone.New()),
		authsome.WithPlugin(consent.New()),
		authsome.WithPlugin(oauth2provider.New()),
		authsome.WithPlugin(agentauth.New(agentauth.WithStore(agentauth.NewMemoryStore()))),
		authsome.WithPlugin(wl),
		authsome.WithPlugin(ssf),
	))
	require.NoError(t, ext.Register(app))
	ctx := context.Background()
	require.NoError(t, ext.Start(ctx))
	t.Cleanup(func() { _ = ext.Stop(ctx) })

	writes := 0
	seenSafe := map[string]bool{}
	for _, r := range app.Router().Routes() {
		if strings.HasPrefix(r.Path, "/_") {
			continue // forge's own routes (/_/health, /_/metrics, ...)
		}
		key := r.Method + " " + r.Path

		if middleware.IsAgentSafe(r) {
			seenSafe[key] = true
			assert.True(t, extensionAgentSafe[key], "%s is marked agent-safe but isn't on the allowlist", key)
		}
		if _, marked := r.Metadata[middleware.AgentSafeMetadataKey]; marked && !extensionAgentSafe[key] {
			t.Errorf("%s carries the agent-safe metadata key", key)
		}

		// Every route authsome mounts carries the interceptor, reads
		// included: the interceptor itself lets reads through, and checking
		// them is what catches a root router (which only serves GETs today)
		// mounted without the guard.
		guarded := false
		for _, i := range r.Interceptors {
			if i.Name() == middleware.DelegatedWriteInterceptorName {
				guarded = true
			}
		}
		assert.True(t, guarded, "%s has no delegated-write interceptor", key)
		assert.False(t, r.SkipInterceptors[middleware.DelegatedWriteInterceptorName], "%s skips the delegated-write interceptor", key)

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			writes++
		}
	}

	assert.Greater(t, writes, 100, "the check has to see the real routes")
	assert.Equal(t, extensionAgentSafe, seenSafe, "the allowlist and the marked routes agree exactly")
}
