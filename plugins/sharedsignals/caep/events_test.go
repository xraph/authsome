package caep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The payload shape Okta actually emits, which uses "subject" rather than
// the "sub_id" the final CAEP spec names.
func TestParseEvent_OktaSessionRevoked(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"iss_sub","iss":"https://org.okta.com","sub":"okta-user-id1"},
		"reason_admin": {"en":"User logout from Okta"},
		"event_timestamp": 1615304991643
	}`)

	ev, err := ParseEvent(EventSessionRevoked, payload)
	require.NoError(t, err)
	assert.Equal(t, EventSessionRevoked, ev.Type)
	assert.Equal(t, "okta-user-id1", ev.Subject.Subject)
	assert.Equal(t, "User logout from Okta", ev.ReasonAdmin["en"])
	assert.Equal(t, int64(1615304991643), ev.EventTimestamp)
}

func TestParseEvent_SubIDPreferredOverSubject(t *testing.T) {
	payload := json.RawMessage(`{
		"sub_id": {"format":"opaque","id":"from-sub-id"},
		"subject": {"format":"opaque","id":"from-subject"}
	}`)

	ev, err := ParseEvent(EventSessionRevoked, payload)
	require.NoError(t, err)
	assert.Equal(t, "from-sub-id", ev.Subject.ID)
}

func TestParseEvent_CredentialChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"iss_sub","iss":"https://i","sub":"u1"},
		"credential_type": "fido2-roaming",
		"change_type": "create",
		"friendly_name": "FIDO_WEBAUTHN",
		"initiating_entity": "user"
	}`)

	ev, err := ParseEvent(EventCredentialChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "fido2-roaming", ev.CredentialType)
	assert.Equal(t, "create", ev.ChangeType)
	assert.Equal(t, "user", ev.InitiatingEntity)
}

func TestParseEvent_AssuranceLevelChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"namespace": "NIST-AAL",
		"current_level": "aal1",
		"previous_level": "aal2",
		"change_direction": "decrease"
	}`)

	ev, err := ParseEvent(EventAssuranceLevelChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "decrease", ev.ChangeDirection)
	assert.Equal(t, "aal1", ev.CurrentLevel)
	assert.Equal(t, "aal2", ev.PreviousLevel)
}

func TestParseEvent_DeviceComplianceChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"previous_status": "compliant",
		"current_status": "not-compliant"
	}`)

	ev, err := ParseEvent(EventDeviceComplianceChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "not-compliant", ev.CurrentStatus)
	assert.Equal(t, "compliant", ev.PreviousStatus)
}

func TestParseEvent_TokenClaimsChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"claims": {"role":"admin"}
	}`)

	ev, err := ParseEvent(EventTokenClaimsChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "admin", ev.Claims["role"])
}

// The SSF verification event carries the stream_id as an opaque subject and a
// state the receiver chose.
func TestParseEvent_Verification(t *testing.T) {
	payload := json.RawMessage(`{
		"sub_id": {"format":"opaque","id":"stream-1"},
		"state": "abc123"
	}`)

	ev, err := ParseEvent(EventVerification, payload)
	require.NoError(t, err)
	assert.Equal(t, "abc123", ev.State)
	assert.Equal(t, "stream-1", ev.Subject.ID)
}

func TestParseEvent_MissingSubject(t *testing.T) {
	_, err := ParseEvent(EventSessionRevoked, json.RawMessage(`{"reason_admin":{"en":"x"}}`))
	require.Error(t, err)
}

func TestIsKnownEventType(t *testing.T) {
	assert.True(t, IsKnownEventType(EventSessionRevoked))
	assert.False(t, IsKnownEventType("https://example.com/not-a-caep-event"))
}
