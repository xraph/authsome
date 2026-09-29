package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// A session or JWT whose user no longer resolves (deleted, or banned, which
// the engine's resolver reports as an error) authenticates nobody. Before
// this the middleware set the session on the context and carried on, so a
// banned user kept every route that checks only for a session.

var errUserGone = errors.New("user is banned")

func serveWithRequireAuth(t *testing.T, mw forge.Middleware, token string) (int, bool) {
	t.Helper()
	var sawSession bool
	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	require.NoError(t, router.GET("/test", func(ctx forge.Context) error {
		_, sawSession = middleware.SessionFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)
	return rec.Code, sawSession
}

func humanSession() *session.Session {
	return &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: id.NewUserID(),
		Token: "tok", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestSessionAuth_UnresolvableUserIsRefused(t *testing.T) {
	sess := humanSession()
	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) { return sess, nil },
		func(_ string) (*user.User, error) { return nil, errUserGone },
		log.NewNoopLogger(),
	)
	code, sawSession := serveWithRequireAuth(t, mw, "tok")
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.False(t, sawSession, "the session must not reach the handler")
}

func TestSessionAuthWithStrategies_UnresolvableUserIsRefused(t *testing.T) {
	sess := humanSession()
	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) { return sess, nil },
		func(_ string) (*user.User, error) { return nil, errUserGone },
		nil,
		log.NewNoopLogger(),
	)
	code, _ := serveWithRequireAuth(t, mw, "tok")
	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestSessionAuth_MachineSessionNeedsNoUser(t *testing.T) {
	sess := humanSession()
	sess.UserID = id.Nil
	sess.PrincipalKind = principal.KindService
	sess.ServiceAccountID = id.NewServiceAccountID()
	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) { return sess, nil },
		func(_ string) (*user.User, error) { t.Fatal("no user lookup for a machine session"); return nil, nil },
		log.NewNoopLogger(),
	)
	code, sawSession := serveWithRequireAuth(t, mw, "tok")
	assert.Equal(t, http.StatusOK, code)
	assert.True(t, sawSession)
}

func TestJWTAuth_UnresolvableUserIsRefused(t *testing.T) {
	uid := id.NewUserID()
	validator := &mockJWTValidator{claims: &tokenformat.TokenClaims{
		UserID: uid.String(), AppID: id.NewAppID().String(),
	}}
	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) { return nil, errors.New("not a session") },
		func(_ string) (*user.User, error) { return nil, errUserGone },
		nil,
		validator,
		log.NewNoopLogger(),
	)
	code, _ := serveWithRequireAuth(t, mw, "a.b.c")
	assert.Equal(t, http.StatusUnauthorized, code, "a verified signature is not enough once the subject is gone")
}
