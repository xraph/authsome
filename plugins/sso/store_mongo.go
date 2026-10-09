package sso

import (
	"context"
	"encoding/json"
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
	ID           string `bson:"_id"`
	AppID        string `bson:"app_id"`
	EnvID        string `bson:"env_id"`
	OrgID        string `bson:"org_id"`
	Provider     string `bson:"provider"`
	Protocol     string `bson:"protocol"`
	Domain       string `bson:"domain"`
	DisplayName  string `bson:"display_name"`
	MetadataURL  string `bson:"metadata_url"`
	ClientID     string `bson:"client_id"`
	ClientSecret string `bson:"client_secret"`
	Issuer       string `bson:"issuer"`
	// SAML fields. AttributeMappings is stored as a JSON object string.
	IDPMetadataXML    string    `bson:"idp_metadata_xml"`
	IDPSSOURL         string    `bson:"idp_sso_url"`
	IDPCertificate    string    `bson:"idp_certificate"`
	EntityID          string    `bson:"entity_id"`
	ACSURL            string    `bson:"acs_url"`
	SPCertificate     string    `bson:"sp_certificate"`
	SPPrivateKey      string    `bson:"sp_private_key"`
	SignRequests      bool      `bson:"sign_requests"`
	AttributeMappings string    `bson:"attribute_mappings"`
	Active            bool      `bson:"active"`
	Enforced          bool      `bson:"enforced"`
	AllowedDomains    []string  `bson:"allowed_domains,omitempty"`
	TrustedFederation bool      `bson:"trusted_federation"`
	CreatedAt         time.Time `bson:"created_at"`
	UpdatedAt         time.Time `bson:"updated_at"`
}

// ssoIdentityDoc is one subject-to-user binding; _id joins the connection
// id and subject so the pair is unique without an extra index.
type ssoIdentityDoc struct {
	ID           string    `bson:"_id"`
	ConnectionID string    `bson:"connection_id"`
	Subject      string    `bson:"subject"`
	UserID       string    `bson:"user_id"`
	CreatedAt    time.Time `bson:"created_at"`
}

const ssoIdentitiesColl = "authsome_sso_identities"

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func ssoDocToConnection(d *ssoConnectionDoc) (*Connection, error) {
	connID, err := id.ParseSSOConnectionID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		ID:             connID,
		AppID:          appID,
		EnvID:          d.EnvID,
		Provider:       d.Provider,
		Protocol:       d.Protocol,
		Domain:         d.Domain,
		DisplayName:    d.DisplayName,
		MetadataURL:    d.MetadataURL,
		ClientID:       d.ClientID,
		ClientSecret:   d.ClientSecret,
		Issuer:         d.Issuer,
		IDPMetadataXML: d.IDPMetadataXML,
		IDPSSOURL:      d.IDPSSOURL,
		IDPCertificate: d.IDPCertificate,
		EntityID:       d.EntityID,
		ACSURL:         d.ACSURL,
		SPCertificate:  d.SPCertificate,
		SPPrivateKey:   d.SPPrivateKey,
		SignRequests:   d.SignRequests,
		Active:         d.Active,
		Enforced:       d.Enforced,
		AllowedDomains: append([]string(nil), d.AllowedDomains...),

		TrustedFederation: d.TrustedFederation,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}

	if d.AttributeMappings != "" {
		if err := json.Unmarshal([]byte(d.AttributeMappings), &c.AttributeMappings); err != nil {
			return nil, err
		}
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

func ssoConnectionToDoc(c *Connection) *ssoConnectionDoc {
	doc := &ssoConnectionDoc{
		ID:             c.ID.String(),
		AppID:          c.AppID.String(),
		EnvID:          c.EnvID,
		Provider:       c.Provider,
		Protocol:       c.Protocol,
		Domain:         c.Domain,
		DisplayName:    c.DisplayName,
		MetadataURL:    c.MetadataURL,
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		Issuer:         c.Issuer,
		IDPMetadataXML: c.IDPMetadataXML,
		IDPSSOURL:      c.IDPSSOURL,
		IDPCertificate: c.IDPCertificate,
		EntityID:       c.EntityID,
		ACSURL:         c.ACSURL,
		SPCertificate:  c.SPCertificate,
		SPPrivateKey:   c.SPPrivateKey,
		SignRequests:   c.SignRequests,
		Active:         c.Active,
		Enforced:       c.Enforced,
		AllowedDomains: append([]string(nil), c.AllowedDomains...),

		TrustedFederation: c.TrustedFederation,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
	if len(c.AttributeMappings) > 0 {
		if b, err := json.Marshal(c.AttributeMappings); err == nil {
			doc.AttributeMappings = string(b)
		}
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

func (s *MongoStore) CreateConnection(ctx context.Context, c *Connection) error {
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

func (s *MongoStore) GetConnection(ctx context.Context, connID id.SSOConnectionID) (*Connection, error) {
	doc := new(ssoConnectionDoc)
	err := s.mdb.Collection(ssoConnectionsColl).FindOne(ctx, bson.M{
		"_id": connID.String(),
	}).Decode(doc)
	if err != nil {
		return nil, ssoMongoError(err)
	}
	return ssoDocToConnection(doc)
}

func (s *MongoStore) GetConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*Connection, error) {
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

func (s *MongoStore) GetConnectionByDomainAndOrg(ctx context.Context, appID id.AppID, orgID id.OrgID, domain string) (*Connection, error) {
	doc := new(ssoConnectionDoc)
	err := s.mdb.Collection(ssoConnectionsColl).FindOne(ctx, bson.M{
		"app_id": appID.String(),
		"org_id": orgID.String(),
		"domain": domain,
		"active": true,
	}).Decode(doc)
	if err != nil {
		return nil, ssoMongoError(err)
	}
	return ssoDocToConnection(doc)
}

func (s *MongoStore) GetConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*Connection, error) {
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

func (s *MongoStore) ListConnections(ctx context.Context, appID id.AppID) ([]*Connection, error) {
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

	result := make([]*Connection, 0, len(docs))
	for i := range docs {
		c, err := ssoDocToConnection(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *MongoStore) UpdateConnection(ctx context.Context, c *Connection) error {
	c.UpdatedAt = time.Now()
	doc := ssoConnectionToDoc(c)
	_, err := s.mdb.Collection(ssoConnectionsColl).UpdateOne(ctx,
		bson.M{"_id": c.ID.String()},
		bson.M{"$set": bson.M{
			"app_id":             doc.AppID,
			"org_id":             doc.OrgID,
			"provider":           doc.Provider,
			"protocol":           doc.Protocol,
			"domain":             doc.Domain,
			"display_name":       doc.DisplayName,
			"metadata_url":       doc.MetadataURL,
			"client_id":          doc.ClientID,
			"client_secret":      doc.ClientSecret,
			"issuer":             doc.Issuer,
			"idp_metadata_xml":   doc.IDPMetadataXML,
			"idp_sso_url":        doc.IDPSSOURL,
			"idp_certificate":    doc.IDPCertificate,
			"entity_id":          doc.EntityID,
			"acs_url":            doc.ACSURL,
			"sp_certificate":     doc.SPCertificate,
			"sp_private_key":     doc.SPPrivateKey,
			"sign_requests":      doc.SignRequests,
			"attribute_mappings": doc.AttributeMappings,
			"active":             doc.Active,
			"enforced":           doc.Enforced,
			"allowed_domains":    doc.AllowedDomains,
			"trusted_federation": doc.TrustedFederation,
			"updated_at":         doc.UpdatedAt,
		}},
	)
	return ssoMongoError(err)
}

func (s *MongoStore) DeleteConnection(ctx context.Context, connID id.SSOConnectionID) error {
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

func (s *MongoStore) GetIdentity(ctx context.Context, connID id.SSOConnectionID, subject string) (*Identity, error) {
	if subject == "" {
		return nil, ErrIdentityNotFound
	}
	doc := new(ssoIdentityDoc)
	err := s.mdb.Collection(ssoIdentitiesColl).FindOne(ctx, bson.M{"_id": identityRowID(connID, subject)}).Decode(doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrIdentityNotFound
		}
		return nil, ssoMongoError(err)
	}
	userID, err := id.ParseUserID(doc.UserID)
	if err != nil {
		return nil, err
	}
	return &Identity{ConnectionID: connID, Subject: doc.Subject, UserID: userID, CreatedAt: doc.CreatedAt}, nil
}

func (s *MongoStore) CreateIdentity(ctx context.Context, ident *Identity) error {
	if ident == nil || ident.Subject == "" {
		return errors.New("sso: identity needs a subject")
	}
	// Round(0) drops the monotonic reading, which the sqlite driver would
	// otherwise write into the column.
	if ident.CreatedAt.IsZero() {
		ident.CreatedAt = time.Now()
	}
	ident.CreatedAt = ident.CreatedAt.UTC().Round(0)
	_, err := s.mdb.Collection(ssoIdentitiesColl).InsertOne(ctx, &ssoIdentityDoc{
		ID:           identityRowID(ident.ConnectionID, ident.Subject),
		ConnectionID: ident.ConnectionID.String(),
		Subject:      ident.Subject,
		UserID:       ident.UserID.String(),
		CreatedAt:    ident.CreatedAt,
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrIdentityExists
		}
		return ssoMongoError(err)
	}
	return nil
}
