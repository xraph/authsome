package authsome_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/settings"
)

// An app that mandates DPoP refuses a sign-in whose client can't prove a
// key. That is a policy refusal, so it counts as "denied", the same as a
// plugin hook's refusal, and not as a bad credential.
func TestDPoPMandateRefusalCountsAsDenied(t *testing.T) {
	reg, scrape := telemetrytest.Registry(t)
	ml := magiclink.New()
	eng, s := newTestEngine(t, authsome.WithMetricsSink(reg), authsome.WithPlugin(ml))
	ctx := context.Background()
	appID := signInApp(t)
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "dpop-denied@example.com", Password: "SecureP@ss123", FirstName: "D"})
	require.NoError(t, err)
	// Mandate DPoP for the app the way the dashboard does.
	raw, err := json.Marshal("required")
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, "dpop.mode", raw, settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	v, err := account.NewVerification(ctx, appID, u.ID, magiclink.VerificationTypeMagicLink, 5*time.Minute)
	require.NoError(t, err)
	require.NoError(t, s.CreateVerification(ctx, v))

	mux := forge.NewRouter()
	require.NoError(t, ml.RegisterRoutes(mux))
	body, err := json.Marshal(map[string]string{"token": v.Token})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/magic-link/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	assert.ElementsMatch(t, []string{
		`forge_authsome_signin_attempts_total{method="magic_link",result="denied"}`,
	}, telemetrytest.Series(scrape(), "forge_authsome_signin_attempts_total"))
}
