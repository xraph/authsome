package id_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestNewAgentID_HasAgentPrefix(t *testing.T) {
	a := id.NewAgentID()
	assert.Equal(t, id.PrefixAgent, a.Prefix())
}

func TestParseAgentID_RejectsForeignPrefix(t *testing.T) {
	grant := id.NewAgentGrantID()

	_, err := id.ParseAgentID(grant.String())

	require.Error(t, err, "an agent grant id must not parse as an agent id")
}

func TestParseAgentGrantID_RoundTrips(t *testing.T) {
	original := id.NewAgentGrantID()

	parsed, err := id.ParseAgentGrantID(original.String())

	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}
