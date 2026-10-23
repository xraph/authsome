package api_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// memberOrgs stands in for the organization plugin: SwitchActiveOrg finds it
// by name and asks it which orgs the user belongs to.
type memberOrgs struct{ orgs map[string][]id.OrgID }

func (m *memberOrgs) Name() string { return "organization" }

func (m *memberOrgs) ListUserOrganizations(_ context.Context, userID id.UserID) ([]*organization.Organization, error) {
	ids := m.orgs[userID.String()]
	out := make([]*organization.Organization, 0, len(ids))
	for _, o := range ids {
		out = append(out, &organization.Organization{ID: o})
	}
	return out, nil
}

// POST /v1/me/switch-org on a JWT app hands back an access token for the new
// org and puts it in the session cookie, so neither a bearer client nor a
// browser keeps acting in the org it just left.
func TestHandleSwitchOrg_JWTAppReturnsTokenForNewOrg(t *testing.T) {
	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)
	orgs := &memberOrgs{orgs: map[string][]id.OrgID{}}
	eng, err := authsome.NewEngine(authsome.WithConfig(testConfig()), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithJWTFormat(testAppIDStr, jwtFmt),
		authsome.WithPlugin(orgs),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)
	handler := withTestKey(api.New(eng).Handler())

	ctx := context.Background()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "switch-jwt@test.com", Password: "SecureP@ss123", FirstName: "S"})
	require.NoError(t, err)
	orgID := id.NewOrgID()
	orgs.orgs[u.ID.String()] = []id.OrgID{orgID}

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/me/switch-org", jsonBody(t, map[string]string{"org_id": orgID.String()}))
	req.Header.Set("Content-Type", "application/json")
	reqCtx := middleware.WithUserID(req.Context(), u.ID)
	reqCtx = middleware.WithUser(reqCtx, u)
	reqCtx = middleware.WithSessionID(reqCtx, sess.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(reqCtx))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var resp api.SwitchOrgResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, sess.ID.String(), resp.SessionID)
	assert.Equal(t, orgID.String(), resp.OrgID)
	require.NotEmpty(t, resp.SessionToken)
	assert.NotEmpty(t, resp.ExpiresAt)

	parts := strings.Split(resp.SessionToken, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	assert.Equal(t, orgID.String(), claims["org_id"])

	var cookieToken string
	for _, c := range rec.Result().Cookies() {
		if c.Value == resp.SessionToken {
			cookieToken = c.Value
		}
	}
	assert.Equal(t, resp.SessionToken, cookieToken, "the session cookie has to carry the new token")
}
