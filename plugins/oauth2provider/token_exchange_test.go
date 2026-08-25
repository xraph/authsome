package oauth2provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
)

// Named TestRFC8693_* rather than TestTokenExchange_*: authcode_test.go
// already owns TestTokenExchange_RejectsMismatchedRedirectURI and friends,
// which are about redeeming an authorization code. A shared prefix would make
// `go test -run TestTokenExchange` ambiguous between two unrelated features.

const (
	exchangeGrant   = "urn:ietf:params:oauth:grant-type:token-exchange"
	accessTokenType = "urn:ietf:params:oauth:token-type:access_token"
	xchgClientID    = "svc-exchange"
	xchgSecret      = "svc-exchange-secret"
)

// recordingEvents captures what the plugin writes.
type recordingEvents struct{ events []*securityevent.Event }

func (r *recordingEvents) RecordSecurityEvent(_ context.Context, e *securityevent.Event) error {
	r.events = append(r.events, e)
	return nil
}

func (r *recordingEvents) QuerySecurityEvents(_ context.Context, _ *securityevent.Query) ([]*securityevent.Event, string, error) {
	return r.events, "", nil
}

// exchangeEngine implements only the plugin.Engine methods this grant touches,
// plus ExchangeToken. Embedding the interface satisfies the type; anything the
// grant does not call stays nil and panics loudly if that ever changes.
type exchangeEngine struct {
	plugin.Engine
	core      store.Store
	events    *recordingEvents
	cfg       account.SessionConfig
	principal *principal.Principal

	// lastExchange records what the handler asked for, so a test can assert on
	// the translation from HTTP parameters into an ExchangeRequest.
	lastExchange *authsome.ExchangeRequest
	issued       *session.Session
	exchangeErr  error
}

func (e *exchangeEngine) Store() store.Store                  { return e.core }
func (e *exchangeEngine) Logger() log.Logger                  { return log.NewNoopLogger() }
func (e *exchangeEngine) SecurityEvents() securityevent.Store { return e.events }

// Nil is a valid answer here: OnInit reads it and falls back to a
// process-local limiter, which is what a test wants anyway.
func (e *exchangeEngine) RateLimiter() ratelimit.Limiter { return nil }

// Nil registry means SessionGuard and AdminGuard attach no middleware, which
// is how the other fixtures in this package register routes without standing
// up auth. The token endpoint is unauthenticated anyway; this grant
// authenticates the client itself.
func (e *exchangeEngine) AuthRegistry() auth.Registry { return nil }

// Nil checker: no route this grant uses is permission-gated.
func (e *exchangeEngine) HasPermission(_ context.Context, _ id.UserID, _, _ string) (bool, error) {
	return false, nil
}

func (e *exchangeEngine) ResolveSessionByToken(token string) (*session.Session, error) {
	return e.core.GetSessionByToken(context.Background(), token)
}

func (e *exchangeEngine) ResolvePrincipal(_ context.Context, _ principal.Ref) (*principal.Principal, error) {
	if e.principal == nil {
		return nil, principal.ErrNotFound
	}
	return e.principal, nil
}

func (e *exchangeEngine) ExchangeToken(_ context.Context, req *authsome.ExchangeRequest) (*session.Session, error) {
	e.lastExchange = req
	return e.issued, e.exchangeErr
}

type xchgFixture struct {
	oauth  oauth2provider.Store
	core   store.Store
	engine *exchangeEngine
	mux    forge.Router
	appID  id.AppID
	events *recordingEvents
}

// newExchangeFixture registers one confidential client holding scopes a and b,
// registered for the exchange grant and linked to a principal of kind agent.
func newExchangeFixture(t *testing.T) *xchgFixture {
	t.Helper()
	ctx := context.Background()

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	oauth := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(oauth)

	core := memory.New()
	events := &recordingEvents{}
	principalID := id.NewServiceAccountID()
	eng := &exchangeEngine{
		core:   core,
		events: events,
		cfg:    account.SessionConfig{TokenTTL: time.Hour, TokenExchangeTTL: 5 * time.Minute},
		// Kind agent on purpose: the handler must use the resolved kind, not
		// the probe kind it looks the principal up with.
		principal: &principal.Principal{
			Ref:    principal.Ref{Kind: principal.KindAgent, ID: principalID.String()},
			Scopes: []string{"a", "b"},
		},
	}
	require.NoError(t, p.OnInit(ctx, eng))

	hashed, err := bcrypt.GenerateFromPassword([]byte(xchgSecret), bcrypt.MinCost)
	require.NoError(t, err)

	appID := id.NewAppID()
	require.NoError(t, oauth.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     xchgClientID,
		ClientSecret: string(hashed),
		Name:         "Exchange client",
		Scopes:       []string{"a", "b"},
		GrantTypes:   []string{exchangeGrant},
		PrincipalID:  principalID,
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	return &xchgFixture{oauth: oauth, core: core, engine: eng, mux: mux, appID: appID, events: events}
}

func (f *xchgFixture) seedSubject(t *testing.T, appID id.AppID, scopes []string, jkt string) string {
	t.Helper()
	tok := "subject-" + id.NewSessionID().String()
	require.NoError(t, f.core.CreateSession(context.Background(), &session.Session{
		ID:        id.NewSessionID(),
		AppID:     appID,
		UserID:    id.NewUserID(),
		Token:     tok,
		Scopes:    scopes,
		DPoPJKT:   jkt,
		ExpiresAt: time.Now().Add(time.Hour),
	}))
	return tok
}

// grantSucceeds makes the stub engine return a plausible delegated session.
func (f *xchgFixture) grantSucceeds(scopes []string) {
	f.engine.issued = &session.Session{
		ID:         id.NewSessionID(),
		AppID:      f.appID,
		UserID:     id.NewUserID(),
		Token:      "issued-" + id.NewSessionID().String(),
		Scopes:     scopes,
		ActorGrant: principal.GrantDelegation,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	f.engine.exchangeErr = nil
}

func (f *xchgFixture) exchange(t *testing.T, extra map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"grant_type":    exchangeGrant,
		"client_id":     xchgClientID,
		"client_secret": xchgSecret,
	}
	for k, v := range extra {
		body[k] = v
	}
	return postToken(t, f.mux, body)
}

func TestRFC8693_ReturnsIssuedTokenTypeAndNoRefreshToken(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, "")

	body := decodeTokenResponseMap(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	assert.Equal(t, accessTokenType, body["issued_token_type"])
	assert.Equal(t, "a", body["scope"])
	assert.NotEmpty(t, body["access_token"])
	// Re-exchange rather than refresh, so the subject stays the only durable
	// credential.
	assert.Empty(t, body["refresh_token"])
}

// The actor must carry the principal's real kind. FindActiveDelegation
// compares the whole Ref by value, so a grant written against kind "agent"
// never matches a request built with the probe kind.
func TestRFC8693_UsesResolvedPrincipalKindNotProbeKind(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, "")

	decodeTokenResponseMap(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	require.NotNil(t, f.engine.lastExchange)
	assert.Equal(t, principal.KindAgent, f.engine.lastExchange.Actor.Kind,
		"actor must carry the resolved kind, not the workload probe kind")
	assert.Equal(t, f.appID, f.engine.lastExchange.AppID)
	assert.Equal(t, []string{"a"}, f.engine.lastExchange.Scopes)
	assert.Equal(t, principal.KindUser, f.engine.lastExchange.RequestedSubject.Kind)
}

// The engine refuses a chained exchange, but only if the caller chain reaches
// it. Leaving CallerActors empty would silently disarm that guard.
func TestRFC8693_ForwardsSubjectChainSoTheEngineGuardFires(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})

	tok := "subject-chained"
	delegationID := id.NewDelegationID()
	require.NoError(t, f.core.CreateSession(context.Background(), &session.Session{
		ID:           id.NewSessionID(),
		AppID:        f.appID,
		UserID:       id.NewUserID(),
		Token:        tok,
		Scopes:       []string{"a"},
		Actors:       principal.Chain{{Kind: principal.KindAgent, ID: "svc_prior"}},
		ActorGrant:   principal.GrantDelegation,
		DelegationID: delegationID,
		ExpiresAt:    time.Now().Add(time.Hour),
	}))

	f.exchange(t, map[string]string{
		"subject_token":      tok,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.NotNil(t, f.engine.lastExchange)
	require.Len(t, f.engine.lastExchange.CallerActors, 1)
	assert.Equal(t, "svc_prior", f.engine.lastExchange.CallerActors[0].ID)
	assert.Equal(t, delegationID, f.engine.lastExchange.CallerDelegationID)
}

// Exchanging a DPoP-bound token would hand back an unbound bearer token with
// the same authority, because the exchange path collects no proof.
func TestRFC8693_RefusesToUnbindADPoPBoundSubject(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})
	subject := f.seedSubject(t, f.appID, []string{"a"}, "some-jkt-thumbprint")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange, "must refuse before reaching the engine")
}

func TestRFC8693_RefusesScopeTheSubjectDoesNotHold(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"b"})
	subject := f.seedSubject(t, f.appID, []string{"a"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "b",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange, "a subject-side refusal must not reach the engine")
}

func TestRFC8693_RefusesScopeTheClientIsNotRegisteredFor(t *testing.T) {
	f := newExchangeFixture(t)
	subject := f.seedSubject(t, f.appID, []string{"a", "c"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "c", // held by the subject, not registered to the client
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange)
}

func TestRFC8693_RefusesEmptyScope(t *testing.T) {
	f := newExchangeFixture(t)
	subject := f.seedSubject(t, f.appID, []string{"a"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesCrossApp(t *testing.T) {
	f := newExchangeFixture(t)
	subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange)
}

func TestRFC8693_RefusesClientWithNoPrincipal(t *testing.T) {
	f := newExchangeFixture(t)
	hashed, err := bcrypt.GenerateFromPassword([]byte("s"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: f.appID, ClientID: "unlinked",
		ClientSecret: string(hashed), Name: "Unlinked", Scopes: []string{"a"},
		GrantTypes: []string{exchangeGrant}, // no PrincipalID
	}))
	subject := f.seedSubject(t, f.appID, []string{"a"}, "")

	rec := postToken(t, f.mux, map[string]string{
		"grant_type": exchangeGrant, "client_id": "unlinked", "client_secret": "s",
		"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "principal")
}

func TestRFC8693_RefusesUnsupportedSubjectTokenType(t *testing.T) {
	f := newExchangeFixture(t)
	subject := f.seedSubject(t, f.appID, []string{"a"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": "urn:ietf:params:oauth:token-type:refresh_token",
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported_token_type")
}

func TestRFC8693_SurfacesEngineRefusalAsInvalidGrant(t *testing.T) {
	f := newExchangeFixture(t)
	f.engine.exchangeErr = authsome.ErrExchangeRefused
	subject := f.seedSubject(t, f.appID, []string{"a"}, "")

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestRFC8693_DiscoveryAdvertisesTheGrant(t *testing.T) {
	_, _, mux := newFixture(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), exchangeGrant)
}
