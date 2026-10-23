package passkey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// A passkey is a sign-in credential on the user's account. A session acting
// for the user without being them (an agent, or a token-exchange session whose
// actor chain names the real caller) must not register one: it would outlive
// the grant and sign in as the user.
func TestRegisterBegin_RefusesDelegatedSessions(t *testing.T) {
	cases := map[string]func(userID id.UserID) *session.Session{
		"agent": func(userID id.UserID) *session.Session {
			return &session.Session{
				ID: id.NewSessionID(), UserID: userID, CreatedAt: time.Now(),
				PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
			}
		},
		"token exchange": func(userID id.UserID) *session.Session {
			return &session.Session{
				ID: id.NewSessionID(), UserID: userID, CreatedAt: time.Now(),
				Actors:     principal.Chain{{Kind: principal.KindWorkload, ID: "svc_actor"}},
				ActorGrant: principal.GrantDelegation,
			}
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			p := New(Config{RPID: "localhost"})
			router := forge.NewRouter()
			require.NoError(t, p.RegisterRoutes(router))

			u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), Email: "pk@example.com"}
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/passkeys/register/begin", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://localhost:3000")
			ctx := middleware.WithUser(req.Context(), u)
			ctx = middleware.WithUserID(ctx, u.ID)
			ctx = middleware.WithSession(ctx, build(u.ID))
			rec := httptest.NewRecorder()
			router.Handler().ServeHTTP(rec, req.WithContext(ctx))

			assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
			assert.NotContains(t, rec.Body.String(), "challenge")
			_, err := p.ceremonies.Get(context.Background(), "passkey:reg:"+u.ID.String())
			assert.Error(t, err, "no registration ceremony is stored")
		})
	}
}
