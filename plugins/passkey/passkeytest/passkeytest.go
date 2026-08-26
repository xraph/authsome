// Package passkeytest provides a backend-agnostic conformance suite for the
// passkey.Store interface. Every backend (memory, sqlite, postgres, mongo) is
// expected to pass the same contract, so behavioral drift between them is
// caught here rather than in production.
//
// Passkeys are a sharper test than most stores: credential ids, public keys
// and AAGUIDs are raw binary, and the sign count is a uint32 that the SQL
// backends have to hold in a signed column.
package passkeytest

import (
	"testing"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/passkey"
)

// Fixture is one backend ready to test, plus the tenant rows its foreign keys
// point at.
type Fixture struct {
	Store  passkey.Store
	AppID  id.AppID
	UserID id.UserID
	// OtherUserID is a second user under the same app, used to prove that
	// credential listing is scoped to one user.
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
		{"CredentialCRUD", testCredentialCRUD},
		{"CredentialNotFound", testCredentialNotFound},
		{"BinaryFieldsRoundTrip", testBinaryFieldsRoundTrip},
		{"TransportRoundTrip", testTransportRoundTrip},
		{"EmptyTransportRoundTrip", testEmptyTransportRoundTrip},
		{"ListIsScopedToUser", testListIsScopedToUser},
		{"UpdateSignCount", testUpdateSignCount},
		{"SignCountHoldsFullUint32", testSignCountHoldsFullUint32},
		{"DeleteCredential", testDeleteCredential},
	}
	for _, tc := range cases {
		if skipSet[tc.name] {
			continue
		}
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newFixture(t)) })
	}
}

func now() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// uniqueBytes returns a credential id that is unique per call and contains
// byte values that break naive text handling: a NUL, a 0xFF, and a byte
// sequence that is not valid UTF-8.
func uniqueBytes() []byte {
	raw := id.NewPasskeyID().String()
	prefix := []byte{0x00, 0xFF, 0xC3, 0x28}
	out := make([]byte, 0, len(prefix)+len(raw))
	out = append(out, prefix...)
	return append(out, raw...)
}

func newCredential(f Fixture, userID id.UserID) *passkey.Credential {
	return &passkey.Credential{
		ID:              id.NewPasskeyID(),
		UserID:          userID,
		AppID:           f.AppID,
		CredentialID:    uniqueBytes(),
		PublicKey:       []byte{0xA5, 0x01, 0x02, 0x03, 0x26, 0x20, 0x01, 0x00, 0xFF},
		AttestationType: "none",
		Transport:       []string{"internal", "hybrid"},
		SignCount:       1,
		AAGUID:          []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DisplayName:     "Test Key",
		CreatedAt:       now(),
		UpdatedAt:       now(),
	}
}
