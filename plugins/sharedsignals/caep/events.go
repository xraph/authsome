package caep

import (
	"encoding/json"
	"fmt"
)

// CAEP and SSF event type URIs.
const (
	EventSessionRevoked         = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTokenClaimsChange      = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change" //nolint:gosec // G101: this is a public CAEP event type URI, not a credential
	EventCredentialChange       = "https://schemas.openid.net/secevent/caep/event-type/credential-change"   //nolint:gosec // G101: this is a public CAEP event type URI, not a credential
	EventAssuranceLevelChange   = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventDeviceComplianceChange = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
	EventRiskLevelChange        = "https://schemas.openid.net/secevent/caep/event-type/risk-level-change"
	EventVerification           = "https://schemas.openid.net/secevent/ssf/event-type/verification"
)

var knownEventTypes = map[string]struct{}{
	EventSessionRevoked:         {},
	EventTokenClaimsChange:      {},
	EventCredentialChange:       {},
	EventAssuranceLevelChange:   {},
	EventDeviceComplianceChange: {},
	EventRiskLevelChange:        {},
	EventVerification:           {},
}

// IsKnownEventType reports whether this build understands the event type.
func IsKnownEventType(t string) bool {
	_, ok := knownEventTypes[t]
	return ok
}

// IsStreamLevel reports whether an event describes the STREAM rather than a
// principal on it. A stream-level event carries no subject at all, so it can
// neither be resolved to a user nor be refused for failing to resolve to one.
// SSF's verification event is the only one today.
func IsStreamLevel(t string) bool { return t == EventVerification }

// Event is one decoded event payload from a SET's events map.
type Event struct {
	Type string
	// Subject is the zero SubjectID for a stream-level event.
	Subject SubjectID
	// RawSubject is the exact subject JSON the transmitter sent, kept
	// verbatim for the audit row. The receiver deliberately keeps offending
	// values out of its error responses on the grounds that they belong in
	// the audit record instead, which only works if the audit record
	// actually gets them. Empty for a stream-level event.
	RawSubject       json.RawMessage
	EventTimestamp   int64
	InitiatingEntity string
	ReasonAdmin      map[string]string
	ReasonUser       map[string]string

	// credential-change
	CredentialType string
	ChangeType     string
	FriendlyName   string

	// assurance-level-change
	Namespace       string
	CurrentLevel    string
	PreviousLevel   string
	ChangeDirection string

	// device-compliance-change
	CurrentStatus  string
	PreviousStatus string

	// token-claims-change
	Claims map[string]any

	// risk-level-change
	RiskReason string
	Principal  string

	// SSF verification
	State string
}

type eventWire struct {
	SubID            json.RawMessage   `json:"sub_id"`
	Subject          json.RawMessage   `json:"subject"`
	EventTimestamp   int64             `json:"event_timestamp"`
	InitiatingEntity string            `json:"initiating_entity"`
	ReasonAdmin      map[string]string `json:"reason_admin"`
	ReasonUser       map[string]string `json:"reason_user"`
	CredentialType   string            `json:"credential_type"`
	ChangeType       string            `json:"change_type"`
	FriendlyName     string            `json:"friendly_name"`
	Namespace        string            `json:"namespace"`
	CurrentLevel     string            `json:"current_level"`
	PreviousLevel    string            `json:"previous_level"`
	ChangeDirection  string            `json:"change_direction"`
	CurrentStatus    string            `json:"current_status"`
	PreviousStatus   string            `json:"previous_status"`
	Claims           map[string]any    `json:"claims"`
	RiskReason       string            `json:"risk_reason"`
	Principal        string            `json:"principal"`
	State            string            `json:"state"`
}

// ParseEvent decodes one event payload. It accepts the subject under either
// `sub_id` (CAEP 1.0 final) or `subject` (what Okta and Google ship today),
// preferring `sub_id` when a payload carries both.
//
// Every CAEP event names a principal and is refused without one. A
// stream-level event (see IsStreamLevel) is the exception: SSF's verification
// event describes the stream itself and carries only `state`, so requiring a
// subject there made the entire verification handshake unprocessable.
func ParseEvent(eventType string, payload json.RawMessage) (Event, error) {
	var w eventWire
	if err := json.Unmarshal(payload, &w); err != nil {
		return Event{}, fmt.Errorf("caep: decode event payload: %w", err)
	}

	rawSubject := w.SubID
	if len(rawSubject) == 0 {
		rawSubject = w.Subject
	}

	var subject SubjectID
	if len(rawSubject) == 0 {
		if !IsStreamLevel(eventType) {
			return Event{}, fmt.Errorf("caep: event has neither sub_id nor subject")
		}
	} else {
		var err error
		subject, err = ParseSubjectID(rawSubject)
		if err != nil {
			return Event{}, err
		}
	}

	return Event{
		Type:             eventType,
		Subject:          subject,
		RawSubject:       rawSubject,
		EventTimestamp:   w.EventTimestamp,
		InitiatingEntity: w.InitiatingEntity,
		ReasonAdmin:      w.ReasonAdmin,
		ReasonUser:       w.ReasonUser,
		CredentialType:   w.CredentialType,
		ChangeType:       w.ChangeType,
		FriendlyName:     w.FriendlyName,
		Namespace:        w.Namespace,
		CurrentLevel:     w.CurrentLevel,
		PreviousLevel:    w.PreviousLevel,
		ChangeDirection:  w.ChangeDirection,
		CurrentStatus:    w.CurrentStatus,
		PreviousStatus:   w.PreviousStatus,
		Claims:           w.Claims,
		RiskReason:       w.RiskReason,
		Principal:        w.Principal,
		State:            w.State,
	}, nil
}
