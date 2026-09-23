package extension

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/forge"
	contract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

func TestContractServerExportsInstalledPlugins(t *testing.T) {
	wardenEngine, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := authsome.NewEngine(authsome.WithStore(memory.New()), authsome.WithWarden(wardenEngine), authsome.WithPlugin(apikey.New()))
	if err != nil {
		t.Fatal(err)
	}
	extension := New()
	extension.engine = engine
	extension.SetLogger(forge.NewNoopLogger())
	router := forge.NewRouter()
	if err := extension.registerContractServer(router); err != nil {
		t.Fatal(err)
	}
	if _, ok := extension.contractReg.Intent("apikey", "apikeys.list", 1); !ok {
		t.Fatal("installed API key contributor missing from contract server")
	}
}

func TestRemoteContractCatalogRegistersAndDispatchesPlugin(t *testing.T) {
	manifest := func(name, intent string) *contract.ContractManifest {
		return &contract.ContractManifest{SchemaVersion: 1, Contributor: contract.Contributor{Name: name, Envelope: contract.EnvelopeSupport{Supports: []string{"v1"}, Preferred: "v1"}}, Intents: []contract.Intent{{Name: intent, Kind: "query", Version: 1, Capability: "read"}}}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/authsome/_forge/contract/manifest" {
			if request.Header.Get("Authorization") != "Bearer service-key" {
				t.Error("missing service authorization")
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"manifests": []*contract.ContractManifest{manifest("auth", "auth.config"), manifest("apikey", "apikeys.list")}})
			return
		}
		if request.URL.Path != "/authsome/_forge/contract/dispatch" {
			http.NotFound(writer, request)
			return
		}
		var envelope contract.Request
		if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		if envelope.Contributor != "apikey" || envelope.Intent != "apikeys.list" {
			t.Errorf("wrong forwarded request: %+v", envelope)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"ok": true, "envelope": "v1", "kind": "query", "data": map[string]any{"keys": []string{"key-1"}}})
	}))
	defer upstream.Close()
	extension := New()
	extension.clientMode = true
	extension.config = Config{PortalURL: upstream.URL + "/authsome", ServiceAPIKey: "service-key"}
	extension.SetLogger(forge.NewNoopLogger())
	registry := contract.NewRegistry()
	remoteDispatcher := dispatcher.New(dispatcher.NoopMetricsEmitter{})
	if err := extension.registerRemoteContractContributor(remoteDispatcher, registry, contract.NewWardenRegistry()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"auth", "apikey"} {
		if !registry.IsRemote(name) {
			t.Fatalf("%s is not registered remotely", name)
		}
	}
	data, _, err := remoteDispatcher.Dispatch(context.Background(), contract.Request{Envelope: "v1", Kind: contract.KindQuery, Contributor: "apikey", Intent: "apikeys.list", IntentVersion: 1, Payload: json.RawMessage(`{}`)}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"keys":["key-1"]}` {
		t.Fatalf("unexpected response: %s", data)
	}
}
