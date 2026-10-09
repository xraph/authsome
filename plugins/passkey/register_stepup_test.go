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
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Registering a passkey mints a sign-in credential, so it is refused on a
// session that did not sign in recently.

func registerBegin(t *testing.T, signedInAt time.Time, withSession bool) int {
	t.Helper()
	p := New(Config{RPID: "localhost"})
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))

	u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), Email: "pk@example.com"}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/passkeys/register/begin", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	ctx := middleware.WithUser(req.Context(), u)
	ctx = middleware.WithUserID(ctx, u.ID)
	if withSession {
		ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), UserID: u.ID, CreatedAt: signedInAt})
	}
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req.WithContext(ctx))
	return rec.Code
}

func TestRegisterBegin_FreshSessionProceeds(t *testing.T) {
	assert.Equal(t, http.StatusOK, registerBegin(t, time.Now().Add(-time.Minute), true))
}

func TestRegisterBegin_StaleSessionIsRefused(t *testing.T) {
	assert.Equal(t, http.StatusForbidden, registerBegin(t, time.Now().Add(-time.Hour), true))
}

func TestRegisterBegin_NoSessionIsRefused(t *testing.T) {
	assert.Equal(t, http.StatusForbidden, registerBegin(t, time.Time{}, false))
}
