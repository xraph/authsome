package webhook

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignBody_RoundTripVerifies(t *testing.T) {
	t.Parallel()
	secret := []byte("super-secret-32-bytes-or-more!!")
	body := []byte(`{"event":"user.created"}`)
	ts := time.Now()

	header := SignBody(secret, ts, body)
	require.Contains(t, header, "t=")
	require.Contains(t, header, "v1=")

	require.NoError(t, VerifySignature(secret, body, header, 0))
}

func TestSignBody_TimestampIsBoundIntoMAC(t *testing.T) {
	t.Parallel()
	secret := []byte("k")
	body := []byte("payload")

	at := time.Unix(1_700_000_000, 0)
	header := SignBody(secret, at, body)

	// Mutate just the timestamp, leaving the v1= signature intact.
	// The new (body, signature) pair must NOT verify, even within
	// tolerance, because the MAC covered the original ts.
	tampered := "t=1700000050," + extractV1(t, header)
	err := verifySignatureAt(secret, body, tampered, time.Hour, at.Add(30*time.Second))
	require.ErrorIs(t, err, ErrSignatureMismatch,
		"swapping the timestamp must invalidate the signature; otherwise an attacker could replay")
}

func TestVerifySignature_TamperedBodyFails(t *testing.T) {
	t.Parallel()
	secret := []byte("k")
	header := SignBody(secret, time.Now(), []byte("original"))
	err := VerifySignature(secret, []byte("ORIGINAL"), header, 0)
	require.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestVerifySignature_WrongSecretFails(t *testing.T) {
	t.Parallel()
	body := []byte("payload")
	header := SignBody([]byte("secret-A"), time.Now(), body)
	err := VerifySignature([]byte("secret-B"), body, header, 0)
	require.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestVerifySignature_StaleTimestampFails(t *testing.T) {
	t.Parallel()
	secret := []byte("k")
	body := []byte("p")
	old := time.Unix(1_700_000_000, 0)

	header := SignBody(secret, old, body)
	err := verifySignatureAt(secret, body, header, time.Minute, old.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrTimestampSkew,
		"signatures older than tolerance must be rejected even if the MAC is correct")
}

func TestVerifySignature_FutureTimestampWithinToleranceOK(t *testing.T) {
	t.Parallel()
	secret := []byte("k")
	body := []byte("p")

	// Sender 30s ahead of receiver — within default tolerance.
	now := time.Unix(1_700_000_000, 0)
	header := SignBody(secret, now.Add(30*time.Second), body)
	require.NoError(t, verifySignatureAt(secret, body, header, time.Minute, now))
}

func TestVerifySignature_MalformedHeaderFails(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"empty":       "",
		"no equals":   "t,1700000000,v1,abc",
		"missing v1":  "t=1700000000",
		"missing t":   "v1=abc",
		"non-int ts":  "t=banana,v1=ab",
		"non-hex sig": "t=1700000000,v1=zzzz",
	}
	for name, header := range cases {
		header := header
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := VerifySignature([]byte("k"), []byte("p"), header, 0)
			require.Error(t, err, "header %q must not verify", header)
		})
	}
}

func TestVerifySignature_UnknownVersionDistinguishable(t *testing.T) {
	t.Parallel()
	// A v2-only signature must surface ErrUnknownVersion so receivers
	// can distinguish "spec mismatch" from "secret mismatch".
	header := "t=1700000000,v2=deadbeef"
	err := VerifySignature([]byte("k"), []byte("p"), header, 0)
	require.True(t, errors.Is(err, ErrUnknownVersion),
		"v2-only header must surface ErrUnknownVersion, got %v", err)
}

func TestSignatureHeader_Constant(t *testing.T) {
	t.Parallel()
	require.Equal(t, "X-Authsome-Signature", SignatureHeader)
	require.Equal(t, "v1", SignatureVersion)
}

// extractV1 returns the 'v1=...' segment of a header so tests can
// rebuild a tampered header without re-implementing the format here.
func extractV1(t *testing.T, header string) string {
	t.Helper()
	for _, part := range splitCSV(header) {
		if len(part) > 3 && part[:3] == "v1=" {
			return part
		}
	}
	t.Fatalf("header %q has no v1= segment", header)
	return ""
}

func splitCSV(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
