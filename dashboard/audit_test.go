package dashboard_test

import (
	"context"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/hook"
)

// captureBus returns a bus whose only handler appends every event to the
// returned slice, standing in for the engine's audit subscriber.
func captureBus() (*hook.Bus, *[]*hook.Event) {
	bus := hook.NewBus(log.NewNoopLogger())
	events := &[]*hook.Event{}
	bus.On("capture", func(_ context.Context, ev *hook.Event) error {
		*events = append(*events, ev)
		return nil
	})
	return bus, events
}

func TestAuditor_RecordPopulatesEvent(t *testing.T) {
	bus, events := captureBus()
	a := dashboard.NewAuditor(bus)

	meta := map[string]string{"app_id": "app-123"}
	a.Record(context.Background(),
		"user.delete",
		hook.SeverityCritical,
		"actor-1",
		"resource-2",
		meta,
	)

	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	ev := (*events)[0]
	if ev.Action != "user.delete" {
		t.Errorf("Action: got %q, want %q", ev.Action, "user.delete")
	}
	if ev.Severity != hook.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", ev.Severity, hook.SeverityCritical)
	}
	if ev.ActorID != "actor-1" {
		t.Errorf("ActorID: got %q, want %q", ev.ActorID, "actor-1")
	}
	if ev.ResourceID != "resource-2" {
		t.Errorf("ResourceID: got %q, want %q", ev.ResourceID, "resource-2")
	}
	if got := ev.Metadata["app_id"]; got != "app-123" {
		t.Errorf("Metadata[app_id]: got %q, want %q", got, "app-123")
	}
	if ev.Tenant != "app-123" {
		t.Errorf("Tenant: got %q, want the app id from metadata", ev.Tenant)
	}
	if ev.Outcome != hook.OutcomeSuccess {
		t.Errorf("Outcome: got %q, want %q (Record defaults to OutcomeSuccess)", ev.Outcome, hook.OutcomeSuccess)
	}
}

func TestAuditor_RecordEmptyActorBecomesUnknown(t *testing.T) {
	bus, events := captureBus()
	a := dashboard.NewAuditor(bus)
	a.Record(context.Background(), "user.delete", hook.SeverityCritical, "", "res", nil)
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	if (*events)[0].ActorID != "unknown" {
		t.Errorf("ActorID: got %q, want %q (empty actor must become explicit 'unknown')", (*events)[0].ActorID, "unknown")
	}
}

func TestAuditor_RecordWithOutcomeOverrides(t *testing.T) {
	bus, events := captureBus()
	a := dashboard.NewAuditor(bus)
	a.RecordWithOutcome(context.Background(), "user.delete", hook.SeverityCritical, hook.OutcomeFailure, "actor", "res",
		map[string]string{"error": "boom"})
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	if (*events)[0].Outcome != hook.OutcomeFailure {
		t.Errorf("Outcome: got %q, want %q", (*events)[0].Outcome, hook.OutcomeFailure)
	}
	if (*events)[0].Reason != "boom" {
		t.Errorf("Reason: got %q, want the error text", (*events)[0].Reason)
	}
}

func TestAuditor_NilAuditorSafe(t *testing.T) {
	t.Helper()
	var a *dashboard.Auditor
	// Must not panic.
	a.Record(context.Background(), "x", hook.SeverityInfo, "", "", nil)
}

func TestAuditor_NilBusSafe(t *testing.T) {
	t.Helper()
	a := dashboard.NewAuditor(nil)
	// Must not panic.
	a.Record(context.Background(), "x", hook.SeverityInfo, "", "", nil)
}
