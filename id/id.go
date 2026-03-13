// Package id defines TypeID-based identity types for all AuthSome entities.
//
// Every entity in AuthSome uses a single ID struct with a prefix that identifies
// the entity type. IDs are K-sortable (UUIDv7-based), globally unique,
// and URL-safe in the format "prefix_suffix".
package id

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"

	"go.jetify.com/typeid/v2"
)

// BSON type constants (avoids importing the mongo-driver bson package).
const (
	bsonTypeString byte = 0x02
	bsonTypeNull   byte = 0x0A
)

// Prefix identifies the entity type encoded in a TypeID.
type Prefix string

// Prefix constants for all AuthSome entity types.
const (
	PrefixUser            Prefix = "ausr"
	PrefixSession         Prefix = "ases"
	PrefixApp             Prefix = "aapp"
	PrefixOrg             Prefix = "aorg"
	PrefixMember          Prefix = "amem"
	PrefixTeam            Prefix = "atm"
	PrefixInvitation      Prefix = "ainv"
	PrefixDevice          Prefix = "adev"
	PrefixWebhook         Prefix = "awhk"
	PrefixNotification    Prefix = "antf"
	PrefixVerification    Prefix = "avrf"
	PrefixPasswordReset   Prefix = "apwr"
	PrefixAPIKey          Prefix = "akey"
	PrefixOAuthConnection Prefix = "aoau"
	PrefixPasskey         Prefix = "apsk"
	PrefixMFA             Prefix = "amfa"
	PrefixEnvironment     Prefix = "aenv"
	PrefixRole            Prefix = "arol"
	PrefixPermission      Prefix = "aprm"
	PrefixRecoveryCode    Prefix = "arec"
	PrefixSSOConnection   Prefix = "asso"
	PrefixConsent         Prefix = "acns"
	PrefixFormConfig      Prefix = "afcf"
	PrefixBrandingConfig  Prefix = "abrd"
	PrefixAppSessionCfg   Prefix = "ascf"
	PrefixOAuth2Client    Prefix = "aoac"
	PrefixAuthCode        Prefix = "aaco"
	PrefixSetting         Prefix = "aset"
	PrefixAppClientConfig Prefix = "aacf"
	PrefixDeviceCode      Prefix = "advc"
	PrefixSCIMConfig      Prefix = "ascm"
	PrefixSCIMToken       Prefix = "asct"
	PrefixSCIMLog         Prefix = "ascl"
)

// ID is the primary identifier type for all AuthSome entities.
// It wraps a TypeID providing a prefix-qualified, globally unique,
// sortable, URL-safe identifier in the format "prefix_suffix".
//
//nolint:recvcheck // Value receivers for read-only methods, pointer receivers for UnmarshalText/Scan.
type ID struct {
	inner typeid.TypeID
	valid bool
}

// Nil is the zero-value ID.
var Nil ID

// ──────────────────────────────────────────────────
// Type aliases for backward compatibility
// ──────────────────────────────────────────────────

// UserID is a type-safe identifier for users (prefix: "ausr").
type UserID = ID

// SessionID is a type-safe identifier for sessions (prefix: "ases").
type SessionID = ID

// AppID is a type-safe identifier for apps (prefix: "aapp").
type AppID = ID

// OrgID is a type-safe identifier for organizations (prefix: "aorg").
type OrgID = ID

// MemberID is a type-safe identifier for organization members (prefix: "amem").
type MemberID = ID

// TeamID is a type-safe identifier for teams (prefix: "atm").
type TeamID = ID

// InvitationID is a type-safe identifier for invitations (prefix: "ainv").
type InvitationID = ID

// DeviceID is a type-safe identifier for devices (prefix: "adev").
type DeviceID = ID

// WebhookID is a type-safe identifier for webhooks (prefix: "awhk").
type WebhookID = ID

// NotificationID is a type-safe identifier for notifications (prefix: "antf").
type NotificationID = ID

// VerificationID is a type-safe identifier for verification tokens (prefix: "avrf").
type VerificationID = ID

// PasswordResetID is a type-safe identifier for password reset tokens (prefix: "apwr").
type PasswordResetID = ID

// APIKeyID is a type-safe identifier for API keys (prefix: "akey").
type APIKeyID = ID

// OAuthConnectionID is a type-safe identifier for OAuth connections (prefix: "aoau").
type OAuthConnectionID = ID

// PasskeyID is a type-safe identifier for passkeys (prefix: "apsk").
type PasskeyID = ID

// MFAID is a type-safe identifier for MFA enrollment records (prefix: "amfa").
type MFAID = ID

// EnvironmentID is a type-safe identifier for environments (prefix: "aenv").
type EnvironmentID = ID

// RoleID is a type-safe identifier for roles (prefix: "arol").
type RoleID = ID

// PermissionID is a type-safe identifier for permissions (prefix: "aprm").
type PermissionID = ID

// RecoveryCodeID is a type-safe identifier for MFA recovery codes (prefix: "arec").
type RecoveryCodeID = ID

// SSOConnectionID is a type-safe identifier for SSO connections (prefix: "asso").
type SSOConnectionID = ID

// ConsentID is a type-safe identifier for consent records (prefix: "acns").
type ConsentID = ID

// FormConfigID is a type-safe identifier for form configurations (prefix: "afcf").
type FormConfigID = ID

// BrandingConfigID is a type-safe identifier for branding configurations (prefix: "abrd").
type BrandingConfigID = ID

// AppSessionConfigID is a type-safe identifier for app session configs (prefix: "ascf").
type AppSessionConfigID = ID

// OAuth2ClientID is a type-safe identifier for OAuth2 clients (prefix: "aoac").
type OAuth2ClientID = ID

// AuthCodeID is a type-safe identifier for authorization codes (prefix: "aaco").
type AuthCodeID = ID

// SettingID is a type-safe identifier for settings (prefix: "aset").
type SettingID = ID

// AppClientConfigID is a type-safe identifier for app client configs (prefix: "aacf").
type AppClientConfigID = ID

// DeviceCodeID is a type-safe identifier for OAuth2 device codes (prefix: "advc").
type DeviceCodeID = ID

// SCIMConfigID is a type-safe identifier for SCIM configurations (prefix: "ascm").
type SCIMConfigID = ID

// SCIMTokenID is a type-safe identifier for SCIM bearer tokens (prefix: "asct").
type SCIMTokenID = ID

// SCIMLogID is a type-safe identifier for SCIM provision logs (prefix: "ascl").
type SCIMLogID = ID

// AnyID is a TypeID that accepts any valid prefix.
type AnyID = ID

// ──────────────────────────────────────────────────
// Core functions
// ──────────────────────────────────────────────────

// New generates a new globally unique ID with the given prefix.
// It panics if prefix is not a valid TypeID prefix (programming error).
func New(prefix Prefix) ID {
	tid, err := typeid.Generate(string(prefix))
	if err != nil {
		panic(fmt.Sprintf("id: invalid prefix %q: %v", prefix, err))
	}

	return ID{inner: tid, valid: true}
}

// Parse parses a TypeID string (e.g., "ausr_01h2xcejqtf2nbrexx3vqjhp41")
// into an ID. Returns an error if the string is not valid.
func Parse(s string) (ID, error) {
	if s == "" {
		return Nil, fmt.Errorf("id: parse %q: empty string", s)
	}

	tid, err := typeid.Parse(s)
	if err != nil {
		return Nil, fmt.Errorf("id: parse %q: %w", s, err)
	}

	return ID{inner: tid, valid: true}, nil
}

// ParseWithPrefix parses a TypeID string and validates that its prefix
// matches the expected value.
func ParseWithPrefix(s string, expected Prefix) (ID, error) {
	parsed, err := Parse(s)
	if err != nil {
		return Nil, err
	}

	if parsed.Prefix() != expected {
		return Nil, fmt.Errorf("id: expected prefix %q, got %q", expected, parsed.Prefix())
	}

	return parsed, nil
}

// MustParse is like Parse but panics on error. Use for hardcoded ID values.
func MustParse(s string) ID {
	parsed, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("id: must parse %q: %v", s, err))
	}

	return parsed
}

// MustParseWithPrefix is like ParseWithPrefix but panics on error.
func MustParseWithPrefix(s string, expected Prefix) ID {
	parsed, err := ParseWithPrefix(s, expected)
	if err != nil {
		panic(fmt.Sprintf("id: must parse with prefix %q: %v", expected, err))
	}

	return parsed
}

// ──────────────────────────────────────────────────
// Convenience constructors
// ──────────────────────────────────────────────────

// NewUserID generates a new unique user ID.
func NewUserID() ID { return New(PrefixUser) }

// NewSessionID generates a new unique session ID.
func NewSessionID() ID { return New(PrefixSession) }

// NewAppID generates a new unique app ID.
func NewAppID() ID { return New(PrefixApp) }

// NewOrgID generates a new unique organization ID.
func NewOrgID() ID { return New(PrefixOrg) }

// NewMemberID generates a new unique member ID.
func NewMemberID() ID { return New(PrefixMember) }

// NewTeamID generates a new unique team ID.
func NewTeamID() ID { return New(PrefixTeam) }

// NewInvitationID generates a new unique invitation ID.
func NewInvitationID() ID { return New(PrefixInvitation) }

// NewDeviceID generates a new unique device ID.
func NewDeviceID() ID { return New(PrefixDevice) }

// NewWebhookID generates a new unique webhook ID.
func NewWebhookID() ID { return New(PrefixWebhook) }

// NewNotificationID generates a new unique notification ID.
func NewNotificationID() ID { return New(PrefixNotification) }

// NewVerificationID generates a new unique verification ID.
func NewVerificationID() ID { return New(PrefixVerification) }

// NewPasswordResetID generates a new unique password reset ID.
func NewPasswordResetID() ID { return New(PrefixPasswordReset) }

// NewAPIKeyID generates a new unique API key ID.
func NewAPIKeyID() ID { return New(PrefixAPIKey) }

// NewOAuthConnectionID generates a new unique OAuth connection ID.
func NewOAuthConnectionID() ID { return New(PrefixOAuthConnection) }

// NewPasskeyID generates a new unique passkey ID.
func NewPasskeyID() ID { return New(PrefixPasskey) }

// NewMFAID generates a new unique MFA enrollment ID.
func NewMFAID() ID { return New(PrefixMFA) }

// NewEnvironmentID generates a new unique environment ID.
func NewEnvironmentID() ID { return New(PrefixEnvironment) }

// NewRoleID generates a new unique role ID.
func NewRoleID() ID { return New(PrefixRole) }

// NewPermissionID generates a new unique permission ID.
func NewPermissionID() ID { return New(PrefixPermission) }

// NewRecoveryCodeID generates a new unique recovery code ID.
func NewRecoveryCodeID() ID { return New(PrefixRecoveryCode) }

// NewSSOConnectionID generates a new unique SSO connection ID.
func NewSSOConnectionID() ID { return New(PrefixSSOConnection) }

// NewConsentID generates a new unique consent record ID.
func NewConsentID() ID { return New(PrefixConsent) }

// NewFormConfigID generates a new unique form config ID.
func NewFormConfigID() ID { return New(PrefixFormConfig) }

// NewBrandingConfigID generates a new unique branding config ID.
func NewBrandingConfigID() ID { return New(PrefixBrandingConfig) }

// NewAppSessionConfigID generates a new unique app session config ID.
func NewAppSessionConfigID() ID { return New(PrefixAppSessionCfg) }

// NewOAuth2ClientID generates a new unique OAuth2 client ID.
func NewOAuth2ClientID() ID { return New(PrefixOAuth2Client) }

// NewAuthCodeID generates a new unique authorization code ID.
func NewAuthCodeID() ID { return New(PrefixAuthCode) }

// NewSettingID generates a new unique setting ID.
func NewSettingID() ID { return New(PrefixSetting) }

// NewAppClientConfigID generates a new unique app client config ID.
func NewAppClientConfigID() ID { return New(PrefixAppClientConfig) }

// NewDeviceCodeID generates a new unique device code ID.
func NewDeviceCodeID() ID { return New(PrefixDeviceCode) }

// NewSCIMConfigID generates a new unique SCIM config ID.
func NewSCIMConfigID() ID { return New(PrefixSCIMConfig) }

// NewSCIMTokenID generates a new unique SCIM token ID.
func NewSCIMTokenID() ID { return New(PrefixSCIMToken) }

// NewSCIMLogID generates a new unique SCIM provision log ID.
func NewSCIMLogID() ID { return New(PrefixSCIMLog) }

// ──────────────────────────────────────────────────
// Convenience parsers
// ──────────────────────────────────────────────────

// ParseUserID parses a string and validates the "ausr" prefix.
func ParseUserID(s string) (ID, error) { return ParseWithPrefix(s, PrefixUser) }

// ParseSessionID parses a string and validates the "ases" prefix.
func ParseSessionID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSession) }

// ParseAppID parses a string and validates the "aapp" prefix.
func ParseAppID(s string) (ID, error) { return ParseWithPrefix(s, PrefixApp) }

// ParseOrgID parses a string and validates the "aorg" prefix.
func ParseOrgID(s string) (ID, error) { return ParseWithPrefix(s, PrefixOrg) }

// ParseMemberID parses a string and validates the "amem" prefix.
func ParseMemberID(s string) (ID, error) { return ParseWithPrefix(s, PrefixMember) }

// ParseTeamID parses a string and validates the "atm" prefix.
func ParseTeamID(s string) (ID, error) { return ParseWithPrefix(s, PrefixTeam) }

// ParseInvitationID parses a string and validates the "ainv" prefix.
func ParseInvitationID(s string) (ID, error) { return ParseWithPrefix(s, PrefixInvitation) }

// ParseDeviceID parses a string and validates the "adev" prefix.
func ParseDeviceID(s string) (ID, error) { return ParseWithPrefix(s, PrefixDevice) }

// ParseWebhookID parses a string and validates the "awhk" prefix.
func ParseWebhookID(s string) (ID, error) { return ParseWithPrefix(s, PrefixWebhook) }

// ParseNotificationID parses a string and validates the "antf" prefix.
func ParseNotificationID(s string) (ID, error) { return ParseWithPrefix(s, PrefixNotification) }

// ParseVerificationID parses a string and validates the "avrf" prefix.
func ParseVerificationID(s string) (ID, error) { return ParseWithPrefix(s, PrefixVerification) }

// ParsePasswordResetID parses a string and validates the "apwr" prefix.
func ParsePasswordResetID(s string) (ID, error) { return ParseWithPrefix(s, PrefixPasswordReset) }

// ParseAPIKeyID parses a string and validates the "akey" prefix.
func ParseAPIKeyID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAPIKey) }

// ParseOAuthConnectionID parses a string and validates the "aoau" prefix.
func ParseOAuthConnectionID(s string) (ID, error) {
	return ParseWithPrefix(s, PrefixOAuthConnection)
}

// ParsePasskeyID parses a string and validates the "apsk" prefix.
func ParsePasskeyID(s string) (ID, error) { return ParseWithPrefix(s, PrefixPasskey) }

// ParseMFAID parses a string and validates the "amfa" prefix.
func ParseMFAID(s string) (ID, error) { return ParseWithPrefix(s, PrefixMFA) }

// ParseEnvironmentID parses a string and validates the "aenv" prefix.
func ParseEnvironmentID(s string) (ID, error) { return ParseWithPrefix(s, PrefixEnvironment) }

// ParseRoleID parses a string and validates the "arol" prefix.
func ParseRoleID(s string) (ID, error) { return ParseWithPrefix(s, PrefixRole) }

// ParsePermissionID parses a string and validates the "aprm" prefix.
func ParsePermissionID(s string) (ID, error) { return ParseWithPrefix(s, PrefixPermission) }

// ParseRecoveryCodeID parses a string and validates the "arec" prefix.
func ParseRecoveryCodeID(s string) (ID, error) { return ParseWithPrefix(s, PrefixRecoveryCode) }

// ParseSSOConnectionID parses a string and validates the "asso" prefix.
func ParseSSOConnectionID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSSOConnection) }

// ParseConsentID parses a string and validates the "acns" prefix.
func ParseConsentID(s string) (ID, error) { return ParseWithPrefix(s, PrefixConsent) }

// ParseFormConfigID parses a string and validates the "afcf" prefix.
func ParseFormConfigID(s string) (ID, error) { return ParseWithPrefix(s, PrefixFormConfig) }

// ParseBrandingConfigID parses a string and validates the "abrd" prefix.
func ParseBrandingConfigID(s string) (ID, error) { return ParseWithPrefix(s, PrefixBrandingConfig) }

// ParseAppSessionConfigID parses a string and validates the "ascf" prefix.
func ParseAppSessionConfigID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAppSessionCfg) }

// ParseOAuth2ClientID parses a string and validates the "aoac" prefix.
func ParseOAuth2ClientID(s string) (ID, error) { return ParseWithPrefix(s, PrefixOAuth2Client) }

// ParseAuthCodeID parses a string and validates the "aaco" prefix.
func ParseAuthCodeID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAuthCode) }

// ParseSettingID parses a string and validates the "aset" prefix.
func ParseSettingID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSetting) }

// ParseAppClientConfigID parses a string and validates the "aacf" prefix.
func ParseAppClientConfigID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAppClientConfig) }

// ParseDeviceCodeID parses a string and validates the "advc" prefix.
func ParseDeviceCodeID(s string) (ID, error) { return ParseWithPrefix(s, PrefixDeviceCode) }

// ParseSCIMConfigID parses a string and validates the "ascm" prefix.
func ParseSCIMConfigID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSCIMConfig) }

// ParseSCIMTokenID parses a string and validates the "asct" prefix.
func ParseSCIMTokenID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSCIMToken) }

// ParseSCIMLogID parses a string and validates the "ascl" prefix.
func ParseSCIMLogID(s string) (ID, error) { return ParseWithPrefix(s, PrefixSCIMLog) }

// ParseAny parses a string into an ID without type checking the prefix.
func ParseAny(s string) (ID, error) { return Parse(s) }

// ──────────────────────────────────────────────────
// ID methods
// ──────────────────────────────────────────────────

// String returns the full TypeID string representation (prefix_suffix).
// Returns an empty string for the Nil ID.
func (i ID) String() string {
	if !i.valid {
		return ""
	}

	return i.inner.String()
}

// Prefix returns the prefix component of this ID.
func (i ID) Prefix() Prefix {
	if !i.valid {
		return ""
	}

	return Prefix(i.inner.Prefix())
}

// IsNil reports whether this ID is the zero value.
func (i ID) IsNil() bool {
	return !i.valid
}

// MarshalText implements encoding.TextMarshaler.
func (i ID) MarshalText() ([]byte, error) {
	if !i.valid {
		return []byte{}, nil
	}

	return []byte(i.inner.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *ID) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*i = Nil

		return nil
	}

	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}

	*i = parsed

	return nil
}

// MarshalBSONValue satisfies bson.ValueMarshaler (mongo-driver v2) so the ID
// is stored as a BSON string instead of an opaque struct. No bson import needed
// because Go uses structural typing for interface satisfaction.
func (i ID) MarshalBSONValue() (bsonType byte, data []byte, err error) {
	if !i.valid {
		return bsonTypeNull, nil, nil
	}

	s := i.inner.String()
	l := len(s) + 1 // length includes null terminator

	buf := make([]byte, 4+len(s)+1)
	binary.LittleEndian.PutUint32(buf, uint32(l))
	copy(buf[4:], s)
	// trailing 0x00 is already zero from make

	return bsonTypeString, buf, nil
}

// UnmarshalBSONValue satisfies bson.ValueUnmarshaler (mongo-driver v2).
func (i *ID) UnmarshalBSONValue(t byte, data []byte) error {
	if t == bsonTypeNull {
		*i = Nil

		return nil
	}

	if t != bsonTypeString {
		return fmt.Errorf("id: cannot unmarshal BSON type 0x%02x into ID", t)
	}

	if len(data) < 5 { //nolint:mnd // 4-byte length + at least 1 null terminator
		*i = Nil

		return nil
	}

	l := binary.LittleEndian.Uint32(data[:4])
	if l <= 1 { // empty string (just null terminator)
		*i = Nil

		return nil
	}

	s := string(data[4 : 4+l-1]) // exclude null terminator

	return i.UnmarshalText([]byte(s))
}

// Value implements driver.Valuer for database storage.
// Returns nil for the Nil ID so that optional foreign key columns store NULL.
func (i ID) Value() (driver.Value, error) {
	if !i.valid {
		return nil, nil //nolint:nilnil // nil is the canonical NULL for driver.Valuer
	}

	return i.inner.String(), nil
}

// Scan implements sql.Scanner for database retrieval.
func (i *ID) Scan(src any) error {
	if src == nil {
		*i = Nil

		return nil
	}

	switch v := src.(type) {
	case string:
		if v == "" {
			*i = Nil

			return nil
		}

		return i.UnmarshalText([]byte(v))
	case []byte:
		if len(v) == 0 {
			*i = Nil

			return nil
		}

		return i.UnmarshalText(v)
	default:
		return fmt.Errorf("id: cannot scan %T into ID", src)
	}
}
