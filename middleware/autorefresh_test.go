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

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// AutoRefreshMiddleware tests
// ──────────────────────────────────────────────────

func TestAutoRefresh_Disabled(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			t.Fatal("refresher should not be called when disabled")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: false}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:        id.NewSessionID(),
		Token:     "original-token",
		ExpiresAt: time.Now().Add(2 * time.Minute), // near expiry
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no refresh header when disabled")
}

func TestAutoRefresh_NotNearExpiry(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			t.Fatal("refresher should not be called when token not near expiry")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "original-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(30 * time.Minute), // far from expiry
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no refresh when not near expiry")
}

func TestAutoRefresh_NearExpiry_RefreshesAndSetsHeaders(t *testing.T) {
	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:            true,
				Threshold:          5 * time.Minute,
				ExposeRefreshToken: true,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute), // within 5-min threshold
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "new-access-token", rec.Header().Get("X-Auth-Token"))
	assert.NotEmpty(t, rec.Header().Get("X-Auth-Token-Expires-At"))
	assert.Equal(t, "new-refresh-token", rec.Header().Get("X-Auth-Refresh-Token"))
}

func TestAutoRefresh_RefreshTokenNotExposedByDefault(t *testing.T) {
	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:            true,
				Threshold:          5 * time.Minute,
				ExposeRefreshToken: false, // default — don't expose
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "new-access-token", rec.Header().Get("X-Auth-Token"), "access token should be in header")
	assert.Empty(t, rec.Header().Get("X-Auth-Refresh-Token"), "refresh token should NOT be in header")
}

func TestAutoRefresh_CookieSetter_Called(t *testing.T) {
	var cookieToken string
	var cookieMaxAge int

	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "refreshed-token",
		RefreshToken: "refreshed-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	setter := func(_ forge.Context, token string, maxAge int) {
		cookieToken = token
		cookieMaxAge = maxAge
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
		setter,
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "refreshed-token", cookieToken, "cookie setter should receive the new token")
	assert.Greater(t, cookieMaxAge, 0, "cookie MaxAge should be positive")
}

func TestAutoRefresh_RefreshFailure_NonFatal(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			return nil, errors.New("refresh failed")
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Response should still be OK even when refresh fails
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no new token header on refresh failure")
}

func TestAutoRefresh_NoSession_NoOp(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			t.Fatal("refresher should not be called without a session")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	// No session injected
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAutoRefresh_ForwardsDPoPProofForBoundSession(t *testing.T) {
	var got middleware.RefreshRequest

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, req middleware.RefreshRequest) (*session.Session, error) {
			got = req
			return &session.Session{
				ID:        id.NewSessionID(),
				Token:     "refreshed-token",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		DPoPJKT:      "bound-thumbprint",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("DPoP", "the-proof")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "old-refresh", got.RefreshToken)
	assert.Equal(t, "the-proof", got.DPoPProof, "the request's proof must reach Refresh")
	assert.Equal(t, "GET", got.Method, "htm is checked against the request method")
	assert.Equal(t, "http://example.com/test", got.RequestURL, "htu is checked against the request URL")
	assert.Equal(t, "refreshed-token", rec.Header().Get("X-Auth-Token"))
}

func TestAutoRefresh_ForwardsClientBindingToRefresher(t *testing.T) {
	var got middleware.RefreshRequest

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, req middleware.RefreshRequest) (*session.Session, error) {
			got = req
			return &session.Session{
				ID:        id.NewSessionID(),
				Token:     "refreshed-token",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.RemoteAddr = "203.0.113.7:44321"
	req.Header.Set("User-Agent", "test-agent/1.0")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "203.0.113.7", got.IPAddress, "IP binding must be checked on auto-refresh")
	assert.Equal(t, "test-agent/1.0", got.UserAgent, "UA binding must be checked on auto-refresh")
}

// TestAutoRefresh_BoundSessionRevalidatesTheSameProof is the regression guard
// for the reason auto-refresh could not simply forward the request's proof.
//
// Two enforcement points look at one proof on one request: the auth
// middleware, and then Engine.Refresh underneath auto-refresh. The replay
// cache records a jti the first time it sees it, so the second look would
// report the request's own proof as a replay of itself. dpop.WithRequestScope,
// installed by the auth middleware, is what stops that, and this test fails
// if that wiring ever comes apart.
//
// The second request below is the control: it proves the replay cache is
// genuinely live, so the first request passing means the scope worked rather
// than the cache being absent.
func TestAutoRefresh_BoundSessionRevalidatesTheSameProof(t *testing.T) {
	key := dpopTestKey(t)
	validator := dpop.NewValidator(dpop.Config{})

	sess := &session.Session{
		ID:           id.NewSessionID(),
		AppID:        id.NewAppID(),
		UserID:       id.NewUserID(),
		Token:        "bound-token",
		RefreshToken: "old-refresh",
		DPoPJKT:      dpopThumbprint(t, key),
		ExpiresAt:    time.Now().Add(2 * time.Minute), // near expiry
	}
	u := &user.User{ID: sess.UserID, AppID: sess.AppID, Email: "dpop@test.com"}

	var refreshErr error
	refresher := func(ctx context.Context, req middleware.RefreshRequest) (*session.Session, error) {
		// Stands in for Engine.verifyRefreshDPoP: re-check the very proof the
		// auth middleware already accepted, against this request's htm/htu.
		p, err := dpop.Parse(req.DPoPProof)
		if err != nil {
			refreshErr = err
			return nil, err
		}
		if err := validator.Validate(ctx, p, dpop.Expectation{
			Method:      req.Method,
			URL:         req.RequestURL,
			ExpectedJKT: sess.DPoPJKT,
		}); err != nil {
			refreshErr = err
			return nil, err
		}
		return &session.Session{
			ID:        sess.ID,
			Token:     "refreshed-token",
			DPoPJKT:   sess.DPoPJKT, // the rotated token keeps the binding
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}, nil
	}

	router := forge.NewRouter()
	router.Use(middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == sess.Token {
				return sess, nil
			}
			return nil, errors.New("invalid")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == sess.UserID.String() {
				return u, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{DPoPValidator: validator},
	))
	router.Use(middleware.AutoRefreshMiddleware(
		refresher,
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	claims := dpopValidClaims("http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash(sess.Token)
	proof := dpopMintProof(t, key, "ES256", claims)

	newReq := func() *http.Request {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
		r.Header.Set("Authorization", "DPoP bound-token")
		r.Header.Set("DPoP", proof)
		return r
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newReq())

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, refreshErr, "the proof must not read as a replay of itself")
	assert.Equal(t, "refreshed-token", rec.Header().Get("X-Auth-Token"),
		"a bound session must still get transparent auto-refresh")

	// Control: the same proof on a second request is a genuine replay.
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, newReq())
	assert.Equal(t, http.StatusUnauthorized, rec2.Code,
		"sanity: the replay cache is live, so the first request passed on merit")
}
