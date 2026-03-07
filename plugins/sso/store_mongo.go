package sso

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

// MongoStore implements sso.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed SSO connection store.
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

type ssoConnectionDoc struct {
	ID          string    `bson:"_id"`
	AppID       string    `bson:"app_id"`
	OrgID       string    `bson:"org_id"`
	Provider    string    `bson:"provider"`
	Protocol    string    `bson:"protocol"`
	Domain      string    `bson:"domain"`
	MetadataURL string    `bson:"metadata_url"`
	ClientID     string    `bson:"client_id"`
	ClientSecret string    `bson:"client_secret"`
	Issuer       string    `bson:"issuer"`
	Active      bool      `bson:"active"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func ssoDocToConnection(d *ssoConnectionDoc) (*SSOConnection, error) {
	connID, err := id.ParseSSOConnectionID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	c := &SSOConnection{
		ID:          connID,
		AppID:       appID,
		Provider:    d.Provider,
		Protocol:    d.Protocol,
		Domain:      d.Domain,
		MetadataURL: d.MetadataURL,
		ClientID:     d.ClientID,
		ClientSecret: d.ClientSecret,
		Issuer:       d.Issuer,
		Active:      d.Active,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}

	if d.OrgID != "" {
		orgID, err := id.ParseOrgID(d.OrgID)
		if err != nil {
			return nil, err
		}
		c.OrgID = orgID
	}

	return c, nil
}

func ssoConnectionToDoc(c *SSOConnection) *ssoConnectionDoc {
	doc := &ssoConnectionDoc{
		ID:          c.ID.String(),
		AppID:       c.AppID.String(),
		Provider:    c.Provider,
		Protocol:    c.Protocol,
		Domain:      c.Domain,
		MetadataURL: c.MetadataURL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Issuer:       c.Issuer,
		Active:      c.Active,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
	if c.OrgID.Prefix() != "" {
		doc.OrgID = c.OrgID.String()
	}
	return doc
}

// ──────────────────────────────────────────────────
// Collection name
// ──────────────────────────────────────────────────

const ssoConnectionsColl = "authsome_sso_connections"

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateSSOConnection(ctx context.Context, c *SSOConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	doc := ssoConnectionToDoc(c)
	_, err := s.mdb.Collection(ssoConnectionsColl).InsertOne(ctx, doc)
	return ssoMongoError(err)
}

func (s *MongoStore) GetSSOConnection(ctx context.Context, connID id.SSOConnectionID) (*SSOConnection, error) {
	doc := new(ssoConnectionDoc)
	err := s.mdb.Collection(ssoConnectionsColl).FindOne(ctx, bson.M{
		"_id": connID.String(),
	}).Decode(doc)
	if err != nil {
		return nil, ssoMongoError(err)
	}
	return ssoDocToConnection(doc)
}

func (s *MongoStore) GetSSOConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*SSOConnection, error) {
	doc := new(ssoConnectionDoc)
	err := s.mdb.Collection(ssoConnectionsColl).FindOne(ctx, bson.M{
		"app_id": appID.String(),
		"domain": domain,
		"active": true,
	}).Decode(doc)
	if err != nil {
		return nil, ssoMongoError(err)
	}
	return ssoDocToConnection(doc)
}

func (s *MongoStore) GetSSOConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*SSOConnection, error) {
	doc := new(ssoConnectionDoc)
	err := s.mdb.Collection(ssoConnectionsColl).FindOne(ctx, bson.M{
		"app_id":   appID.String(),
		"provider": provider,
		"active":   true,
	}).Decode(doc)
	if err != nil {
		return nil, ssoMongoError(err)
	}
	return ssoDocToConnection(doc)
}

func (s *MongoStore) ListSSOConnections(ctx context.Context, appID id.AppID) ([]*SSOConnection, error) {
	cursor, err := s.mdb.Collection(ssoConnectionsColl).Find(ctx, bson.M{
		"app_id": appID.String(),
	})
	if err != nil {
		return nil, ssoMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []ssoConnectionDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, ssoMongoError(err)
	}

	result := make([]*SSOConnection, 0, len(docs))
	for i := range docs {
		c, err := ssoDocToConnection(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *MongoStore) UpdateSSOConnection(ctx context.Context, c *SSOConnection) error {
	c.UpdatedAt = time.Now()
	doc := ssoConnectionToDoc(c)
	_, err := s.mdb.Collection(ssoConnectionsColl).UpdateOne(ctx,
		bson.M{"_id": c.ID.String()},
		bson.M{"$set": bson.M{
			"app_id":       doc.AppID,
			"org_id":       doc.OrgID,
			"provider":     doc.Provider,
			"protocol":     doc.Protocol,
			"domain":       doc.Domain,
			"metadata_url": doc.MetadataURL,
			"client_id":     doc.ClientID,
			"client_secret": doc.ClientSecret,
			"issuer":        doc.Issuer,
			"active":       doc.Active,
			"updated_at":   doc.UpdatedAt,
		}},
	)
	return ssoMongoError(err)
}

func (s *MongoStore) DeleteSSOConnection(ctx context.Context, connID id.SSOConnectionID) error {
	_, err := s.mdb.Collection(ssoConnectionsColl).DeleteOne(ctx, bson.M{
		"_id": connID.String(),
	})
	return ssoMongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func ssoIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func ssoMongoError(err error) error {
	if err == nil {
		return nil
	}
	if ssoIsNoDocuments(err) {
		return ErrConnectionNotFound
	}
	return err
}
