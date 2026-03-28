package waitlist

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

// MongoStore implements waitlist.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed waitlist store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{
		db:  db,
		mdb: mongodriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Mongo document model
// ──────────────────────────────────────────────────

type waitlistDoc struct {
	ID        string    `bson:"_id"`
	AppID     string    `bson:"app_id"`
	Email     string    `bson:"email"`
	Name      string    `bson:"name"`
	Status    string    `bson:"status"`
	UserID    string    `bson:"user_id,omitempty"`
	IPAddress string    `bson:"ip_address"`
	Note      string    `bson:"note"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func waitlistDocToEntry(d *waitlistDoc) (*WaitlistEntry, error) {
	entryID, err := id.ParseWaitlistID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	e := &WaitlistEntry{
		ID:        entryID,
		AppID:     appID,
		Email:     d.Email,
		Name:      d.Name,
		Status:    WaitlistStatus(d.Status),
		IPAddress: d.IPAddress,
		Note:      d.Note,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}

	if d.UserID != "" {
		uid, err := id.ParseUserID(d.UserID)
		if err != nil {
			return nil, err
		}
		e.UserID = &uid
	}

	return e, nil
}

func entryToWaitlistDoc(e *WaitlistEntry) *waitlistDoc {
	d := &waitlistDoc{
		ID:        e.ID.String(),
		AppID:     e.AppID.String(),
		Email:     e.Email,
		Name:      e.Name,
		Status:    string(e.Status),
		IPAddress: e.IPAddress,
		Note:      e.Note,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.UserID != nil {
		d.UserID = e.UserID.String()
	}
	return d
}

// ──────────────────────────────────────────────────
// Collection name
// ──────────────────────────────────────────────────

const waitlistColl = "authsome_waitlist_entries"

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateEntry(ctx context.Context, e *WaitlistEntry) error {
	now := time.Now()
	if e.ID.IsNil() {
		e.ID = id.NewWaitlistID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	doc := entryToWaitlistDoc(e)
	_, err := s.mdb.Collection(waitlistColl).InsertOne(ctx, doc)
	return waitlistMongoError(err)
}

func (s *MongoStore) GetEntry(ctx context.Context, entryID id.WaitlistID) (*WaitlistEntry, error) {
	doc := new(waitlistDoc)
	err := s.mdb.Collection(waitlistColl).FindOne(ctx, bson.M{
		"_id": entryID.String(),
	}).Decode(doc)
	if err != nil {
		return nil, waitlistMongoError(err)
	}
	return waitlistDocToEntry(doc)
}

func (s *MongoStore) GetEntryByEmail(ctx context.Context, appID id.AppID, email string) (*WaitlistEntry, error) {
	doc := new(waitlistDoc)
	err := s.mdb.Collection(waitlistColl).FindOne(ctx, bson.M{
		"app_id": appID.String(),
		"email":  strings.ToLower(email),
	}).Decode(doc)
	if err != nil {
		return nil, waitlistMongoError(err)
	}
	return waitlistDocToEntry(doc)
}

func (s *MongoStore) UpdateEntryStatus(ctx context.Context, entryID id.WaitlistID, status WaitlistStatus, note string) error {
	now := time.Now()
	res, err := s.mdb.Collection(waitlistColl).UpdateOne(ctx,
		bson.M{"_id": entryID.String()},
		bson.M{
			"$set": bson.M{
				"status":     string(status),
				"note":       note,
				"updated_at": now,
			},
		},
	)
	if err != nil {
		return waitlistMongoError(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) ListEntries(ctx context.Context, q *WaitlistQuery) (*WaitlistList, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	filter := bson.M{}
	if q.AppID.Prefix() != "" {
		filter["app_id"] = q.AppID.String()
	}
	if q.Email != "" {
		filter["email"] = strings.ToLower(q.Email)
	}
	if q.Status != "" {
		filter["status"] = string(q.Status)
	}
	if q.Cursor != "" {
		filter["_id"] = bson.M{"$gt": q.Cursor}
	}

	opts := options.Find().
		SetSort(bson.M{"_id": 1}).
		SetLimit(int64(limit + 1))

	cursor, err := s.mdb.Collection(waitlistColl).Find(ctx, filter, opts)
	if err != nil {
		return nil, waitlistMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []waitlistDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, waitlistMongoError(err)
	}

	var nextCursor string
	if len(docs) > limit {
		nextCursor = docs[limit-1].ID
		docs = docs[:limit]
	}

	entries := make([]*WaitlistEntry, 0, len(docs))
	for i := range docs {
		e, err := waitlistDocToEntry(&docs[i])
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return &WaitlistList{
		Entries:    entries,
		Total:      len(entries),
		NextCursor: nextCursor,
	}, nil
}

func (s *MongoStore) CountByStatus(ctx context.Context, appID id.AppID) (pending, approved, rejected int, err error) {
	coll := s.mdb.Collection(waitlistColl)

	countStatus := func(status WaitlistStatus) (int, error) {
		n, countErr := coll.CountDocuments(ctx, bson.M{
			"app_id": appID.String(),
			"status": string(status),
		})
		if countErr != nil {
			return 0, waitlistMongoError(countErr)
		}
		return int(n), nil
	}

	pending, err = countStatus(StatusPending)
	if err != nil {
		return
	}
	approved, err = countStatus(StatusApproved)
	if err != nil {
		return
	}
	rejected, err = countStatus(StatusRejected)
	return
}

func (s *MongoStore) DeleteEntry(ctx context.Context, entryID id.WaitlistID) error {
	res, err := s.mdb.Collection(waitlistColl).DeleteOne(ctx, bson.M{
		"_id": entryID.String(),
	})
	if err != nil {
		return waitlistMongoError(err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func waitlistIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func waitlistMongoError(err error) error {
	if err == nil {
		return nil
	}
	if waitlistIsNoDocuments(err) {
		return ErrNotFound
	}
	if strings.Contains(err.Error(), "duplicate key") {
		return ErrDuplicateEmail
	}
	return err
}
