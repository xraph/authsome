package extension

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xraph/forge"
	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

// TestRegisterRemoteContractContributor_PortalURLWithBasePath guards the
// production wiring shape: PortalURL is passed in already containing the
// authsome basePath (e.g. http://identity:7902/authsome), matching how
// twinos sets TWINOS_AUTH_IDENTITY_URL, the SDK's authclient.NewClient,
// the client API proxy, and the legacy dashboard contributor fetch.
//
// Regression: an earlier version of registerRemoteContractContributor did
// `remoteBaseURL := portalURL + basePath`, producing a doubled /authsome
// path that 404'd silently. The Warn-and-return-nil failure mode meant the
// only symptom was "intent auth.login not registered" surfacing later when
// a user clicked Sign In on the dashboard.
func TestRegisterRemoteContractContributor_PortalURLWithBasePath(t *testing.T) {
	t.Parallel()

	var manifestHits, dispatchHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/authsome/_forge/contract/manifest", func(w http.ResponseWriter, _ *http.Request) {
		manifestHits++
		w.Header().Set("Content-Type", "application/json")
		// Minimal valid manifest shape — schemaVersion 1, the auth
		// contributor, three intents. JSON form (not YAML) so the
		// transport decoder path is identical to production.
		_, _ = io.WriteString(w, `{
			"schemaVersion": 1,
			"contributor": {"name": "auth", "envelope": {"supports": ["v1"], "preferred": "v1"}},
			"intents": [
				{"name": "auth.login",  "kind": "command", "version": 1, "capability": "write"},
				{"name": "auth.logout", "kind": "command", "version": 1, "capability": "write"},
				{"name": "auth.config", "kind": "query",   "version": 1, "capability": "read"}
			]
		}`)
	})
	mux.HandleFunc("/authsome/_forge/contract/dispatch", func(w http.ResponseWriter, _ *http.Request) {
		dispatchHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true,"envelope":"v1","kind":"command","data":{"echoed":true}}`)
	})
	// Catch-all so the wrong URL (e.g. /authsome/authsome/_forge/contract/manifest)
	// is loudly visible in failed test output instead of returning Go's
	// default 404 with no context.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unexpected upstream path: "+r.URL.Path, http.StatusNotFound)
	})
	upstream := httptest.NewServer(mux)
	defer upstream.Close()

	e := New()
	e.clientMode = true
	// Mirror twinos: PortalURL already includes /authsome (the basePath).
	e.config = Config{BasePath: "/authsome", PortalURL: upstream.URL + "/authsome"}
	// e.Logger() panics until the extension is wired into a forge app;
	// inject a noop logger so the success/failure log lines don't crash.
	e.BaseExtension.SetLogger(forge.NewNoopLogger())

	hostReg := dashcontract.NewRegistry()
	hostWreg := dashcontract.NewWardenRegistry()
	hostDisp := dispatcher.New(dispatcher.NoopMetricsEmitter{})

	if err := e.registerRemoteContractContributor(hostDisp, hostReg, hostWreg); err != nil {
		t.Fatalf("registerRemoteContractContributor: %v", err)
	}

	// The function returns nil on fetch failure too (non-fatal). Assert the
	// intent actually landed in the registry — that's the real success signal.
	if _, ok := hostReg.Intent("auth", "auth.login", 1); !ok {
		t.Fatalf("auth.login not registered in host registry; manifestHits=%d", manifestHits)
	}
	if manifestHits != 1 {
		t.Errorf("manifest endpoint hits = %d, want 1 (double-join regression?)", manifestHits)
	}

	// Round-trip a dispatch through the forwarding dispatcher to confirm the
	// remote endpoint baseURL is what the registry will use at request time.
	data, _, err := hostDisp.Dispatch(context.Background(), dashcontract.Request{
		Envelope:      "v1",
		Kind:          dashcontract.KindCommand,
		Contributor:   "auth",
		Intent:        "auth.login",
		IntentVersion: 1,
		Payload:       json.RawMessage(`{"email":"a@b","password":"x"}`),
	}, dashcontract.Principal{})
	if err != nil {
		t.Fatalf("dispatch auth.login: %v", err)
	}
	if !strings.Contains(string(data), `"echoed":true`) {
		t.Errorf("dispatch data = %s; want upstream echo", data)
	}
	if dispatchHits != 1 {
		t.Errorf("dispatch endpoint hits = %d, want 1", dispatchHits)
	}
}

