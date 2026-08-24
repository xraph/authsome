package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Audience enforcement tests (RFC 8707)
//
// Both trySessionAuth (opaque token path, via
// AuthMiddlewareWithStrategies) and tryJWTAuth (JWT path, via
// AuthMiddlewareWithJWT) must independently enforce the audience
// check. A caller must not be able to bypass the restriction simply
// by picking whichever token format skips it.
// ──────────────────────────────────────────────────

func audienceResolverFor(expected []string) middleware.ExpectedAudienceResolver {
	if expected == nil {
		return nil
	}
	return func(_ context.Context) []string {
		return expected
	}
}

func audienceTestCases() []struct {
	name           string
	tokenAudience  []string
	expected       []string
	wantAuthorized bool
} {
	return []struct {
		name           string
		tokenAudience  []string
		expected       []string
		wantAuthorized bool
	}{
		{
			name:           "no resolver configured, audienced token still passes",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       nil,
			wantAuthorized: true,
		},
		{
			name:           "unaudienced token passes an audience check",
			tokenAudience:  nil,
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "matching audience passes",
			tokenAudience:  []string{"https://api.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "one of several audiences matching is enough",
			tokenAudience:  []string{"https://files.example.com", "https://api.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "disjoint audience is refused",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: false,
		},
	}
}

// TestAuthMiddleware_Audience_OpaqueSession exercises trySessionAuth,
// the opaque-token path. Each subtest asserts whether a *user* ended
// up in context, which is only ever set once resolveUser succeeds
// downstream of the audience check, so a subtest here genuinely
// distinguishes "authenticated" from "not authenticated": if the
// audience gate were absent (always pass), the disjoint case would
// wrongly authenticate; if it always failed, the passing cases would
// wrongly reject.
func TestAuthMiddleware_Audience_OpaqueSession(t *testing.T) {
	for _, tc := range audienceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			testUserID := id.NewUserID()
			testAppID := id.NewAppID()
			testSessID := id.NewSessionID()

			testSession := &session.Session{
				ID:       testSessID,
				AppID:    testAppID,
				UserID:   testUserID,
				Token:    "audience-token",
				Audience: tc.tokenAudience,
			}
			testUser := &user.User{
				ID:    testUserID,
				AppID: testAppID,
				Email: "audience@test.com",
			}

			mw := middleware.AuthMiddlewareWithStrategies(
				func(token string) (*session.Session, error) {
					if token == "audience-token" {
						return testSession, nil
					}
					return nil, errors.New("invalid")
				},
				func(userIDStr string) (*user.User, error) {
					if userIDStr == testUserID.String() {
						return testUser, nil
					}
					return nil, errors.New("not found")
				},
				nil, // no strategy fallback: isolates the session path
				log.NewNoopLogger(),
				middleware.SessionBindingConfig{
					ExpectedAudienceResolver: audienceResolverFor(tc.expected),
				},
			)

			var gotUser bool
			router := forge.NewRouter()
			router.Use(mw)
			router.GET("/test", func(ctx forge.Context) error {
				_, gotUser = middleware.UserFrom(ctx.Context())
				return ctx.NoContent(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer audience-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "middleware always passes through to next handler")
			assert.Equal(t, tc.wantAuthorized, gotUser,
				"expected authenticated=%v for token audience %v against expected %v",
				tc.wantAuthorized, tc.tokenAudience, tc.expected)
		})
	}
}

// TestAuthMiddleware_Audience_JWT exercises tryJWTAuth, the JWT path.
// Same distinguishing logic as the opaque-session case above, but
// driven through AuthMiddlewareWithJWT with a mock JWT validator that
// returns claims carrying the case's token audience.
func TestAuthMiddleware_Audience_JWT(t *testing.T) {
	for _, tc := range audienceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			testUserID := id.NewUserID()
			testAppID := id.NewAppID()
			testSessID := id.NewSessionID()

			validator := &mockJWTValidator{
				claims: &tokenformat.TokenClaims{
					UserID:    testUserID.String(),
					AppID:     testAppID.String(),
					SessionID: testSessID.String(),
					Audience:  tc.tokenAudience,
				},
			}

			mw := middleware.AuthMiddlewareWithJWT(
				func(_ string) (*session.Session, error) {
					return nil, errors.New("not found")
				},
				func(userIDStr string) (*user.User, error) {
					if userIDStr == testUserID.String() {
						return &user.User{ID: testUserID, AppID: testAppID, Email: "jwt-audience@test.com"}, nil
					}
					return nil, errors.New("not found")
				},
				nil, // no strategy fallback: isolates the JWT path
				validator,
				log.NewNoopLogger(),
				middleware.SessionBindingConfig{
					ExpectedAudienceResolver: audienceResolverFor(tc.expected),
				},
			)

			var gotUser bool
			router := forge.NewRouter()
			router.Use(mw)
			router.GET("/test", func(ctx forge.Context) error {
				_, gotUser = middleware.UserFrom(ctx.Context())
				return ctx.NoContent(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer header.payload.signature")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "middleware always passes through to next handler")
			assert.Equal(t, tc.wantAuthorized, gotUser,
				"expected authenticated=%v for token audience %v against expected %v",
				tc.wantAuthorized, tc.tokenAudience, tc.expected)
		})
	}
}

// TestAuthMiddleware_Audience_BareConstructor exercises the audience
// guard inlined directly into AuthMiddleware's own session-resolution
// block (as opposed to trySessionAuth, which the strategy/JWT
// constructors delegate to). AuthMiddleware is exported and does not
// route through trySessionAuth, so it needed its own copy of the
// check; this covers the three cases that distinguish "the check is
// wired here too" from "it silently does nothing": no resolver
// configured still passes, an unaudienced legacy token still passes,
// and a disjoint audience is refused. The refusal case is the one
// that actually proves the check is wired, since the other two would
// also pass with no check at all.
func TestAuthMiddleware_Audience_BareConstructor(t *testing.T) {
	cases := []struct {
		name           string
		tokenAudience  []string
		expected       []string
		wantAuthorized bool
	}{
		{
			name:           "no resolver configured, audienced token still passes",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       nil,
			wantAuthorized: true,
		},
		{
			name:           "unaudienced token passes an audience check",
			tokenAudience:  nil,
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "disjoint audience is refused",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testUserID := id.NewUserID()
			testAppID := id.NewAppID()
			testSessID := id.NewSessionID()

			testSession := &session.Session{
				ID:       testSessID,
				AppID:    testAppID,
				UserID:   testUserID,
				Token:    "bare-audience-token",
				Audience: tc.tokenAudience,
			}
			testUser := &user.User{
				ID:    testUserID,
				AppID: testAppID,
				Email: "bare-audience@test.com",
			}

			mw := middleware.AuthMiddleware(
				func(token string) (*session.Session, error) {
					if token == "bare-audience-token" {
						return testSession, nil
					}
					return nil, errors.New("invalid")
				},
				func(userIDStr string) (*user.User, error) {
					if userIDStr == testUserID.String() {
						return testUser, nil
					}
					return nil, errors.New("not found")
				},
				log.NewNoopLogger(),
				middleware.SessionBindingConfig{
					ExpectedAudienceResolver: audienceResolverFor(tc.expected),
				},
			)

			var gotUser bool
			router := forge.NewRouter()
			router.Use(mw)
			router.GET("/test", func(ctx forge.Context) error {
				_, gotUser = middleware.UserFrom(ctx.Context())
				return ctx.NoContent(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer bare-audience-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "middleware always passes through to next handler")
			assert.Equal(t, tc.wantAuthorized, gotUser,
				"expected authenticated=%v for token audience %v against expected %v",
				tc.wantAuthorized, tc.tokenAudience, tc.expected)
		})
	}
}
