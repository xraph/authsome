package bridge

import (
	"context"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/hook"
)

func TestBusChronicleForwardsToTheBus(t *testing.T) {
	bus := hook.NewBus(log.NewNoopLogger())
	var got *hook.Event
	bus.On("capture", func(_ context.Context, ev *hook.Event) error { got = ev; return nil })

	c := NewBusChronicle(bus)
	ctx := hook.WithRequestInfo(context.Background(), hook.RequestInfo{IP: "203.0.113.9", RequestID: "req_1"})
	err := c.Record(ctx, &AuditEvent{
		Action: "mfa.enroll", Resource: "mfa", ResourceID: "enr_1", ActorID: "user_1", Tenant: "app_1",
		Outcome: OutcomeSuccess, Severity: SeverityWarning, Category: "auth", Metadata: map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("event did not reach the bus")
	}
	if got.Action != "mfa.enroll" || got.Tenant != "app_1" || got.Severity != SeverityWarning || got.Metadata["k"] != "v" {
		t.Errorf("event fields lost: %+v", got)
	}
	if got.IP != "203.0.113.9" || got.RequestID != "req_1" {
		t.Errorf("request info not applied by the bus: %+v", got)
	}
}

func TestBusChronicleNilSafe(t *testing.T) {
	var c *BusChronicle
	if err := c.Record(context.Background(), &AuditEvent{Action: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := NewBusChronicle(nil).Record(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
