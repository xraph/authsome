//go:build integration

package mongo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// TestDeleteApp_CascadesChildren verifies the Mongo DeleteApp removes every
// app-scoped row (users, organizations, org members) rather than orphaning
// them. Requires AUTHSOME_MONGO_URI (a replica set for the transaction).
func TestDeleteApp_CascadesChildren(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	now := time.Now()

	appID := id.NewAppID()
	require.NoError(t, s.CreateApp(ctx, &app.App{
		ID:        appID,
		Name:      "Cascade",
		Slug:      "cascade-" + appID.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}))

	u := &user.User{ID: id.NewUserID(), AppID: appID, Email: "cascade@test.com", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateUser(ctx, u))

	org := &organization.Organization{ID: id.NewOrgID(), AppID: appID, Name: "Org", Slug: "org-" + appID.String(), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.CreateOrganization(ctx, org))
	require.NoError(t, s.CreateMember(ctx, &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID, Role: organization.RoleOwner, CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.DeleteApp(ctx, appID))

	// The app-scoped rows must all be gone.
	_, err := s.GetUser(ctx, u.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "user must be cascaded")

	_, err = s.GetOrganization(ctx, org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "organization must be cascaded")

	members, err := s.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "org members must be cascaded")

	// A second delete of the same app is idempotent (nothing left to remove).
	assert.NoError(t, s.DeleteApp(ctx, appID))
}
