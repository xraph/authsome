package contract

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
	"github.com/xraph/forge/extensions/dashboard/contract/remote"
)

// mockUpstream stands up an httptest.Server that emulates a remote authsome
// service exposing /_forge/contract/{manifest,dispatch}. The dispatch
// handler captures the incoming envelope so tests can assert the wire
// payload reached the upstream verbatim. Slice (m) verification.
func mockUpstream(t *testing.T) (string, *[]dashcontract.Request, func()) {
	t.Helper()
	captured := []dashcontract.Request{}
	mux := http.NewServeMux()
	mux.HandleFunc("/authsome/_forge/contract/manifest", func(w http.ResponseWriter, _ *http.Request) {
		// Hand-rolled minimal auth manifest mirroring the real one. We can't
		// invoke our own Register() here because it requires a live engine;
		// the wire path is the same regardless of which dispatcher is
		// behind the manifest.
		yaml := `
schemaVersion: 1
contributor: { name: auth, envelope: { supports: [v1], preferred: v1 } }
intents:
  - { name: auth.login,  kind: command, version: 1, capability: write }
  - { name: auth.logout, kind: command, version: 1, capability: write }
  - { name: auth.config, kind: query,   version: 1, capability: read  }
`
		m, err := loader.Load(strings.NewReader(yaml), "mock.yaml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/authsome/_forge/contract/dispatch", func(w http.ResponseWriter, r *http.Request) {
		var req dashcontract.Request
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		captured = append(captured, req)
		// Echo a success envelope so the forwarding path returns cleanly.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dashcontract.Response{
			OK: true, Envelope: "v1", Kind: req.Kind,
			Data: json.RawMessage(`{"echoed":true}`),
		})
	})
	srv := httptest.NewServer(mux)
	return srv.URL, &captured, srv.Close
}

// TestEndToEnd_RemoteAuthLoginRoutes proves the slice-(m) plumbing fixes
// the "intent auth.login not registered" symptom. Sequence:
//
//  1. Stand up a mock upstream emulating authsome's /_forge/contract endpoints.
//  2. Fetch the manifest like authsome's client-mode wiring does.
//  3. Register it as a remote on the host registry.
//  4. Install the forwarding dispatcher.
//  5. Dispatch auth.login through the host dispatcher → expect the envelope
//     to land on the upstream.
func TestEndToEnd_RemoteAuthLoginRoutes(t *testing.T) {
	upURL, captured, closer := mockUpstream(t)
	defer closer()

	hostReg := dashcontract.NewRegistry()
	hostWreg := dashcontract.NewWardenRegistry()
	hostDisp := dispatcher.New(dispatcher.NoopMetricsEmitter{})

	// Match what registerRemoteContractContributor does in client mode:
	// the base is PortalURL + BasePath ("/authsome").
	remoteBase := upURL + "/authsome"

	m, err := remote.FetchManifest(context.Background(), remoteBase, "", nil)
	if err != nil {
		t.Fatalf("fetch manifest: %v", err)
	}
	if validateErr := loader.Validate(m, hostWreg); validateErr != nil {
		t.Fatalf("validate: %v", validateErr)
	}
	if regErr := hostReg.RegisterRemote(m, dashcontract.RemoteEndpoint{BaseURL: remoteBase}); regErr != nil {
		t.Fatalf("register remote: %v", regErr)
	}
	hostDisp.SetRemoteDispatcher(remote.NewForwardingDispatcher(hostReg))

	// No local handler for auth.login — the forwarding dispatcher should
	// ship it to the upstream.
	data, _, err := hostDisp.Dispatch(context.Background(), dashcontract.Request{
		Envelope:      "v1",
		Kind:          dashcontract.KindCommand,
		Contributor:   "auth",
		Intent:        "auth.login",
		IntentVersion: 1,
		Payload:       json.RawMessage(`{"email":"a@b","password":"x"}`),
	}, dashcontract.Principal{})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if string(data) != `{"echoed":true}` {
		t.Errorf("data = %s; want upstream echo", data)
	}
	if len(*captured) != 1 {
		t.Fatalf("upstream saw %d requests; want 1", len(*captured))
	}
	got := (*captured)[0]
	if got.Contributor != "auth" || got.Intent != "auth.login" {
		t.Errorf("upstream got %s/%s; want auth/auth.login", got.Contributor, got.Intent)
	}
	if string(got.Payload) != `{"email":"a@b","password":"x"}` {
		t.Errorf("upstream payload not forwarded verbatim: %s", got.Payload)
	}
}

// TestEndToEnd_UpstreamManifestExposesAuthLogin guards the upstream
// manifest format authsome's standalone service serves. If a future change
// drops auth.login from the manifest, this catches it before it ships.
func TestEndToEnd_UpstreamManifestExposesAuthLogin(t *testing.T) {
	upURL, _, closer := mockUpstream(t)
	defer closer()
	m, err := remote.FetchManifest(context.Background(), upURL+"/authsome", "", nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	have := map[string]bool{}
	for _, in := range m.Intents {
		have[in.Name] = true
	}
	for _, want := range []string{"auth.login", "auth.logout", "auth.config"} {
		if !have[want] {
			t.Errorf("manifest missing %q intent", want)
		}
	}
}

// TestEndToEnd_HostFallsThroughOnUnknownIntent confirms intents the
// upstream doesn't declare still surface as CodeNotFound from the host —
// the forwarding dispatcher routes by contributor, and the upstream
// returns its own error envelope for unknown intents.
func TestEndToEnd_HostFallsThroughOnUnknownIntent(t *testing.T) {
	upURL, _, closer := mockUpstream(t)
	defer closer()
	hostReg := dashcontract.NewRegistry()
	hostWreg := dashcontract.NewWardenRegistry()
	hostDisp := dispatcher.New(dispatcher.NoopMetricsEmitter{})
	remoteBase := upURL + "/authsome"
	m, _ := remote.FetchManifest(context.Background(), remoteBase, "", nil)
	_ = loader.Validate(m, hostWreg)
	_ = hostReg.RegisterRemote(m, dashcontract.RemoteEndpoint{BaseURL: remoteBase})
	hostDisp.SetRemoteDispatcher(remote.NewForwardingDispatcher(hostReg))

	// "unknown.contributor" was never registered. Forwarding dispatcher
	// returns CodeNotFound (its own, not the upstream's) because the
	// registry has no endpoint for the contributor.
	_, _, err := hostDisp.Dispatch(context.Background(), dashcontract.Request{
		Envelope: "v1", Kind: dashcontract.KindQuery,
		Contributor: "unknown", Intent: "x", IntentVersion: 1,
	}, dashcontract.Principal{})
	var ce *dashcontract.Error
	if !errors.As(err, &ce) || ce.Code != dashcontract.CodeNotFound {
		t.Errorf("expected CodeNotFound for unknown contributor, got %v", err)
	}
}
