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

// When the session check loads the row for a JWT, the row's active org has to
// match the token's org_id. A mismatch means the org was switched or cleared
// after the token was signed, and the request would otherwise be scoped to an
// org the session no longer has.
func TestJWTAuth_SessionChecker_OrgMustMatchRow(t *testing.T) {
	userID, appID, sessID := id.NewUserID(), id.NewAppID(), id.NewSessionID()
	tokenOrg, otherOrg := id.NewOrgID(), id.NewOrgID()

	cases := []struct {
		name     string
		claimOrg id.OrgID
		rowOrg   id.OrgID
		admitted bool
	}{
		{"same org", tokenOrg, tokenOrg, true},
		{"neither has one", id.OrgID{}, id.OrgID{}, true},
		{"row switched to another org", tokenOrg, otherOrg, false},
		{"row cleared", tokenOrg, id.OrgID{}, false},
		{"token predates a switch", id.OrgID{}, tokenOrg, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &mockJWTValidator{claims: &tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: sessID.String(),
				OrgID:     tc.claimOrg.String(),
			}}
			mw := middleware.AuthMiddlewareWithJWT(
				func(_ context.Context, _ string) (*session.Session, error) {
					return nil, errors.New("not found")
				},
				func(_ context.Context, _ string) (*user.User, error) {
					return &user.User{ID: userID, AppID: appID}, nil
				},
				nil,
				validator,
				log.NewNoopLogger(),
				middleware.SessionBindingConfig{
					JWTSessionChecker: func(_, _ string) (*session.Session, error) {
						return &session.Session{ID: sessID, AppID: appID, UserID: userID, OrgID: tc.rowOrg}, nil
					},
				},
			)

			var gotUser bool
			var gotOrg string
			router := forge.NewRouter()
			router.Use(mw)
			router.GET("/test", func(ctx forge.Context) error {
				_, gotUser = middleware.UserFrom(ctx.Context())
				if o, ok := middleware.OrgIDFrom(ctx.Context()); ok {
					gotOrg = o.String()
				}
				return ctx.NoContent(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer header.payload.signature")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.admitted, gotUser)
			if tc.admitted {
				assert.Equal(t, tc.claimOrg.String(), gotOrg)
			}
		})
	}
}
