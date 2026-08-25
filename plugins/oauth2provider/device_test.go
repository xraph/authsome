package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"

	"golang.org/x/crypto/bcrypt"
)

const (
	deviceConfID     = "device-conf-client"
	deviceConfSecret = "device-conf-secret-value"
	devicePublicID   = "device-pub-client"
)

// registerDeviceClients adds a confidential and a public client to an existing
// fixture, both registered for the device grant.
func registerDeviceClients(t *testing.T, st oauth2provider.Store) {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(deviceConfSecret), bcrypt.MinCost)
	require.NoError(t, err)

	appID := id.NewAppID()
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     deviceConfID,
		ClientSecret: string(hashed),
		Name:         "Device Confidential",
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"urn:ietf:params:oauth:grant-type:device_code"},
	}))
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      appID,
		ClientID:   devicePublicID,
		Name:       "Device Public",
		Scopes:     []string{"openid", "profile"},
		GrantTypes: []string{"urn:ietf:params:oauth:grant-type:device_code"},
		Public:     true,
	}))
}

// startDeviceFlow drives POST /device/authorize and returns the device_code.
func startDeviceFlow(t *testing.T, mux forge.Router, clientID string) string {
	t.Helper()
	form := url.Values{"client_id": {clientID}, "scope": {"openid profile"}}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp struct {
		DeviceCode string `json:"device_code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.DeviceCode)
	return resp.DeviceCode
}

// approveDeviceCode marks a pending code authorized, standing in for the user
// approving at the verification URI.
func approveDeviceCode(t *testing.T, st oauth2provider.Store, deviceCode string) {
	t.Helper()
	dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), deviceCode)
	require.NoError(t, err)
	dc.Status = oauth2provider.DeviceCodeStatusAuthorized
	dc.UserID = id.NewUserID()
	require.NoError(t, st.UpdateDeviceCode(context.Background(), dc))
}

func pollDeviceToken(t *testing.T, mux forge.Router, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body["grant_type"] = "urn:ietf:params:oauth:grant-type:device_code"
	return postToken(t, mux, body)
}

// oauthErrorCode pulls the RFC 6749 error code out of a token endpoint failure.
func oauthErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), "body: %s", rec.Body.String())
	return resp.Error
}

// ── client authentication (RFC 8628 §3.4 → RFC 6749 §3.2.1) ────────────

// The device code is not a client credential. It travels in every poll, so it
// lands in access logs and proxies, and the client_id beside it is public by
// definition. A confidential client that skips its secret must be turned away.
func TestDeviceCodeGrant_RejectsConfidentialClientWithoutSecret(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, deviceConfID)
	approveDeviceCode(t, st, deviceCode)

	rec := pollDeviceToken(t, mux, map[string]string{
		"device_code": deviceCode,
		"client_id":   deviceConfID,
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_client", oauthErrorCode(t, rec))
}

func TestDeviceCodeGrant_RejectsConfidentialClientWithWrongSecret(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, deviceConfID)
	approveDeviceCode(t, st, deviceCode)

	rec := pollDeviceToken(t, mux, map[string]string{
		"device_code":   deviceCode,
		"client_id":     deviceConfID,
		"client_secret": "not-the-secret",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_client", oauthErrorCode(t, rec))
}

func TestDeviceCodeGrant_AcceptsConfidentialClientWithSecret(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, deviceConfID)
	approveDeviceCode(t, st, deviceCode)

	rec := pollDeviceToken(t, mux, map[string]string{
		"device_code":   deviceCode,
		"client_id":     deviceConfID,
		"client_secret": deviceConfSecret,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
}

// A public client has no secret to present, so the flow it has always used
// keeps working. This is the guard on the change above.
func TestDeviceCodeGrant_PublicClientNeedsNoSecret(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, devicePublicID)
	approveDeviceCode(t, st, deviceCode)

	rec := pollDeviceToken(t, mux, map[string]string{
		"device_code": deviceCode,
		"client_id":   devicePublicID,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
}

// ── ordering: authentication runs before anything observable ────────────

// authorization_pending, expired_token and invalid_grant each say something
// different about the device code presented. An unauthenticated caller must
// not get to tell them apart, so the answer stays invalid_client whether the
// code is real and pending or made up on the spot.
func TestDeviceCodeGrant_UnauthenticatedPollLeaksNothingAboutTheCode(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	live := startDeviceFlow(t, mux, deviceConfID) // pending, never approved

	pending := pollDeviceToken(t, mux, map[string]string{
		"device_code": live,
		"client_id":   deviceConfID,
	})
	fabricated := pollDeviceToken(t, mux, map[string]string{
		"device_code": "not-a-real-device-code",
		"client_id":   deviceConfID,
	})

	assert.Equal(t, http.StatusUnauthorized, pending.Code, "body: %s", pending.Body.String())
	assert.Equal(t, "invalid_client", oauthErrorCode(t, pending))
	assert.Equal(t, pending.Code, fabricated.Code)
	assert.Equal(t, oauthErrorCode(t, pending), oauthErrorCode(t, fabricated))
}

// The slow_down branch writes: it pushes the code's polling interval up by
// five seconds and stamps LastPolledAt. Reachable without credentials, that
// lets anyone holding a leaked device code ratchet the real device's interval
// on every request until the code times out.
func TestDeviceCodeGrant_UnauthenticatedPollCannotRatchetTheInterval(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, deviceConfID)
	before, err := st.GetDeviceCodeByDeviceCode(context.Background(), deviceCode)
	require.NoError(t, err)

	for range 5 {
		rec := pollDeviceToken(t, mux, map[string]string{
			"device_code": deviceCode,
			"client_id":   deviceConfID,
		})
		require.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	}

	after, err := st.GetDeviceCodeByDeviceCode(context.Background(), deviceCode)
	require.NoError(t, err)
	assert.Equal(t, before.Interval, after.Interval)
	assert.True(t, after.LastPolledAt.IsZero(), "an unauthenticated poll must not be recorded")
}
