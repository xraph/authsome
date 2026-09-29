package mfa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// A deleted user's second factors go with the account.
func TestOnBeforeUserDelete_RemovesEnrollments(t *testing.T) {
	p := New()
	st := NewMemoryStore()
	p.SetStore(st)
	ctx := context.Background()
	userID, other := id.NewUserID(), id.NewUserID()
	require.NoError(t, st.CreateEnrollment(ctx, &Enrollment{ID: id.NewMFAID(), UserID: userID, Method: "totp", Secret: "s", Verified: true}))
	require.NoError(t, st.CreateEnrollment(ctx, &Enrollment{ID: id.NewMFAID(), UserID: other, Method: "totp", Secret: "o", Verified: true}))

	require.NoError(t, p.OnBeforeUserDelete(ctx, userID))

	mine, err := st.ListEnrollments(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, mine)
	theirs, err := st.ListEnrollments(ctx, other)
	require.NoError(t, err)
	assert.Len(t, theirs, 1, "another user's enrollments stay")
}
