package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/middleware"
)

const otherAppPublishableKey = "pk_test_other_app_boundary"

// seedOtherApp creates a second, non-platform app with its own publishable key
// so a test can present that key against a session minted in the platform app.
func seedOtherApp(t *testing.T, eng *authsome.Engine) {
	t.Helper()
	now := time.Now()
	require.NoError(t, eng.Store().CreateApp(context.Background(), &app.App{
		ID:             otherAppID(t),
		Name:           "Other",
		Slug:           "other",
		PublishableKey: otherAppPublishableKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))
}

// realChainHandler builds the production middleware order: the auth
// middleware runs before routing through UseGlobal, and the API's own
// publishable-key middleware runs at route level after it. Tests that inject
// a user straight into the context cannot see the ordering bug this guards.
func realChainHandler(t *testing.T, a interface {
	RegisterRoutes(forge.Router) error
}, eng *authsome.Engine) http.Handler {
	t.Helper()
	root := forge.NewRouter()
	root.UseGlobal(eng.AuthMiddleware())
	require.NoError(t, a.RegisterRoutes(root))
	return root.Handler()
}

func TestAdminScopeCannotBeSwitchedWithAnotherAppsPublishableKey(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := realChainHandler(t, a, eng)
	seedOtherApp(t, eng)

	// The first sign-up on the platform app is promoted to platform owner and
	// therefore holds manage:user for the platform app.
	_, ownerToken, _ := signUp(t, eng, "boundary-owner@test.com", "SecureP@ss1")
	foreign := seedForeignUser(t, eng, "boundary-foreign@test.com")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/admin/users/"+foreign.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set(middleware.PublishableKeyHeader, otherAppPublishableKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusOK, rec.Code, "another app's publishable key must not move the caller's tenant; body=%s", rec.Body.String())
	assert.Contains(t, []int{http.StatusUnauthorized, http.StatusNotFound}, rec.Code, "body=%s", rec.Body.String())
}

func TestAdminScopeFollowsTheSessionWithTheCallersOwnKey(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := realChainHandler(t, a, eng)
	seedOtherApp(t, eng)

	_, ownerToken, _ := signUp(t, eng, "boundary-owner2@test.com", "SecureP@ss1")
	_, victimToken, _ := signUp(t, eng, "boundary-victim@test.com", "SecureP@ss1")
	victimID := userIDFor(t, eng, victimToken)
	foreign := seedForeignUser(t, eng, "boundary-foreign2@test.com")

	get := func(target string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/admin/users/"+target, nil)
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		req.Header.Set(middleware.PublishableKeyHeader, testPublishableKey)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, get(victimID.String()), "a user in the caller's app is readable")
	assert.Equal(t, http.StatusNotFound, get(foreign.ID.String()), "a user in another app is not")
}
