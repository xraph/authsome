package scim

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"

	"golang.org/x/crypto/bcrypt"
)

const (
	scimConfigsColl = "authsome_scim_configs"
	scimTokensColl  = "authsome_scim_tokens" // #nosec G101 -- a collection name
	scimLogsColl    = "authsome_scim_provision_logs"
)

type scimConfigDoc struct {
	ID          string            `bson:"_id"`
	AppID       string            `bson:"app_id"`
	OrgID       string            `bson:"org_id,omitempty"`
	Name        string            `bson:"name"`
	Enabled     bool              `bson:"enabled"`
	AutoCreate  bool              `bson:"auto_create"`
	AutoSuspend bool              `bson:"auto_suspend"`
	GroupSync   bool              `bson:"group_sync"`
	DefaultRole string            `bson:"default_role"`
	Metadata    map[string]string `bson:"metadata,omitempty"`
	CreatedAt   time.Time         `bson:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at"`
}

type scimTokenDoc struct {
	ID          string     `bson:"_id"`
	ConfigID    string     `bson:"config_id"`
	Name        string     `bson:"name"`
	TokenHash   string     `bson:"token_hash"`
	TokenLookup string     `bson:"token_lookup,omitempty"`
	LastUsedAt  *time.Time `bson:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `bson:"expires_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
}

type scimLogDoc struct {
	ID           string    `bson:"_id"`
	ConfigID     string    `bson:"config_id"`
	Action       string    `bson:"action"`
	ResourceType string    `bson:"resource_type"`
	ExternalID   string    `bson:"external_id,omitempty"`
	InternalID   string    `bson:"internal_id,omitempty"`
	Status       string    `bson:"status"`
	Detail       string    `bson:"detail,omitempty"`
	CreatedAt    time.Time `bson:"created_at"`
}

// MongoStore implements Store on MongoDB. The plugin has no mongo migration
// group, so the store installs its own indexes before its first write.
type MongoStore struct {
	db      *grove.DB
	mdb     *mongodriver.MongoDB
	indexMu sync.Mutex
	indexed bool
}

// NewMongoStore creates a SCIM store on the mongo driver.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{db: db, mdb: mongodriver.Unwrap(db)}
}

var _ Store = (*MongoStore)(nil)

func (s *MongoStore) ensureIndexes(ctx context.Context) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	if s.indexed {
		return nil
	}
	if _, err := s.mdb.Collection(scimConfigsColl).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "org_id", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("scim/mongo: config indexes: %w", err)
	}
	if _, err := s.mdb.Collection(scimTokensColl).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "config_id", Value: 1}}},
		{
			Keys: bson.D{{Key: "token_lookup", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"token_lookup": bson.M{"$gt": ""}}),
		},
	}); err != nil {
		return fmt.Errorf("scim/mongo: token indexes: %w", err)
	}
	if _, err := s.mdb.Collection(scimLogsColl).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "config_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}); err != nil {
		return fmt.Errorf("scim/mongo: log indexes: %w", err)
	}
	s.indexed = true
	return nil
}

func isNoDocs(err error) bool { return errors.Is(err, mongo.ErrNoDocuments) }

func configToDoc(c *SCIMConfig) *scimConfigDoc {
	d := &scimConfigDoc{
		ID: c.ID.String(), AppID: c.AppID.String(), Name: c.Name, Enabled: c.Enabled, AutoCreate: c.AutoCreate,
		AutoSuspend: c.AutoSuspend, GroupSync: c.GroupSync, DefaultRole: c.DefaultRole, Metadata: c.Metadata,
		CreatedAt: c.CreatedAt.UTC().Round(0), UpdatedAt: c.UpdatedAt.UTC().Round(0),
	}
	if !c.OrgID.IsNil() {
		d.OrgID = c.OrgID.String()
	}
	return d
}

func docToConfig(d *scimConfigDoc) (*SCIMConfig, error) {
	cfgID, err := id.ParseSCIMConfigID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}
	c := &SCIMConfig{
		ID: cfgID, AppID: appID, Name: d.Name, Enabled: d.Enabled, AutoCreate: d.AutoCreate,
		AutoSuspend: d.AutoSuspend, GroupSync: d.GroupSync, DefaultRole: d.DefaultRole, Metadata: d.Metadata,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	if d.OrgID != "" {
		if orgID, perr := id.ParseOrgID(d.OrgID); perr == nil {
			c.OrgID = orgID
		}
	}
	return c, nil
}

func tokenToDoc(t *Token) *scimTokenDoc {
	d := &scimTokenDoc{
		ID: t.ID.String(), ConfigID: t.ConfigID.String(), Name: t.Name, TokenHash: t.TokenHash,
		TokenLookup: t.TokenLookup, CreatedAt: t.CreatedAt.UTC().Round(0),
	}
	if t.LastUsedAt != nil {
		v := t.LastUsedAt.UTC().Round(0)
		d.LastUsedAt = &v
	}
	if t.ExpiresAt != nil {
		v := t.ExpiresAt.UTC().Round(0)
		d.ExpiresAt = &v
	}
	return d
}

func docToToken(d *scimTokenDoc) (*Token, error) {
	tokID, err := id.ParseSCIMTokenID(d.ID)
	if err != nil {
		return nil, err
	}
	cfgID, err := id.ParseSCIMConfigID(d.ConfigID)
	if err != nil {
		return nil, err
	}
	return &Token{
		ID: tokID, ConfigID: cfgID, Name: d.Name, TokenHash: d.TokenHash, TokenLookup: d.TokenLookup,
		LastUsedAt: d.LastUsedAt, ExpiresAt: d.ExpiresAt, CreatedAt: d.CreatedAt,
	}, nil
}

func logToDoc(l *ProvisionLog) *scimLogDoc {
	return &scimLogDoc{
		ID: l.ID.String(), ConfigID: l.ConfigID.String(), Action: l.Action, ResourceType: l.ResourceType,
		ExternalID: l.ExternalID, InternalID: l.InternalID, Status: l.Status, Detail: l.Detail,
		CreatedAt: l.CreatedAt.UTC().Round(0),
	}
}

func docToLog(d *scimLogDoc) (*ProvisionLog, error) {
	logID, err := id.ParseSCIMLogID(d.ID)
	if err != nil {
		return nil, err
	}
	cfgID, err := id.ParseSCIMConfigID(d.ConfigID)
	if err != nil {
		return nil, err
	}
	return &ProvisionLog{
		ID: logID, ConfigID: cfgID, Action: d.Action, ResourceType: d.ResourceType, ExternalID: d.ExternalID,
		InternalID: d.InternalID, Status: d.Status, Detail: d.Detail, CreatedAt: d.CreatedAt,
	}, nil
}

// ── configs ──

func (s *MongoStore) CreateConfig(ctx context.Context, c *SCIMConfig) error {
	if err := s.ensureIndexes(ctx); err != nil {
		return err
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	_, err := s.mdb.Collection(scimConfigsColl).InsertOne(ctx, configToDoc(c))
	if err != nil {
		return fmt.Errorf("scim/mongo: create config: %w", err)
	}
	return nil
}

func (s *MongoStore) GetConfig(ctx context.Context, configID id.SCIMConfigID) (*SCIMConfig, error) {
	d := new(scimConfigDoc)
	if err := s.mdb.Collection(scimConfigsColl).FindOne(ctx, bson.M{"_id": configID.String()}).Decode(d); err != nil {
		if isNoDocs(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configID)
		}
		return nil, fmt.Errorf("scim/mongo: get config: %w", err)
	}
	return docToConfig(d)
}

func (s *MongoStore) UpdateConfig(ctx context.Context, c *SCIMConfig) error {
	c.UpdatedAt = time.Now()
	d := configToDoc(c)
	res, err := s.mdb.Collection(scimConfigsColl).UpdateOne(ctx, bson.M{"_id": d.ID}, bson.M{"$set": bson.M{
		"name": d.Name, "enabled": d.Enabled, "auto_create": d.AutoCreate, "auto_suspend": d.AutoSuspend,
		"group_sync": d.GroupSync, "default_role": d.DefaultRole, "metadata": d.Metadata, "updated_at": d.UpdatedAt,
	}})
	if err != nil {
		return fmt.Errorf("scim/mongo: update config: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("%w: %s", ErrConfigNotFound, c.ID)
	}
	return nil
}

func (s *MongoStore) DeleteConfig(ctx context.Context, configID id.SCIMConfigID) error {
	if _, err := s.mdb.Collection(scimConfigsColl).DeleteOne(ctx, bson.M{"_id": configID.String()}); err != nil {
		return fmt.Errorf("scim/mongo: delete config: %w", err)
	}
	// Tokens and logs cascade with the configuration, as the SQL schema does.
	_, _ = s.mdb.Collection(scimTokensColl).DeleteMany(ctx, bson.M{"config_id": configID.String()}) //nolint:errcheck // best-effort cascade
	_, _ = s.mdb.Collection(scimLogsColl).DeleteMany(ctx, bson.M{"config_id": configID.String()})   //nolint:errcheck // best-effort cascade
	return nil
}

func (s *MongoStore) listConfigs(ctx context.Context, filter bson.M) ([]*SCIMConfig, error) {
	cur, err := s.mdb.Collection(scimConfigsColl).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("scim/mongo: list configs: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]*SCIMConfig, 0)
	for cur.Next(ctx) {
		d := new(scimConfigDoc)
		if err := cur.Decode(d); err != nil {
			return nil, fmt.Errorf("scim/mongo: decode config: %w", err)
		}
		c, convErr := docToConfig(d)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, c)
	}
	return out, cur.Err()
}

func (s *MongoStore) ListConfigs(ctx context.Context, appID string) ([]*SCIMConfig, error) {
	return s.listConfigs(ctx, bson.M{"app_id": appID})
}

func (s *MongoStore) ListConfigsByOrg(ctx context.Context, orgID id.OrgID) ([]*SCIMConfig, error) {
	return s.listConfigs(ctx, bson.M{"org_id": orgID.String()})
}

// ── tokens ──

func (s *MongoStore) CreateToken(ctx context.Context, t *Token) error {
	if err := s.ensureIndexes(ctx); err != nil {
		return err
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	if _, err := s.mdb.Collection(scimTokensColl).InsertOne(ctx, tokenToDoc(t)); err != nil {
		return fmt.Errorf("scim/mongo: create token: %w", err)
	}
	return nil
}

func (s *MongoStore) GetToken(ctx context.Context, tokenID id.SCIMTokenID) (*Token, error) {
	d := new(scimTokenDoc)
	if err := s.mdb.Collection(scimTokensColl).FindOne(ctx, bson.M{"_id": tokenID.String()}).Decode(d); err != nil {
		if isNoDocs(err) {
			return nil, fmt.Errorf("%w: %s", ErrTokenNotFound, tokenID)
		}
		return nil, fmt.Errorf("scim/mongo: get token: %w", err)
	}
	return docToToken(d)
}

func (s *MongoStore) UpdateToken(ctx context.Context, t *Token) error {
	d := tokenToDoc(t)
	set := bson.M{"name": d.Name, "last_used_at": d.LastUsedAt, "expires_at": d.ExpiresAt}
	if d.TokenLookup != "" {
		set["token_lookup"] = d.TokenLookup
	}
	res, err := s.mdb.Collection(scimTokensColl).UpdateOne(ctx, bson.M{"_id": d.ID}, bson.M{"$set": set})
	if err != nil {
		return fmt.Errorf("scim/mongo: update token: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("%w: %s", ErrTokenNotFound, t.ID)
	}
	return nil
}

func (s *MongoStore) ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*Token, error) {
	cur, err := s.mdb.Collection(scimTokensColl).Find(ctx, bson.M{"config_id": configID.String()}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("scim/mongo: list tokens: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]*Token, 0)
	for cur.Next(ctx) {
		d := new(scimTokenDoc)
		if err := cur.Decode(d); err != nil {
			return nil, fmt.Errorf("scim/mongo: decode token: %w", err)
		}
		t, convErr := docToToken(d)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, t)
	}
	return out, cur.Err()
}

func (s *MongoStore) DeleteToken(ctx context.Context, tokenID id.SCIMTokenID) error {
	if _, err := s.mdb.Collection(scimTokensColl).DeleteOne(ctx, bson.M{"_id": tokenID.String()}); err != nil {
		return fmt.Errorf("scim/mongo: delete token: %w", err)
	}
	return nil
}

// FindTokenByPlaintext resolves by the lookup digest and bcrypt-compares
// that document; documents without a lookup are compared one by one and
// upgraded on a match.
func (s *MongoStore) FindTokenByPlaintext(ctx context.Context, plaintext string) (*Token, *SCIMConfig, error) {
	lookup := store.HashToken(plaintext)
	d := new(scimTokenDoc)
	err := s.mdb.Collection(scimTokensColl).FindOne(ctx, bson.M{"token_lookup": lookup}).Decode(d)
	switch {
	case err == nil:
		if bcrypt.CompareHashAndPassword([]byte(d.TokenHash), []byte(plaintext)) != nil {
			return nil, nil, ErrTokenNotFound
		}
	case isNoDocs(err):
		cur, ferr := s.mdb.Collection(scimTokensColl).Find(ctx, bson.M{"token_lookup": bson.M{"$in": bson.A{nil, ""}}})
		if ferr != nil {
			return nil, nil, fmt.Errorf("scim/mongo: scan legacy tokens: %w", ferr)
		}
		defer cur.Close(ctx)
		d = nil
		for cur.Next(ctx) {
			cand := new(scimTokenDoc)
			if derr := cur.Decode(cand); derr != nil {
				continue
			}
			if bcrypt.CompareHashAndPassword([]byte(cand.TokenHash), []byte(plaintext)) == nil {
				d = cand
				break
			}
		}
		if d == nil {
			return nil, nil, ErrTokenNotFound
		}
		_, _ = s.mdb.Collection(scimTokensColl).UpdateOne(ctx, //nolint:errcheck // best-effort upgrade
			bson.M{"_id": d.ID, "token_lookup": bson.M{"$in": bson.A{nil, ""}}},
			bson.M{"$set": bson.M{"token_lookup": lookup}})
		d.TokenLookup = lookup
	default:
		return nil, nil, fmt.Errorf("scim/mongo: find token: %w", err)
	}
	t, err := docToToken(d)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := s.GetConfig(ctx, t.ConfigID)
	if err != nil {
		return nil, nil, err
	}
	return t, cfg, nil
}

// ── logs ──

func (s *MongoStore) CreateLog(ctx context.Context, l *ProvisionLog) error {
	if err := s.ensureIndexes(ctx); err != nil {
		return err
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	if _, err := s.mdb.Collection(scimLogsColl).InsertOne(ctx, logToDoc(l)); err != nil {
		return fmt.Errorf("scim/mongo: create log: %w", err)
	}
	return nil
}

func (s *MongoStore) appConfigIDs(ctx context.Context, appID string) ([]string, error) {
	configs, err := s.ListConfigs(ctx, appID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(configs))
	for _, c := range configs {
		ids = append(ids, c.ID.String())
	}
	return ids, nil
}

func (s *MongoStore) listLogs(ctx context.Context, filter bson.M, limit int) ([]*ProvisionLog, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts = opts.SetLimit(int64(limit))
	}
	cur, err := s.mdb.Collection(scimLogsColl).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("scim/mongo: list logs: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]*ProvisionLog, 0)
	for cur.Next(ctx) {
		d := new(scimLogDoc)
		if err := cur.Decode(d); err != nil {
			return nil, fmt.Errorf("scim/mongo: decode log: %w", err)
		}
		l, convErr := docToLog(d)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, l)
	}
	return out, cur.Err()
}

func (s *MongoStore) ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*ProvisionLog, error) {
	return s.listLogs(ctx, bson.M{"config_id": configID.String()}, limit)
}

func (s *MongoStore) ListAllLogs(ctx context.Context, appID string, limit int) ([]*ProvisionLog, error) {
	ids, err := s.appConfigIDs(ctx, appID)
	if err != nil {
		return nil, err
	}
	return s.listLogs(ctx, bson.M{"config_id": bson.M{"$in": ids}}, limit)
}

func countStatuses(logs []*ProvisionLog) (success, errs, skipped int) {
	for _, l := range logs {
		switch l.Status {
		case LogStatusSuccess:
			success++
		case LogStatusError:
			errs++
		case LogStatusSkipped:
			skipped++
		}
	}
	return success, errs, skipped
}

func (s *MongoStore) CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, errs, skipped int, err error) {
	logs, err := s.ListLogs(ctx, configID, 0)
	if err != nil {
		return 0, 0, 0, err
	}
	success, errs, skipped = countStatuses(logs)
	return success, errs, skipped, nil
}

func (s *MongoStore) CountAllLogsByStatus(ctx context.Context, appID string) (success, errs, skipped int, err error) {
	logs, err := s.ListAllLogs(ctx, appID, 0)
	if err != nil {
		return 0, 0, 0, err
	}
	success, errs, skipped = countStatuses(logs)
	return success, errs, skipped, nil
}
