package oauth2provider

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

// MongoStore implements oauth2provider.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed OAuth2 store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{
		db:  db,
		mdb: mongodriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Mongo document models
// ──────────────────────────────────────────────────

type oauth2ClientDoc struct {
	ID           string   `bson:"_id"`
	AppID        string   `bson:"app_id"`
	Name         string   `bson:"name"`
	ClientID     string   `bson:"client_id"`
	ClientSecret string   `bson:"client_secret"`
	RedirectURIs []string `bson:"redirect_uris"`
	Scopes       []string `bson:"scopes"`
	GrantTypes   []string `bson:"grant_types"`
	Public       bool     `bson:"public"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
}

type authCodeDoc struct {
	ID                  string    `bson:"_id"`
	Code                string    `bson:"code"`
	ClientID            string    `bson:"client_id"`
	UserID              string    `bson:"user_id"`
	AppID               string    `bson:"app_id"`
	RedirectURI         string    `bson:"redirect_uri"`
	Scopes              []string  `bson:"scopes"`
	CodeChallenge       string    `bson:"code_challenge"`
	CodeChallengeMethod string    `bson:"code_challenge_method"`
	ExpiresAt           time.Time `bson:"expires_at"`
	Consumed            bool      `bson:"consumed"`
	CreatedAt           time.Time `bson:"created_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func oauth2ClientDocToModel(d *oauth2ClientDoc) (*OAuth2Client, error) {
	clientID, err := id.ParseOAuth2ClientID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	redirectURIs := d.RedirectURIs
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	scopes := d.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	grantTypes := d.GrantTypes
	if grantTypes == nil {
		grantTypes = []string{}
	}

	return &OAuth2Client{
		ID:           clientID,
		AppID:        appID,
		Name:         d.Name,
		ClientID:     d.ClientID,
		ClientSecret: d.ClientSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       d.Public,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}, nil
}

func oauth2ClientToDoc(c *OAuth2Client) *oauth2ClientDoc {
	redirectURIs := c.RedirectURIs
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	scopes := c.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	grantTypes := c.GrantTypes
	if grantTypes == nil {
		grantTypes = []string{}
	}

	return &oauth2ClientDoc{
		ID:           c.ID.String(),
		AppID:        c.AppID.String(),
		Name:         c.Name,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       c.Public,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func authCodeDocToModel(d *authCodeDoc) (*AuthorizationCode, error) {
	codeID, err := id.ParseAuthCodeID(d.ID)
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

	scopes := d.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return &AuthorizationCode{
		ID:                  codeID,
		Code:                d.Code,
		ClientID:            d.ClientID,
		UserID:              userID,
		AppID:               appID,
		RedirectURI:         d.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       d.CodeChallenge,
		CodeChallengeMethod: d.CodeChallengeMethod,
		ExpiresAt:           d.ExpiresAt,
		Consumed:            d.Consumed,
		CreatedAt:           d.CreatedAt,
	}, nil
}

func authCodeToDoc(c *AuthorizationCode) *authCodeDoc {
	scopes := c.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return &authCodeDoc{
		ID:                  c.ID.String(),
		Code:                c.Code,
		ClientID:            c.ClientID,
		UserID:              c.UserID.String(),
		AppID:               c.AppID.String(),
		RedirectURI:         c.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
		ExpiresAt:           c.ExpiresAt,
		Consumed:            c.Consumed,
		CreatedAt:           c.CreatedAt,
	}
}

type deviceCodeDoc struct {
	ID              string    `bson:"_id"`
	DeviceCode      string    `bson:"device_code"`
	UserCode        string    `bson:"user_code"`
	ClientID        string    `bson:"client_id"`
	AppID           string    `bson:"app_id"`
	Scopes          []string  `bson:"scopes"`
	VerificationURI string    `bson:"verification_uri"`
	ExpiresAt       time.Time `bson:"expires_at"`
	Interval        int       `bson:"interval"`
	Status          string    `bson:"status"`
	UserID          string    `bson:"user_id"`
	LastPolledAt    time.Time `bson:"last_polled_at,omitempty"`
	CreatedAt       time.Time `bson:"created_at"`
}

// ──────────────────────────────────────────────────
// Device code converters
// ──────────────────────────────────────────────────

func deviceCodeDocToModel(d *deviceCodeDoc) (*DeviceCode, error) {
	dcID, err := id.ParseDeviceCodeID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}

	var userID id.UserID
	if d.UserID != "" {
		userID, err = id.ParseUserID(d.UserID)
		if err != nil {
			return nil, err
		}
	}

	scopes := d.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return &DeviceCode{
		ID:              dcID,
		DeviceCode:      d.DeviceCode,
		UserCode:        d.UserCode,
		ClientID:        d.ClientID,
		AppID:           appID,
		Scopes:          scopes,
		VerificationURI: d.VerificationURI,
		ExpiresAt:       d.ExpiresAt,
		Interval:        d.Interval,
		Status:          d.Status,
		UserID:          userID,
		LastPolledAt:    d.LastPolledAt,
		CreatedAt:       d.CreatedAt,
	}, nil
}

func deviceCodeToDoc(dc *DeviceCode) *deviceCodeDoc {
	scopes := dc.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return &deviceCodeDoc{
		ID:              dc.ID.String(),
		DeviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		ClientID:        dc.ClientID,
		AppID:           dc.AppID.String(),
		Scopes:          scopes,
		VerificationURI: dc.VerificationURI,
		ExpiresAt:       dc.ExpiresAt,
		Interval:        dc.Interval,
		Status:          dc.Status,
		UserID:          dc.UserID.String(),
		LastPolledAt:    dc.LastPolledAt,
		CreatedAt:       dc.CreatedAt,
	}
}

// ──────────────────────────────────────────────────
// Collection names
// ──────────────────────────────────────────────────

const (
	oauth2ClientsColl     = "authsome_oauth2_clients"
	oauth2AuthCodesColl   = "authsome_oauth2_auth_codes"
	oauth2DeviceCodesColl = "authsome_oauth2_device_codes"
)

// ──────────────────────────────────────────────────
// Client methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateClient(ctx context.Context, c *OAuth2Client) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	doc := oauth2ClientToDoc(c)
	_, err := s.mdb.Collection(oauth2ClientsColl).InsertOne(ctx, doc)
	return oauth2MongoError(err)
}

func (s *MongoStore) GetClient(ctx context.Context, clientID string) (*OAuth2Client, error) {
	doc := new(oauth2ClientDoc)
	err := s.mdb.Collection(oauth2ClientsColl).FindOne(ctx, bson.M{
		"client_id": clientID,
	}).Decode(doc)
	if err != nil {
		return nil, oauth2MongoError(err)
	}
	return oauth2ClientDocToModel(doc)
}

func (s *MongoStore) GetClientByID(ctx context.Context, clientID id.OAuth2ClientID) (*OAuth2Client, error) {
	doc := new(oauth2ClientDoc)
	err := s.mdb.Collection(oauth2ClientsColl).FindOne(ctx, bson.M{
		"_id": clientID.String(),
	}).Decode(doc)
	if err != nil {
		return nil, oauth2MongoError(err)
	}
	return oauth2ClientDocToModel(doc)
}

func (s *MongoStore) ListClients(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	cursor, err := s.mdb.Collection(oauth2ClientsColl).Find(ctx, bson.M{
		"app_id": appID.String(),
	})
	if err != nil {
		return nil, oauth2MongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []oauth2ClientDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, oauth2MongoError(err)
	}

	result := make([]*OAuth2Client, 0, len(docs))
	for i := range docs {
		c, err := oauth2ClientDocToModel(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *MongoStore) DeleteClient(ctx context.Context, clientID id.OAuth2ClientID) error {
	_, err := s.mdb.Collection(oauth2ClientsColl).DeleteOne(ctx, bson.M{
		"_id": clientID.String(),
	})
	return oauth2MongoError(err)
}

// ──────────────────────────────────────────────────
// Auth code methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateAuthCode(ctx context.Context, code *AuthorizationCode) error {
	now := time.Now()
	if code.CreatedAt.IsZero() {
		code.CreatedAt = now
	}
	doc := authCodeToDoc(code)
	_, err := s.mdb.Collection(oauth2AuthCodesColl).InsertOne(ctx, doc)
	return oauth2MongoError(err)
}

func (s *MongoStore) GetAuthCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	doc := new(authCodeDoc)
	err := s.mdb.Collection(oauth2AuthCodesColl).FindOne(ctx, bson.M{
		"code": code,
	}).Decode(doc)
	if err != nil {
		return nil, oauth2MongoError(err)
	}
	return authCodeDocToModel(doc)
}

func (s *MongoStore) ConsumeAuthCode(ctx context.Context, code string) error {
	_, err := s.mdb.Collection(oauth2AuthCodesColl).UpdateOne(ctx,
		bson.M{"code": code},
		bson.M{"$set": bson.M{"consumed": true}},
	)
	return oauth2MongoError(err)
}

// ──────────────────────────────────────────────────
// Device code methods (RFC 8628)
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	if dc.CreatedAt.IsZero() {
		dc.CreatedAt = time.Now()
	}
	doc := deviceCodeToDoc(dc)
	_, err := s.mdb.Collection(oauth2DeviceCodesColl).InsertOne(ctx, doc)
	return oauth2MongoError(err)
}

func (s *MongoStore) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCode, error) {
	doc := new(deviceCodeDoc)
	err := s.mdb.Collection(oauth2DeviceCodesColl).FindOne(ctx, bson.M{
		"device_code": deviceCode,
	}).Decode(doc)
	if err != nil {
		if oauth2IsNoDocuments(err) {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2MongoError(err)
	}
	return deviceCodeDocToModel(doc)
}

func (s *MongoStore) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCode, error) {
	doc := new(deviceCodeDoc)
	err := s.mdb.Collection(oauth2DeviceCodesColl).FindOne(ctx, bson.M{
		"user_code": userCode,
	}).Decode(doc)
	if err != nil {
		if oauth2IsNoDocuments(err) {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2MongoError(err)
	}
	return deviceCodeDocToModel(doc)
}

func (s *MongoStore) UpdateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	_, err := s.mdb.Collection(oauth2DeviceCodesColl).UpdateOne(ctx,
		bson.M{"_id": dc.ID.String()},
		bson.M{"$set": bson.M{
			"status":         dc.Status,
			"user_id":        dc.UserID.String(),
			"last_polled_at": dc.LastPolledAt,
		}},
	)
	return oauth2MongoError(err)
}

func (s *MongoStore) DeleteExpiredDeviceCodes(ctx context.Context) error {
	_, err := s.mdb.Collection(oauth2DeviceCodesColl).DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})
	return oauth2MongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func oauth2IsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func oauth2MongoError(err error) error {
	if err == nil {
		return nil
	}
	if oauth2IsNoDocuments(err) {
		return ErrClientNotFound
	}
	return err
}
