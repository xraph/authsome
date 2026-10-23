package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/settings"
)

// An agent's authority is its grant. The admin routes check permissions
// against its delegating user, so an agent acting for an admin would carry
// that admin's whole role set into user management, role assignment and app
// settings. Every admin route refuses an agent or delegated session; the
// admin's own session still gets through.
func TestAdminRoutes_RefuseAgentSession(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "agent-admin-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	_, targetToken, _ := signUp(t, eng, "agent-admin-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)
	ctx := context.Background()

	do := func(method, path, body string, asOwner bool) *httptest.ResponseRecorder {
		t.Helper()
		var r *bytes.Reader
		if body != "" {
			r = bytes.NewReader([]byte(body))
		} else {
			r = bytes.NewReader(nil)
		}
		req := httptest.NewRequestWithContext(ctx, method, path, r)
		req.Header.Set("Content-Type", "application/json")
		if asOwner {
			req = asOwnSession(t, asAdmin(t, req, eng, owner), owner)
		} else {
			req = asAgentOf(t, req, owner)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	roleCount := func() int {
		roles, err := eng.ListUserRoles(ctx, target)
		require.NoError(t, err)
		return len(roles)
	}

	t.Run("role assign", func(t *testing.T) {
		path := "/v1/roles/" + roleIDBySlug(t, eng, "admin") + "/assign"
		body := `{"user_id":"` + target.String() + `"}`
		before := roleCount()

		rec := do(http.MethodPost, path, body, false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Equal(t, before, roleCount(), "no role is assigned")

		rec = do(http.MethodPost, path, body, true)
		assert.Equal(t, http.StatusOK, rec.Code, "the owner's own session still assigns; body=%s", rec.Body.String())
	})

	t.Run("ban", func(t *testing.T) {
		path := "/v1/admin/users/" + target.String() + "/ban"
		rec := do(http.MethodPost, path, `{"reason":"agent"}`, false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		u, err := eng.AdminGetUser(ctx, target)
		require.NoError(t, err)
		assert.False(t, u.Banned, "the user is not banned")

		rec = do(http.MethodPost, path, `{"reason":"owner"}`, true)
		assert.Less(t, rec.Code, 300, "the owner's own session still bans; body=%s", rec.Body.String())
	})

	t.Run("delete", func(t *testing.T) {
		_, victimToken, _ := signUp(t, eng, "agent-admin-victim@test.com", "SecureP@ss123")
		victim := userIDFor(t, eng, victimToken)
		path := "/v1/admin/users/" + victim.String()

		rec := do(http.MethodDelete, path, "", false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		_, err := eng.AdminGetUser(ctx, victim)
		require.NoError(t, err, "the user survives")

		rec = do(http.MethodDelete, path, "", true)
		assert.Less(t, rec.Code, 300, "the owner's own session still deletes; body=%s", rec.Body.String())
	})

	t.Run("platform owner grant", func(t *testing.T) {
		_, candToken, _ := signUp(t, eng, "agent-admin-candidate@test.com", "SecureP@ss123")
		cand := userIDFor(t, eng, candToken)
		isOwner := func() bool {
			roles, err := eng.ListUserRoles(ctx, cand)
			require.NoError(t, err)
			for _, r := range roles {
				if r.Slug == rbac.PlatformOwnerSlug {
					return true
				}
			}
			return false
		}
		body := `{"user_id":"` + cand.String() + `"}`

		rec := do(http.MethodPost, "/v1/admin/platform/owners", body, false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.False(t, isOwner(), "nobody becomes a platform owner")

		rec = do(http.MethodPost, "/v1/admin/platform/owners", body, true)
		assert.Less(t, rec.Code, 300, "the owner's own session still grants; body=%s", rec.Body.String())
		assert.True(t, isOwner())
	})

	t.Run("list users", func(t *testing.T) {
		rec := do(http.MethodGet, "/v1/admin/users?app_id="+testAppIDStr, "", false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.NotContains(t, rec.Body.String(), target.String())

		rec = do(http.MethodGet, "/v1/admin/users?app_id="+testAppIDStr, "", true)
		assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("settings", func(t *testing.T) {
		appID, err := id.ParseAppID(testAppIDStr)
		require.NoError(t, err)
		path := "/v1/admin/settings/values/" + scopedSettingKey
		body := `{"value":false,"scope":"app","scope_id":"` + appID.String() + `"}`
		current := func() bool {
			v, err := settings.Get(ctx, eng.Settings(), authsome.SettingJWTRequireActiveSession, settings.ResolveOpts{AppID: appID.String()})
			require.NoError(t, err)
			return v
		}
		require.True(t, current())

		rec := do(http.MethodPut, path, body, false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.True(t, current(), "the setting is unchanged")

		rec = do(http.MethodPut, path, body, true)
		assert.Equal(t, http.StatusOK, rec.Code, "the owner's own session still writes it; body=%s", rec.Body.String())
		assert.False(t, current())
	})

	t.Run("bulk session revoke", func(t *testing.T) {
		_, victimToken, _ := signUp(t, eng, "agent-admin-bulk@test.com", "SecureP@ss123")
		victim := userIDFor(t, eng, victimToken)
		count := func() int {
			list, err := eng.Store().ListUserSessions(ctx, victim)
			require.NoError(t, err)
			return len(list)
		}
		before := count()
		require.Positive(t, before)
		path := "/v1/admin/bulk/sessions?user_id=" + victim.String()

		rec := do(http.MethodDelete, path, "", false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Equal(t, before, count(), "no session is revoked")

		rec = do(http.MethodDelete, path, "", true)
		assert.Less(t, rec.Code, 300, "the owner's own session still revokes; body=%s", rec.Body.String())
		assert.Less(t, count(), before)
	})

	t.Run("app session config", func(t *testing.T) {
		appID, err := id.ParseAppID(testAppIDStr)
		require.NoError(t, err)
		path := "/v1/admin/apps/" + appID.String() + "/session-config"
		maxSessions := func() *int {
			cfg, err := eng.Store().GetAppSessionConfig(ctx, appID)
			if err != nil || cfg == nil {
				return nil
			}
			return cfg.MaxActiveSessions
		}

		rec := do(http.MethodPut, path, `{"max_active_sessions":3}`, false)
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		assert.Nil(t, maxSessions(), "the config is unchanged")

		rec = do(http.MethodPut, path, `{"max_active_sessions":3}`, true)
		assert.Less(t, rec.Code, 300, "the owner's own session still sets it; body=%s", rec.Body.String())
		require.NotNil(t, maxSessions())
		assert.Equal(t, 3, *maxSessions())
	})
}
