package sharedsignals

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
)

// MongoStore implements Store on MongoDB.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore builds a MongoDB-backed store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{db: db, mdb: mongodriver.Unwrap(db)}
}

var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Documents
// ──────────────────────────────────────────────────

type inboundStreamDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_inbound_streams"`

	ID                    string     `bson:"_id"`
	AppID                 string     `bson:"app_id"`
	EnvID                 string     `bson:"env_id"`
	Name                  string     `bson:"name"`
	Issuer                string     `bson:"issuer"`
	Audience              string     `bson:"audience"`
	JWKSURI               string     `bson:"jwks_uri"`
	PushPathHash          string     `bson:"push_path_hash"`
	PushTokenHash         string     `bson:"push_token_hash"`
	AllowedEventTypes     []string   `bson:"allowed_event_types"`
	AllowedSubjectFormats []string   `bson:"allowed_subject_formats"`
	VerifiedDomains       []string   `bson:"verified_domains"`
	ActionOverrides       string     `bson:"action_overrides"`
	EnforcementMode       string     `bson:"enforcement_mode"`
	Status                string     `bson:"status"`
	MaxActionsPerHour     int        `bson:"max_actions_per_hour"`
	PendingVerifyState    string     `bson:"pending_verify_state"`
	LastVerifiedAt        *time.Time `bson:"last_verified_at,omitempty"`
	CreatedAt             time.Time  `bson:"created_at"`
	UpdatedAt             time.Time  `bson:"updated_at"`
}

type subjectLinkDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_subject_links"`

	ID         string    `bson:"_id"`
	AppID      string    `bson:"app_id"`
	EnvID      string    `bson:"env_id"`
	Issuer     string    `bson:"issuer"`
	Subject    string    `bson:"subject"`
	UserID     string    `bson:"user_id"`
	Source     string    `bson:"source"`
	CreatedAt  time.Time `bson:"created_at"`
	LastSeenAt time.Time `bson:"last_seen_at"`
}

type receivedEventDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_received_events"`

	ID             string    `bson:"_id"`
	StreamID       string    `bson:"stream_id"`
	JTI            string    `bson:"jti"`
	EventType      string    `bson:"event_type"`
	SubjectJSON    string    `bson:"subject_json"`
	ResolvedUserID string    `bson:"resolved_user_id"`
	Outcome        string    `bson:"outcome"`
	ActionTaken    string    `bson:"action_taken"`
	Error          string    `bson:"error"`
	ReceivedAt     time.Time `bson:"received_at"`
}

type signalDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_signals"`

	ID        string    `bson:"_id"`
	AppID     string    `bson:"app_id"`
	EnvID     string    `bson:"env_id"`
	UserID    string    `bson:"user_id"`
	StreamID  string    `bson:"stream_id"`
	EventType string    `bson:"event_type"`
	Severity  int       `bson:"severity"`
	Reason    string    `bson:"reason"`
	EventAt   time.Time `bson:"event_at"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func inboundStreamToDoc(s *InboundStream) *inboundStreamDoc {
	return &inboundStreamDoc{
		ID: s.ID.String(), AppID: s.AppID.String(), EnvID: s.EnvID.String(),
		Name: s.Name, Issuer: s.Issuer, Audience: s.Audience, JWKSURI: s.JWKSURI,
		PushPathHash: s.PushPathHash, PushTokenHash: s.PushTokenHash,
		AllowedEventTypes:     s.AllowedEventTypes,
		AllowedSubjectFormats: s.AllowedSubjectFormats,
		VerifiedDomains:       s.VerifiedDomains,
		ActionOverrides:       encodeJSON(s.ActionOverrides, "{}"),
		EnforcementMode:       s.EnforcementMode, Status: s.Status,
		MaxActionsPerHour:  s.MaxActionsPerHour,
		PendingVerifyState: s.PendingVerifyState, LastVerifiedAt: s.LastVerifiedAt,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func docToInboundStream(d *inboundStreamDoc) (*InboundStream, error) {
	streamID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeStringMap(d.ActionOverrides)
	if err != nil {
		return nil, err
	}
	return &InboundStream{
		ID: streamID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		Name: d.Name, Issuer: d.Issuer, Audience: d.Audience, JWKSURI: d.JWKSURI,
		PushPathHash: d.PushPathHash, PushTokenHash: d.PushTokenHash,
		AllowedEventTypes:     d.AllowedEventTypes,
		AllowedSubjectFormats: d.AllowedSubjectFormats,
		VerifiedDomains:       d.VerifiedDomains,
		ActionOverrides:       overrides,
		EnforcementMode:       d.EnforcementMode, Status: d.Status,
		MaxActionsPerHour:  d.MaxActionsPerHour,
		PendingVerifyState: d.PendingVerifyState, LastVerifiedAt: d.LastVerifiedAt,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}, nil
}

func subjectLinkToDoc(l *SubjectLink) *subjectLinkDoc {
	return &subjectLinkDoc{
		ID: l.ID.String(), AppID: l.AppID.String(), EnvID: l.EnvID.String(),
		Issuer: l.Issuer, Subject: l.Subject, UserID: l.UserID.String(),
		Source: l.Source, CreatedAt: l.CreatedAt, LastSeenAt: l.LastSeenAt,
	}
}

func docToSubjectLink(d *subjectLinkDoc) (*SubjectLink, error) {
	linkID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &SubjectLink{
		ID: linkID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		Issuer: d.Issuer, Subject: d.Subject, UserID: parseOptionalID(d.UserID),
		Source: d.Source, CreatedAt: d.CreatedAt, LastSeenAt: d.LastSeenAt,
	}, nil
}

func receivedEventToDoc(e *ReceivedEvent) *receivedEventDoc {
	resolved := ""
	if !e.ResolvedUserID.IsNil() {
		resolved = e.ResolvedUserID.String()
	}
	return &receivedEventDoc{
		ID: e.ID.String(), StreamID: e.StreamID.String(), JTI: e.JTI,
		EventType: e.EventType, SubjectJSON: e.SubjectJSON, ResolvedUserID: resolved,
		Outcome: e.Outcome, ActionTaken: e.ActionTaken, Error: e.Error,
		ReceivedAt: e.ReceivedAt,
	}
}

func docToReceivedEvent(d *receivedEventDoc) (*ReceivedEvent, error) {
	eventID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &ReceivedEvent{
		ID: eventID, StreamID: parseOptionalID(d.StreamID), JTI: d.JTI,
		EventType: d.EventType, SubjectJSON: d.SubjectJSON,
		ResolvedUserID: parseOptionalID(d.ResolvedUserID),
		Outcome:        d.Outcome, ActionTaken: d.ActionTaken, Error: d.Error,
		ReceivedAt: d.ReceivedAt,
	}, nil
}

func signalToDoc(s *Signal) *signalDoc {
	return &signalDoc{
		ID: s.ID.String(), AppID: s.AppID.String(), EnvID: s.EnvID.String(),
		UserID: s.UserID.String(), StreamID: s.StreamID.String(),
		EventType: s.EventType, Severity: s.Severity, Reason: s.Reason,
		EventAt: s.EventAt, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt,
	}
}

func docToSignal(d *signalDoc) (*Signal, error) {
	signalID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &Signal{
		ID: signalID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		UserID: parseOptionalID(d.UserID), StreamID: parseOptionalID(d.StreamID),
		EventType: d.EventType, Severity: d.Severity, Reason: d.Reason,
		EventAt: d.EventAt, ExpiresAt: d.ExpiresAt, CreatedAt: d.CreatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func mongoErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}

func isMongoDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

func (s *MongoStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.mdb.Collection(colInboundStreams).InsertOne(ctx, inboundStreamToDoc(in))
	return mongoErr(err)
}

func (s *MongoStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	d := new(inboundStreamDoc)
	if err := s.mdb.Collection(colInboundStreams).
		FindOne(ctx, bson.M{"_id": streamID.String()}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToInboundStream(d)
}

func (s *MongoStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	d := new(inboundStreamDoc)
	if err := s.mdb.Collection(colInboundStreams).
		FindOne(ctx, bson.M{"push_path_hash": hash}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToInboundStream(d)
}

func (s *MongoStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	cur, err := s.mdb.Collection(colInboundStreams).Find(ctx,
		bson.M{"app_id": appID.String()},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, mongoErr(err)
	}
	defer cur.Close(ctx) //nolint:errcheck // cursor close

	out := make([]*InboundStream, 0)
	for cur.Next(ctx) {
		d := new(inboundStreamDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		converted, cerr := docToInboundStream(d)
		if cerr != nil {
			return nil, cerr
		}
		out = append(out, converted)
	}
	return out, cur.Err()
}

func (s *MongoStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.mdb.Collection(colInboundStreams).ReplaceOne(ctx,
		bson.M{"_id": in.ID.String()}, inboundStreamToDoc(in))
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.mdb.Collection(colInboundStreams).
		DeleteOne(ctx, bson.M{"_id": streamID.String()})
	if err != nil {
		return mongoErr(err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now
	filter := bson.M{
		"app_id": l.AppID.String(), "env_id": l.EnvID.String(),
		"issuer": l.Issuer, "subject": l.Subject,
	}
	_, err := s.mdb.Collection(colSubjectLinks).ReplaceOne(ctx, filter,
		subjectLinkToDoc(l), options.Replace().SetUpsert(true))
	return mongoErr(err)
}

func (s *MongoStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	d := new(subjectLinkDoc)
	if err := s.mdb.Collection(colSubjectLinks).FindOne(ctx, bson.M{
		"app_id": appID.String(), "env_id": envID.String(),
		"issuer": issuer, "subject": subject,
	}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToSubjectLink(d)
}

func (s *MongoStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.mdb.Collection(colReceivedEvents).InsertOne(ctx, receivedEventToDoc(e))
	if isMongoDuplicate(err) {
		return ErrDuplicateJTI
	}
	return mongoErr(err)
}

func (s *MongoStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.mdb.Collection(colReceivedEvents).ReplaceOne(ctx,
		bson.M{"_id": e.ID.String()}, receivedEventToDoc(e))
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) CountActionsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	n, err := s.mdb.Collection(colReceivedEvents).CountDocuments(ctx, bson.M{
		"stream_id":    streamID.String(),
		"action_taken": bson.M{"$ne": ""},
		"received_at":  bson.M{"$gt": since},
	})
	if err != nil {
		return 0, mongoErr(err)
	}
	return int(n), nil
}

func (s *MongoStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.mdb.Collection(colSignals).InsertOne(ctx, signalToDoc(sig))
	return mongoErr(err)
}

func (s *MongoStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, userID id.UserID, now time.Time) ([]*Signal, error) {
	cur, err := s.mdb.Collection(colSignals).Find(ctx, bson.M{
		"app_id": appID.String(), "env_id": envID.String(), "user_id": userID.String(),
		"expires_at": bson.M{"$gt": now},
	}, options.Find().SetSort(bson.D{{Key: "severity", Value: -1}}))
	if err != nil {
		return nil, mongoErr(err)
	}
	defer cur.Close(ctx) //nolint:errcheck // cursor close

	out := make([]*Signal, 0)
	for cur.Next(ctx) {
		d := new(signalDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		converted, cerr := docToSignal(d)
		if cerr != nil {
			return nil, cerr
		}
		out = append(out, converted)
	}
	return out, cur.Err()
}
