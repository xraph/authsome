package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xraph/forge/extensions/dashboard/contract"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

func agentCtx() context.Context {
	return middleware.WithSession(context.Background(), &session.Session{
		ID: id.NewSessionID(), UserID: id.NewUserID(),
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	})
}

// Every command refuses an agent session before it touches the service,
// whatever the service would have answered.
func TestCommands_RefuseDelegatedSessions(t *testing.T) {
	f := &fakeService{}
	deps := Deps{Plugin: f}
	ctx := agentCtx()
	p := contract.Principal{}
	calls := map[string]func() error{
		"update": func() error {
			_, err := clientsUpdateHandler(deps)(ctx, UpdateClientInput{ID: "oa2c_1"}, p)
			return err
		},
		"delete": func() error {
			_, err := clientsDeleteHandler(deps)(ctx, ClientIDInput{ID: "oa2c_1"}, p)
			return err
		},
		"rotateSecret": func() error {
			_, err := clientsRotateSecretHandler(deps)(ctx, ClientIDInput{ID: "oa2c_1"}, p)
			return err
		},
		"keys.rotate": func() error {
			_, err := keysRotateHandler(deps)(ctx, struct{}{}, p)
			return err
		},
	}
	for name, call := range calls {
		err := call()
		expectCode(t, err, contract.CodePermissionDenied)
		var ce *contract.Error
		if errors.As(err, &ce) && ce.Message != middleware.DelegatedSessionMessage {
			t.Errorf("%s: message = %q", name, ce.Message)
		}
	}
	if f.deleteCall != 0 || f.rotateSecretCall != 0 || f.lastID != "" {
		t.Fatalf("the service was reached for a delegated session: %+v", f)
	}
}

// Queries do not refuse delegated sessions: reading is allowed.
func TestQueries_AllowDelegatedSessions(t *testing.T) {
	f := &fakeService{getClient: Client{ID: "oa2c_1"}}
	if _, err := clientsDetailHandler(Deps{Plugin: f})(agentCtx(), ClientIDInput{ID: "oa2c_1"}, contract.Principal{}); err != nil {
		t.Fatalf("detail for an agent session: %v", err)
	}
}

func TestClientsList_ProjectsEveryClientWithoutNils(t *testing.T) {
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.FixedZone("x", 3600))
	f := &fakeService{listed: []Client{
		{ID: "oa2c_1", ClientID: "pub1", Name: "One", Public: true, FirstParty: true, CreatedAt: at,
			RedirectURIs: []string{"https://a/cb"}, Scopes: []string{"openid"}, GrantTypes: []string{"authorization_code"}},
		{ID: "oa2c_2", ClientID: "pub2", Name: "Two"},
	}}
	out, err := clientsListHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Clients) != 2 {
		t.Fatalf("clients = %d, want 2", len(out.Clients))
	}
	one, two := out.Clients[0], out.Clients[1]
	if one.ID != "oa2c_1" || one.ClientID != "pub1" || one.Name != "One" || !one.Public || !one.FirstParty ||
		one.CreatedAt != "2026-03-04T04:06:07Z" || one.RedirectURIs[0] != "https://a/cb" || one.Scopes[0] != "openid" ||
		one.GrantTypes[0] != "authorization_code" {
		t.Errorf("first = %+v", one)
	}
	if two.RedirectURIs == nil || two.Scopes == nil || two.GrantTypes == nil {
		t.Errorf("second has nil slices, which serialise as null: %+v", two)
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(b)), `secret"`) {
		t.Fatalf("list response mentions a secret: %s", b)
	}
}

func TestClientsList_EmptyIsAnEmptyArray(t *testing.T) {
	out, err := clientsListHandler(Deps{Plugin: &fakeService{}})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if string(b) != `{"clients":[]}` {
		t.Fatalf("body = %s", b)
	}
}

func TestClientsList_StoreFailureIsInternal(t *testing.T) {
	_, err := clientsListHandler(Deps{Plugin: &fakeService{listErr: errors.New("dial tcp 10.0.0.1:5432")}})(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeInternal)
	if strings.Contains(err.Error(), "10.0.0.1") {
		t.Fatalf("error leaks the cause: %v", err)
	}
}

func TestClientsDetail_CarriesNoSecret(t *testing.T) {
	f := &fakeService{getClient: Client{ID: "oa2c_1", ClientID: "pub", Resources: []string{"https://api"}, DPoPMode: "required", AuthMethod: "client_secret_basic"}}
	out, err := clientsDetailHandler(Deps{Plugin: f})(context.Background(), ClientIDInput{ID: "oa2c_1"}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if f.lastID != "oa2c_1" || out.Resources[0] != "https://api" || out.DPoPMode != "required" {
		t.Fatalf("out = %+v, service saw %q", out, f.lastID)
	}
	b, _ := json.Marshal(out)
	if strings.Contains(strings.ToLower(string(b)), `secret"`) {
		t.Fatalf("detail response mentions a secret: %s", b)
	}
}

func TestClientsCreate_PassesTrimmedNameAndFields(t *testing.T) {
	f := &fakeService{created: Created{ID: "oa2c_1", ClientID: "pub"}}
	in := CreateClientInput{
		Name: "  Web  ", Public: true, FirstParty: true, RedirectURIs: []string{"https://a/cb"},
		Scopes: []string{"openid"}, GrantTypes: []string{"authorization_code"}, Resources: []string{"https://api"},
	}
	if _, err := clientsCreateHandler(Deps{Plugin: f})(context.Background(), in, contract.Principal{}); err != nil {
		t.Fatal(err)
	}
	got := f.lastCreate
	if got.Name != "Web" || !got.Public || !got.FirstParty || got.RedirectURIs[0] != "https://a/cb" ||
		got.Scopes[0] != "openid" || got.GrantTypes[0] != "authorization_code" || got.Resources[0] != "https://api" {
		t.Fatalf("service got %+v", got)
	}
}

func TestClientsUpdate_PassesOnlyWhatWasSent(t *testing.T) {
	name := "Renamed"
	f := &fakeService{updated: Client{ID: "oa2c_1", Name: name}}
	out, err := clientsUpdateHandler(Deps{Plugin: f})(context.Background(), UpdateClientInput{ID: "oa2c_1", Name: &name}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	u := f.lastUpdate
	if f.lastID != "oa2c_1" || u.Name == nil || *u.Name != name || u.RedirectURIs != nil || u.Scopes != nil || u.GrantTypes != nil || u.Resources != nil {
		t.Fatalf("service got id %q, %+v", f.lastID, u)
	}
	if out.ID != "oa2c_1" || out.Name != name || out.Resources == nil {
		t.Fatalf("out = %+v", out)
	}
}

func TestClientsUpdate_MapsErrors(t *testing.T) {
	for _, c := range []struct {
		err  error
		code contract.ErrorCode
		msg  string
	}{
		{&ValidationError{Msg: "grant_type \"x\" is not supported"}, contract.CodeBadRequest, "grant_type \"x\" is not supported"},
		{fmt.Errorf("load: %w", ErrClientNotFound), contract.CodeNotFound, "oauth2 client not found"},
		{errors.New("boom"), contract.CodeInternal, "internal error"},
	} {
		_, err := clientsUpdateHandler(Deps{Plugin: &fakeService{updateErr: c.err}})(context.Background(), UpdateClientInput{ID: "oa2c_1"}, contract.Principal{})
		expectCode(t, err, c.code)
		var ce *contract.Error
		if errors.As(err, &ce) && ce.Message != c.msg {
			t.Errorf("%v: message = %q, want %q", c.err, ce.Message, c.msg)
		}
	}
}

func TestClientsDelete_AcksWithTheID(t *testing.T) {
	f := &fakeService{}
	out, err := clientsDeleteHandler(Deps{Plugin: f})(context.Background(), ClientIDInput{ID: "oa2c_9"}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.ID != "oa2c_9" || f.lastID != "oa2c_9" {
		t.Fatalf("out = %+v, service saw %q", out, f.lastID)
	}
	b, _ := json.Marshal(out)
	if string(b) != `{"ok":true,"id":"oa2c_9"}` {
		t.Fatalf("body = %s", b)
	}
}

func TestClientsDelete_NotFound(t *testing.T) {
	_, err := clientsDeleteHandler(Deps{Plugin: &fakeService{deleteErr: ErrClientNotFound}})(context.Background(), ClientIDInput{ID: "x"}, contract.Principal{})
	expectCode(t, err, contract.CodeNotFound)
}

func TestClientsRotateSecret_CarriesTheNewSecret(t *testing.T) {
	f := &fakeService{rotatedSecret: RotatedClient{ID: "oa2c_1", ClientID: "pub", ClientSecret: "fresh"}}
	out, err := clientsRotateSecretHandler(Deps{Plugin: f})(context.Background(), ClientIDInput{ID: "oa2c_1"}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if out != (RotatedSecret{ID: "oa2c_1", ClientID: "pub", ClientSecret: "fresh"}) || f.lastID != "oa2c_1" {
		t.Fatalf("out = %+v", out)
	}
}

func TestClientsRotateSecret_PublicClientIsBadRequest(t *testing.T) {
	f := &fakeService{rotateSecretErr: &ValidationError{Msg: "a public client has no client_secret to rotate"}}
	_, err := clientsRotateSecretHandler(Deps{Plugin: f})(context.Background(), ClientIDInput{ID: "oa2c_1"}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)
	var ce *contract.Error
	if errors.As(err, &ce) && ce.Message != "a public client has no client_secret to rotate" {
		t.Fatalf("message = %q", ce.Message)
	}
}

func TestIssuerInfo_Projects(t *testing.T) {
	f := &fakeService{issuer: IssuerDetails{
		Issuer: "https://iss/apps/acme", DiscoveryURL: "https://iss/apps/acme/.well-known/openid-configuration",
		JWKSURI: "https://iss/apps/acme/v1/oauth/jwks", AppIssuer: true, Algs: []string{"ES256"},
	}}
	out, err := issuerInfoHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	want := IssuerInfoResponse{
		Issuer: "https://iss/apps/acme", DiscoveryURL: "https://iss/apps/acme/.well-known/openid-configuration",
		JWKSURI: "https://iss/apps/acme/v1/oauth/jwks", AppIssuer: true, Algs: []string{"ES256"},
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out = %+v, want %+v", out, want)
	}
}

func TestIssuerInfo_NilAlgsIsAnEmptyArray(t *testing.T) {
	out, err := issuerInfoHandler(Deps{Plugin: &fakeService{}})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if !strings.Contains(string(b), `"algs":[]`) {
		t.Fatalf("body = %s", b)
	}
}

func TestIssuerInfo_LookupFailureIsInternal(t *testing.T) {
	_, err := issuerInfoHandler(Deps{Plugin: &fakeService{issuerErr: errors.New("settings down")}})(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeInternal)
}

func TestKeysList_Projects(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	retires := created.Add(48 * time.Hour)
	f := &fakeService{keys: []KeyInfo{
		{KID: "a", Alg: "RS256", Source: "managed", Status: "active", CreatedAt: created},
		{KID: "b", Alg: "ES256", Source: "operator", Status: "retiring", CreatedAt: created, RetiresAt: &retires},
	}}
	out, err := keysListHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	want := KeyListResponse{Issuer: "https://iss", Keys: []KeySummary{
		{KID: "a", Alg: "RS256", Source: "managed", Status: "active", CreatedAt: "2026-01-01T00:00:00Z"},
		{KID: "b", Alg: "ES256", Source: "operator", Status: "retiring", CreatedAt: "2026-01-01T00:00:00Z", RetiresAt: "2026-01-03T00:00:00Z"},
	}}
	if out.Issuer != want.Issuer || len(out.Keys) != 2 || out.Keys[0] != want.Keys[0] || out.Keys[1] != want.Keys[1] {
		t.Fatalf("out = %+v", out)
	}
	b, _ := json.Marshal(out.Keys[0])
	if strings.Contains(string(b), "retiresAt") {
		t.Fatalf("an active key carries retiresAt: %s", b)
	}
}

func TestKeysList_EmptyIsAnEmptyArray(t *testing.T) {
	out, err := keysListHandler(Deps{Plugin: &fakeService{}})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if string(b) != `{"issuer":"https://iss","keys":[]}` {
		t.Fatalf("body = %s", b)
	}
}

func TestKeysRotate_NoSigningKeyIsUnavailable(t *testing.T) {
	_, err := keysRotateHandler(Deps{Plugin: &fakeService{rotateErr: ErrNoSigningKey}})(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func TestMapErr_PassesThroughAndUnwraps(t *testing.T) {
	if mapErr(nil) != nil {
		t.Fatal("nil should stay nil")
	}
	own := &contract.Error{Code: contract.CodeConflict, Message: "mine"}
	if got := mapErr(fmt.Errorf("wrapped: %w", own)); !errors.Is(got, own) || got.Error() != own.Error() {
		t.Fatalf("a contract error should pass through as itself, got %v", got)
	}
	for _, c := range []struct {
		err  error
		code contract.ErrorCode
		msg  string
	}{
		{fmt.Errorf("a: %w", ErrOperatorKey), contract.CodeBadRequest, "the signing key for this issuer is managed by configuration"},
		{fmt.Errorf("a: %w", ErrSharedKey), contract.CodeBadRequest, "the deployment issuer's signing key is shared by every app and rotates on its own schedule"},
		{fmt.Errorf("a: %w", ErrKeyTooNew), contract.CodeBadRequest, "the current signing key is less than an hour old; try again later"},
		{fmt.Errorf("a: %w", ErrNoSigningKey), contract.CodeUnavailable, "no id_token signing key is available; configure an encryption key or an operator key"},
		{fmt.Errorf("a: %w", &ValidationError{Msg: "bad uri"}), contract.CodeBadRequest, "bad uri"},
	} {
		err := mapErr(c.err)
		expectCode(t, err, c.code)
		var ce *contract.Error
		if errors.As(err, &ce) && ce.Message != c.msg {
			t.Errorf("%v: message = %q, want %q", c.err, ce.Message, c.msg)
		}
	}
}

func TestValidationError_MessageIsTheText(t *testing.T) {
	if (&ValidationError{Msg: "name required"}).Error() != "name required" {
		t.Fatal("ValidationError.Error should be its message")
	}
}
