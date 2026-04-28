package authsome_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
)

// fakeOrgPlugin satisfies plugin.Plugin (for registry lookup) AND
// the unexported orgMembershipChecker contract used by SwitchActiveOrg.
// We can't reference orgMembershipChecker by name (engine-internal),
// but the registry lookup uses a type assertion so any struct exposing
// the right method satisfies it.
type fakeOrgPlugin struct {
	orgs map[string][]id.OrgID // userID string -> org IDs the user is a member of
}

func (f *fakeOrgPlugin) Name() string { return "organization" }

func (f *fakeOrgPlugin) ListUserOrganizations(_ context.Context, userID id.UserID) ([]*organization.Organization, error) {
	orgIDs := f.orgs[userID.String()]
	out := make([]*organization.Organization, 0, len(orgIDs))
	for _, oid := range orgIDs {
		out = append(out, &organization.Organization{ID: oid})
	}
	return out, nil
}

func TestSwitchActiveOrg_unknownSession_returnsError(t *testing.T) {
	eng, _ := newTestEngine(t, authsome.WithPlugin(&fakeOrgPlugin{}))
	_, err := eng.SwitchActiveOrg(context.Background(), id.NewSessionID(), id.NewOrgID())
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "get session"), "want 'get session' wrap, got %v", err)
}

func TestSwitchActiveOrg_clearsActiveOrg(t *testing.T) {
	eng, store := newTestEngine(t, authsome.WithPlugin(&fakeOrgPlugin{}))
	ctx := context.Background()

	sess := newSeedSession(t, store, ctx, id.NewOrgID())

	// Empty newOrgID is allowed and clears the active org.
	updated, err := eng.SwitchActiveOrg(ctx, sess.ID, id.OrgID{})
	require.NoError(t, err)
	require.True(t, updated.OrgID.IsNil(), "OrgID should be cleared, got %s", updated.OrgID)

	// Persisted in the store.
	stored, err := store.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, stored.OrgID.IsNil(), "stored session OrgID not cleared")
}

func TestSwitchActiveOrg_membershipRequired(t *testing.T) {
	uid := id.NewUserID()
	memberOrg := id.NewOrgID()
	otherOrg := id.NewOrgID()

	eng, store := newTestEngine(t, authsome.WithPlugin(&fakeOrgPlugin{
		orgs: map[string][]id.OrgID{
			uid.String(): {memberOrg},
		},
	}))
	ctx := context.Background()

	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: uid,
		Token:  "tok",
	}
	require.NoError(t, store.CreateSession(ctx, sess))

	// Switching to an org the user is NOT a member of should fail.
	_, err := eng.SwitchActiveOrg(ctx, sess.ID, otherOrg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a member")

	// Switching to a member org succeeds.
	updated, err := eng.SwitchActiveOrg(ctx, sess.ID, memberOrg)
	require.NoError(t, err)
	require.Equal(t, memberOrg.String(), updated.OrgID.String())
}

func TestSwitchActiveOrg_pluginMissing_returnsError(t *testing.T) {
	// No org plugin registered → switch with a non-empty newOrgID
	// must error rather than panic on the nil plugin.
	eng, store := newTestEngine(t)
	ctx := context.Background()
	sess := newSeedSession(t, store, ctx, id.OrgID{})

	_, err := eng.SwitchActiveOrg(ctx, sess.ID, id.NewOrgID())
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization plugin")
}

// newSeedSession persists a fresh session for a synthetic user and
// returns it. The session has no OrgID by default unless the caller
// passes one.
func newSeedSession(t *testing.T, store interface {
	CreateSession(ctx context.Context, s *session.Session) error
}, ctx context.Context, orgID id.OrgID) *session.Session {
	t.Helper()
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
		OrgID:  orgID,
		Token:  "tok",
	}
	require.NoError(t, store.CreateSession(ctx, sess))
	return sess
}
