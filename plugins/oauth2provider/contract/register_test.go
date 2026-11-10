package contract

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xraph/forge/extensions/dashboard"
	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/idempotency"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
)

// allIntents is every intent the manifest declares, with the kind the
// dashboard is told to send it as.
var allIntents = map[string]contract.IntentKind{
	"oauth2.clients.list":         contract.IntentKindQuery,
	"oauth2.clients.detail":       contract.IntentKindQuery,
	"oauth2.clients.create":       contract.IntentKindCommand,
	"oauth2.clients.update":       contract.IntentKindCommand,
	"oauth2.clients.delete":       contract.IntentKindCommand,
	"oauth2.clients.rotateSecret": contract.IntentKindCommand,
	"oauth2.issuer.info":          contract.IntentKindQuery,
	"oauth2.keys.list":            contract.IntentKindQuery,
	"oauth2.keys.rotate":          contract.IntentKindCommand,
}

// registered registers the contract against a real registry and a dispatcher
// that keeps idempotency entries, the way the dashboard extension wires it.
func registered(t *testing.T, svc OAuth2Service) (*dispatcher.Dispatcher, contract.Registry) {
	t.Helper()
	d := dispatcher.NewWithOptions(dispatcher.NoopMetricsEmitter{},
		dispatcher.WithIdempotencyStore(dashboard.AdaptIdempotencyStore(idempotency.NewInMemoryStore())))
	reg := contract.NewRegistry()
	if err := Register(d, reg, contract.NewWardenRegistry(), Deps{Engine: secutil.NewTestEngine(t), Plugin: svc}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return d, reg
}

func operator(appID id.AppID) contract.Principal {
	p := contract.PrincipalFor(&dashauth.UserInfo{Subject: "operator"})
	p.Claims["app_id"] = appID.String()
	return p
}

func dispatch(t *testing.T, d *dispatcher.Dispatcher, p contract.Principal, kind contract.IntentKind, intent, payload, key string) (json.RawMessage, error) {
	t.Helper()
	k := contract.KindQuery
	if kind == contract.IntentKindCommand {
		k = contract.KindCommand
	}
	data, _, err := d.Dispatch(context.Background(), contract.Request{
		Envelope: "v1", Kind: k, Contributor: "oauth2provider", Intent: intent, IntentVersion: 1,
		Payload: json.RawMessage(payload), IdempotencyKey: key,
	}, p)
	return data, err
}

func TestRegister_AllNineIntentsWithTheirKinds(t *testing.T) {
	_, reg := registered(t, &fakeService{})
	if _, ok := reg.Contributor("oauth2provider"); !ok {
		t.Fatal("oauth2provider contributor is not registered")
	}
	for name, kind := range allIntents {
		in, ok := reg.Intent("oauth2provider", name, 1)
		if !ok {
			t.Errorf("intent %s v1 is not registered", name)
			continue
		}
		if in.Kind != kind {
			t.Errorf("%s kind = %q, want %q", name, in.Kind, kind)
		}
	}
}

// Every intent reaches a handler: an unregistered one would answer NOT_FOUND
// with "not registered", and a handler whose plugin is missing answers
// UNAVAILABLE, so getting UNAVAILABLE from each proves the dispatcher holds
// a handler for all nine.
func TestRegister_DispatcherHoldsAHandlerPerIntent(t *testing.T) {
	d := dispatcher.New(dispatcher.NoopMetricsEmitter{})
	if err := Register(d, contract.NewRegistry(), contract.NewWardenRegistry(), Deps{Engine: secutil.NewTestEngine(t)}); err != nil {
		t.Fatalf("register: %v", err)
	}
	for name, kind := range allIntents {
		_, err := dispatch(t, d, operator(id.NewAppID()), kind, name, `{"id":"x","name":"n"}`, "")
		var ce *contract.Error
		if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
			t.Errorf("%s: err = %v, want UNAVAILABLE from its handler", name, err)
		}
	}
}

// create and rotateSecret answer with a plaintext secret, so a replay with
// the same idempotency key must not hand it out again or mint a second one.
func TestRegister_SecretCommandsAreNeverReplayed(t *testing.T) {
	for _, c := range []struct {
		intent  string
		payload string
		calls   func(*fakeService) int
	}{
		{"oauth2.clients.create", `{"name":"web"}`, func(f *fakeService) int { return f.createCall }},
		{"oauth2.clients.rotateSecret", `{"id":"oa2c_1"}`, func(f *fakeService) int { return f.rotateSecretCall }},
	} {
		t.Run(c.intent, func(t *testing.T) {
			f := &fakeService{
				created:       Created{ID: "oa2c_1", ClientID: "pub", ClientSecret: "s3cr3t"},
				rotatedSecret: RotatedClient{ID: "oa2c_1", ClientID: "pub", ClientSecret: "s3cr3t"},
			}
			d, _ := registered(t, f)
			p := operator(id.NewAppID())

			data, err := dispatch(t, d, p, contract.IntentKindCommand, c.intent, c.payload, "key-1")
			if err != nil {
				t.Fatalf("first dispatch: %v", err)
			}
			if !strings.Contains(string(data), "s3cr3t") {
				t.Fatalf("first answer = %s, want the secret", data)
			}

			data, err = dispatch(t, d, p, contract.IntentKindCommand, c.intent, c.payload, "key-1")
			var ce *contract.Error
			if !errors.As(err, &ce) || ce.Code != contract.CodeConflict || ce.Details[dispatcher.ReasonDetail] != dispatcher.ReasonAlreadyRan {
				t.Fatalf("replay: data = %s, err = %v, want CONFLICT %s", data, err, dispatcher.ReasonAlreadyRan)
			}
			if n := c.calls(f); n != 1 {
				t.Fatalf("service ran %d times, want 1", n)
			}
		})
	}
}

// A command that carries no secret replays its stored answer instead of
// running again, which is what sets the two secret commands apart.
func TestRegister_OrdinaryCommandReplays(t *testing.T) {
	f := &fakeService{}
	d, _ := registered(t, f)
	p := operator(id.NewAppID())
	first, err := dispatch(t, d, p, contract.IntentKindCommand, "oauth2.clients.delete", `{"id":"oa2c_1"}`, "key-1")
	if err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	second, err := dispatch(t, d, p, contract.IntentKindCommand, "oauth2.clients.delete", `{"id":"oa2c_1"}`, "key-1")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if string(first) != string(second) || f.deleteCall != 1 {
		t.Fatalf("first = %s, second = %s, delete ran %d times", first, second, f.deleteCall)
	}
}

func TestRegister_ScopesToThePrincipalsApp(t *testing.T) {
	f := &fakeService{}
	d, _ := registered(t, f)
	appID := id.NewAppID()
	if _, err := dispatch(t, d, operator(appID), contract.IntentKindQuery, "oauth2.clients.list", `{}`, ""); err != nil {
		t.Fatal(err)
	}
	if f.lastApp.String() != appID.String() {
		t.Fatalf("service saw app %s, want %s", f.lastApp, appID)
	}
}

func TestRegister_RequiresEngine(t *testing.T) {
	err := Register(dispatcher.New(dispatcher.NoopMetricsEmitter{}), contract.NewRegistry(), contract.NewWardenRegistry(), Deps{Plugin: &fakeService{}})
	if err == nil || !strings.Contains(err.Error(), "Engine is required") {
		t.Fatalf("err = %v, want Engine is required", err)
	}
}

func TestRegister_TwiceIsAnError(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	d := dispatcher.New(dispatcher.NoopMetricsEmitter{})
	reg := contract.NewRegistry()
	deps := Deps{Engine: eng, Plugin: &fakeService{}}
	if err := Register(d, reg, contract.NewWardenRegistry(), deps); err != nil {
		t.Fatal(err)
	}

	// The same registry refuses the manifest a second time.
	err := Register(dispatcher.New(dispatcher.NoopMetricsEmitter{}), reg, contract.NewWardenRegistry(), deps)
	if err == nil || !strings.Contains(err.Error(), "register manifest") {
		t.Fatalf("second manifest: err = %v", err)
	}

	// A fresh registry over the same dispatcher fails on the first handler,
	// and the error names the intent.
	err = Register(d, contract.NewRegistry(), contract.NewWardenRegistry(), deps)
	if err == nil || !strings.Contains(err.Error(), "register oauth2.clients.list") {
		t.Fatalf("second handlers: err = %v", err)
	}
}
