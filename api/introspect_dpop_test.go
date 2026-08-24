package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// introspect posts token to /v1/introspect and returns the decoded body.
func introspect(t *testing.T, router http.Handler, token string) map[string]any {
	t.Helper()

	body, err := json.Marshal(map[string]string{"token": token})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	return resp
}

// TestIntrospect_BoundSessionCarriesConfirmation covers RFC 9449 section 7.3.
// A resource server that validates tokens by introspection has no other way to
// learn that the token is bound, so without cnf it accepts a stolen one.
func TestIntrospect_BoundSessionCarriesConfirmation(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	_, token, _ := signUp(t, eng, "bound-introspect@test.com", "SecureP@ss1")

	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)
	sess.DPoPJKT = "test-thumbprint-value"
	require.NoError(t, eng.Store().UpdateSession(context.Background(), sess))

	resp := introspect(t, router, token)

	assert.Equal(t, true, resp["active"])
	cnf, ok := resp["cnf"].(map[string]any)
	require.True(t, ok, "bound token must introspect with a cnf claim, got %v", resp["cnf"])
	assert.Equal(t, "test-thumbprint-value", cnf["jkt"])
}

// TestIntrospect_UnboundSessionOmitsConfirmation is the other half of the
// contract. An unbound token must look exactly as it did before cnf existed,
// so a resource server keying off the claim's presence is not misled.
func TestIntrospect_UnboundSessionOmitsConfirmation(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	_, token, _ := signUp(t, eng, "unbound-introspect@test.com", "SecureP@ss1")

	resp := introspect(t, router, token)

	assert.Equal(t, true, resp["active"])
	assert.NotContains(t, resp, "cnf", "an unbound token must carry no cnf claim")
}
