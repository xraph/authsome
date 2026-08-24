// Package sharedsignals implements the OpenID Shared Signals Framework and
// CAEP. It receives Security Event Tokens from an upstream identity provider
// and turns them into session revocations and durable risk signals.
package sharedsignals

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// Store errors.
var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("sharedsignals: not found")
	// ErrDuplicateJTI is returned when a (stream_id, jti) pair already
	// exists. Callers treat this as a replay and answer 202, so it must be
	// distinguishable from every other write failure.
	ErrDuplicateJTI = errors.New("sharedsignals: duplicate jti")
)

// Stream status values, matching the SSF stream status vocabulary.
const (
	StatusEnabled  = "enabled"
	StatusPaused   = "paused"
	StatusDisabled = "disabled"
)

// Enforcement modes. Observe records what would have happened without doing it.
const (
	EnforcementEnforce = "enforce"
	EnforcementObserve = "observe"
)

// Received-event outcomes.
const (
	OutcomePending    = "pending"
	OutcomeApplied    = "applied"
	OutcomeIgnored    = "ignored"
	OutcomeUnresolved = "unresolved"
	OutcomeRejected   = "rejected"
)

// Subject link sources.
const (
	SourceSSO    = "sso"
	SourceSocial = "social"
	SourceSCIM   = "scim"
	SourceManual = "manual"
)

// InboundStream is one identity provider we accept events from. Everything
// the receiver trusts about a SET comes from this row, never from the token.
type InboundStream struct {
	ID    id.SSFStreamID
	AppID id.AppID
	EnvID id.EnvironmentID
	Name  string

	// Issuer is the exact iss value a SET must carry.
	Issuer string
	// Audience is the aud value a SET must include.
	Audience string
	// JWKSURI is where this stream's verification keys live.
	JWKSURI string

	// PushPathHash is SHA-256 of the secret URL segment. The plaintext is
	// shown once at creation and never stored.
	PushPathHash string
	// PushTokenHash is SHA-256 of the bearer token the transmitter must send.
	PushTokenHash string

	AllowedEventTypes     []string
	AllowedSubjectFormats []string
	VerifiedDomains       []string
	ActionOverrides       map[string]string

	EnforcementMode   string
	Status            string
	MaxActionsPerHour int

	// PendingVerifyState is the state we sent to the transmitter's
	// verification endpoint and expect echoed back in a verification event.
	PendingVerifyState string
	LastVerifiedAt     *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// SubjectLink binds an upstream (issuer, subject) pair to an authsome user.
// This is what makes the iss_sub subject format resolvable.
type SubjectLink struct {
	ID         id.SSFLinkID
	AppID      id.AppID
	EnvID      id.EnvironmentID
	Issuer     string
	Subject    string
	UserID     id.UserID
	Source     string
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// ReceivedEvent is both the replay guard and the audit trail for one SET.
type ReceivedEvent struct {
	ID             id.SSFEventID
	StreamID       id.SSFStreamID
	JTI            string
	EventType      string
	SubjectJSON    string
	ResolvedUserID id.UserID
	Outcome        string
	ActionTaken    string
	Error          string
	ReceivedAt     time.Time
}

// Signal is durable risk state. An event that arrives while nobody is signing
// in has to survive until the sign-in that cares about it.
type Signal struct {
	ID        id.SSFSignalID
	AppID     id.AppID
	EnvID     id.EnvironmentID
	UserID    id.UserID
	StreamID  id.SSFStreamID
	EventType string
	Severity  int
	Reason    string
	EventAt   time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Store persists everything the receiver needs.
type Store interface {
	CreateInboundStream(ctx context.Context, s *InboundStream) error
	GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error)
	GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error)
	ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error)
	UpdateInboundStream(ctx context.Context, s *InboundStream) error
	DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error

	UpsertSubjectLink(ctx context.Context, l *SubjectLink) error
	GetSubjectLink(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		issuer, subject string) (*SubjectLink, error)

	// InsertReceivedEvent returns ErrDuplicateJTI when (stream_id, jti)
	// already exists.
	InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	// CountActionsSince counts events on a stream that actually did
	// something since a cutoff. It backs the circuit breaker.
	CountActionsSince(ctx context.Context, streamID id.SSFStreamID, since time.Time) (int, error)

	CreateSignal(ctx context.Context, s *Signal) error
	ListActiveSignals(ctx context.Context, appID id.AppID, userID id.UserID,
		now time.Time) ([]*Signal, error)
}
