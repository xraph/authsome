package telemetry_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

func TestAuthMethodFoldsToTheBoundedSet(t *testing.T) {
	for raw, want := range map[string]string{
		"password":           "password",
		"password+mfa":       "password",
		"passkey":            "passkey",
		"magiclink":          "magic_link",
		"phone":              "phone",
		"social:github":      "social",
		"social:github+mfa":  "social",
		"sso:acme-okta":      "sso",
		"sso:acme-okta+mfa":  "sso",
		"email_verification": "email_verification",
		"my-custom-plugin":   "other",
		"":                   "other",
	} {
		assert.Equal(t, want, telemetry.AuthMethod(raw), raw)
	}
}

func TestBoundedKeepsConstantsAndFoldsFreeText(t *testing.T) {
	assert.Equal(t, "auth.signin", telemetry.Bounded("auth.signin"))
	assert.Equal(t, "org.member.add", telemetry.Bounded("org.member.add"))
	assert.Equal(t, "", telemetry.Bounded(""))
	assert.Equal(t, "other", telemetry.Bounded("GET /v1/users/ausr_01jf0000000000000000000000"))
	assert.Equal(t, "other", telemetry.Bounded("alice@example.com"))
	assert.Equal(t, "other", telemetry.Bounded("Auth.SignIn"))
}

// Fail puts the error's type on the span and never its text.
func TestFailKeepsErrorTextOffTheSpan(t *testing.T) {
	rec := telemetrytest.Spans(t)
	_, span := telemetry.Start(context.Background(), "op")
	err := fmt.Errorf("lookup alice@example.com: %w", errors.New("password hunter2 rejected"))
	telemetry.FailErr(span, err)
	span.End()

	got := telemetrytest.Authsome(rec)
	require.Len(t, got, 1)
	assert.Equal(t, codes.Error, got[0].Status().Code)
	assert.Equal(t, "*errors.errorString", got[0].Status().Description)
	text := telemetrytest.Text(got[0])
	assert.NotContains(t, text, "alice@example.com")
	assert.NotContains(t, text, "hunter2")
}

func TestDetachDropsTheSpanButKeepsCancellation(t *testing.T) {
	rec := telemetrytest.Spans(t)
	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := telemetry.Start(ctx, "boot")
	detached := telemetry.Detach(ctx)
	span.End()

	assert.False(t, trace.SpanContextFromContext(detached).IsValid())
	cancel()
	assert.Error(t, detached.Err(), "cancellation still reaches the detached ctx")

	_, root := telemetry.Start(detached, "work")
	root.End()
	work := telemetrytest.Named(telemetrytest.Authsome(rec), "work")
	require.Len(t, work, 1)
	assert.False(t, work[0].Parent().IsValid(), "a span under a detached ctx is a root")
}

func TestSignInAttemptCountsOnce(t *testing.T) {
	reg, scrape := telemetrytest.Registry(t)
	m := telemetry.NewMetrics(reg)

	// Handed off: IssueSession counts it, so Done records nothing.
	a := m.StartSignIn(telemetry.MethodMagicLink)
	a.HandOff()
	a.Done(errors.New("ignored"))

	m.StartSignIn(telemetry.MethodMagicLink).Done(forge.Unauthorized("invalid or expired magic link"))
	m.StartSignIn(telemetry.MethodPhone).Done(forge.NewHTTPError(http.StatusTooManyRequests, "slow down"))
	m.StartSignIn(telemetry.MethodSSO).Done(errors.New("store down"))

	refused := m.StartSignIn(telemetry.MethodSocial)
	refused.Refuse(telemetry.SignInDenied)
	refused.Done(nil)

	var nilMetrics *telemetry.Metrics
	nilMetrics.StartSignIn(telemetry.MethodPasskey).Done(nil)

	assert.ElementsMatch(t, []string{
		`forge_authsome_signin_attempts_total{method="magic_link",result="invalid"}`,
		`forge_authsome_signin_attempts_total{method="phone",result="rate_limited"}`,
		`forge_authsome_signin_attempts_total{method="sso",result="error"}`,
		`forge_authsome_signin_attempts_total{method="social",result="denied"}`,
	}, telemetrytest.Series(scrape(), "forge_authsome_signin_"))
}

// A nil *Metrics is what an engine without a sink holds; nothing on it may
// panic.
func TestNilMetricsRecordNothing(t *testing.T) {
	var m *telemetry.Metrics
	assert.NotPanics(t, func() {
		m.SignIn("password", "ok")
		m.MFAVerification("signin", "totp", "ok")
		m.Refresh("ok")
		m.PasswordHash("hash", "bcrypt", 0)
		m.APIKeyAuth("ok")
		m.OAuth2Token("client_credentials", "ok")
		m.ProviderRequest("email", "ok", 0)
		m.RetentionRowsDeleted("sessions", 1)
		m.RetentionSweep("ok")
		m.RetentionJob("hubspot", "done")
		m.ControlDegraded("ratelimit")
		m.Throttled("route")
		m.RateLimiterError("route")
		m.AuditRecordFailed()
		m.Event("auth.signin", "session", "success")
	})
	assert.Nil(t, telemetry.From(struct{}{}))
}
