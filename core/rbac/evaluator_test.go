package rbac

import "testing"

func TestEvaluator_Evaluate(t *testing.T) {
	p := &Policy{Subject: "user", Actions: []string{"read", "write"}, Resource: "project:*"}
	e := NewEvaluator()

	ok := e.Evaluate(p, &Context{Subject: "user", Action: "read", Resource: "project:123"})
	if !ok {
		t.Fatalf("expected allowed for matching action and resource")
	}

	// wrong subject
	if e.Evaluate(p, &Context{Subject: "role:admin", Action: "read", Resource: "project:123"}) {
		t.Fatalf("expected deny for different subject")
	}

	// wrong action
	if e.Evaluate(p, &Context{Subject: "user", Action: "delete", Resource: "project:123"}) {
		t.Fatalf("expected deny for action not in policy")
	}

	// condition check
	p.Condition = "role = admin"

	ok = e.Evaluate(p, &Context{Subject: "user", Action: "read", Resource: "project:123", Vars: map[string]string{"role": "admin"}})
	if !ok {
		t.Fatalf("expected allowed when condition matches")
	}

	ok = e.Evaluate(p, &Context{Subject: "user", Action: "read", Resource: "project:123", Vars: map[string]string{"role": "user"}})
	if ok {
		t.Fatalf("expected deny when condition fails")
	}
}
