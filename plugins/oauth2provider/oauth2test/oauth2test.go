// Package oauth2test provides a backend-agnostic conformance suite for the
// oauth2provider.Store interface. Every backend (memory, sqlite, postgres,
// mongo) is expected to pass the same contract, so behavioral drift between
// implementations is caught here rather than in production.
//
// It mirrors store/storetest, which does the same job for the core store.
package oauth2test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// Fixture is one backend ready to test: a migrated plugin store plus the
// tenant rows its foreign keys point at. The SQL backends reference
// authsome_apps and authsome_users, so a suite cannot invent those ids.
type Fixture struct {
	Store oauth2provider.Store
	AppID id.AppID
	// UserID must exist in authsome_users for backends that enforce the
	// auth-code foreign key.
	UserID id.UserID
	// OtherAppID is a second tenant, used to prove app scoping. Backends that
	// cannot seed one may leave it zero and the scoping case is skipped.
	OtherAppID id.AppID
}

// OtherUserID is a user id that owns nothing in the fixture, for proving
// per-user scoping without a second seeded user.
func (f Fixture) OtherUserID() id.UserID { return id.NewUserID() }

// Factory builds a fresh, empty, migrated fixture for a single test.
type Factory func(t *testing.T) Fixture

// RunConformance runs every contract test against fixtures from newFixture.
// Names in skip are not run, for backends that legitimately cannot support a
// case; skipping anything else is a bug in the backend, not in the suite.
func RunConformance(t *testing.T, newFixture Factory, skip ...string) {
	t.Helper()
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}
	cases := []struct {
		name string
		fn   func(t *testing.T, f Fixture)
	}{
		{"ClientCRUD", testClientCRUD},
		{"ClientNotFound", testClientNotFound},
		{"ClientListIsAppScoped", testClientListIsAppScoped},
		{"ClientEmptySlicesRoundTrip", testClientEmptySlicesRoundTrip},
		{"ClientSlicesRoundTrip", testClientSlicesRoundTrip},
		{"UpdateClientPersistsEveryField", testUpdateClientPersistsEveryField},
		{"UpdateClientOnMissingClient", testUpdateClientOnMissingClient},
		{"UnlinkedClientHasNilPrincipal", testUnlinkedClientHasNilPrincipal},
		{"AuthCodeRoundTrip", testAuthCodeRoundTrip},
		{"ConsumeAuthCodeOnce", testConsumeAuthCodeOnce},
		{"ConsumeAuthCodeReplay", testConsumeAuthCodeReplay},
		{"ConsumeAuthCodeIsAtomic", testConsumeAuthCodeIsAtomic},
		{"ConsumeAuthCodeUnknown", testConsumeAuthCodeUnknown},
		{"AuthCodes/NonceAndAuthTimeRoundTrip", testAuthCodeNonceAndAuthTimeRoundTrip},
		{"DeviceCodes/AuthTimeRoundTrip", testDeviceCodeAuthTimeRoundTrip},
		{"DeviceCodeRoundTrip", testDeviceCodeRoundTrip},
		{"DeviceCodeUpdate", testDeviceCodeUpdate},
		{"DeleteExpiredDeviceCodes", testDeleteExpiredDeviceCodes},
		{"DeleteExpiredAuthCodes", testDeleteExpiredAuthCodes},
		{"DeleteExpiredDeviceCodesBefore", testDeleteExpiredDeviceCodesBefore},
		{"AuthCodeStoredAsHash", testAuthCodeStoredAsHash},
		{"DeviceCodeStoredAsHash", testDeviceCodeStoredAsHash},
		{"GrantLifecycle", testGrantLifecycle},
		{"ClientFirstPartyRoundTrip", testClientFirstPartyRoundTrip},
		{"SigningKeys/CreateAndList", testSigningKeysCreateAndList},
		{"SigningKeys/DuplicateGenerationConflicts", testSigningKeysDuplicateGenerationConflicts},
		{"SigningKeys/ScopesAreIsolated", testSigningKeysScopesAreIsolated},
		{"SigningKeys/Retire", testSigningKeysRetire},
		{"SigningKeys/RetireUnknownIsNoop", testSigningKeysRetireUnknownIsNoop},
		{"SigningKeys/ConcurrentCreateOneWins", testSigningKeysConcurrentCreateOneWins},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// unique returns a per-call unique string so sub-tests sharing one backend
// (mongo and postgres reuse a single container) never collide.
func unique(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, id.NewOAuth2ClientID().String())
}

func newClient(appID id.AppID) *oauth2provider.OAuth2Client {
	return &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		Name:         "Test Client",
		ClientID:     unique("cid"),
		ClientSecret: "hashed-secret",
		RedirectURIs: []string{"https://example.test/cb"},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code"},
		Resources:    []string{"https://api.example.test"},
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}
}

// ──────────────────────────────────────────────────
// Signing keys
// ──────────────────────────────────────────────────

func newSigningKey(scope string, generation int64) *oauth2provider.SigningKey {
	return &oauth2provider.SigningKey{
		ID:         id.NewOAuth2SigningKeyID(),
		Scope:      scope,
		Alg:        "RS256",
		PrivateKey: "sealed-private-key",
		PublicJWK:  []byte(`{"kty":"RSA","kid":"k","alg":"RS256","use":"sig"}`),
		Generation: generation,
		CreatedAt:  now(),
	}
}

// sameJSON compares two JSON documents by value. Postgres JSONB normalizes
// whitespace and key order, so a byte comparison would fail there for a
// document that round-tripped faithfully.
func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("invalid JSON %q: %v", a, err)
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("invalid JSON %q: %v", b, err)
	}
	return reflect.DeepEqual(av, bv)
}

func testSigningKeysCreateAndList(t *testing.T, f Fixture) {
	ctx := context.Background()
	scope := unique("scope")
	older, newer := newSigningKey(scope, 1), newSigningKey(scope, 2)
	// Insert out of order so the result can only be ordered by the store.
	for _, k := range []*oauth2provider.SigningKey{newer, older} {
		if err := f.Store.CreateSigningKey(ctx, k); err != nil {
			t.Fatalf("CreateSigningKey gen %d: %v", k.Generation, err)
		}
	}

	got, err := f.Store.ListSigningKeys(ctx, scope)
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if len(got) != 2 || got[0].Generation != 2 || got[1].Generation != 1 {
		t.Fatalf("want generations [2 1], got %+v", got)
	}
	k := got[0]
	if k.ID != newer.ID || k.Scope != scope || k.Alg != "RS256" ||
		k.PrivateKey != newer.PrivateKey || !sameJSON(t, k.PublicJWK, newer.PublicJWK) {
		t.Fatalf("round trip mismatch: got %+v want %+v", k, newer)
	}
	if !k.CreatedAt.Equal(newer.CreatedAt) {
		t.Fatalf("CreatedAt: got %v want %v", k.CreatedAt, newer.CreatedAt)
	}
	if k.RetiresAt != nil {
		t.Fatalf("a new key must not be retired, got %v", k.RetiresAt)
	}
}

func testSigningKeysDuplicateGenerationConflicts(t *testing.T, f Fixture) {
	ctx := context.Background()
	scope := unique("scope")
	if err := f.Store.CreateSigningKey(ctx, newSigningKey(scope, 1)); err != nil {
		t.Fatalf("CreateSigningKey: %v", err)
	}
	err := f.Store.CreateSigningKey(ctx, newSigningKey(scope, 1))
	if !errors.Is(err, oauth2provider.ErrSigningKeyConflict) {
		t.Fatalf("duplicate generation: want ErrSigningKeyConflict, got %v", err)
	}
	got, err := f.Store.ListSigningKeys(ctx, scope)
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("the loser must not be stored, got %d keys", len(got))
	}
}

func testSigningKeysScopesAreIsolated(t *testing.T, f Fixture) {
	ctx := context.Background()
	a, b := unique("scope"), unique("scope")
	// The same generation in two scopes is fine: uniqueness is per scope.
	for _, scope := range []string{a, b, ""} {
		if err := f.Store.CreateSigningKey(ctx, newSigningKey(scope, 1)); err != nil {
			t.Fatalf("CreateSigningKey scope %q: %v", scope, err)
		}
	}
	for _, scope := range []string{a, b} {
		got, err := f.Store.ListSigningKeys(ctx, scope)
		if err != nil {
			t.Fatalf("ListSigningKeys: %v", err)
		}
		if len(got) != 1 || got[0].Scope != scope {
			t.Fatalf("scope %q: want its own single key, got %+v", scope, got)
		}
	}
	got, err := f.Store.ListSigningKeys(ctx, unique("empty"))
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unknown scope must list nothing, got %d", len(got))
	}
}

func testSigningKeysRetire(t *testing.T, f Fixture) {
	ctx := context.Background()
	scope := unique("scope")
	retired, active := newSigningKey(scope, 1), newSigningKey(scope, 2)
	for _, k := range []*oauth2provider.SigningKey{retired, active} {
		if err := f.Store.CreateSigningKey(ctx, k); err != nil {
			t.Fatalf("CreateSigningKey: %v", err)
		}
	}
	at := now().Add(time.Hour)
	if err := f.Store.RetireSigningKey(ctx, retired.ID, at); err != nil {
		t.Fatalf("RetireSigningKey: %v", err)
	}
	got, err := f.Store.ListSigningKeys(ctx, scope)
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("retiring must not delete, got %d keys", len(got))
	}
	if got[0].RetiresAt != nil {
		t.Fatalf("the other key must stay active, got %v", got[0].RetiresAt)
	}
	if got[1].RetiresAt == nil || !got[1].RetiresAt.Equal(at) {
		t.Fatalf("RetiresAt: got %v want %v", got[1].RetiresAt, at)
	}

	// Retiring again overwrites the earlier time.
	later := at.Add(time.Hour)
	if err = f.Store.RetireSigningKey(ctx, retired.ID, later); err != nil {
		t.Fatalf("RetireSigningKey again: %v", err)
	}
	got, err = f.Store.ListSigningKeys(ctx, scope)
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if got[1].RetiresAt == nil || !got[1].RetiresAt.Equal(later) {
		t.Fatalf("re-retire must overwrite RetiresAt: got %v want %v", got[1].RetiresAt, later)
	}
}

func testSigningKeysRetireUnknownIsNoop(t *testing.T, f Fixture) {
	if err := f.Store.RetireSigningKey(context.Background(), id.NewOAuth2SigningKeyID(), now()); err != nil {
		t.Fatalf("retiring an unknown key must be a no-op, got %v", err)
	}
}

func testSigningKeysConcurrentCreateOneWins(t *testing.T, f Fixture) {
	ctx := context.Background()
	scope := unique("scope")
	const writers = 8
	errs := make([]error, writers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = f.Store.CreateSigningKey(ctx, newSigningKey(scope, 1))
		}()
	}
	close(start)
	wg.Wait()

	wins := 0
	for _, err := range errs {
		switch {
		case err == nil:
			wins++
		case !errors.Is(err, oauth2provider.ErrSigningKeyConflict):
			t.Fatalf("loser must get ErrSigningKeyConflict, got %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("exactly one writer must win, got %d", wins)
	}
	got, err := f.Store.ListSigningKeys(ctx, scope)
	if err != nil {
		t.Fatalf("ListSigningKeys: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want one stored key, got %d", len(got))
	}
}
