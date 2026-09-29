package authsome_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/rbac"
)

// A role change reaches the user's live sessions at once: the stamped
// roles on every session row are rewritten, so a revoked role stops
// working on the next request rather than at the next refresh.

func stampedRoles(t *testing.T, eng *authsome.Engine, uid id.UserID) []string {
	t.Helper()
	sessions, err := eng.Store().ListUserSessions(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	return sessions[0].Roles
}

func TestRoleChange_RestampsLiveSessions(t *testing.T) {
	eng, _ := newBootstrapEngine(t)
	ctx := context.Background()
	appID := eng.PlatformAppID()

	// A second verified user so this one is a plain user, not the owner.
	_ = signUpVerified(t, eng, "owner@example.com")
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "member@example.com", Password: "SecureP@ss123", FirstName: "M"})
	require.NoError(t, err)
	secutil.VerifyEmail(t, eng, u.ID)
	assert.NotContains(t, stampedRoles(t, eng, u.ID), "admin")

	admin, err := eng.GetRoleBySlug(ctx, appID, "admin")
	require.NoError(t, err)
	require.NotNil(t, admin)
	require.NoError(t, eng.AssignUserRole(ctx, &rbac.UserRole{UserID: u.ID.String(), RoleID: admin.ID}))
	assert.Contains(t, stampedRoles(t, eng, u.ID), "admin", "a grant reaches the live session")

	suffix := admin.ID[strings.IndexByte(admin.ID, '_')+1:]
	roleID, err := id.ParseRoleID("arol_" + suffix)
	require.NoError(t, err)
	require.NoError(t, eng.UnassignUserRole(ctx, u.ID, roleID))
	assert.NotContains(t, stampedRoles(t, eng, u.ID), "admin", "a revocation reaches the live session")
}
