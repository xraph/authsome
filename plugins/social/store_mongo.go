package social

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
)

// MongoStore implements social.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed social/OAuth store.
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

type oauthConnectionDoc struct {
	ID             string            `bson:"_id"`
	AppID          string            `bson:"app_id"`
	UserID         string            `bson:"user_id"`
	Provider       string            `bson:"provider"`
	ProviderUserID string            `bson:"provider_user_id"`
	Email          string            `bson:"email"`
	AccessToken    string            `bson:"access_token"`
	RefreshToken   string            `bson:"refresh_token"`
	ExpiresAt      *time.Time        `bson:"expires_at,omitempty"`
	Metadata       map[string]string `bson:"metadata,omitempty"`
	CreatedAt      time.Time         `bson:"created_at"`
	UpdatedAt      time.Time         `bson:"updated_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func oauthDocToConnection(d *oauthConnectionDoc) (*OAuthConnection, error) {
	connID, err := id.ParseOAuthConnectionID(d.ID)
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

	md := d.Metadata
	if md == nil {
		md = make(map[string]string)
	}

	c := &OAuthConnection{
		ID:             connID,
		AppID:          appID,
		UserID:         userID,
		Provider:       d.Provider,
		ProviderUserID: d.ProviderUserID,
		Email:          d.Email,
		AccessToken:    d.AccessToken,
		RefreshToken:   d.RefreshToken,
		Metadata:       md,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
	if d.ExpiresAt != nil {
		c.ExpiresAt = *d.ExpiresAt
	}
	return c, nil
}

func oauthConnectionToDoc(c *OAuthConnection) *oauthConnectionDoc {
	md := c.Metadata
	if md == nil {
		md = make(map[string]string)
	}

	doc := &oauthConnectionDoc{
		ID:             c.ID.String(),
		AppID:          c.AppID.String(),
		UserID:         c.UserID.String(),
		Provider:       c.Provider,
		ProviderUserID: c.ProviderUserID,
		Email:          c.Email,
		AccessToken:    c.AccessToken,
		RefreshToken:   c.RefreshToken,
		Metadata:       md,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
	if !c.ExpiresAt.IsZero() {
		doc.ExpiresAt = &c.ExpiresAt
	}
	return doc
}

// ──────────────────────────────────────────────────
// Collection name
// ──────────────────────────────────────────────────

const oauthConnectionsColl = "authsome_oauth_connections"

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateOAuthConnection(ctx context.Context, c *OAuthConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	doc := oauthConnectionToDoc(c)
	_, err := s.mdb.Collection(oauthConnectionsColl).InsertOne(ctx, doc)
	return socialMongoError(err)
}

func (s *MongoStore) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	doc := new(oauthConnectionDoc)
	err := s.mdb.Collection(oauthConnectionsColl).FindOne(ctx, bson.M{
		"provider":         provider,
		"provider_user_id": providerUserID,
	}).Decode(doc)
	if err != nil {
		return nil, socialMongoError(err)
	}
	return oauthDocToConnection(doc)
}

func (s *MongoStore) GetOAuthConnectionsByUserID(ctx context.Context, userID id.UserID) ([]*OAuthConnection, error) {
	cursor, err := s.mdb.Collection(oauthConnectionsColl).Find(ctx, bson.M{
		"user_id": userID.String(),
	})
	if err != nil {
		return nil, socialMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []oauthConnectionDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, socialMongoError(err)
	}

	result := make([]*OAuthConnection, 0, len(docs))
	for i := range docs {
		c, err := oauthDocToConnection(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *MongoStore) DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error {
	_, err := s.mdb.Collection(oauthConnectionsColl).DeleteOne(ctx, bson.M{
		"_id": connID.String(),
	})
	return socialMongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func socialIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func socialMongoError(err error) error {
	if err == nil {
		return nil
	}
	if socialIsNoDocuments(err) {
		return ErrConnectionNotFound
	}
	return err
}
