package sharedsignals

import (
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Grove models
// ──────────────────────────────────────────────────

type inboundStreamModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_inbound_streams,alias:sis"`

	ID    string `grove:"id,pk"`
	AppID string `grove:"app_id,notnull"`
	EnvID string `grove:"env_id,notnull"`
	Name  string `grove:"name,notnull"`

	Issuer   string `grove:"issuer,notnull"`
	Audience string `grove:"audience,notnull"`
	JWKSURI  string `grove:"jwks_uri,notnull"`

	PushPathHash  string `grove:"push_path_hash,notnull"`
	PushTokenHash string `grove:"push_token_hash,notnull"`

	AllowedEventTypes     string `grove:"allowed_event_types,notnull"`
	AllowedSubjectFormats string `grove:"allowed_subject_formats,notnull"`
	VerifiedDomains       string `grove:"verified_domains,notnull"`
	ActionOverrides       string `grove:"action_overrides,notnull"`

	EnforcementMode   string `grove:"enforcement_mode,notnull"`
	Status            string `grove:"status,notnull"`
	MaxActionsPerHour int    `grove:"max_actions_per_hour,notnull"`

	PendingVerifyState string     `grove:"pending_verify_state,notnull"`
	LastVerifiedAt     *time.Time `grove:"last_verified_at"`

	CreatedAt time.Time `grove:"created_at,notnull"`
	UpdatedAt time.Time `grove:"updated_at,notnull"`
}

type subjectLinkModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_subject_links,alias:ssl"`

	ID         string    `grove:"id,pk"`
	AppID      string    `grove:"app_id,notnull"`
	EnvID      string    `grove:"env_id,notnull"`
	Issuer     string    `grove:"issuer,notnull"`
	Subject    string    `grove:"subject,notnull"`
	UserID     string    `grove:"user_id,notnull"`
	Source     string    `grove:"source,notnull"`
	CreatedAt  time.Time `grove:"created_at,notnull"`
	LastSeenAt time.Time `grove:"last_seen_at,notnull"`
}

type receivedEventModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_received_events,alias:sre"`

	ID             string    `grove:"id,pk"`
	StreamID       string    `grove:"stream_id,notnull"`
	JTI            string    `grove:"jti,notnull"`
	EventType      string    `grove:"event_type,notnull"`
	SubjectJSON    string    `grove:"subject_json,notnull"`
	ResolvedUserID string    `grove:"resolved_user_id,notnull"`
	Outcome        string    `grove:"outcome,notnull"`
	ActionTaken    string    `grove:"action_taken,notnull"`
	Error          string    `grove:"error,notnull"`
	ReceivedAt     time.Time `grove:"received_at,notnull"`
}

type signalModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_signals,alias:ssg"`

	ID        string    `grove:"id,pk"`
	AppID     string    `grove:"app_id,notnull"`
	EnvID     string    `grove:"env_id,notnull"`
	UserID    string    `grove:"user_id,notnull"`
	StreamID  string    `grove:"stream_id,notnull"`
	EventType string    `grove:"event_type,notnull"`
	Severity  int       `grove:"severity,notnull"`
	Reason    string    `grove:"reason,notnull"`
	EventAt   time.Time `grove:"event_at,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull"`
}

// ──────────────────────────────────────────────────
// Mappers
// ──────────────────────────────────────────────────

// encodeJSON marshals a value to a JSON string, falling back to an empty
// container so a NOT NULL column always has something valid in it.
func encodeJSON(v any, empty string) string {
	if v == nil {
		return empty
	}
	b, err := json.Marshal(v)
	if err != nil {
		return empty
	}
	return string(b)
}

func decodeStrings(s string) ([]string, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeStringMap(s string) (map[string]string, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// parseOptionalID parses an ID that may legitimately be absent, returning the
// zero ID rather than an error for an empty column.
func parseOptionalID(s string) id.ID {
	if s == "" {
		return id.Nil
	}
	parsed, err := id.Parse(s)
	if err != nil {
		return id.Nil
	}
	return parsed
}

func fromInboundStream(s *InboundStream) *inboundStreamModel {
	return &inboundStreamModel{
		ID:                    s.ID.String(),
		AppID:                 s.AppID.String(),
		EnvID:                 s.EnvID.String(),
		Name:                  s.Name,
		Issuer:                s.Issuer,
		Audience:              s.Audience,
		JWKSURI:               s.JWKSURI,
		PushPathHash:          s.PushPathHash,
		PushTokenHash:         s.PushTokenHash,
		AllowedEventTypes:     encodeJSON(s.AllowedEventTypes, "[]"),
		AllowedSubjectFormats: encodeJSON(s.AllowedSubjectFormats, "[]"),
		VerifiedDomains:       encodeJSON(s.VerifiedDomains, "[]"),
		ActionOverrides:       encodeJSON(s.ActionOverrides, "{}"),
		EnforcementMode:       s.EnforcementMode,
		Status:                s.Status,
		MaxActionsPerHour:     s.MaxActionsPerHour,
		PendingVerifyState:    s.PendingVerifyState,
		LastVerifiedAt:        s.LastVerifiedAt,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
}

func toInboundStream(m *inboundStreamModel) (*InboundStream, error) {
	streamID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.Parse(m.AppID)
	if err != nil {
		return nil, err
	}
	events, err := decodeStrings(m.AllowedEventTypes)
	if err != nil {
		return nil, err
	}
	formats, err := decodeStrings(m.AllowedSubjectFormats)
	if err != nil {
		return nil, err
	}
	domains, err := decodeStrings(m.VerifiedDomains)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeStringMap(m.ActionOverrides)
	if err != nil {
		return nil, err
	}
	return &InboundStream{
		ID:                    streamID,
		AppID:                 appID,
		EnvID:                 parseOptionalID(m.EnvID),
		Name:                  m.Name,
		Issuer:                m.Issuer,
		Audience:              m.Audience,
		JWKSURI:               m.JWKSURI,
		PushPathHash:          m.PushPathHash,
		PushTokenHash:         m.PushTokenHash,
		AllowedEventTypes:     events,
		AllowedSubjectFormats: formats,
		VerifiedDomains:       domains,
		ActionOverrides:       overrides,
		EnforcementMode:       m.EnforcementMode,
		Status:                m.Status,
		MaxActionsPerHour:     m.MaxActionsPerHour,
		PendingVerifyState:    m.PendingVerifyState,
		LastVerifiedAt:        m.LastVerifiedAt,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}, nil
}

func fromSubjectLink(l *SubjectLink) *subjectLinkModel {
	return &subjectLinkModel{
		ID:         l.ID.String(),
		AppID:      l.AppID.String(),
		EnvID:      l.EnvID.String(),
		Issuer:     l.Issuer,
		Subject:    l.Subject,
		UserID:     l.UserID.String(),
		Source:     l.Source,
		CreatedAt:  l.CreatedAt,
		LastSeenAt: l.LastSeenAt,
	}
}

func toSubjectLink(m *subjectLinkModel) (*SubjectLink, error) {
	linkID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.Parse(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.Parse(m.UserID)
	if err != nil {
		return nil, err
	}
	return &SubjectLink{
		ID:         linkID,
		AppID:      appID,
		EnvID:      parseOptionalID(m.EnvID),
		Issuer:     m.Issuer,
		Subject:    m.Subject,
		UserID:     userID,
		Source:     m.Source,
		CreatedAt:  m.CreatedAt,
		LastSeenAt: m.LastSeenAt,
	}, nil
}

func fromReceivedEvent(e *ReceivedEvent) *receivedEventModel {
	resolved := ""
	if !e.ResolvedUserID.IsNil() {
		resolved = e.ResolvedUserID.String()
	}
	return &receivedEventModel{
		ID:             e.ID.String(),
		StreamID:       e.StreamID.String(),
		JTI:            e.JTI,
		EventType:      e.EventType,
		SubjectJSON:    e.SubjectJSON,
		ResolvedUserID: resolved,
		Outcome:        e.Outcome,
		ActionTaken:    e.ActionTaken,
		Error:          e.Error,
		ReceivedAt:     e.ReceivedAt,
	}
}

func toReceivedEvent(m *receivedEventModel) (*ReceivedEvent, error) {
	eventID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	// StreamID is half the (stream_id, jti) replay-guard tuple, so a
	// corrupt value must fail loudly rather than silently becoming "no
	// stream".
	streamID, err := id.Parse(m.StreamID)
	if err != nil {
		return nil, err
	}
	return &ReceivedEvent{
		ID:             eventID,
		StreamID:       streamID,
		JTI:            m.JTI,
		EventType:      m.EventType,
		SubjectJSON:    m.SubjectJSON,
		ResolvedUserID: parseOptionalID(m.ResolvedUserID),
		Outcome:        m.Outcome,
		ActionTaken:    m.ActionTaken,
		Error:          m.Error,
		ReceivedAt:     m.ReceivedAt,
	}, nil
}

func fromSignal(s *Signal) *signalModel {
	return &signalModel{
		ID:        s.ID.String(),
		AppID:     s.AppID.String(),
		EnvID:     s.EnvID.String(),
		UserID:    s.UserID.String(),
		StreamID:  s.StreamID.String(),
		EventType: s.EventType,
		Severity:  s.Severity,
		Reason:    s.Reason,
		EventAt:   s.EventAt,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
}

func toSignal(m *signalModel) (*Signal, error) {
	signalID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.Parse(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.Parse(m.UserID)
	if err != nil {
		return nil, err
	}
	return &Signal{
		ID:        signalID,
		AppID:     appID,
		EnvID:     parseOptionalID(m.EnvID),
		UserID:    userID,
		StreamID:  parseOptionalID(m.StreamID),
		EventType: m.EventType,
		Severity:  m.Severity,
		Reason:    m.Reason,
		EventAt:   m.EventAt,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}, nil
}
