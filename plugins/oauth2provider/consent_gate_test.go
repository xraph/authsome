package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

type recordingGate struct {
	called   bool
	clientID string
	scopes   []string
	err      error
}

func (g *recordingGate) Evaluate(_ context.Context, clientID string, _ id.UserID, _ id.OrgID, scopes []string) error {
	g.called = true
	g.clientID = clientID
	g.scopes = scopes
	return g.err
}

func TestConsentGate_GateIsConsulted(t *testing.T) {
	gate := &recordingGate{}
	p := oauth2provider.New(oauth2provider.Config{ConsentGate: gate})

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), []string{"invoices:read"})

	require.NoError(t, err)
	assert.True(t, gate.called)
	assert.Equal(t, "client_abc", gate.clientID)
	assert.Equal(t, []string{"invoices:read"}, gate.scopes)
}

func TestConsentGate_RefusalPropagates(t *testing.T) {
	denied := errors.New("org policy blocks this agent")
	p := oauth2provider.New(oauth2provider.Config{ConsentGate: &recordingGate{err: denied}})

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), nil)

	require.ErrorIs(t, err, denied)
}

// Without a gate the provider must behave exactly as it does today.
func TestConsentGate_SetterWiresGate(t *testing.T) {
	gate := &recordingGate{}
	p := oauth2provider.New()
	p.SetConsentGate(gate)

	require.NoError(t, p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), nil))
	assert.True(t, gate.called, "the setter must wire the gate as effectively as Config does")
}

func TestEvaluateConsent_NoGateAllows(t *testing.T) {
	p := oauth2provider.New()

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), []string{"anything"})

	require.NoError(t, err)
}

// ── end-to-end: a refusal must stop the credential, not just be reachable ──
//
// The tests above only prove that Config/SetConsentGate forward to
// EvaluateConsent. They pass even if the call inside handleAuthorize or
// handleDeviceComplete is deleted. These drive the real HTTP handlers so a
// removed gate call fails them.

// countingAuthCodeStore wraps a Store to record CreateAuthCode calls, so a
// test can assert that a refused authorization never persisted a code — an
// error response with a stored code behind it would still be a leak.
type countingAuthCodeStore struct {
	oauth2provider.Store
	authCodesCreated int
}

func (s *countingAuthCodeStore) CreateAuthCode(ctx context.Context, code *oauth2provider.AuthorizationCode) error {
	s.authCodesCreated++
	return s.Store.CreateAuthCode(ctx, code)
}

func TestConsentGate_RefusalBlocksAuthorizationCode(t *testing.T) {
	p, st, mux := newFixture(t)
	counting := &countingAuthCodeStore{Store: st}
	p.SetOAuth2Store(counting)
	p.SetConsentGate(&recordingGate{err: errors.New("org policy blocks this agent")})

	rec := authorize(t, mux, baseAuthorizeQuery(confidentialID))

	assert.NotEqual(t, http.StatusFound, rec.Code,
		"a refused authorization must not redirect with a code; body: %s", rec.Body.String())
	assert.Equal(t, 0, counting.authCodesCreated,
		"a refused authorization must never persist an authorization code")
}

// deviceAuthorize drives POST /v1/oauth/device/authorize for clientID and
// decodes the response.
func deviceAuthorize(t *testing.T, mux forge.Router, clientID string) *oauth2provider.DeviceAuthResponse {
	t.Helper()
	body, err := json.Marshal(map[string]string{"client_id": clientID})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/device/authorize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp oauth2provider.DeviceAuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return &resp
}

// deviceComplete drives POST /v1/oauth/device/complete as a signed-in user.
func deviceComplete(t *testing.T, mux forge.Router, userCode, action string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"user_code": userCode, "action": action})
	require.NoError(t, err)
	ctx := middleware.WithUserID(context.Background(), id.NewUserID())
	req := httptest.NewRequestWithContext(ctx, "POST",
		"/v1/oauth/device/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestConsentGate_RefusalBlocksDeviceApproval(t *testing.T) {
	p, st, mux := newFixture(t)

	const deviceClientID = "device-client"
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      id.NewAppID(),
		ClientID:   deviceClientID,
		Name:       "Device Client",
		Scopes:     []string{"openid"},
		GrantTypes: []string{"device_code"},
	}))

	p.SetConsentGate(&recordingGate{err: errors.New("org policy blocks this agent")})

	auth := deviceAuthorize(t, mux, deviceClientID)
	rec := deviceComplete(t, mux, auth.UserCode, "approve")

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"a refused device approval must not report success; body: %s", rec.Body.String())

	dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), auth.DeviceCode)
	require.NoError(t, err)
	assert.NotEqual(t, oauth2provider.DeviceCodeStatusAuthorized, dc.Status,
		"a refused approval must not leave the device code authorized")
}
