package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/plugin"
)

// spansSince returns the authsome spans that ended after the first n.
func spansSince(rec *tracetest.SpanRecorder, n int) []sdktrace.ReadOnlySpan {
	return telemetrytest.OnlyAuthsome(rec.Ended()[n:])
}

func signInApp(t *testing.T) id.AppID {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	return appID
}

// shape is what a span shows an operator: its name and its attributes.
type shape struct {
	name  string
	attrs map[string]string
}

func shapes(spans []sdktrace.ReadOnlySpan) []shape {
	out := make([]shape, len(spans))
	for i, s := range spans {
		out[i] = shape{s.Name(), telemetrytest.Attrs(s)}
	}
	return out
}

func TestSignInIsTracedAndCounted(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, _ := newTestEngine(t, authsome.WithMetricsSink(reg))
	ctx := context.Background()
	appID := signInApp(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "trace-ok@example.com", Password: "SecureP@ss123", FirstName: "T"})
	require.NoError(t, err)
	assert.Len(t, telemetrytest.Named(telemetrytest.Authsome(rec), "hash password"), 1, "sign-up hashes once")

	n := len(rec.Ended())
	_, sess, err := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "trace-ok@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)
	require.NotNil(t, sess)
	got := spansSince(rec, n)
	assert.Equal(t, []string{"verify password", "issue session"}, telemetrytest.Names(got))
	issue := telemetrytest.Named(got, "issue session")[0]
	assert.Equal(t, trace.SpanKindInternal, issue.SpanKind())
	assert.Equal(t, map[string]string{
		"authsome.auth.method":    "password",
		"authsome.mfa.verified":   "false",
		"authsome.mfa.required":   "false",
		"authsome.result":         "ok",
		"authsome.session.format": "opaque",
	}, telemetrytest.Attrs(issue))

	body := scrape()
	assert.Contains(t, body, `forge_authsome_signin_attempts_total{method="password",result="ok"} 1`)
	assert.Contains(t, body, `forge_authsome_password_hash_duration_seconds_count{algorithm="bcrypt",operation="hash"} 1`)
	assert.Contains(t, body, `forge_authsome_password_hash_duration_seconds_count{algorithm="bcrypt",operation="verify"} 1`)
	assert.Contains(t, body, `forge_authsome_events_total{action="auth.signin",outcome="success",resource="session"} 1`)
}

// A wrong password and an identifier nobody holds must look the same in a
// trace and on a metric: same spans, same attributes, same result.
func TestSignInDoesNotTellAnUnknownIdentifierFromAWrongPassword(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, _ := newTestEngine(t, authsome.WithMetricsSink(reg))
	ctx := context.Background()
	appID := signInApp(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "known@example.com", Password: "SecureP@ss123", FirstName: "K"})
	require.NoError(t, err)

	n := len(rec.Ended())
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "known@example.com", Password: "wrong-password"})
	require.ErrorIs(t, err, account.ErrInvalidCredentials)
	wrong := spansSince(rec, n)

	n = len(rec.Ended())
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "nobody-here@example.com", Password: "wrong-password"})
	require.ErrorIs(t, err, account.ErrInvalidCredentials)
	unknown := spansSince(rec, n)

	require.NotEmpty(t, wrong)
	assert.Equal(t, shapes(wrong), shapes(unknown))
	for _, s := range append(wrong, unknown...) {
		assert.Equal(t, codes.Unset, s.Status().Code, "a refused password is an answer, not a failure")
	}
	assert.Contains(t, scrape(), `forge_authsome_signin_attempts_total{method="password",result="invalid"} 2`)
}

// refusingPlugin turns every sign-in away in its before hook.
type refusingPlugin struct{}

func (refusingPlugin) Name() string { return "test-refuser" }
func (refusingPlugin) OnBeforeSignIn(context.Context, *account.SignInRequest) error {
	return errors.New("risk score too high for alice@example.com")
}

var _ plugin.BeforeSignIn = refusingPlugin{}

func TestPluginHooksAreSpansAndARefusalIsDenied(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, _ := newTestEngine(t, authsome.WithMetricsSink(reg), authsome.WithPlugin(refusingPlugin{}))
	ctx := context.Background()
	appID := signInApp(t)

	_, _, err := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "alice@example.com", Password: "whatever-it-is"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "risk score too high", "the caller sees the hook's error unchanged")

	hooks := telemetrytest.Named(telemetrytest.Authsome(rec), "run plugin hook")
	require.Len(t, hooks, 1)
	assert.Equal(t, map[string]string{
		"authsome.plugin.name": "test-refuser",
		"authsome.plugin.hook": "before_sign_in",
		"authsome.result":      "refused",
	}, telemetrytest.Attrs(hooks[0]))
	assert.NotContains(t, telemetrytest.Text(hooks[0]), "alice@example.com")
	assert.Contains(t, scrape(), `forge_authsome_signin_attempts_total{method="password",result="denied"} 1`)
}

func TestRefreshIsTracedAndCountedByResult(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, _ := newTestEngine(t, authsome.WithMetricsSink(reg))
	ctx := context.Background()
	appID := signInApp(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "refresh-trace@example.com", Password: "SecureP@ss123", FirstName: "R"})
	require.NoError(t, err)
	spent := sess.RefreshToken

	n := len(rec.Ended())
	_, err = eng.Refresh(ctx, spent)
	require.NoError(t, err)
	// Presenting the spent token again is a replay, and the caller still
	// gets the generic answer.
	_, err = eng.Refresh(ctx, spent)
	require.ErrorIs(t, err, account.ErrInvalidCredentials)
	_, err = eng.Refresh(ctx, "not-a-refresh-token")
	require.ErrorIs(t, err, account.ErrInvalidCredentials)

	refreshes := telemetrytest.Named(spansSince(rec, n), "refresh session")
	require.Len(t, refreshes, 3)
	results := make([]string, 0, len(refreshes))
	for _, s := range refreshes {
		attrs := telemetrytest.Attrs(s)
		assert.Equal(t, "refresh_token", attrs["authsome.refresh.via"])
		results = append(results, attrs["authsome.result"])
	}
	assert.Equal(t, []string{"ok", "replay_detected", "invalid"}, results)

	body := scrape()
	for _, want := range []string{
		`forge_authsome_sessions_refreshed_total{result="ok"} 1`,
		`forge_authsome_sessions_refreshed_total{result="replay_detected"} 1`,
		`forge_authsome_sessions_refreshed_total{result="invalid"} 1`,
	} {
		assert.Contains(t, body, want)
	}
}
