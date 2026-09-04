package retention

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
)

// MongoStore implements Store on MongoDB. It shares neither models nor
// mappers with the SQL-backed stores: Mongo gets its own bson documents.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB

	// logger reports documents the mappers cannot read. Never nil: the
	// constructor installs a no-op one, and SetLogger replaces it.
	logger log.Logger
}

// NewMongoStore builds a MongoDB-backed store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{db: db, mdb: mongodriver.Unwrap(db), logger: log.NewNoopLogger()}
}

// SetLogger installs the plugin's logger. Optional: the store works
// without it, it just cannot tell anyone about a document it had to skip.
func (s *MongoStore) SetLogger(l log.Logger) {
	if l != nil {
		s.logger = l
	}
}

var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Documents
// ──────────────────────────────────────────────────

type outboxDoc struct {
	grove.BaseModel `grove:"table:authsome_retention_outbox"`

	ID       string            `bson:"_id"`
	AppID    string            `bson:"app_id"`
	EnvID    string            `bson:"env_id"`
	UserID   string            `bson:"user_id"`
	Provider string            `bson:"provider"`
	Kind     string            `bson:"kind"`
	Payload  map[string]string `bson:"payload"`
	// Do not add omitempty here: the partial unique index in
	// MongoMigrations matches on idempotency_key existing as a string
	// ($gt: ""), not on it being non-empty ($ne would cover missing/null
	// too, but Mongo rejects $ne in a partial filter). Omitting the field
	// on a zero value would drop that document out of the index's coverage
	// and silently stop enforcing uniqueness for it.
	IdempotencyKey string     `bson:"idempotency_key"`
	State          string     `bson:"state"`
	Attempts       int        `bson:"attempts"`
	NextAttemptAt  time.Time  `bson:"next_attempt_at"`
	InFlightUntil  *time.Time `bson:"in_flight_until,omitempty"`
	LastError      string     `bson:"last_error"`
	CreatedAt      time.Time  `bson:"created_at"`
}

type contactRefDoc struct {
	grove.BaseModel `grove:"table:authsome_retention_contact_ref"`

	ID               string    `bson:"_id"`
	AppID            string    `bson:"app_id"`
	EnvID            string    `bson:"env_id"`
	UserID           string    `bson:"user_id"`
	Provider         string    `bson:"provider"`
	RemoteObjectType string    `bson:"remote_object_type"`
	RemoteID         string    `bson:"remote_id"`
	SyncedAt         time.Time `bson:"synced_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func jobToDoc(j *Job) *outboxDoc {
	d := &outboxDoc{
		ID: j.ID.String(), AppID: j.AppID.String(), EnvID: j.EnvID.String(),
		UserID: j.UserID.String(), Provider: j.Provider, Kind: j.Kind,
		Payload: j.Payload, IdempotencyKey: j.IdempotencyKey,
		State: j.State, Attempts: j.Attempts,
		NextAttemptAt: j.NextAttemptAt, LastError: j.LastError, CreatedAt: j.CreatedAt,
	}
	if !j.InFlightUntil.IsZero() {
		until := j.InFlightUntil
		d.InFlightUntil = &until
	}
	return d
}

func docToJob(d *outboxDoc) (*Job, error) {
	jobID, err := id.ParseRetentionJobID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID: jobID, AppID: appID, UserID: userID, Provider: d.Provider, Kind: d.Kind,
		IdempotencyKey: d.IdempotencyKey, State: d.State, Attempts: d.Attempts,
		NextAttemptAt: d.NextAttemptAt, LastError: d.LastError, CreatedAt: d.CreatedAt,
	}
	if d.InFlightUntil != nil {
		j.InFlightUntil = *d.InFlightUntil
	}
	// The empty environment is the zero id.EnvironmentID, whose String() is
	// "". ParseEnvironmentID("") would fail, so only parse a non-empty field.
	if d.EnvID != "" {
		envID, err := id.ParseEnvironmentID(d.EnvID)
		if err != nil {
			return nil, err
		}
		j.EnvID = envID
	}
	j.Payload = d.Payload
	if j.Payload == nil {
		j.Payload = make(map[string]string)
	}
	return j, nil
}

func refToDoc(r *ContactRef) *contactRefDoc {
	return &contactRefDoc{
		ID: r.ID.String(), AppID: r.AppID.String(), EnvID: r.EnvID.String(),
		UserID: r.UserID.String(), Provider: r.Provider,
		RemoteObjectType: r.RemoteObjectType, RemoteID: r.RemoteID, SyncedAt: r.SyncedAt,
	}
}

func docToRef(d *contactRefDoc) (*ContactRef, error) {
	refID, err := id.ParseRetentionRefID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	r := &ContactRef{
		ID: refID, AppID: appID, UserID: userID, Provider: d.Provider,
		RemoteObjectType: d.RemoteObjectType, RemoteID: d.RemoteID, SyncedAt: d.SyncedAt,
	}
	// Same empty-environment guard as docToJob.
	if d.EnvID != "" {
		envID, err := id.ParseEnvironmentID(d.EnvID)
		if err != nil {
			return nil, err
		}
		r.EnvID = envID
	}
	return r, nil
}

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

// mongoErr maps a driver miss onto ErrNotFound so callers never see
// mongo.ErrNoDocuments.
func mongoErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}

// isMongoDuplicate reports whether err is a duplicate-key failure, which
// Enqueue treats as success rather than a fault.
func isMongoDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

// Enqueue inserts a pending job, treating a duplicate idempotency key as
// success. The partial unique index on idempotency_key (see MongoMigrations)
// is what makes this safe under concurrency.
func (s *MongoStore) Enqueue(ctx context.Context, j *Job) error {
	if j.State == "" {
		j.State = StatePending
	}
	_, err := s.mdb.Collection(colOutbox).InsertOne(ctx, jobToDoc(j))
	if err != nil {
		if isMongoDuplicate(err) {
			return nil
		}
		return mongoErr(err)
	}
	return nil
}

// claimFilter matches a job that is due: pending and past its next attempt,
// or in_flight with an expired lease. The second clause is what recovers
// work from a process that died mid delivery.
func claimFilter(now time.Time) bson.M {
	return bson.M{
		"$or": bson.A{
			bson.M{"state": StatePending, "next_attempt_at": bson.M{"$lte": now}},
			bson.M{"state": StateInFlight, "in_flight_until": bson.M{"$lt": now}},
		},
	}
}

// ClaimDue loops FindOneAndUpdate with the due-or-expired filter, sorted by
// next_attempt_at then created_at, until it has limit documents or the
// filter stops matching. Each FindOneAndUpdate is atomic per document, which
// is what the contract needs: two callers racing on the same candidate
// cannot both claim it, because the second one's filter no longer matches
// once the first has flipped the state.
//
// limit <= 0 means "no limit," matching every other backend: the loop then
// runs until the filter is exhausted instead of stopping at a count.
func (s *MongoStore) ClaimDue(ctx context.Context, limit int, lease time.Duration,
	now time.Time) ([]*Job, error) {
	until := now.Add(lease)
	update := bson.M{"$set": bson.M{"state": StateInFlight, "in_flight_until": until}}
	// Before, not After. The updated document says in_flight whichever
	// clause matched, so it cannot tell a fresh pending claim from a
	// reclaimed expired lease; the pre-update document can. The two fields
	// the update changes are then written back onto the mapped job below,
	// which is exactly what SqliteStore.ClaimDue does with its own
	// pre-update rows. See Job.Reclaimed.
	opts := options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "next_attempt_at", Value: 1}, {Key: "created_at", Value: 1}}).
		SetReturnDocument(options.Before)

	coll := s.mdb.Collection(colOutbox)
	var out []*Job
	for limit <= 0 || len(out) < limit {
		d := new(outboxDoc)
		err := coll.FindOneAndUpdate(ctx, claimFilter(now), update, opts).Decode(d)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				break
			}
			return nil, mongoErr(err)
		}
		j, jerr := docToJob(d)
		if jerr != nil {
			// One unreadable document must not discard the batch. It is
			// already in_flight, so returning here strands every document
			// claimed with it until their leases expire, and the same
			// poison document is then re-claimed and stalls the queue
			// again, forever. Skipping it means it comes back on every
			// claim once its own lease expires: noisy, but it blocks
			// nothing behind it, and the live lease keeps this loop from
			// re-matching it on the very next iteration.
			s.logger.Warn("retention: skipping unreadable outbox document",
				log.String("job_id", d.ID),
				log.String("error", jerr.Error()))
			continue
		}
		j.Reclaimed = d.State == StateInFlight
		j.State = StateInFlight
		j.InFlightUntil = until
		out = append(out, j)
	}
	return out, nil
}

// MarkDone completes a job. now is intentionally unused: there is no
// completed_at field in the document.
func (s *MongoStore) MarkDone(ctx context.Context, jobID id.RetentionJobID, _ time.Time) error {
	res, err := s.mdb.Collection(colOutbox).UpdateOne(ctx,
		bson.M{"_id": jobID.String()},
		bson.M{"$set": bson.M{"state": StateDone, "last_error": ""}})
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkRetry returns a job to pending with an incremented attempt count.
func (s *MongoStore) MarkRetry(ctx context.Context, jobID id.RetentionJobID,
	nextAttemptAt time.Time, lastErr string) error {
	res, err := s.mdb.Collection(colOutbox).UpdateOne(ctx,
		bson.M{"_id": jobID.String()},
		bson.M{
			"$set": bson.M{
				"state":           StatePending,
				"next_attempt_at": nextAttemptAt,
				"last_error":      lastErr,
			},
			"$unset": bson.M{"in_flight_until": ""},
			"$inc":   bson.M{"attempts": 1},
		})
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkDeferred returns a job to pending at nextAttemptAt without touching
// the attempt count. See the Store interface for why that is a separate
// method rather than a flag on MarkRetry.
func (s *MongoStore) MarkDeferred(ctx context.Context, jobID id.RetentionJobID,
	nextAttemptAt time.Time, reason string) error {
	res, err := s.mdb.Collection(colOutbox).UpdateOne(ctx,
		bson.M{"_id": jobID.String()},
		bson.M{
			"$set": bson.M{
				"state":           StatePending,
				"next_attempt_at": nextAttemptAt,
				"last_error":      reason,
			},
			"$unset": bson.M{"in_flight_until": ""},
		})
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkDead parks a job permanently after too many attempts.
func (s *MongoStore) MarkDead(ctx context.Context, jobID id.RetentionJobID, lastErr string) error {
	res, err := s.mdb.Collection(colOutbox).UpdateOne(ctx,
		bson.M{"_id": jobID.String()},
		bson.M{"$set": bson.M{"state": StateDead, "last_error": lastErr}})
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkSuppressed records that the job was deliberately not delivered.
func (s *MongoStore) MarkSuppressed(ctx context.Context, jobID id.RetentionJobID, reason string) error {
	res, err := s.mdb.Collection(colOutbox).UpdateOne(ctx,
		bson.M{"_id": jobID.String()},
		bson.M{"$set": bson.M{"state": StateSuppressed, "last_error": reason}})
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// PurgeTerminal deletes terminal documents older than their class cutoff.
//
// A TTL index is the obvious Mongo-native alternative and it was rejected.
// Partial TTL indexes on an ordinary collection work fine -- expireAfterSeconds
// alongside partialFilterExpression has been supported since MongoDB 3.2 --
// so "terminal states only" was never the blocker. What kills it: TTL
// indexes are single-field, and MongoDB refuses two indexes with the same
// key specification differing only in options, so `created_at` cannot carry
// both a 30-day window for done and a 180-day one for the audit states.
// Beyond that the reaper runs on its own roughly-60-second schedule, only on
// a primary, and would put the retention window in a migration constant
// instead of the plugin's config: Mongo would be the one backend whose
// purge behaviour the conformance suite cannot observe. An explicit
// DeleteMany keeps all four backends answering to the same contract.
func (s *MongoStore) PurgeTerminal(ctx context.Context, doneBefore, auditBefore time.Time) (int, error) {
	coll := s.mdb.Collection(colOutbox)
	total := 0
	for _, c := range purgeClasses(doneBefore, auditBefore) {
		res, err := coll.DeleteMany(ctx, bson.M{
			"state":      c.State,
			"created_at": bson.M{"$lt": c.Before},
		})
		if err != nil {
			return total, mongoErr(err)
		}
		total += int(res.DeletedCount)
	}
	return total, nil
}

// GetJob fetches one job. Returns ErrNotFound when absent.
func (s *MongoStore) GetJob(ctx context.Context, jobID id.RetentionJobID) (*Job, error) {
	d := new(outboxDoc)
	if err := s.mdb.Collection(colOutbox).
		FindOne(ctx, bson.M{"_id": jobID.String()}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToJob(d)
}

// ListDead returns dead-lettered jobs for an app, newest first.
func (s *MongoStore) ListDead(ctx context.Context, appID id.AppID, limit int) ([]*Job, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}
	cur, err := s.mdb.Collection(colOutbox).Find(ctx,
		bson.M{"app_id": appID.String(), "state": StateDead}, findOpts)
	if err != nil {
		return nil, mongoErr(err)
	}
	defer func() { _ = cur.Close(ctx) }()

	out := make([]*Job, 0)
	for cur.Next(ctx) {
		d := new(outboxDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		j, jerr := docToJob(d)
		if jerr != nil {
			return nil, jerr
		}
		out = append(out, j)
	}
	return out, cur.Err()
}

// GetRef returns the contact ref. Returns ErrNotFound when absent.
func (s *MongoStore) GetRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	d := new(contactRefDoc)
	if err := s.mdb.Collection(colContactRef).FindOne(ctx, refFilter(appID, envID, userID, provider)).
		Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToRef(d)
}

func refFilter(appID id.AppID, envID id.EnvironmentID, userID id.UserID, provider string) bson.M {
	return bson.M{
		"app_id": appID.String(), "env_id": envID.String(),
		"user_id": userID.String(), "provider": provider,
	}
}

// updateRefByFilter updates the mutable fields of the ref addressed by the
// unique (app_id, env_id, user_id, provider) tuple. Addressing it by the
// tuple rather than by an _id read a moment ago is what lets the insert
// path fall back onto it after losing a race.
func (s *MongoStore) updateRefByFilter(ctx context.Context, r *ContactRef) error {
	_, err := s.mdb.Collection(colContactRef).UpdateOne(ctx,
		refFilter(r.AppID, r.EnvID, r.UserID, r.Provider),
		bson.M{"$set": bson.M{
			"remote_object_type": r.RemoteObjectType,
			"remote_id":          r.RemoteID,
			"synced_at":          r.SyncedAt,
		}})
	return mongoErr(err)
}

// PutRef inserts or updates the contact ref for the unique tuple. It looks
// up the existing document by the unique (app_id, env_id, user_id, provider)
// tuple first: an insert when absent, otherwise an update of the fields that
// can change.
//
// Find-then-insert is not atomic, so two workers can both miss and both
// insert. That is likely rather than theoretical: a signup enqueues a
// contact_upsert and an activity_log for the same user in the same breath.
// The loser gets a duplicate key on the unique tuple index, so it falls
// back to the update instead of returning an error that would cost the job
// its retry budget for a document that is already where it should be.
func (s *MongoStore) PutRef(ctx context.Context, r *ContactRef) error {
	coll := s.mdb.Collection(colContactRef)
	filter := refFilter(r.AppID, r.EnvID, r.UserID, r.Provider)

	existing := new(contactRefDoc)
	err := coll.FindOne(ctx, filter).Decode(existing)
	switch {
	case err == nil:
		return s.updateRefByFilter(ctx, r)
	case errors.Is(err, mongo.ErrNoDocuments):
		_, ierr := coll.InsertOne(ctx, refToDoc(r))
		if ierr != nil && isMongoDuplicate(ierr) {
			return s.updateRefByFilter(ctx, r)
		}
		return mongoErr(ierr)
	default:
		return mongoErr(err)
	}
}

// DeleteRef removes the contact ref. Deleting an absent ref is not an error.
func (s *MongoStore) DeleteRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) error {
	_, err := s.mdb.Collection(colContactRef).
		DeleteOne(ctx, refFilter(appID, envID, userID, provider))
	return mongoErr(err)
}

// ListRefsForUser returns every ref held for the user across all apps and
// providers. Deliberately unscoped by app: a data-subject export covers the
// person, not one app's view of them.
func (s *MongoStore) ListRefsForUser(ctx context.Context, userID id.UserID) ([]*ContactRef, error) {
	cur, err := s.mdb.Collection(colContactRef).Find(ctx, bson.M{"user_id": userID.String()})
	if err != nil {
		return nil, mongoErr(err)
	}
	defer func() { _ = cur.Close(ctx) }()

	out := make([]*ContactRef, 0)
	for cur.Next(ctx) {
		d := new(contactRefDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		r, rerr := docToRef(d)
		if rerr != nil {
			return nil, rerr
		}
		out = append(out, r)
	}
	return out, cur.Err()
}
