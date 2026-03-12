package passkey

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

// MongoStore implements passkey.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed passkey store.
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

type credentialDoc struct {
	ID              string    `bson:"_id"`
	UserID          string    `bson:"user_id"`
	AppID           string    `bson:"app_id"`
	CredentialID    []byte    `bson:"credential_id"`
	PublicKey       []byte    `bson:"public_key"`
	AttestationType string    `bson:"attestation_type"`
	Transport       string    `bson:"transport"` // comma-separated
	SignCount       int       `bson:"sign_count"`
	AAGUID          []byte    `bson:"aaguid,omitempty"`
	DisplayName     string    `bson:"display_name"`
	CreatedAt       time.Time `bson:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func credentialDocToCredential(d *credentialDoc) (*Credential, error) {
	pkID, err := id.ParsePasskeyID(d.ID)
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

	var transport []string
	if d.Transport != "" {
		transport = strings.Split(d.Transport, ",")
	}

	return &Credential{
		ID:              pkID,
		UserID:          userID,
		AppID:           appID,
		CredentialID:    d.CredentialID,
		PublicKey:       d.PublicKey,
		AttestationType: d.AttestationType,
		Transport:       transport,
		SignCount:       uint32(d.SignCount), //nolint:gosec // G115: sign count validated range
		AAGUID:          d.AAGUID,
		DisplayName:     d.DisplayName,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}, nil
}

func credentialToDoc(c *Credential) *credentialDoc {
	return &credentialDoc{
		ID:              c.ID.String(),
		UserID:          c.UserID.String(),
		AppID:           c.AppID.String(),
		CredentialID:    c.CredentialID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       strings.Join(c.Transport, ","),
		SignCount:       int(c.SignCount),
		AAGUID:          c.AAGUID,
		DisplayName:     c.DisplayName,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Collection name
// ──────────────────────────────────────────────────

const passkeyCredentialsColl = "authsome_passkey_credentials"

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateCredential(ctx context.Context, c *Credential) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	doc := credentialToDoc(c)
	_, err := s.mdb.Collection(passkeyCredentialsColl).InsertOne(ctx, doc)
	return passkeyMongoError(err)
}

func (s *MongoStore) GetCredential(ctx context.Context, credentialID []byte) (*Credential, error) {
	doc := new(credentialDoc)
	err := s.mdb.Collection(passkeyCredentialsColl).FindOne(ctx, bson.M{
		"credential_id": credentialID,
	}).Decode(doc)
	if err != nil {
		return nil, passkeyMongoError(err)
	}
	return credentialDocToCredential(doc)
}

func (s *MongoStore) ListUserCredentials(ctx context.Context, userID id.UserID) ([]*Credential, error) {
	cursor, err := s.mdb.Collection(passkeyCredentialsColl).Find(ctx, bson.M{
		"user_id": userID.String(),
	})
	if err != nil {
		return nil, passkeyMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []credentialDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, passkeyMongoError(err)
	}

	result := make([]*Credential, 0, len(docs))
	for i := range docs {
		c, err := credentialDocToCredential(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *MongoStore) DeleteCredential(ctx context.Context, credentialID []byte) error {
	_, err := s.mdb.Collection(passkeyCredentialsColl).DeleteOne(ctx, bson.M{
		"credential_id": credentialID,
	})
	return passkeyMongoError(err)
}

func (s *MongoStore) UpdateSignCount(ctx context.Context, credentialID []byte, count uint32) error {
	now := time.Now()
	_, err := s.mdb.Collection(passkeyCredentialsColl).UpdateOne(ctx,
		bson.M{"credential_id": credentialID},
		bson.M{"$set": bson.M{
			"sign_count": int(count),
			"updated_at": now,
		}},
	)
	return passkeyMongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func passkeyIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func passkeyMongoError(err error) error {
	if err == nil {
		return nil
	}
	if passkeyIsNoDocuments(err) {
		return ErrCredentialNotFound
	}
	return err
}
