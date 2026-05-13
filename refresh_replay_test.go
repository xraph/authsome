package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

// TestRefresh_ChainedRotations exercises the legitimate happy path: two
// consecutive Refresh calls each succeed and produce fresh tokens, while the
// session FamilyID is preserved across rotations.
func TestRefresh_ChainedRotations(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "chain@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Chain User",
	})
	require.NoError(t, err)
	require.False(t, sess.FamilyID.IsNil(), "fresh sign-in must mint a FamilyID")
	originalFamily := sess.FamilyID.String()
	tok1 := sess.RefreshToken

	// First rotation
	s2, err := eng.Refresh(ctx, tok1)
	require.NoError(t, err)
	assert.NotEqual(t, tok1, s2.RefreshToken)
	assert.Equal(t, originalFamily, s2.FamilyID.String(), "family inherits across rotation")
	tok2 := s2.RefreshToken

	// Second rotation off the freshly-minted token works.
	s3, err := eng.Refresh(ctx, tok2)
	require.NoError(t, err)
	assert.NotEqual(t, tok2, s3.RefreshToken)
	assert.Equal(t, originalFamily, s3.FamilyID.String())
}

// TestRefresh_ReplayRevokesFamily exercises the attack path: after a single
// rotation, replaying the OLD token must (a) be rejected with a generic
// ErrInvalidCredentials and (b) cascade-revoke the active session in the
// family. Subsequent attempts to use the freshly-minted token are also
// refused (the family is dead).
func TestRefresh_ReplayRevokesFamily(t *testing.T) {
	eng, s := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "replay@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Replay User",
	})
	require.NoError(t, err)
	originalRefresh := sess.RefreshToken

	// Legitimate rotation by the real client.
	s2, err := eng.Refresh(ctx, originalRefresh)
	require.NoError(t, err)
	require.NotEmpty(t, s2.RefreshToken)
	survivorID := s2.ID
	rotatedToken := s2.RefreshToken

	// Attacker replays the original refresh token.
	_, err = eng.Refresh(ctx, originalRefresh)
	assert.ErrorIs(t, err, account.ErrInvalidCredentials,
		"replayed token must yield generic ErrInvalidCredentials")

	// The family's surviving session is now revoked.
	_, err = s.GetSession(ctx, survivorID)
	assert.ErrorIs(t, err, store.ErrNotFound,
		"replay detection must cascade-revoke siblings in the family")

	// Even the legitimate-but-now-orphaned rotated token is dead.
	_, err = eng.Refresh(ctx, rotatedToken)
	assert.Error(t, err,
		"rotated token belonging to a revoked family must not be usable")
}

// TestRefresh_UnrelatedSessionUntouched ensures family cascade revocation
// is scoped — sessions in a *different* family (e.g. a second login) are
// not affected when one family is poisoned.
func TestRefresh_UnrelatedSessionUntouched(t *testing.T) {
	eng, s := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// Two independent sign-ins for the same user produce two families.
	_, sessA, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "two-fams@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Two Families",
	})
	require.NoError(t, err)
	tokenA := sessA.RefreshToken

	_, sessB, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "two-fams@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	require.NotEqual(t, sessA.FamilyID.String(), sessB.FamilyID.String(),
		"distinct sign-ins must mint distinct FamilyIDs")

	// Rotate family A once, then replay.
	_, err = eng.Refresh(ctx, tokenA)
	require.NoError(t, err)
	_, err = eng.Refresh(ctx, tokenA)
	require.ErrorIs(t, err, account.ErrInvalidCredentials)

	// Family B's session is untouched and its refresh still works.
	got, err := s.GetSession(ctx, sessB.ID)
	require.NoError(t, err)
	assert.Equal(t, sessB.ID.String(), got.ID.String())

	s3, err := eng.Refresh(ctx, sessB.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, sessB.FamilyID.String(), s3.FamilyID.String())
}
