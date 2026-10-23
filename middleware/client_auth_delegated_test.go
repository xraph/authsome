package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// A client-mode service learns who holds a token from introspection and puts
// no session on the context, so DenyDelegatedSession and RefuseDelegated saw
// nothing there and let an agent token through as its user. The delegated
// flag introspection reports now reaches both.
func TestClientAuth_DelegatedTokenIsRefusedByTheGuard(t *testing.T) {
	cases := map[string]struct {
		kind      string
		delegated bool
		want      int
	}{
		"user's own":                       {"user", false, http.StatusOK},
		"older identity server":            {"", false, http.StatusOK},
		"agent":                            {"agent", true, http.StatusForbidden},
		"token exchange":                   {"user", true, http.StatusForbidden},
		"service account acting as itself": {"service_account", false, http.StatusOK},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			userID := id.NewUserID().String()
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(authclient.IntrospectResponse{
					Active: true, UserID: userID, AppID: id.NewAppID().String(), SessionID: id.NewSessionID().String(),
					User:          &authclient.IntrospectUser{ID: userID, Email: "client@test.com"},
					PrincipalKind: tc.kind, Delegated: tc.delegated,
				})
			}))
			t.Cleanup(identity.Close)

			minted, refused := false, false
			router := forge.NewRouter()
			router.Use(middleware.ClientAuthMiddleware(authclient.NewClient(identity.URL), log.NewNoopLogger()))
			require.NoError(t, router.POST("/guarded", func(ctx forge.Context) error {
				minted = true
				return ctx.NoContent(http.StatusOK)
			}, forge.WithMiddleware(middleware.DenyDelegatedSession())))
			require.NoError(t, router.POST("/handler-checks", func(ctx forge.Context) error {
				if errors.Is(middleware.RefuseDelegated(ctx.Context()), middleware.ErrDelegatedSession) {
					refused = true
					return forge.Forbidden(middleware.DelegatedSessionMessage)
				}
				return ctx.NoContent(http.StatusOK)
			}))

			for _, path := range []string{"/guarded", "/handler-checks"} {
				req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, nil)
				req.Header.Set("Authorization", "Bearer client-token")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				assert.Equal(t, tc.want, rec.Code, "%s body=%s", path, rec.Body.String())
			}
			assert.Equal(t, tc.want == http.StatusOK, minted, "the guarded handler runs only for a token that isn't delegated")
			assert.Equal(t, tc.want != http.StatusOK, refused)
		})
	}
}

// A JWT carrying an RFC 8693 act claim is a delegated token for its sub. The
// JWT path puts no session on the context either, so the claim itself has to
// reach the guard.
func TestJWTAuth_ActClaimIsRefusedByTheGuard(t *testing.T) {
	userID, appID := id.NewUserID(), id.NewAppID()
	for name, act := range map[string]*tokenformat.ActClaim{
		"no act claim": nil,
		"act claim":    {Subject: id.NewServiceAccountID().String()},
	} {
		t.Run(name, func(t *testing.T) {
			mw := middleware.AuthMiddlewareWithJWT(
				func(context.Context, string) (*session.Session, error) { return nil, errors.New("not found") },
				func(context.Context, string) (*user.User, error) {
					return &user.User{ID: userID, AppID: appID, Email: "jwt@test.com"}, nil
				},
				&mockStrategyAuth{},
				&mockJWTValidator{claims: &tokenformat.TokenClaims{
					UserID: userID.String(), AppID: appID.String(), SessionID: id.NewSessionID().String(), Act: act,
				}},
				log.NewNoopLogger(),
			)
			router := forge.NewRouter()
			router.Use(mw)
			require.NoError(t, router.POST("/guarded", func(ctx forge.Context) error {
				return ctx.NoContent(http.StatusOK)
			}, forge.WithMiddleware(middleware.DenyDelegatedSession())))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/guarded", nil)
			req.Header.Set("Authorization", "Bearer a.b.c")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			want := http.StatusOK
			if act != nil {
				want = http.StatusForbidden
			}
			assert.Equal(t, want, rec.Code, "body=%s", rec.Body.String())
		})
	}
}

// An identity server that predates principal_kind sends neither field, and
// the client then can't tell a delegated token from the user's own. That
// fails open, so the middleware says so in the log, once, rather than on
// every request.
func TestClientAuth_WarnsOnceWhenIntrospectionHasNoPrincipalKind(t *testing.T) {
	for name, kind := range map[string]string{"older identity server": "", "current identity server": "user"} {
		t.Run(name, func(t *testing.T) {
			tl, ok := log.NewTestLogger().(*log.TestLogger)
			require.True(t, ok)
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(authclient.IntrospectResponse{
					Active: true, UserID: id.NewUserID().String(), AppID: id.NewAppID().String(),
					SessionID: id.NewSessionID().String(), PrincipalKind: kind,
				})
			}))
			t.Cleanup(identity.Close)

			router := forge.NewRouter()
			router.Use(middleware.ClientAuthMiddleware(authclient.NewClient(identity.URL), tl))
			require.NoError(t, router.GET("/x", func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }))
			for _, token := range []string{"t1", "t2", "t3"} {
				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				router.ServeHTTP(httptest.NewRecorder(), req)
			}

			warned := 0
			for _, l := range tl.GetLogsByLevel("WARN") {
				if strings.Contains(l.Message, "principal_kind") {
					warned++
				}
			}
			if kind == "" {
				assert.Equal(t, 1, warned, "warned once, not per request")
			} else {
				assert.Zero(t, warned)
			}
		})
	}
}
