package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
)

// appAdmin signs up a user who is NOT a platform owner (the bootstrap
// promotes only the first verified sign-up) and grants them the app's admin
// role, so tests can exercise the ceilings a tenant admin runs into.
func appAdmin(t *testing.T, eng *authsome.Engine, email string) id.UserID {
	t.Helper()
	// Burn the platform-owner slot first so this user is a plain admin.
	signUp(t, eng, "owner-slot-"+email, "SecureP@ss123")
	_, token, _ := signUp(t, eng, email, "SecureP@ss123")
	uid := userIDFor(t, eng, token)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	adminRole, err := eng.GetRoleBySlug(context.Background(), appID, "admin")
	require.NoError(t, err)
	require.NotNil(t, adminRole)
	require.NoError(t, eng.AssignUserRole(context.Background(), &rbac.UserRole{UserID: uid.String(), RoleID: adminRole.ID}))

	roles, err := eng.ListUserRoles(context.Background(), uid)
	require.NoError(t, err)
	for _, r := range roles {
		require.NotEqual(t, rbac.PlatformOwnerSlug, r.Slug, "fixture must not be a platform owner")
	}
	return uid
}

func roleIDBySlug(t *testing.T, eng *authsome.Engine, slug string) string {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	r, err := eng.GetRoleBySlug(context.Background(), appID, slug)
	require.NoError(t, err)
	require.NotNil(t, r)
	// Seeded roles carry Warden ids; the HTTP handlers accept the authsome
	// form, which shares the suffix.
	if i := strings.IndexByte(r.ID, '_'); i >= 0 {
		return "arol_" + r.ID[i+1:]
	}
	return r.ID
}

func postJSON(t *testing.T, handler http.Handler, eng *authsome.Engine, as id.UserID, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, as)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAssignRole_AdminCannotGrantOwner(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "ceiling-admin@test.com")
	_, targetToken, _ := signUp(t, eng, "ceiling-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+roleIDBySlug(t, eng, "owner")+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "an admin must not grant ownership; body=%s", rec.Body.String())
}

func TestAssignRole_AdminCannotAssignToSelf(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "ceiling-self@test.com")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+roleIDBySlug(t, eng, "admin")+"/assign",
		`{"user_id":"`+admin.String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "a caller must not change their own roles; body=%s", rec.Body.String())
}

func TestAddPermission_AdminCannotGrantWhatTheyDoNotHold(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "ceiling-perm@test.com")

	rec := postJSON(t, handler, eng, admin, "/v1/roles/"+roleIDBySlug(t, eng, "admin")+"/permissions",
		`{"action":"manage","resource":"settings"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "settings:manage is not held by an admin; body=%s", rec.Body.String())
}

func TestAssignRole_OwnerGrantsAdmin(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "ceiling-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	_, targetToken, _ := signUp(t, eng, "ceiling-target2@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)

	rec := postJSON(t, handler, eng, owner, "/v1/roles/"+roleIDBySlug(t, eng, "admin")+"/assign",
		`{"user_id":"`+target.String()+`"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "a platform owner holds everything admin confers; body=%s", rec.Body.String())
}
