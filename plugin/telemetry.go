package plugin

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/internal/telemetry"
)

// spanRunPluginHook is the span each plugin's auth hook runs in.
const spanRunPluginHook = "run plugin hook"

// Values of authsome.plugin.hook on the run plugin hook span.
const (
	hookBeforeSignUp        = "before_sign_up"
	hookAfterSignUp         = "after_sign_up"
	hookBeforeSignIn        = "before_sign_in"
	hookAfterSignIn         = "after_sign_in"
	hookBeforePrincipalAuth = "before_principal_auth"
	hookAfterPrincipalAuth  = "after_principal_auth"
	hookBeforeSessionCreate = "before_session_create"
	hookAfterSessionCreate  = "after_session_create"
)

// Values of authsome.result on the run plugin hook span.
const (
	hookResultOK      = "ok"
	hookResultRefused = "refused"
	hookResultError   = "error"
)

// runHook calls one plugin's hook inside a run plugin hook span named for
// the plugin and the hook, so a slow risk check or geo lookup shows up as
// itself in the sign-in trace. A before hook that returns an error refuses
// the operation, which is an answer, so the span says "refused" and is not
// marked failed. An after hook's error is a fault: the span is marked
// failed with the error's type and never its text.
func runHook(ctx context.Context, hook, pluginName string, before bool, fn func(context.Context) error) error {
	ctx, span := telemetry.Start(ctx, spanRunPluginHook, trace.WithAttributes(
		telemetry.AttrPluginName.String(pluginName),
		telemetry.AttrPluginHook.String(hook),
	))
	defer span.End()
	err := fn(ctx)
	switch {
	case err == nil:
		span.SetAttributes(telemetry.AttrResult.String(hookResultOK))
	case before:
		span.SetAttributes(telemetry.AttrResult.String(hookResultRefused))
	default:
		span.SetAttributes(telemetry.AttrResult.String(hookResultError))
		telemetry.FailErr(span, err)
	}
	return err
}
