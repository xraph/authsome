package dashboard_test

import (
	"context"
	"testing"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/internal/secutil"
)

func TestAuditor_RecordPopulatesEvent(t *testing.T) {
	ch := secutil.NewBufferedChronicle()
	a := dashboard.NewAuditor(ch)

	meta := map[string]string{"app_id": "app-123"}
	a.Record(context.Background(),
		"user.delete",
		bridge.SeverityCritical,
		"actor-1",
		"resource-2",
		meta,
	)

	events := ch.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Action != "user.delete" {
		t.Errorf("Action: got %q, want %q", ev.Action, "user.delete")
	}
	if ev.Severity != bridge.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", ev.Severity, bridge.SeverityCritical)
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
	if ev.Outcome != bridge.OutcomeSuccess {
		t.Errorf("Outcome: got %q, want %q (Record defaults to OutcomeSuccess)", ev.Outcome, bridge.OutcomeSuccess)
	}
}

func TestAuditor_RecordEmptyActorBecomesUnknown(t *testing.T) {
	ch := secutil.NewBufferedChronicle()
	a := dashboard.NewAuditor(ch)
	a.Record(context.Background(), "user.delete", bridge.SeverityCritical, "", "res", nil)
	events := ch.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ActorID != "unknown" {
		t.Errorf("ActorID: got %q, want %q (empty actor must become explicit 'unknown')", events[0].ActorID, "unknown")
	}
}

func TestAuditor_RecordWithOutcomeOverrides(t *testing.T) {
	ch := secutil.NewBufferedChronicle()
	a := dashboard.NewAuditor(ch)
	a.RecordWithOutcome(context.Background(), "user.delete", bridge.SeverityCritical, bridge.OutcomeFailure, "actor", "res", nil)
	events := ch.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Outcome != bridge.OutcomeFailure {
		t.Errorf("Outcome: got %q, want %q", events[0].Outcome, bridge.OutcomeFailure)
	}
}

func TestAuditor_NilAuditorSafe(t *testing.T) {
	var a *dashboard.Auditor
	// Must not panic.
	a.Record(context.Background(), "x", bridge.SeverityInfo, "", "", nil)
}

func TestAuditor_NilChronicleSafe(t *testing.T) {
	a := dashboard.NewAuditor(nil)
	// Must not panic.
	a.Record(context.Background(), "x", bridge.SeverityInfo, "", "", nil)
}
