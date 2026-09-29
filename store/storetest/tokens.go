package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// legacySessionSeeder is the test seam every backend exposes to write a
// session the way releases before token hashing did: plaintext, no hashes.
type legacySessionSeeder interface {
	SeedLegacySession(ctx context.Context, sess *session.Session) error
}

// testSessionTokensStoredAsHashes proves a session read back by id carries no
// plaintext, that both hashes are the shared digest, and that a lookup by
// either plaintext returns exactly that plaintext and nothing else.
func testSessionTokensStoredAsHashes(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "hashed@test.com")
	sess := seedSession(t, s, tn, u.ID, "plain-access", "plain-refresh")

	assert.Equal(t, store.HashToken("plain-access"), sess.TokenHash, "create fills the caller's access hash")
	assert.Equal(t, store.HashToken("plain-refresh"), sess.RefreshTokenHash, "create fills the caller's refresh hash")
	assert.Equal(t, "plain-access", sess.Token, "create leaves the caller's plaintext alone")

	byID, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "no access plaintext at rest")
	assert.Empty(t, byID.RefreshToken, "no refresh plaintext at rest")
	assert.Equal(t, store.HashToken("plain-access"), byID.TokenHash)
	assert.Equal(t, store.HashToken("plain-refresh"), byID.RefreshTokenHash)

	byTok, err := s.GetSessionByToken(ctx, "plain-access")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byTok.ID.String())
	assert.Equal(t, "plain-access", byTok.Token, "the presented plaintext rides back on the session")
	assert.Empty(t, byTok.RefreshToken, "the other credential stays hidden")
	assert.Equal(t, store.HashToken("plain-access"), byTok.TokenHash)

	byRef, err := s.GetSessionByRefreshToken(ctx, "plain-refresh")
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), byRef.ID.String())
	assert.Equal(t, "plain-refresh", byRef.RefreshToken)
	assert.Empty(t, byRef.Token)
	assert.Equal(t, store.HashToken("plain-access"), byRef.TokenHash, "the access hash is what the refresh CAS keys on")

	_, err = s.GetSessionByToken(ctx, store.HashToken("plain-access"))
	assert.ErrorIs(t, err, store.ErrNotFound, "presenting the stored digest itself must not authenticate")
	_, err = s.GetSessionByToken(ctx, "")
	assert.ErrorIs(t, err, store.ErrNotFound, "an empty token never matches a row with empty plaintext")

	// An update through a session read by id must not erase the hashes.
	byID.UserAgent = "updated"
	require.NoError(t, s.UpdateSession(ctx, byID))
	again, err := s.GetSessionByToken(ctx, "plain-access")
	require.NoError(t, err)
	assert.Equal(t, "updated", again.UserAgent)
}

// testLegacyPlaintextSessionUpgrades proves a row written before hashing is
// still found by its plaintext, is rewritten as hashes on first use, and is
// then found by hash alone.
func testLegacyPlaintextSessionUpgrades(t *testing.T, s store.Store) {
	seeder, ok := s.(legacySessionSeeder)
	require.True(t, ok, "every backend must expose SeedLegacySession for this case")
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "legacy@test.com")
	sess := seedSession(t, s, tn, u.ID, "legacy-access", "legacy-refresh")
	require.NoError(t, s.DeleteSession(ctx, sess.ID))
	sess.TokenHash, sess.RefreshTokenHash = "", ""
	require.NoError(t, seeder.SeedLegacySession(ctx, sess))

	got, err := s.GetSessionByToken(ctx, "legacy-access")
	require.NoError(t, err, "a legacy plaintext row must still authenticate")
	assert.Equal(t, sess.ID.String(), got.ID.String())
	assert.Equal(t, store.HashToken("legacy-access"), got.TokenHash, "the upgraded row's hash rides back")

	byID, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "the first lookup rewrites the row without plaintext")
	assert.Empty(t, byID.RefreshToken)
	assert.Equal(t, store.HashToken("legacy-access"), byID.TokenHash)
	assert.Equal(t, store.HashToken("legacy-refresh"), byID.RefreshTokenHash, "the refresh token is upgraded in the same write")

	byRef, err := s.GetSessionByRefreshToken(ctx, "legacy-refresh")
	require.NoError(t, err, "the upgraded row is found by hash")
	assert.Equal(t, sess.ID.String(), byRef.ID.String())

	// A legacy row first touched through its refresh token upgrades too.
	other := seedSession(t, s, tn, u.ID, "legacy-access-2", "legacy-refresh-2")
	require.NoError(t, s.DeleteSession(ctx, other.ID))
	other.TokenHash, other.RefreshTokenHash = "", ""
	require.NoError(t, seeder.SeedLegacySession(ctx, other))
	_, err = s.GetSessionByRefreshToken(ctx, "legacy-refresh-2")
	require.NoError(t, err)
	byID, err = s.GetSession(ctx, other.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token)
	assert.Equal(t, store.HashToken("legacy-access-2"), byID.TokenHash)
}

// testRevokeFamilyUsesStoredHashes proves family revocation marks each
// sibling's refresh hash revoked without ever needing the plaintext.
func testRevokeFamilyUsesStoredHashes(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "family@test.com")
	a := seedSession(t, s, tn, u.ID, "fam-access-a", "fam-refresh-a")
	b := seedSession(t, s, tn, u.ID, "fam-access-b", "fam-refresh-b")
	b.FamilyID = a.FamilyID
	require.NoError(t, s.UpdateSession(ctx, b))

	require.NoError(t, s.RevokeRefreshTokenFamily(ctx, a.FamilyID, session.RevokeReasonReplayDetected))

	for _, plain := range []string{"fam-refresh-a", "fam-refresh-b"} {
		revoked, err := s.IsRefreshTokenRevoked(ctx, store.HashToken(plain))
		require.NoError(t, err)
		assert.True(t, revoked, "sibling %s must be in the revoked set", plain)
	}
	_, err := s.GetSession(ctx, a.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	_, err = s.GetSession(ctx, b.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// legacyCredentialSeeder is the test seam every backend exposes to write
// verifications, password resets and invitations the way releases before
// token hashing did: plaintext, no hash.
type legacyCredentialSeeder interface {
	SeedLegacyVerification(ctx context.Context, v *account.Verification) error
	SeedLegacyPasswordReset(ctx context.Context, pr *account.PasswordReset) error
	SeedLegacyInvitation(ctx context.Context, inv *organization.Invitation) error
}

func seedVerification(t *testing.T, s store.Store, tn tenant, userID id.UserID, token string) *account.Verification {
	t.Helper()
	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: userID,
		Token: token, Type: account.VerificationEmail,
		ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, s.CreateVerification(context.Background(), v))
	return v
}

func seedPasswordReset(t *testing.T, s store.Store, tn tenant, userID id.UserID, token string) *account.PasswordReset {
	t.Helper()
	pr := &account.PasswordReset{
		ID: id.NewPasswordResetID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: userID,
		Token: token, ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, s.CreatePasswordReset(context.Background(), pr))
	return pr
}

func seedInvitation(t *testing.T, s store.Store, orgID id.OrgID, inviter id.UserID, token string) *organization.Invitation {
	t.Helper()
	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: orgID, Email: token + "@invite.test", Role: organization.RoleMember,
		InviterID: inviter, Status: organization.InvitationPending, Token: token,
		ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, s.CreateInvitation(context.Background(), inv))
	return inv
}

// testCredentialTokensStoredAsHashes proves verifications, password resets
// and invitations keep only a digest at rest, answer to their plaintext, and
// consume exactly once.
func testCredentialTokensStoredAsHashes(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "cred-hash@test.com")

	v := seedVerification(t, s, tn, u.ID, "verify-plain")
	assert.Equal(t, store.HashToken("verify-plain"), v.TokenHash, "create fills the caller's hash")
	assert.Equal(t, "verify-plain", v.Token, "create leaves the caller's plaintext alone")
	byUser, err := s.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, byUser.Token, "no plaintext at rest")
	assert.Equal(t, store.HashToken("verify-plain"), byUser.TokenHash)
	byTok, err := s.GetVerification(ctx, "verify-plain")
	require.NoError(t, err)
	assert.Equal(t, v.ID.String(), byTok.ID.String())
	assert.Equal(t, "verify-plain", byTok.Token, "the presented plaintext rides back")
	_, err = s.GetVerification(ctx, store.HashToken("verify-plain"))
	assert.ErrorIs(t, err, store.ErrNotFound, "the digest itself is not a credential")
	byUser.Attempts = 2
	require.NoError(t, s.UpdateVerification(ctx, byUser), "an update through a hash-only read must find the row")
	again, err := s.GetVerification(ctx, "verify-plain")
	require.NoError(t, err)
	assert.Equal(t, 2, again.Attempts)
	require.NoError(t, s.ConsumeVerification(ctx, "verify-plain"))
	assert.ErrorIs(t, s.ConsumeVerification(ctx, "verify-plain"), store.ErrNotFound, "single use")

	pr := seedPasswordReset(t, s, tn, u.ID, "reset-plain")
	assert.Equal(t, store.HashToken("reset-plain"), pr.TokenHash)
	got, err := s.GetPasswordReset(ctx, "reset-plain")
	require.NoError(t, err)
	assert.Equal(t, pr.ID.String(), got.ID.String())
	assert.Equal(t, "reset-plain", got.Token)
	assert.Equal(t, store.HashToken("reset-plain"), got.TokenHash)
	_, err = s.GetPasswordReset(ctx, store.HashToken("reset-plain"))
	assert.ErrorIs(t, err, store.ErrNotFound)
	require.NoError(t, s.ConsumePasswordReset(ctx, "reset-plain"))
	got, err = s.GetPasswordReset(ctx, "reset-plain")
	require.NoError(t, err)
	assert.True(t, got.Consumed)

	org := &organization.Organization{ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Hashes", Slug: "hashes-" + suffix(tn.AppID.String()), CreatedBy: u.ID, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateOrganization(ctx, org))
	inv := seedInvitation(t, s, org.ID, u.ID, "invite-plain")
	assert.Equal(t, store.HashToken("invite-plain"), inv.TokenHash)
	byID, err := s.GetInvitation(ctx, inv.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "no plaintext at rest")
	assert.Equal(t, store.HashToken("invite-plain"), byID.TokenHash)
	byInvTok, err := s.GetInvitationByToken(ctx, "invite-plain")
	require.NoError(t, err)
	assert.Equal(t, inv.ID.String(), byInvTok.ID.String())
	assert.Equal(t, "invite-plain", byInvTok.Token)
	_, err = s.GetInvitationByToken(ctx, store.HashToken("invite-plain"))
	assert.ErrorIs(t, err, store.ErrNotFound)
	byID.Status = organization.InvitationAccepted
	require.NoError(t, s.UpdateInvitation(ctx, byID), "an update through a hash-only read keeps the hash")
	byInvTok, err = s.GetInvitationByToken(ctx, "invite-plain")
	require.NoError(t, err)
	assert.Equal(t, organization.InvitationAccepted, byInvTok.Status)
}

// testLegacyPlaintextCredentialsUpgrade proves rows written before hashing
// still answer to their plaintext, are rewritten as hashes on first use, and
// are then found by hash alone.
func testLegacyPlaintextCredentialsUpgrade(t *testing.T, s store.Store) {
	seeder, ok := s.(legacyCredentialSeeder)
	require.True(t, ok, "every backend must expose the legacy credential seams")
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "cred-legacy@test.com")

	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: u.ID,
		Token: "legacy-verify", Type: account.VerificationEmail,
		ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, seeder.SeedLegacyVerification(ctx, v))
	got, err := s.GetVerification(ctx, "legacy-verify")
	require.NoError(t, err, "a legacy plaintext verification must still be found")
	assert.Equal(t, v.ID.String(), got.ID.String())
	byUser, err := s.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, byUser.Token, "the first lookup rewrites the row without plaintext")
	assert.Equal(t, store.HashToken("legacy-verify"), byUser.TokenHash)
	require.NoError(t, s.ConsumeVerification(ctx, "legacy-verify"), "consume finds the upgraded row by hash")

	pr := &account.PasswordReset{
		ID: id.NewPasswordResetID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: u.ID,
		Token: "legacy-reset", ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, seeder.SeedLegacyPasswordReset(ctx, pr))
	require.NoError(t, s.ConsumePasswordReset(ctx, "legacy-reset"), "consume must reach a legacy plaintext row")
	gotPR, err := s.GetPasswordReset(ctx, "legacy-reset")
	require.NoError(t, err)
	assert.True(t, gotPR.Consumed)
	assert.Equal(t, store.HashToken("legacy-reset"), gotPR.TokenHash)

	org := &organization.Organization{ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Legacy", Slug: "legacy-" + suffix(tn.AppID.String()), CreatedBy: u.ID, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateOrganization(ctx, org))
	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: org.ID, Email: "legacy@invite.test", Role: organization.RoleMember,
		InviterID: u.ID, Status: organization.InvitationPending, Token: "legacy-invite",
		ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, seeder.SeedLegacyInvitation(ctx, inv))
	gotInv, err := s.GetInvitationByToken(ctx, "legacy-invite")
	require.NoError(t, err, "a legacy plaintext invitation must still be found")
	assert.Equal(t, inv.ID.String(), gotInv.ID.String())
	byID, err := s.GetInvitation(ctx, inv.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token, "the first lookup rewrites the row without plaintext")
	assert.Equal(t, store.HashToken("legacy-invite"), byID.TokenHash)
}

// testHashLegacyTokensConverts proves the start-up sweep rewrites every kind
// of plaintext credential row and reports when nothing is left.
func testHashLegacyTokensConverts(t *testing.T, s store.Store) {
	sessSeeder, ok := s.(legacySessionSeeder)
	require.True(t, ok)
	credSeeder, ok := s.(legacyCredentialSeeder)
	require.True(t, ok)
	ctx := context.Background()

	// Drain anything an earlier case left behind so the counts below are exact.
	for range 100 {
		n, err := s.HashLegacyTokens(ctx, 100)
		require.NoError(t, err)
		if n == 0 {
			break
		}
	}

	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "sweep@test.com")
	sess := seedSession(t, s, tn, u.ID, "sweep-access", "sweep-refresh")
	require.NoError(t, s.DeleteSession(ctx, sess.ID))
	sess.TokenHash, sess.RefreshTokenHash = "", ""
	require.NoError(t, sessSeeder.SeedLegacySession(ctx, sess))
	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: u.ID,
		Token: "sweep-verify", Type: account.VerificationEmail, ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, credSeeder.SeedLegacyVerification(ctx, v))
	pr := &account.PasswordReset{
		ID: id.NewPasswordResetID(), AppID: tn.AppID, EnvID: tn.EnvID, UserID: u.ID,
		Token: "sweep-reset", ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, credSeeder.SeedLegacyPasswordReset(ctx, pr))
	org := &organization.Organization{ID: id.NewOrgID(), AppID: tn.AppID, EnvID: tn.EnvID, Name: "Sweep", Slug: "sweep-" + suffix(tn.AppID.String()), CreatedBy: u.ID, CreatedAt: now(), UpdatedAt: now()}
	require.NoError(t, s.CreateOrganization(ctx, org))
	inv := &organization.Invitation{
		ID: id.NewInvitationID(), OrgID: org.ID, Email: "sweep@invite.test", Role: organization.RoleMember,
		InviterID: u.ID, Status: organization.InvitationPending, Token: "sweep-invite",
		ExpiresAt: now().Add(time.Hour), CreatedAt: now(),
	}
	require.NoError(t, credSeeder.SeedLegacyInvitation(ctx, inv))

	// A batch smaller than the backlog converts exactly that many.
	n, err := s.HashLegacyTokens(ctx, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "the batch bound is honoured")
	n, err = s.HashLegacyTokens(ctx, 100)
	require.NoError(t, err)
	assert.EqualValues(t, 3, n, "the rest of the backlog converts in one call")
	n, err = s.HashLegacyTokens(ctx, 100)
	require.NoError(t, err)
	assert.Zero(t, n, "a complete sweep reports nothing left")

	got, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Token)
	assert.Equal(t, store.HashToken("sweep-access"), got.TokenHash)
	assert.Equal(t, store.HashToken("sweep-refresh"), got.RefreshTokenHash)
	_, err = s.GetSessionByRefreshToken(ctx, "sweep-refresh")
	require.NoError(t, err, "a converted session answers to its plaintext by hash")
	byUser, err := s.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, byUser.Token)
	assert.Equal(t, store.HashToken("sweep-verify"), byUser.TokenHash)
	gotPR, err := s.GetPasswordReset(ctx, "sweep-reset")
	require.NoError(t, err)
	assert.Equal(t, store.HashToken("sweep-reset"), gotPR.TokenHash)
	byID, err := s.GetInvitation(ctx, inv.ID)
	require.NoError(t, err)
	assert.Empty(t, byID.Token)
	assert.Equal(t, store.HashToken("sweep-invite"), byID.TokenHash)
}
