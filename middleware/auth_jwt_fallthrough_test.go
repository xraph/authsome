package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// A JWT that is refused below the signature check must not be rescued by the
// opaque session lookup.
//
// Access tokens are stored as sess.Token, so the same string that failed as a
// JWT is a live key into the session store. If a refusal falls through, the
// second attempt re-runs the checks against the session ROW instead of the
// CLAIMS: sess.Audience instead of claims.Audience, sess.AppID instead of
// claims.AppID, and the JWT session cross-check not at all. The request is
// then authenticated by the very credential that was just denied.
//
// Both resolutions of this compile and pass the surrounding tests, which is
// why the property needs pinning here rather than reviewing. These tests drive
// the exported middleware and assert on the response, so they survive whatever
// signature tryJWTAuth ends up with.

// jwtFallthroughToken carries two dots so tokenformat.IsJWT routes it down the
// JWT path. It is also the session row's Token, which is the whole point.
const jwtFallthroughToken = "header.payload.signature"

const (
	jwtFallthroughClientIP = "203.0.113.7"
	jwtFallthroughDeviceUA = "enrolled-device/1.0"
)

type jwtFallthroughValidator struct {
	claims *tokenformat.TokenClaims
	err    error
}

func (v *jwtFallthroughValidator) ValidateJWT(_ string) (*tokenformat.TokenClaims, error) {
	return v.claims, v.err
}

type jwtFallthroughCase struct {
	name string

	// claims is what the JWT carries. Each case makes exactly one check refuse.
	claims *tokenformat.TokenClaims

	// rescue is the live session row keyed by jwtFallthroughToken. It is built
	// to PASS the check the claims fail, so a fallthrough authenticates.
	rescue *session.Session

	bind middleware.SessionBindingConfig

	// requestAppID, when set, goes on the context the way the publishable-key
	// middleware would put it there.
	requestAppID id.AppID

	remoteAddr string
	userAgent  string
}

func jwtFallthroughRescue(appID id.AppID, userID id.UserID) *session.Session {
	return &session.Session{
		ID:        id.NewSessionID(),
		AppID:     appID,
		UserID:    userID,
		Token:     jwtFallthroughToken,
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

// jwtFallthroughCases returns one case per reason a JWT is refused past the
// signature check. Keeping them separate means a partial regression cannot
// hide behind a passing sibling.
func jwtFallthroughCases() []jwtFallthroughCase {
	var cases []jwtFallthroughCase

	// 1. Audience (RFC 8707). The claims name a resource this deployment does
	//    not answer to; the session row names one it does.
	{
		appID, userID := id.NewAppID(), id.NewUserID()
		rescue := jwtFallthroughRescue(appID, userID)
		rescue.Audience = []string{"https://api.example.com"}
		cases = append(cases, jwtFallthroughCase{
			name: "audience",
			claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: id.NewSessionID().String(),
				Audience:  []string{"https://elsewhere.example.com"},
			},
			rescue: rescue,
			bind: middleware.SessionBindingConfig{
				ExpectedAudienceResolver: func(_ context.Context, _ string) []string {
					return []string{"https://api.example.com"}
				},
			},
		})
	}

	// 2. Publishable-key app match. The JWT was minted under one app and the
	//    caller presents another app's publishable key. The session row is
	//    bound to the app the key resolves to, so it passes the same check.
	{
		requestAppID, tokenAppID := id.NewAppID(), id.NewAppID()
		userID := id.NewUserID()
		cases = append(cases, jwtFallthroughCase{
			name: "publishable key app mismatch",
			claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     tokenAppID.String(),
				SessionID: id.NewSessionID().String(),
			},
			rescue:       jwtFallthroughRescue(requestAppID, userID),
			requestAppID: requestAppID,
		})
	}

	// 3. Revoked session. The sid in the claims is gone from the store. The
	//    opaque lookup is keyed by token, not sid, so it still finds a row.
	{
		appID, userID := id.NewAppID(), id.NewUserID()
		cases = append(cases, jwtFallthroughCase{
			name: "revoked session lookup",
			claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: id.NewSessionID().String(),
			},
			rescue: jwtFallthroughRescue(appID, userID),
			bind: middleware.SessionBindingConfig{
				JWTSessionChecker: func(_ string) (*session.Session, error) {
					return nil, errors.New("session revoked")
				},
			},
		})
	}

	// 4. IP binding. The session the sid points at was created from another
	//    IP. The row the token resolves to carries the caller's own IP.
	{
		appID, userID := id.NewAppID(), id.NewUserID()
		rescue := jwtFallthroughRescue(appID, userID)
		rescue.IPAddress = jwtFallthroughClientIP
		bound := &session.Session{
			ID:        id.NewSessionID(),
			AppID:     appID,
			UserID:    userID,
			IPAddress: "198.51.100.9",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		cases = append(cases, jwtFallthroughCase{
			name: "ip binding",
			claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: id.NewSessionID().String(),
			},
			rescue: rescue,
			bind: middleware.SessionBindingConfig{
				BindToIP: true,
				JWTSessionChecker: func(_ string) (*session.Session, error) {
					return bound, nil
				},
			},
			remoteAddr: jwtFallthroughClientIP + ":41234",
		})
	}

	// 5. Device binding. Same shape as IP, on User-Agent.
	{
		appID, userID := id.NewAppID(), id.NewUserID()
		rescue := jwtFallthroughRescue(appID, userID)
		rescue.UserAgent = jwtFallthroughDeviceUA
		bound := &session.Session{
			ID:        id.NewSessionID(),
			AppID:     appID,
			UserID:    userID,
			UserAgent: "some-other-device/9.9",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		cases = append(cases, jwtFallthroughCase{
			name: "device binding",
			claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: id.NewSessionID().String(),
			},
			rescue: rescue,
			bind: middleware.SessionBindingConfig{
				BindToDevice: true,
				JWTSessionChecker: func(_ string) (*session.Session, error) {
					return bound, nil
				},
			},
			userAgent: jwtFallthroughDeviceUA,
		})
	}

	return cases
}

// run drives the exported middleware end to end and hands back the response.
// Nothing here touches tryJWTAuth or its return type.
func (tc jwtFallthroughCase) run(t *testing.T, validator middleware.JWTValidator) *httptest.ResponseRecorder {
	t.Helper()

	resolveSession := func(token string) (*session.Session, error) {
		if token == jwtFallthroughToken {
			return tc.rescue, nil
		}
		return nil, errors.New("no session for token")
	}
	resolveUser := func(userID string) (*user.User, error) {
		if userID == tc.rescue.UserID.String() {
			return &user.User{ID: tc.rescue.UserID, AppID: tc.rescue.AppID, Email: "fallthrough@test.com"}, nil
		}
		return nil, errors.New("no such user")
	}

	router := forge.NewRouter()
	if !tc.requestAppID.IsNil() {
		appID := tc.requestAppID
		router.Use(func(next forge.Handler) forge.Handler {
			return func(ctx forge.Context) error {
				ctx.WithContext(middleware.WithAppID(ctx.Context(), appID))
				return next(ctx)
			}
		})
	}
	router.Use(middleware.AuthMiddlewareWithJWT(
		resolveSession, resolveUser, nil, validator, log.NewNoopLogger(), tc.bind,
	))
	router.Use(middleware.RequireAuth())
	router.GET("/probe", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+jwtFallthroughToken)
	if tc.remoteAddr != "" {
		req.RemoteAddr = tc.remoteAddr
	}
	if tc.userAgent != "" {
		req.Header.Set("User-Agent", tc.userAgent)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestJWTRefusal_DoesNotFallThroughToSessionLookup is the regression. A JWT
// refused for any reason below the signature must leave the request
// unauthenticated, not merely without a JWT identity.
// A related gap this file does NOT cover, recorded here because this is where
// somebody will look. tryJWTAuth populates AppID, UserID and SessionID but
// never calls WithSession, so no *session.Session reaches the context on the
// JWT path. Guards that read the session rather than the ids depend entirely
// on the global AuthMiddleware having populated it first. agentauth's
// denyAgentPrincipal is one such guard: it is correct today only because
// nothing issues an agent a JWT credential. The day something does, that deny
// fails open, and no test here will say so.
func TestJWTRefusal_DoesNotFallThroughToSessionLookup(t *testing.T) {
	t.Parallel()
	for _, tc := range jwtFallthroughCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := tc.run(t, &jwtFallthroughValidator{claims: tc.claims})
			require.Equal(t, http.StatusUnauthorized, rec.Code,
				"a JWT refused on %s must end unauthenticated; the opaque session "+
					"lookup rescued it and checked the session row instead of the claims", tc.name)
		})
	}
}

// TestJWTRefusal_GuardSessionRowWouldHaveAuthenticated keeps the test above
// honest. Same rows, same config, but the validator does not recognise the
// token, so the fallthrough is legitimate and must succeed. Without this, a
// middleware that authenticated nothing at all would pass the regression test.
func TestJWTRefusal_GuardSessionRowWouldHaveAuthenticated(t *testing.T) {
	t.Parallel()
	for _, tc := range jwtFallthroughCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := tc.run(t, &jwtFallthroughValidator{err: errors.New("not a token this deployment issued")})
			require.Equal(t, http.StatusOK, rec.Code,
				"harness check: the %s session row must be one the opaque lookup would "+
					"authenticate, otherwise the regression test passes for the wrong reason", tc.name)
		})
	}
}

// TestJWTRefusal_ValidJWTStillAuthenticates guards the other direction: the
// refusal path must not have been widened into refusing everything.
func TestJWTRefusal_ValidJWTStillAuthenticates(t *testing.T) {
	t.Parallel()
	appID, userID := id.NewAppID(), id.NewUserID()
	tc := jwtFallthroughCase{
		claims: &tokenformat.TokenClaims{UserID: userID.String(), AppID: appID.String()},
		rescue: jwtFallthroughRescue(appID, userID),
	}
	rec := tc.run(t, &jwtFallthroughValidator{claims: tc.claims})
	require.Equal(t, http.StatusOK, rec.Code, "a JWT that passes every check must authenticate")
}
