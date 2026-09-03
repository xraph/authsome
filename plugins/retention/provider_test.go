package retention

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestCapabilityHas(t *testing.T) {
	both := CapContacts | CapActivities
	assert.True(t, both.Has(CapContacts))
	assert.True(t, both.Has(CapActivities))

	contactsOnly := CapContacts
	assert.True(t, contactsOnly.Has(CapContacts))
	assert.False(t, contactsOnly.Has(CapActivities),
		"a contacts-only CRM must not advertise activity logging")
}

func TestRetryableUnwrapsProviderError(t *testing.T) {
	pe := &ProviderError{Err: errors.New("429 slow down"), Retryable: true, RetryAfter: 30 * time.Second}
	ok, after := Retryable(pe)
	require.True(t, ok)
	assert.Equal(t, 30*time.Second, after)
}

func TestRetryableWrappedProviderError(t *testing.T) {
	pe := &ProviderError{Err: errors.New("boom"), Retryable: true}
	ok, after := Retryable(fmt.Errorf("hubspot: %w", pe))
	require.True(t, ok, "Retryable must see through fmt.Errorf wrapping")
	assert.Zero(t, after)
}

func TestRetryablePlainErrorIsTerminal(t *testing.T) {
	ok, after := Retryable(errors.New("unclassified"))
	assert.False(t, ok, "an unclassified error must not be retried forever")
	assert.Zero(t, after)
}

func TestRemoteRefEmpty(t *testing.T) {
	assert.True(t, RemoteRef{}.IsZero())
	assert.False(t, RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "501"}.IsZero())
}

func TestRetentionIDPrefixes(t *testing.T) {
	j := id.NewRetentionJobID()
	_, err := id.ParseRetentionJobID(j.String())
	require.NoError(t, err)

	_, err = id.ParseRetentionRefID(j.String())
	assert.Error(t, err, "a job id must not parse as a ref id")
}
