package hook_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/authsome/hook"
)

func TestEmitEnrichesFromRequestInfo(t *testing.T) {
	bus := newBus()
	var got *hook.Event
	bus.On("capture", func(_ context.Context, ev *hook.Event) error { got = ev; return nil })

	ctx := hook.WithRequestInfo(context.Background(), hook.RequestInfo{
		IP: "203.0.113.9", UserAgent: "curl/8.0", RequestID: "req_1", SessionID: "sess_1",
	})
	bus.Emit(ctx, &hook.Event{Action: "x", Resource: "y", Tenant: "app_1", Err: errors.New("boom")})

	if got == nil {
		t.Fatal("handler did not run")
	}
	if got.IP != "203.0.113.9" || got.UserAgent != "curl/8.0" || got.RequestID != "req_1" || got.SessionID != "sess_1" {
		t.Fatalf("request info not applied: %+v", got)
	}
	if got.Outcome != hook.OutcomeFailure {
		t.Errorf("Outcome = %q, want failure when Err is set", got.Outcome)
	}
	if got.Reason != "boom" {
		t.Errorf("Reason = %q, want the error text", got.Reason)
	}
	if got.Severity != hook.SeverityInfo {
		t.Errorf("Severity = %q, want default info", got.Severity)
	}
	if got.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestEmitDoesNotOverwriteExplicitFields(t *testing.T) {
	bus := newBus()
	var got *hook.Event
	bus.On("capture", func(_ context.Context, ev *hook.Event) error { got = ev; return nil })

	ctx := hook.WithRequestInfo(context.Background(), hook.RequestInfo{IP: "1.1.1.1", SessionID: "ctx"})
	bus.Emit(ctx, &hook.Event{
		Action: "x", Resource: "y", Tenant: "app_1",
		IP: "9.9.9.9", SessionID: "explicit",
		Severity: hook.SeverityCritical, Outcome: hook.OutcomeFailure,
	})

	if got.IP != "9.9.9.9" || got.SessionID != "explicit" || got.Severity != hook.SeverityCritical || got.Outcome != hook.OutcomeFailure {
		t.Fatalf("explicit fields were overwritten: %+v", got)
	}
}

func TestEmitCriticalReturnsFirstHandlerError(t *testing.T) {
	bus := newBus()
	bus.On("ok", func(context.Context, *hook.Event) error { return nil })
	bus.On("fails", func(context.Context, *hook.Event) error { return errors.New("sink down") })
	ran := false
	bus.On("after", func(context.Context, *hook.Event) error { ran = true; return nil })

	err := bus.EmitCritical(context.Background(), &hook.Event{Action: "x", Resource: "y", Tenant: "app_1"})
	if err == nil || err.Error() != `hook "fails": sink down` {
		t.Fatalf("err = %v", err)
	}
	if !ran {
		t.Error("later handlers must still run so metrics and notifications see the event")
	}
}
