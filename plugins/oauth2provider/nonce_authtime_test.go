package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/store/storetest"
)

// sessionCtx is a signed-in human session created at createdAt.
func sessionCtx(userID id.UserID, appID id.AppID, createdAt time.Time) context.Context {
	sess := &session.Session{ID: id.NewSessionID(), AppID: appID, UserID: userID, CreatedAt: createdAt}
	ctx := middleware.WithSession(context.Background(), sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	ctx = middleware.WithAppID(ctx, appID)
	return middleware.WithUserID(ctx, userID)
}

func authorizeWith(ctx context.Context, mux http.Handler, q url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAuthorize_NonceRoundTripsToCode(t *testing.T) {
	_, st, mux := newFixture(t)
	q := baseAuthorizeQuery(confidentialID)
	q.Set("nonce", "n-123")

	code := codeFrom(t, authorize(t, mux, q))

	ac, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, "n-123", ac.Nonce)
}

func TestAuthorize_NoNonceLeavesItEmpty(t *testing.T) {
	_, st, mux := newFixture(t)
	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))
	ac, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Empty(t, ac.Nonce)
}

func TestAuthorize_NonceTooLong(t *testing.T) {
	p, st, mux := newFixture(t)
	counting := &countingAuthCodeStore{Store: st}
	p.SetOAuth2Store(counting)

	q := baseAuthorizeQuery(confidentialID)
	q.Set("nonce", strings.Repeat("a", 513))
	rec := authorize(t, mux, q)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_request", oauthErrorCode(t, rec))
	assert.Equal(t, 0, counting.authCodesCreated, "no code may be stored for a refused request")

	q.Set("nonce", strings.Repeat("a", 512))
	assert.Equal(t, http.StatusFound, authorize(t, mux, q).Code, "512 bytes is the limit, not over it")
}

func TestAuthorize_NonceTooLongBeforeConsentRecord(t *testing.T) {
	_, mux, _ := newConsentFixture(t)
	q := consentQuery("openid profile")
	q.Set("nonce", strings.Repeat("a", 513))
	rec := authorizeAs(t, mux, id.NewUserID(), q)
	require.Equal(t, http.StatusBadRequest, rec.Code, "no consent redirect for a refused request; body: %s", rec.Body.String())
}

func TestAuthorize_AuthTimeIsSessionCreatedAt(t *testing.T) {
	_, st, mux := newFixture(t)
	client, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	loggedIn := time.Now().Add(-time.Hour).Truncate(time.Second)

	ctx := sessionCtx(id.NewUserID(), client.AppID, loggedIn)
	code := codeFrom(t, authorizeWith(ctx, mux, baseAuthorizeQuery(confidentialID)))

	ac, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.True(t, ac.AuthTime.Equal(loggedIn), "auth_time %v, want session created_at %v", ac.AuthTime, loggedIn)
}

func TestAuthorize_AuthTimeWithoutSessionIsNow(t *testing.T) {
	_, st, mux := newFixture(t)
	before := time.Now().Add(-time.Second)
	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))
	ac, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), ac.AuthTime, 5*time.Second)
	assert.True(t, ac.AuthTime.After(before))
}

// The nonce and auth_time belong to the /authorize request, so the consent
// round trip has to carry them, and auth_time stays the login time rather than
// the moment the user clicked approve.
func TestConsent_CarriesNonceAndAuthTimeToCode(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()
	loggedIn := time.Now().Add(-90 * time.Minute).Truncate(time.Second)
	ctx := sessionCtx(user, appID, loggedIn)

	q := consentQuery("openid profile")
	q.Set("nonce", "n-consent")
	cid := consentIDFrom(t, authorizeWith(ctx, mux, q))

	page := getConsent(t, mux, user, cid, "")
	require.Equal(t, http.StatusOK, page.Code, page.Body.String())
	csrf := csrfFromPage(t, page.Body.String())

	approved := decide(t, mux, user, url.Values{"cid": {cid}, "csrf": {csrf}, "decision": {"approve"}})
	code := codeFrom(t, approved)

	ac, err := st.GetAuthCode(context.Background(), code)
	require.NoError(t, err)
	assert.Equal(t, "n-consent", ac.Nonce)
	assert.True(t, ac.AuthTime.Equal(loggedIn), "auth_time %v, want %v", ac.AuthTime, loggedIn)
}

func TestDeviceComplete_RecordsAuthTime(t *testing.T) {
	p, st, mux := newFixture(t)
	_ = p
	const deviceClientID = "device-authtime-client"
	appID := id.NewAppID()
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: deviceClientID, FirstParty: true,
		Name: "Device", Scopes: []string{"openid"}, GrantTypes: []string{"device_code"},
	}))
	auth := deviceAuthorize(t, mux, deviceClientID)
	loggedIn := time.Now().Add(-3 * time.Hour).Truncate(time.Second)

	body, err := json.Marshal(map[string]string{"user_code": auth.UserCode, "action": "approve"})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(sessionCtx(id.NewUserID(), appID, loggedIn), http.MethodPost,
		"/v1/oauth/device/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), auth.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, oauth2provider.DeviceCodeStatusAuthorized, dc.Status)
	assert.True(t, dc.AuthTime.Equal(loggedIn), "auth_time %v, want %v", dc.AuthTime, loggedIn)
}

// A code minted before nonce and auth_time existed has neither. It must still
// redeem; id_token minting treats the empty nonce and zero time as absent.
func TestAuthCode_LegacyRowWithoutNonceRedeems(t *testing.T) {
	_, st, mux := newFixture(t)
	client, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	require.NoError(t, st.CreateAuthCode(context.Background(), &oauth2provider.AuthorizationCode{
		ID: id.NewAuthCodeID(), Code: "legacy-code-1", ClientID: confidentialID, UserID: fixtureUserID, AppID: client.AppID,
		RedirectURI: registeredURI, Scopes: []string{"openid"},
		ExpiresAt: time.Now().Add(time.Minute), CreatedAt: time.Now(),
	}))

	rec := postToken(t, mux, map[string]string{
		"grant_type": "authorization_code", "code": "legacy-code-1", "client_id": confidentialID,
		"client_secret": confidentialSecret, "redirect_uri": registeredURI,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// The SQL half of the same property: a row whose nonce and auth_time columns
// are NULL (every row written before the migration) scans to the empty nonce
// and the zero time instead of erroring.
func TestAuthCode_LegacySQLiteRowWithNullColumns(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "legacy.db") + "?cache=shared&_pragma=foreign_keys(1)"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	core := sqlitestore.New(db)
	require.NoError(t, core.Migrate(ctx, oauth2provider.SqliteMigrations))
	tn := storetest.SeedTenant(t, core)
	userID := storetest.SeedUser(t, core, tn, "legacy-null@example.test")
	st := oauth2provider.NewSqliteStore(db)

	require.NoError(t, st.CreateAuthCode(ctx, &oauth2provider.AuthorizationCode{
		ID: id.NewAuthCodeID(), Code: "legacy-null", ClientID: "c", UserID: userID, AppID: tn.AppID,
		RedirectURI: "https://x.test/cb", Scopes: []string{"openid"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}))
	// What a row written before the migration looks like.
	_, err = sdb.NewRaw(`UPDATE authsome_oauth2_auth_codes SET nonce = NULL, auth_time = NULL`).Exec(ctx)
	require.NoError(t, err)

	got, err := st.GetAuthCode(ctx, "legacy-null")
	require.NoError(t, err)
	assert.Empty(t, got.Nonce)
	assert.True(t, got.AuthTime.IsZero())
	ok, err := st.ConsumeAuthCode(ctx, "legacy-null")
	require.NoError(t, err)
	assert.True(t, ok)
}
