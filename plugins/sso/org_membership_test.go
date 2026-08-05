package sso

import (
	"context"
	"testing"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store/memory"
)

// TestEnsureOrgMembership verifies the org-level SSO enrollment: the first SSO
// login into an org-scoped connection adds the user as a member, and repeated
// logins are idempotent (no duplicate membership).
func TestEnsureOrgMembership(t *testing.T) {
	mem := memory.New()
	p := New()
	p.SetStore(mem)

	ctx := context.Background()
	orgID := id.NewOrgID()
	userID := id.NewUserID()

	// First SSO login → user is enrolled into the org.
	p.ensureOrgMembership(ctx, orgID, userID)

	members, err := mem.ListMembers(ctx, orgID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member after enrollment, got %d", len(members))
	}
	if members[0].UserID != userID {
		t.Errorf("member UserID = %s, want %s", members[0].UserID, userID)
	}
	if members[0].OrgID != orgID {
		t.Errorf("member OrgID = %s, want %s", members[0].OrgID, orgID)
	}

	// Second login for the same user → idempotent, still exactly one member.
	p.ensureOrgMembership(ctx, orgID, userID)
	members, err = mem.ListMembers(ctx, orgID)
	if err != nil {
		t.Fatalf("ListMembers (2nd): %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected membership to stay idempotent (1), got %d", len(members))
	}

	// A different user logging in via the same connection is added alongside.
	otherUser := id.NewUserID()
	p.ensureOrgMembership(ctx, orgID, otherUser)
	members, err = mem.ListMembers(ctx, orgID)
	if err != nil {
		t.Fatalf("ListMembers (3rd): %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members after a second user, got %d", len(members))
	}
}
