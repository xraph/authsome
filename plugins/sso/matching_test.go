package sso

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	memory "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

func seedUserWithEmail(t *testing.T, s *memory.Store, appID id.AppID, envID id.EnvironmentID, email string, verified bool, passwordHash string) *user.User {
	t.Helper()
	u := &user.User{
		ID:            id.NewUserID(),
		AppID:         appID,
		EnvID:         envID,
		Email:         email,
		EmailVerified: verified,
		PasswordHash:  passwordHash,
	}
	row := &user.UserEmail{
		ID:        id.NewUserEmailID(),
		UserID:    u.ID,
		AppID:     appID,
		EnvID:     envID,
		Email:     email,
		Verified:  verified,
		IsPrimary: true,
		Source:    "test",
	}
	require.NoError(t, s.CreateUserWithPrimaryEmail(context.Background(), u, row))
	return u
}

// TestLinkableExistingUser_RefusesUnverified is the account-takeover guard: an
// SSO login whose email matches a *pre-existing but unverified* local account
// must NOT link to it. Otherwise an attacker who pre-registers the victim's
// email (unverified) would capture the victim's later SSO login.
func TestLinkableExistingUser_RefusesUnverified(t *testing.T) {
	s := memory.New()
	p := New()
	p.SetStore(s)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	// A self-registered account: unverified email AND a password credential an
	// attacker could have set. Linking SSO to it must still be refused.
	seedUserWithEmail(t, s, appID, envID, "victim@corp.com", false, "attacker-set-password-hash")

	got, err := p.linkableExistingUser(context.Background(), appID, envID, "victim@corp.com")

	require.Error(t, err, "linking to an unverified password-bearing account must be refused")
	assert.Nil(t, got)
}

// TestLinkableExistingUser_LinksVerified confirms the legitimate path: a
// verified local account is linked to.
func TestLinkableExistingUser_LinksVerified(t *testing.T) {
	s := memory.New()
	p := New()
	p.SetStore(s)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	u := seedUserWithEmail(t, s, appID, envID, "member@corp.com", true, "")

	got, err := p.linkableExistingUser(context.Background(), appID, envID, "member@corp.com")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID.String(), got.ID.String())
}

// TestLinkableExistingUser_NoMatchCreatesFresh confirms that when no local
// account exists, the guard signals "no match" (nil user, nil error) so the
// caller creates a fresh SSO user.
func TestLinkableExistingUser_NoMatchCreatesFresh(t *testing.T) {
	s := memory.New()
	p := New()
	p.SetStore(s)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	got, err := p.linkableExistingUser(context.Background(), appID, envID, "nobody@corp.com")

	require.NoError(t, err)
	assert.Nil(t, got)
}
