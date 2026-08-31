package sso

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/authsome/id"
	memstore "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

// TestLinkableExistingUser covers the SSO account-linking guard, including the
// password-credential rule: an invited-but-never-activated account (no password)
// is safe to link on SSO, while a self-registered account (has a password) is not.
func TestLinkableExistingUser(t *testing.T) {
	ctx := context.Background()
	appID := id.NewAppID()
	var envID id.EnvironmentID // nil env → memory store matches app-wide

	newPlugin := func() (*Plugin, *memstore.Store) {
		st := memstore.New()
		return &Plugin{store: st}, st
	}
	seed := func(t *testing.T, st *memstore.Store, email, passwordHash string, verified bool) {
		t.Helper()
		u := &user.User{
			ID:           id.NewUserID(),
			AppID:        appID,
			Email:        email,
			PasswordHash: passwordHash,
		}
		pe := user.NewPrimaryEmail(u, "test")
		pe.Verified = verified
		if err := st.CreateUserWithPrimaryEmail(ctx, u, pe); err != nil {
			t.Fatalf("seed %q: %v", email, err)
		}
	}

	t.Run("no account returns nil,nil (caller creates fresh)", func(t *testing.T) {
		p, _ := newPlugin()
		u, err := p.linkableExistingUser(ctx, appID, envID, "nobody@example.com")
		if u != nil || err != nil {
			t.Fatalf("got (%v, %v), want (nil, nil)", u, err)
		}
	})

	t.Run("verified email links", func(t *testing.T) {
		p, st := newPlugin()
		seed(t, st, "verified@example.com", "pwhash", true)
		u, err := p.linkableExistingUser(ctx, appID, envID, "verified@example.com")
		if u == nil || err != nil {
			t.Fatalf("got (%v, %v), want linked", u, err)
		}
	})

	t.Run("unverified invited (no password) links and gets verified", func(t *testing.T) {
		p, st := newPlugin()
		seed(t, st, "invited@example.com", "", false)
		u, err := p.linkableExistingUser(ctx, appID, envID, "invited@example.com")
		if u == nil || err != nil {
			t.Fatalf("got (%v, %v), want linked", u, err)
		}
		rec, _ := st.GetUserEmailRecord(ctx, appID, envID, "invited@example.com")
		if rec == nil || !rec.Verified {
			t.Fatal("email should be marked verified after linking an invited account")
		}
	})

	t.Run("unverified self-signup (has password) is refused", func(t *testing.T) {
		p, st := newPlugin()
		seed(t, st, "attacker@example.com", "pwhash", false)
		_, err := p.linkableExistingUser(ctx, appID, envID, "attacker@example.com")
		if !errors.Is(err, errUnverifiedSSOLink) {
			t.Fatalf("got %v, want errUnverifiedSSOLink", err)
		}
	})
}
