package rbac

import "testing"

func TestParser_Parse(t *testing.T) {
	p := NewParser()

	policy, err := p.Parse("user:read,write on project:* where owner = true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if policy.Subject != "user" {
		t.Fatalf("subject mismatch: %s", policy.Subject)
	}

	if len(policy.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(policy.Actions))
	}

	if policy.Actions[0] != "read" || policy.Actions[1] != "write" {
		t.Fatalf("actions mismatch: %+v", policy.Actions)
	}

	if policy.Resource != "project:*" {
		t.Fatalf("resource mismatch: %s", policy.Resource)
	}

	if policy.Condition != "owner = true" {
		t.Fatalf("condition mismatch: %s", policy.Condition)
	}
}
