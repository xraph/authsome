package consent

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

// MongoStore implements consent.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed consent store.
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

type consentDoc struct {
	ID        string     `bson:"_id"`
	UserID    string     `bson:"user_id"`
	AppID     string     `bson:"app_id"`
	Purpose   string     `bson:"purpose"`
	Granted   bool       `bson:"granted"`
	Version   string     `bson:"version"`
	IPAddress string     `bson:"ip_address"`
	GrantedAt time.Time  `bson:"granted_at"`
	RevokedAt *time.Time `bson:"revoked_at,omitempty"`
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func consentDocToModel(d *consentDoc) (*Consent, error) {
	consentID, err := id.ParseConsentID(d.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	return &Consent{
		ID:        consentID,
		UserID:    userID,
		AppID:     appID,
		Purpose:   d.Purpose,
		Granted:   d.Granted,
		Version:   d.Version,
		IPAddress: d.IPAddress,
		GrantedAt: d.GrantedAt,
		RevokedAt: d.RevokedAt,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}, nil
}

func consentToDoc(c *Consent) *consentDoc {
	return &consentDoc{
		ID:        c.ID.String(),
		UserID:    c.UserID.String(),
		AppID:     c.AppID.String(),
		Purpose:   c.Purpose,
		Granted:   c.Granted,
		Version:   c.Version,
		IPAddress: c.IPAddress,
		GrantedAt: c.GrantedAt,
		RevokedAt: c.RevokedAt,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Collection name
// ──────────────────────────────────────────────────

const consentsColl = "authsome_consents"

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) GrantConsent(ctx context.Context, c *Consent) error {
	now := time.Now()
	if c.ID.IsNil() {
		c.ID = id.NewConsentID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	doc := consentToDoc(c)
	_, err := s.mdb.Collection(consentsColl).UpdateOne(ctx,
		bson.M{
			"user_id": doc.UserID,
			"app_id":  doc.AppID,
			"purpose": doc.Purpose,
		},
		bson.M{
			"$set": bson.M{
				"_id":        doc.ID,
				"granted":    doc.Granted,
				"version":    doc.Version,
				"ip_address": doc.IPAddress,
				"granted_at": doc.GrantedAt,
				"revoked_at": doc.RevokedAt,
				"updated_at": doc.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"user_id":    doc.UserID,
				"app_id":     doc.AppID,
				"purpose":    doc.Purpose,
				"created_at": doc.CreatedAt,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return consentMongoError(err)
}

func (s *MongoStore) RevokeConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) error {
	now := time.Now()
	res, err := s.mdb.Collection(consentsColl).UpdateOne(ctx,
		bson.M{
			"user_id": userID.String(),
			"app_id":  appID.String(),
			"purpose": purpose,
		},
		bson.M{
			"$set": bson.M{
				"granted":    false,
				"revoked_at": now,
				"updated_at": now,
			},
		},
	)
	if err != nil {
		return consentMongoError(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) GetConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (*Consent, error) {
	doc := new(consentDoc)
	err := s.mdb.Collection(consentsColl).FindOne(ctx, bson.M{
		"user_id": userID.String(),
		"app_id":  appID.String(),
		"purpose": purpose,
	}).Decode(doc)
	if err != nil {
		return nil, consentMongoError(err)
	}
	return consentDocToModel(doc)
}

func (s *MongoStore) ListConsents(ctx context.Context, q *Query) ([]*Consent, string, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	filter := bson.M{}
	if q.UserID.Prefix() != "" {
		filter["user_id"] = q.UserID.String()
	}
	if q.AppID.Prefix() != "" {
		filter["app_id"] = q.AppID.String()
	}
	if q.Purpose != "" {
		filter["purpose"] = q.Purpose
	}
	if q.Cursor != "" {
		filter["_id"] = bson.M{"$gt": q.Cursor}
	}

	opts := options.Find().
		SetSort(bson.M{"_id": 1}).
		SetLimit(int64(limit + 1))

	cursor, err := s.mdb.Collection(consentsColl).Find(ctx, filter, opts)
	if err != nil {
		return nil, "", consentMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []consentDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, "", consentMongoError(err)
	}

	var nextCursor string
	if len(docs) > limit {
		nextCursor = docs[limit-1].ID
		docs = docs[:limit]
	}

	result := make([]*Consent, 0, len(docs))
	for i := range docs {
		c, err := consentDocToModel(&docs[i])
		if err != nil {
			return nil, "", err
		}
		result = append(result, c)
	}

	return result, nextCursor, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func consentIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func consentMongoError(err error) error {
	if err == nil {
		return nil
	}
	if consentIsNoDocuments(err) {
		return ErrNotFound
	}
	return err
}
