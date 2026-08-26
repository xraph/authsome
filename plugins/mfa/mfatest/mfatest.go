// Package mfatest provides a backend-agnostic conformance suite for the
// mfa.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
package mfatest

import (
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
)

// Fixture is one backend ready to test, plus the users its rows hang off.
type Fixture struct {
	Store  mfa.Store
	AppID  id.AppID
	UserID id.UserID
	// OtherUserID is a second user under the same app, used to prove that
	// enrollments and recovery codes are scoped to one user.
	OtherUserID id.UserID
}

// Factory builds a fresh, empty, migrated fixture for a single test.
type Factory func(t *testing.T) Fixture

// RunConformance runs every contract test against fixtures from newFixture.
func RunConformance(t *testing.T, newFixture Factory, skip ...string) {
	t.Helper()
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}
	cases := []struct {
		name string
		fn   func(t *testing.T, f Fixture)
	}{
		{"EnrollmentCRUD", testEnrollmentCRUD},
		{"EnrollmentNotFound", testEnrollmentNotFound},
		{"SecretRoundTrip", testSecretRoundTrip},
		{"UpdateEnrollmentVerified", testUpdateEnrollmentVerified},
		{"EnrollmentLookupIsScopedToUser", testEnrollmentLookupIsScopedToUser},
		{"ListEnrollmentsIsScopedToUser", testListEnrollmentsIsScopedToUser},
		{"DeleteEnrollment", testDeleteEnrollment},
		{"RecoveryCodesRoundTrip", testRecoveryCodesRoundTrip},
		{"UnusedRecoveryCodeHasNoUsedAt", testUnusedRecoveryCodeHasNoUsedAt},
		{"ConsumeRecoveryCodeMarksUsed", testConsumeRecoveryCodeMarksUsed},
		{"ConsumeRecoveryCodeIsSingleUse", testConsumeRecoveryCodeIsSingleUse},
		{"RecoveryCodesAreScopedToUser", testRecoveryCodesAreScopedToUser},
		{"DeleteRecoveryCodes", testDeleteRecoveryCodes},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// newEnrollment builds a TOTP enrollment. The secret is the RFC 4648 test
// vector, not a credential.
func newEnrollment(userID id.UserID) *mfa.Enrollment {
	return &mfa.Enrollment{ // #nosec G101 -- fixed test vector, not a real secret
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "JBSWY3DPEHPK3PXP",
		CreatedAt: now(),
		UpdatedAt: now(),
	}
}

func newRecoveryCode(userID id.UserID) *mfa.RecoveryCode {
	return &mfa.RecoveryCode{
		ID:        id.NewRecoveryCodeID(),
		UserID:    userID,
		CodeHash:  "$2a$04$abcdefghijklmnopqrstuvwxyz012345678901234567890123456",
		CreatedAt: now(),
	}
}
