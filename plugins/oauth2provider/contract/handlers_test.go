package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "oauth2provider/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "oauth2provider" {
		t.Errorf("contributor name = %q, want oauth2provider", m.Contributor.Name)
	}
	want := []string{
		"oauth2.clients.list", "oauth2.clients.detail", "oauth2.clients.create",
		"oauth2.clients.update", "oauth2.clients.delete", "oauth2.clients.rotateSecret",
		"oauth2.issuer.info", "oauth2.keys.list", "oauth2.keys.rotate",
	}
	got := map[string]bool{}
	for _, in := range m.Intents {
		got[in.Name] = true
	}
	if len(m.Intents) != len(want) {
		t.Errorf("intents = %d, want %d", len(m.Intents), len(want))
	}
	for _, n := range want {
		if !got[n] {
			t.Errorf("missing intent %q", n)
		}
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "oauth2provider/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

// fakeService answers from fields, so each test sets only what it needs. It
// records the app and inputs it was called with.
type fakeService struct {
	createErr        error
	created          Created
	rotateErr        error
	rotated          KeyInfo
	keys             []KeyInfo
	keysErr          error
	issuer           IssuerDetails
	issuerErr        error
	getClient        Client
	getErr           error
	listed           []Client
	listErr          error
	updated          Client
	updateErr        error
	deleteErr        error
	rotatedSecret    RotatedClient
	rotateSecretErr  error
	createCall       int
	deleteCall       int
	rotateSecretCall int
	lastApp          id.AppID
	lastID           string
	lastCreate       CreateParams
	lastUpdate       UpdateParams
}

func (f *fakeService) CreateClientFor(_ context.Context, appID id.AppID, in CreateParams) (Created, error) {
	f.createCall++
	f.lastApp, f.lastCreate = appID, in
	return f.created, f.createErr
}
func (f *fakeService) ListClientsFor(_ context.Context, appID id.AppID) ([]Client, error) {
	f.lastApp = appID
	return f.listed, f.listErr
}
func (f *fakeService) GetClientFor(_ context.Context, appID id.AppID, clientID string) (Client, error) {
	f.lastApp, f.lastID = appID, clientID
	return f.getClient, f.getErr
}
func (f *fakeService) UpdateClientFor(_ context.Context, appID id.AppID, clientID string, in UpdateParams) (Client, error) {
	f.lastApp, f.lastID, f.lastUpdate = appID, clientID, in
	return f.updated, f.updateErr
}
func (f *fakeService) DeleteClientFor(_ context.Context, appID id.AppID, clientID string) error {
	f.deleteCall++
	f.lastApp, f.lastID = appID, clientID
	return f.deleteErr
}
func (f *fakeService) RotateClientSecretFor(_ context.Context, appID id.AppID, clientID string) (RotatedClient, error) {
	f.rotateSecretCall++
	f.lastApp, f.lastID = appID, clientID
	return f.rotatedSecret, f.rotateSecretErr
}
func (f *fakeService) SigningKeys(context.Context, id.AppID) (string, []KeyInfo, error) {
	return "https://iss", f.keys, f.keysErr
}
func (f *fakeService) RotateSigningKey(context.Context, id.AppID) (KeyInfo, error) {
	return f.rotated, f.rotateErr
}
func (f *fakeService) IssuerInfo(context.Context, id.AppID) (IssuerDetails, error) {
	return f.issuer, f.issuerErr
}

func TestClientsList_UnavailableWhenPluginNil(t *testing.T) {
	h := clientsListHandler(Deps{})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func TestClientsCreate_RejectsBlankName(t *testing.T) {
	f := &fakeService{}
	h := clientsCreateHandler(Deps{Plugin: f})
	_, err := h(context.Background(), CreateClientInput{Name: "   "}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)
	if f.createCall != 0 {
		t.Fatalf("service was called %d times for a blank name", f.createCall)
	}
}

func TestClientsCreate_RefusesDelegated(t *testing.T) {
	ctx := middleware.WithSession(context.Background(), &session.Session{
		ID: id.NewSessionID(), UserID: id.NewUserID(),
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	})
	f := &fakeService{}
	_, err := clientsCreateHandler(Deps{Plugin: f})(ctx, CreateClientInput{Name: "x"}, contract.Principal{})
	expectCode(t, err, contract.CodePermissionDenied)
	if f.createCall != 0 {
		t.Fatal("service was called for a delegated session")
	}
}

func TestClientsCreate_MapsValidationAndCarriesSecret(t *testing.T) {
	h := clientsCreateHandler(Deps{Plugin: &fakeService{createErr: &ValidationError{Msg: "redirect_uris required"}}})
	_, err := h(context.Background(), CreateClientInput{Name: "x"}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)

	h = clientsCreateHandler(Deps{Plugin: &fakeService{created: Created{ID: "oa2c_1", ClientID: "pub", ClientSecret: "s3"}}})
	out, err := h(context.Background(), CreateClientInput{Name: "x"}, contract.Principal{})
	if err != nil || out.ID != "oa2c_1" || out.ClientID != "pub" || out.ClientSecret != "s3" {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
}

func TestClientsCreate_PublicClientOmitsSecret(t *testing.T) {
	h := clientsCreateHandler(Deps{Plugin: &fakeService{created: Created{ID: "oa2c_1", ClientID: "pub"}}})
	out, err := h(context.Background(), CreateClientInput{Name: "x", Public: true}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), "clientSecret") {
		t.Fatalf("public client response carries clientSecret: %s", b)
	}
}

func TestClientsDetail_NotFound(t *testing.T) {
	h := clientsDetailHandler(Deps{Plugin: &fakeService{getErr: ErrClientNotFound}})
	_, err := h(context.Background(), ClientIDInput{ID: "x"}, contract.Principal{})
	expectCode(t, err, contract.CodeNotFound)
}

func TestClientsDetail_Projects(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	h := clientsDetailHandler(Deps{Plugin: &fakeService{getClient: Client{
		ID: "oa2c_1", ClientID: "pub", Name: "n", CreatedAt: at, UpdatedAt: at, AuthMethod: "none",
	}}})
	out, err := h(context.Background(), ClientIDInput{ID: "oa2c_1"}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if out.CreatedAt != "2026-01-02T03:04:05Z" || out.AuthMethod != "none" || out.RedirectURIs == nil || out.Resources == nil {
		t.Fatalf("out = %+v", out)
	}
}

func TestKeysRotate_OperatorKeyIsBadRequest(t *testing.T) {
	h := keysRotateHandler(Deps{Plugin: &fakeService{rotateErr: ErrOperatorKey}})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)
}

func TestKeysRotate_SharedKeyIsBadRequest(t *testing.T) {
	h := keysRotateHandler(Deps{Plugin: &fakeService{rotateErr: ErrSharedKey}})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Message != "the deployment issuer's signing key is shared by every app and rotates on its own schedule" {
		t.Fatalf("message = %q", ce.Message)
	}
}

func TestKeysRotate_KeyTooNewIsBadRequest(t *testing.T) {
	h := keysRotateHandler(Deps{Plugin: &fakeService{rotateErr: ErrKeyTooNew}})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeBadRequest)
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Message != "the current signing key is less than an hour old; try again later" {
		t.Fatalf("message = %q", ce.Message)
	}
}

func TestKeysRotate_ReturnsPostRotationList(t *testing.T) {
	ret := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	f := &fakeService{rotated: KeyInfo{KID: "new"}, keys: []KeyInfo{
		{KID: "new", Alg: "RS256", Source: "managed", Status: "active", CreatedAt: ret},
		{KID: "old", Alg: "RS256", Source: "managed", Status: "retiring", CreatedAt: ret, RetiresAt: &ret},
	}}
	out, err := keysRotateHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil || len(out.Keys) != 2 || out.Issuer != "https://iss" || out.Keys[1].RetiresAt == "" || out.Keys[0].RetiresAt != "" {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
}

func TestKeysRotate_DegradedListKeepsIssuer(t *testing.T) {
	f := &fakeService{rotated: KeyInfo{KID: "new"}, keysErr: errors.New("boom"), issuer: IssuerDetails{Issuer: "https://iss2"}}
	out, err := keysRotateHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil || len(out.Keys) != 1 || out.Keys[0].KID != "new" || out.Issuer != "https://iss2" {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
	f.issuerErr = errors.New("boom")
	out, err = keysRotateHandler(Deps{Plugin: f})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil || out.Issuer != "" || len(out.Keys) != 1 {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
}

func TestKeys_NoSigningKeyIsUnavailable(t *testing.T) {
	h := keysListHandler(Deps{Plugin: &fakeService{keysErr: ErrNoSigningKey}})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	expectCode(t, err, contract.CodeUnavailable)
}

func TestMapErr_NeverEchoesInternalErrors(t *testing.T) {
	err := mapErr(errors.New("pq: password authentication failed for user x"))
	expectCode(t, err, contract.CodeInternal)
	var ce *contract.Error
	errors.As(err, &ce)
	if ce.Message != "internal error" {
		t.Fatalf("message = %q", ce.Message)
	}
}

func expectCode(t *testing.T, err error, want contract.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %q, got nil", want)
	}
	var ce *contract.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *contract.Error, got %T: %v", err, err)
	}
	if ce.Code != want {
		t.Fatalf("error code = %q, want %q (message=%q)", ce.Code, want, ce.Message)
	}
}
