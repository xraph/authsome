package magiclink_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
)

// TestMagicLink_SendThenVerify_Succeeds is the end-to-end flow: /send must
// create the verification against the REAL user (resolved by email), so that
// /verify can resolve them and mint a session. Previously /send used a random
// user id, making every real magic-link verification fail with a 500.
func TestMagicLink_SendThenVerify_Succeeds(t *testing.T) {
	p, s, mailer := newTestPlugin(t)
	u := createTestUser(t, s)

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	handler := mux.Handler()

	// Send.
	sendReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/send", jsonBody(t, map[string]string{"email": u.Email}))
	sendReq.Header.Set("Content-Type", "application/json")
	sendRec := httptest.NewRecorder()
	handler.ServeHTTP(sendRec, sendReq)
	require.Equal(t, http.StatusOK, sendRec.Code, "send body=%s", sendRec.Body.String())
	require.Len(t, mailer.sent, 1, "a magic link should be sent to the registered email")
	token := mailer.sent[0].Token
	require.NotEmpty(t, token)

	// Verify.
	verifyReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/verify", jsonBody(t, map[string]string{"token": token}))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyRec := httptest.NewRecorder()
	handler.ServeHTTP(verifyRec, verifyReq)

	require.Equal(t, http.StatusOK, verifyRec.Code, "verify must succeed for a real user; body=%s", verifyRec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(verifyRec.Body).Decode(&resp))
	assert.NotEmpty(t, resp["session_token"], "verify must return a session token")
}

// TestMagicLink_Send_UnknownEmail_UniformSuccessNoLink pins anti-enumeration:
// sending for an unregistered email returns the same success response but does
// not actually send a link (nor create a token bound to a phantom user).
func TestMagicLink_Send_UnknownEmail_UniformSuccessNoLink(t *testing.T) {
	p, _, mailer := newTestPlugin(t)

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	handler := mux.Handler()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/send", jsonBody(t, map[string]string{"email": "nobody@example.com"}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "send must return a uniform success even for unknown emails; body=%s", rec.Body.String())
	assert.Empty(t, mailer.sent, "no magic link should be sent to an unregistered email")
}
