package sso

import (
	"context"
	"testing"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	memstore "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

// TestOnBeforeSignIn_Enforcement covers the SSO-required veto and its owner
// break-glass.
func TestOnBeforeSignIn_Enforcement(t *testing.T) {
	ctx := context.Background()
	appID := id.NewAppID()
	orgID := id.NewOrgID()

	setup := func(t *testing.T, enforced, asOwner bool) (*Plugin, string) {
		t.Helper()
		ssoMem := NewMemoryStore()
		coreMem := memstore.New()
		p := &Plugin{ssoStore: ssoMem, store: coreMem}

		if err := ssoMem.CreateConnection(ctx, &Connection{
			ID:       id.NewSSOConnectionID(),
			AppID:    appID,
			OrgID:    orgID,
			Provider: "example.com",
			Protocol: "saml",
			Domain:   "example.com",
			Active:   true,
			Enforced: enforced,
		}); err != nil {
			t.Fatalf("seed connection: %v", err)
		}

		email := "user@example.com"
		u := &user.User{ID: id.NewUserID(), AppID: appID, Email: email}
		if err := coreMem.CreateUserWithPrimaryEmail(ctx, u, user.NewPrimaryEmail(u, "test")); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if asOwner {
			if err := coreMem.CreateMember(ctx, &organization.Member{
				ID:     id.NewMemberID(),
				OrgID:  orgID,
				UserID: u.ID,
				Role:   organization.RoleOwner,
			}); err != nil {
				t.Fatalf("seed member: %v", err)
			}
		}
		return p, email
	}

	req := func(email string) *account.SignInRequest {
		return &account.SignInRequest{AppID: appID, Email: email, Password: "x"}
	}

	t.Run("enforced domain vetoes password login", func(t *testing.T) {
		p, email := setup(t, true, false)
		if err := p.OnBeforeSignIn(ctx, req(email)); err == nil {
			t.Fatal("expected password login to be vetoed for an enforced domain")
		}
	})

	t.Run("owner bypasses enforcement (break-glass)", func(t *testing.T) {
		p, email := setup(t, true, true)
		if err := p.OnBeforeSignIn(ctx, req(email)); err != nil {
			t.Fatalf("owner should bypass, got %v", err)
		}
	})

	t.Run("non-enforced domain passes through", func(t *testing.T) {
		p, email := setup(t, false, false)
		if err := p.OnBeforeSignIn(ctx, req(email)); err != nil {
			t.Fatalf("non-enforced domain must pass, got %v", err)
		}
	})

	t.Run("unrelated domain passes through", func(t *testing.T) {
		p, _ := setup(t, true, false)
		if err := p.OnBeforeSignIn(ctx, req("someone@example.net")); err != nil {
			t.Fatalf("unrelated domain must pass, got %v", err)
		}
	})
}
