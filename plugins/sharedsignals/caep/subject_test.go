package caep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubjectID_IssSub(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"format":"iss_sub","iss":"https://org.okta.com","sub":"okta-user-id1"}`))
	require.NoError(t, err)
	assert.Equal(t, "iss_sub", got.Format)
	assert.Equal(t, "https://org.okta.com", got.Issuer)
	assert.Equal(t, "okta-user-id1", got.Subject)
	assert.False(t, got.IsComplex())
}

func TestParseSubjectID_Email(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(`{"format":"email","email":"a@b.com"}`))
	require.NoError(t, err)
	assert.Equal(t, "email", got.Format)
	assert.Equal(t, "a@b.com", got.Email)
}

func TestParseSubjectID_Complex(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"user":{"format":"iss_sub","iss":"https://i","sub":"u1"},` +
			`"session":{"format":"opaque","id":"sess-9"}}`))
	require.NoError(t, err)
	assert.True(t, got.IsComplex())

	user, ok := got.Member("user")
	require.True(t, ok)
	assert.Equal(t, "u1", user.Subject)

	sess, ok := got.Member("session")
	require.True(t, ok)
	assert.Equal(t, "sess-9", sess.ID)

	_, ok = got.Member("device")
	assert.False(t, ok)
}

func TestParseSubjectID_Aliases(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"format":"aliases","identifiers":[` +
			`{"format":"email","email":"a@b.com"},` +
			`{"format":"iss_sub","iss":"https://i","sub":"u1"}]}`))
	require.NoError(t, err)
	assert.Equal(t, "aliases", got.Format)
	require.Len(t, got.Identifiers, 2)
	assert.Equal(t, "a@b.com", got.Identifiers[0].Email)
	assert.Equal(t, "u1", got.Identifiers[1].Subject)
}

// Nested aliases are forbidden by RFC 9493 and would let a sender build an
// arbitrarily deep structure for us to walk.
func TestParseSubjectID_NestedAliasesRejected(t *testing.T) {
	_, err := ParseSubjectID(json.RawMessage(
		`{"format":"aliases","identifiers":[{"format":"aliases","identifiers":[]}]}`))
	require.Error(t, err)
}

func TestParseSubjectID_Malformed(t *testing.T) {
	_, err := ParseSubjectID(json.RawMessage(`"just-a-string"`))
	require.Error(t, err)
}
