package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// /v1/signin checks the DPoP mandate before SignIn runs, so the handler
// counts that refusal itself, as "denied" like every other method does.
func TestSignIn_DPoPMandateRefusalCountsAsDenied(t *testing.T) {
	reg, scrape := telemetrytest.Registry(t)
	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(authsome.WithConfig(testConfig()), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s), authsome.WithWarden(w), authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr), authsome.WithMetricsSink(reg))
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })
	secutil.RelaxAuthDefaults(t, eng)

	const email = "dpop-denied-api@example.com"
	signUp(t, eng, email, "SecureP@ss123")
	setSignInDPoPMode(t, eng, "required")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin",
		bytes.NewReader([]byte(`{"email":"`+email+`","password":"SecureP@ss123"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newAPIWithRouter(t, eng).ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	assert.ElementsMatch(t, []string{
		`forge_authsome_signin_attempts_total{method="password",result="denied"}`,
	}, telemetrytest.Series(scrape(), "forge_authsome_signin_attempts_total"))
}
