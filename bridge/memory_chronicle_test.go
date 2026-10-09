package bridge

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryChronicleRecordsAndFails(t *testing.T) {
	m := NewMemoryChronicle()
	if err := m.Record(context.Background(), &AuditEvent{Action: "a", Metadata: map[string]string{"k": "v"}}); err != nil {
		t.Fatal(err)
	}
	got := m.Events()
	if len(got) != 1 || got[0].Action != "a" || got[0].Metadata["k"] != "v" {
		t.Fatalf("Events = %+v", got)
	}
	m.FailWith(errors.New("down"))
	if err := m.Record(context.Background(), &AuditEvent{Action: "b"}); err == nil {
		t.Fatal("expected failure")
	}
	m.Reset()
	if len(m.Events()) != 0 {
		t.Fatal("Reset should clear events")
	}
}
