package chronicleadapter

import (
	"context"
	"testing"

	"github.com/xraph/chronicle"
	"github.com/xraph/chronicle/audit"
	"github.com/xraph/chronicle/store"
	"github.com/xraph/chronicle/store/memory"

	"github.com/xraph/authsome/bridge"
)

func TestAdapterMapsEveryField(t *testing.T) {
	mem := memory.New()
	c, err := chronicle.New(chronicle.WithStore(store.NewAdapter(mem)))
	if err != nil {
		t.Fatal(err)
	}
	a := New(c)
	err = a.Record(context.Background(), &bridge.AuditEvent{
		Action: "auth.signin", Resource: "session", ResourceID: "sess_1",
		ActorID: "user_1", Tenant: "app_1", OrgID: "org_1",
		Outcome: bridge.OutcomeSuccess, Severity: bridge.SeverityInfo, Category: "auth",
		Reason: "ok", IP: "203.0.113.9", UserAgent: "curl/8.0", RequestID: "req_1", SessionID: "sess_1",
		Metadata: map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := mem.Query(context.Background(), &audit.Query{AppID: "app_1"})
	if err != nil || len(res.Events) != 1 {
		t.Fatalf("query: %v, %d events", err, len(res.Events))
	}
	ev := res.Events[0]
	if ev.AppID != "app_1" || ev.TenantID != "org_1" || ev.UserID != "user_1" {
		t.Errorf("scope: %+v", ev)
	}
	if ev.IP != "203.0.113.9" || ev.UserAgent != "curl/8.0" || ev.RequestID != "req_1" || ev.SessionID != "sess_1" {
		t.Errorf("correlation: %+v", ev)
	}
	if ev.Metadata["k"] != "v" || ev.Reason != "ok" {
		t.Errorf("payload: %+v", ev)
	}
}
