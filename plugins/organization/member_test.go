package organization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
)

// TestRemoveMember_VetoedByBeforeHookLeavesMemberInPlace proves the mutation
// the reviewer found: deleting the EmitBeforeMemberRemove call (or its
// error-check) from RemoveMember previously left the removal succeeding
// unconditionally, with nothing in this package noticing. If a
// BeforeMemberRemove subscriber errors, the member must still be a member
// afterwards — the removal has to actually abort, not just report failure
// while removing anyway.
func TestRemoveMember_VetoedByBeforeHookLeavesMemberInPlace(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()

	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  s.org.ID,
		UserID: id.NewUserID(),
		Role:   organization.RoleMember,
	}
	require.NoError(t, s.plugin.AddMember(ctx, m))

	vetoErr := errors.New("synthetic before-member-remove veto")
	secutil.OnBeforeMemberRemove(t, s.eng, func(_ context.Context, _ *organization.Member) error {
		return vetoErr
	})

	err := s.plugin.RemoveMember(ctx, m.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, vetoErr)

	got, err := s.eng.Store().GetMember(ctx, m.ID)
	require.NoError(t, err, "member must still exist after a vetoed removal")
	require.NotNil(t, got)
	assert.Equal(t, m.UserID, got.UserID)
}

// TestRemoveMember_RetryAfterSuccessIsIdempotent pins the fix for the
// idempotency regression the reviewer found: store.DeleteMember is a
// documented no-op on an already-missing row, matching every SQL backend, so
// a retried RemoveMember must not turn into an error just because
// RemoveMember now resolves the member first.
func TestRemoveMember_RetryAfterSuccessIsIdempotent(t *testing.T) {
	s := newTxTestSetup(t)
	ctx := context.Background()

	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  s.org.ID,
		UserID: id.NewUserID(),
		Role:   organization.RoleMember,
	}
	require.NoError(t, s.plugin.AddMember(ctx, m))

	require.NoError(t, s.plugin.RemoveMember(ctx, m.ID))

	// A retried delete of an already-removed member (e.g. a client retry
	// after a dropped response) must still succeed.
	err := s.plugin.RemoveMember(ctx, m.ID)
	assert.NoError(t, err, "retried RemoveMember on an already-removed member must be idempotent")
}
