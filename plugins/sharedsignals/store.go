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
	// ErrDuplicateJTI is returned when a (stream_id, jti, event_type)
	// triple already exists. Callers treat this as a replay and answer 202,
	// so it must be distinguishable from every other write failure.
	//
	// The key is the triple, not just (stream_id, jti), because RFC 8417
	// keys a SET's `events` object by event type URI, so a single delivery
	// carries at most one event of a given type but may legitimately carry
	// several different types under the same jti. Keying on (stream_id,
	// jti) alone would make the second event in a multi-event SET collide
	// with the row the first one just inserted, on the very first delivery.
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

// ReceivedEventFilter narrows a ListReceivedEvents query. A received event
// carries no app_id of its own -- the stream it arrived on is what binds it
// to a tenant -- so StreamID is mandatory and every backend resolves the
// stream before it reads a single row.
type ReceivedEventFilter struct {
	StreamID id.SSFStreamID
	// Since and Until bound received_at, half-open: [Since, Until). Either
	// may be zero, meaning unbounded on that side.
	Since time.Time
	Until time.Time
	// Limit caps the rows returned. Zero or less means
	// DefaultReceivedEventLimit; anything above MaxReceivedEventLimit is
	// clamped down to it.
	Limit int
}

// Row limits for ListReceivedEvents. The audit trail grows without bound --
// a busy stream writes a row per delivery -- so a caller that names no limit
// gets a page, not the table, and one that names an enormous limit does not
// get to turn a dashboard request into a full scan.
const (
	DefaultReceivedEventLimit = 50
	MaxReceivedEventLimit     = 500
)

// normalized applies the limit policy above. Every backend calls it, so a
// missing or absurd limit means the same thing on all four.
func (f ReceivedEventFilter) normalized() ReceivedEventFilter {
	switch {
	case f.Limit <= 0:
		f.Limit = DefaultReceivedEventLimit
	case f.Limit > MaxReceivedEventLimit:
		f.Limit = MaxReceivedEventLimit
	}
	return f
}

// streamOwnedBy confirms streamID exists and belongs to appID, returning
// ErrNotFound when it does not. It mirrors Plugin.streamInCallerApp,
// including the choice to answer a cross-tenant hit identically to a miss:
// anything else confirms to the caller that the ID is real for some OTHER
// tenant, which is the fact an IDOR probe is trying to learn.
//
// It takes the lookup as an interface rather than a Store so each backend
// can hand it its own receiver, which keeps the tenant rule written once
// instead of four times.
func streamOwnedBy(ctx context.Context, lookup interface {
	GetInboundStream(context.Context, id.SSFStreamID) (*InboundStream, error)
}, appID id.AppID, streamID id.SSFStreamID) error {
	stream, err := lookup.GetInboundStream(ctx, streamID)
	if err != nil {
		return err
	}
	if stream.AppID.String() != appID.String() {
		return ErrNotFound
	}
	return nil
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

	// InsertReceivedEvent returns ErrDuplicateJTI when (stream_id, jti,
	// event_type) already exists.
	InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	// DeleteReceivedEvent removes one received-event row by ID. Its only
	// caller is the servePush pipeline undoing InsertReceivedEvent's dedupe
	// row after processing that event failed for an infrastructure reason
	// rather than a policy one -- a transmitter retry must actually be
	// reprocessed, not read back later as a replay of a delivery that was
	// never really handled.
	DeleteReceivedEvent(ctx context.Context, id id.SSFEventID) error
	// GetReceivedEvent loads one audit row by ID, scoped to appID. The
	// row itself has no app_id, so the scope is enforced through the
	// stream it arrived on; a row belonging to another tenant answers
	// ErrNotFound, never a distinguishable "forbidden".
	GetReceivedEvent(ctx context.Context, appID id.AppID,
		eventID id.SSFEventID) (*ReceivedEvent, error)
	// ListReceivedEvents returns one stream's audit rows newest first,
	// bounded by the filter's time window and row limit. A stream that
	// does not belong to appID answers ErrNotFound rather than an empty
	// list, so a probe cannot tell "not yours" from "yours but quiet".
	ListReceivedEvents(ctx context.Context, appID id.AppID,
		f ReceivedEventFilter) ([]*ReceivedEvent, error)
	// CountEventsSince counts every event RECORDED for a stream since a
	// cutoff, whatever outcome it reached. It backs the circuit breaker.
	//
	// It deliberately does not filter on action_taken. Counting only the
	// events that produced an action left the entire signal-only half of
	// the matrix outside the breaker: an authentic but hostile transmitter
	// could push unlimited risk-level-change events at HIGH, raising every
	// user's risk score, while the counter stayed at zero. The breaker
	// exists to bound authentic traffic, so it has to count all of it.
	CountEventsSince(ctx context.Context, streamID id.SSFStreamID, since time.Time) (int, error)

	CreateSignal(ctx context.Context, s *Signal) error
	ListActiveSignals(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		userID id.UserID, now time.Time) ([]*Signal, error)
}
