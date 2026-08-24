package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/store/memory"
)

// decodeTokenResponse reads the token endpoint's JSON body.
func decodeTokenResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

// The scopes a grant issues have to survive onto the session row. Without
// this they exist only in the JWT claims and the response body, so an opaque
// token loses them and token exchange has no subject-side ceiling to narrow
// against.
func TestAuthCodeGrantStampsScopesOnSession(t *testing.T) {
	p, _, mux := newFixture(t)
	core := memory.New()
	p.SetStore(core)

	q := baseAuthorizeQuery(confidentialID)
	q.Set("scope", "openid profile")
	code := codeFrom(t, authorize(t, mux, q))

	body := decodeTokenResponse(t, postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  registeredURI,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	}))

	sess, err := core.GetSessionByToken(context.Background(), body["access_token"].(string))
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile"}, sess.Scopes)
}
