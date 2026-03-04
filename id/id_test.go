package id

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────
// ID generation tests
// ──────────────────────────────────────────────────

func TestNewUserID_HasCorrectPrefix(t *testing.T) {
	id := NewUserID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixUser)+"_"))
}

func TestNewSessionID_HasCorrectPrefix(t *testing.T) {
	id := NewSessionID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixSession)+"_"))
}

func TestNewAppID_HasCorrectPrefix(t *testing.T) {
	id := NewAppID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixApp)+"_"))
}

func TestNewOrgID_HasCorrectPrefix(t *testing.T) {
	id := NewOrgID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixOrg)+"_"))
}

func TestNewAPIKeyID_HasCorrectPrefix(t *testing.T) {
	id := NewAPIKeyID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixAPIKey)+"_"))
}

func TestNewMFAID_HasCorrectPrefix(t *testing.T) {
	id := NewMFAID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixMFA)+"_"))
}

func TestNewDeviceID_HasCorrectPrefix(t *testing.T) {
	id := NewDeviceID()
	assert.True(t, strings.HasPrefix(id.String(), string(PrefixDevice)+"_"))
}

// ──────────────────────────────────────────────────
// Uniqueness
// ──────────────────────────────────────────────────

func TestNewUserID_Unique(t *testing.T) {
	a := NewUserID()
	b := NewUserID()
	assert.NotEqual(t, a.String(), b.String())
}

func TestNewSessionID_Unique(t *testing.T) {
	a := NewSessionID()
	b := NewSessionID()
	assert.NotEqual(t, a.String(), b.String())
}

// ──────────────────────────────────────────────────
// Parse round-trip tests
// ──────────────────────────────────────────────────

func TestParseUserID_RoundTrip(t *testing.T) {
	original := NewUserID()
	parsed, err := ParseUserID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}

func TestParseSessionID_RoundTrip(t *testing.T) {
	original := NewSessionID()
	parsed, err := ParseSessionID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}

func TestParseAppID_RoundTrip(t *testing.T) {
	original := NewAppID()
	parsed, err := ParseAppID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}

func TestParseOrgID_RoundTrip(t *testing.T) {
	original := NewOrgID()
	parsed, err := ParseOrgID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}

func TestParseAPIKeyID_RoundTrip(t *testing.T) {
	original := NewAPIKeyID()
	parsed, err := ParseAPIKeyID(original.String())
	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}

// ──────────────────────────────────────────────────
// Parse error cases
// ──────────────────────────────────────────────────

func TestParseUserID_WrongPrefix(t *testing.T) {
	sessID := NewSessionID()
	_, err := ParseUserID(sessID.String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected prefix")
}

func TestParseUserID_InvalidString(t *testing.T) {
	_, err := ParseUserID("not-a-valid-id")
	require.Error(t, err)
}

func TestParseSessionID_WrongPrefix(t *testing.T) {
	userID := NewUserID()
	_, err := ParseSessionID(userID.String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected prefix")
}

func TestParse_EmptyString(t *testing.T) {
	_, err := Parse("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty string")
}

func TestParseWithPrefix_WrongPrefix(t *testing.T) {
	userID := NewUserID()
	_, err := ParseWithPrefix(userID.String(), PrefixApp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected prefix")
}

func TestParseAny_AcceptsAnyPrefix(t *testing.T) {
	userID := NewUserID()
	parsed, err := ParseAny(userID.String())
	require.NoError(t, err)
	assert.Equal(t, userID.String(), parsed.String())

	sessID := NewSessionID()
	parsed2, err := ParseAny(sessID.String())
	require.NoError(t, err)
	assert.Equal(t, sessID.String(), parsed2.String())
}

// ──────────────────────────────────────────────────
// Nil ID
// ──────────────────────────────────────────────────

func TestNilID(t *testing.T) {
	assert.True(t, Nil.IsNil())
	assert.Equal(t, "", Nil.String())
	assert.Equal(t, Prefix(""), Nil.Prefix())

	var zero ID
	assert.True(t, zero.IsNil())
	assert.Equal(t, "", zero.String())
}

func TestNewID_IsNotNil(t *testing.T) {
	id := NewUserID()
	assert.False(t, id.IsNil())
}

// ──────────────────────────────────────────────────
// MustParse
// ──────────────────────────────────────────────────

func TestMustParse_Valid(t *testing.T) {
	original := NewUserID()
	parsed := MustParse(original.String())
	assert.Equal(t, original.String(), parsed.String())
}

func TestMustParse_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustParse("invalid")
	})
}

func TestMustParseWithPrefix_Panics(t *testing.T) {
	userID := NewUserID()
	assert.Panics(t, func() {
		MustParseWithPrefix(userID.String(), PrefixApp)
	})
}

// ──────────────────────────────────────────────────
// MarshalText / UnmarshalText
// ──────────────────────────────────────────────────

func TestMarshalUnmarshalText_RoundTrip(t *testing.T) {
	original := NewUserID()
	data, err := original.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, original.String(), string(data))

	var restored ID
	err = restored.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, original.String(), restored.String())
	assert.False(t, restored.IsNil())
}

func TestMarshalText_Nil(t *testing.T) {
	data, err := Nil.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte{}, data)
}

func TestUnmarshalText_Empty(t *testing.T) {
	var id ID
	err := id.UnmarshalText([]byte{})
	require.NoError(t, err)
	assert.True(t, id.IsNil())
}

// ──────────────────────────────────────────────────
// Value / Scan (database driver)
// ──────────────────────────────────────────────────

func TestValueScan_RoundTrip(t *testing.T) {
	original := NewAppID()
	val, err := original.Value()
	require.NoError(t, err)
	assert.Equal(t, original.String(), val)

	var scanned ID
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, original.String(), scanned.String())
	assert.False(t, scanned.IsNil())
}

func TestValue_Nil(t *testing.T) {
	val, err := Nil.Value()
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestScan_Nil(t *testing.T) {
	var id ID
	err := id.Scan(nil)
	require.NoError(t, err)
	assert.True(t, id.IsNil())
}

func TestScan_EmptyString(t *testing.T) {
	var id ID
	err := id.Scan("")
	require.NoError(t, err)
	assert.True(t, id.IsNil())
}

func TestScan_Bytes(t *testing.T) {
	original := NewOrgID()
	var scanned ID
	err := scanned.Scan([]byte(original.String()))
	require.NoError(t, err)
	assert.Equal(t, original.String(), scanned.String())
}

func TestScan_EmptyBytes(t *testing.T) {
	var id ID
	err := id.Scan([]byte{})
	require.NoError(t, err)
	assert.True(t, id.IsNil())
}

func TestScan_UnsupportedType(t *testing.T) {
	var id ID
	err := id.Scan(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot scan")
}

// ──────────────────────────────────────────────────
// String() produces non-empty output
// ──────────────────────────────────────────────────

func TestIDString_NotEmpty(t *testing.T) {
	ids := []struct {
		name string
		fn   func() ID
	}{
		{"UserID", func() ID { return NewUserID() }},
		{"SessionID", func() ID { return NewSessionID() }},
		{"AppID", func() ID { return NewAppID() }},
		{"OrgID", func() ID { return NewOrgID() }},
		{"MemberID", func() ID { return NewMemberID() }},
		{"TeamID", func() ID { return NewTeamID() }},
		{"DeviceID", func() ID { return NewDeviceID() }},
		{"APIKeyID", func() ID { return NewAPIKeyID() }},
		{"MFAID", func() ID { return NewMFAID() }},
		{"RoleID", func() ID { return NewRoleID() }},
		{"PermissionID", func() ID { return NewPermissionID() }},
	}

	for _, tc := range ids {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.fn()
			assert.NotEmpty(t, id.String())
		})
	}
}

func TestBSONRoundTrip(t *testing.T) {
	original := NewUserID()

	// Marshal to BSON.
	bsonType, data, err := original.MarshalBSONValue()
	require.NoError(t, err)
	assert.Equal(t, byte(0x02), bsonType, "expected BSON string type")

	// Unmarshal back.
	var restored ID
	require.NoError(t, restored.UnmarshalBSONValue(bsonType, data))
	assert.Equal(t, original.String(), restored.String())

	// Nil round-trip.
	var nilID ID
	bsonType, data, err = nilID.MarshalBSONValue()
	require.NoError(t, err)
	assert.Equal(t, byte(0x0A), bsonType, "expected BSON null type")

	var restored2 ID
	require.NoError(t, restored2.UnmarshalBSONValue(bsonType, data))
	assert.True(t, restored2.IsNil())
}

func TestBSONUnmarshalInvalidType(t *testing.T) {
	var restored ID
	err := restored.UnmarshalBSONValue(0x01, []byte{0x00, 0x00, 0x00, 0x00})
	assert.Error(t, err)
}
