package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// DELETE /v1/sessions signs the caller out everywhere except the session
// that made the request.
func TestRevokeOtherSessions_KeepsCaller(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	ctx := context.Background()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	u, first, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "revoke-others@test.com", Password: "SecureP@ss123", FirstName: "R"})
	require.NoError(t, err)
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "revoke-others@test.com", Password: "SecureP@ss123"})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/v1/sessions", nil)
	reqCtx := middleware.WithUserID(req.Context(), u.ID)
	reqCtx = middleware.WithUser(reqCtx, u)
	reqCtx = middleware.WithSessionID(reqCtx, first.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(reqCtx))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	left, err := eng.Store().ListUserSessions(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, left, 1)
	assert.Equal(t, first.ID, left[0].ID)
}

func TestRevokeOtherSessions_RequiresAuth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := withTestKey(a.Handler())
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
