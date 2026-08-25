package agentauth

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

// MongoStore implements agentauth.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed agentauth store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{
		db:  db,
		mdb: mongodriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Collection names
// ──────────────────────────────────────────────────

const (
	agentsColl        = "authsome_agents"
	agentGrantsColl   = "authsome_agent_grants"
	agentPoliciesColl = "authsome_agent_policies"
)

// ──────────────────────────────────────────────────
// Mongo document models
// ──────────────────────────────────────────────────

type agentDoc struct {
	ID    string `bson:"_id"`
	AppID string `bson:"app_id"`
	// OrgID intentionally has no omitempty: it is queried by equality
	// (ListAgents), and MongoDB's {org_id: ""} filter does not match a
	// document where the field is entirely absent, only one where it is
	// stored as "". Always writing it keeps zero-org agents findable.
	OrgID       string    `bson:"org_id"`
	ClientID    string    `bson:"client_id"`
	Name        string    `bson:"name"`
	Description string    `bson:"description,omitempty"`
	LogoURI     string    `bson:"logo_uri,omitempty"`
	Origin      string    `bson:"origin"`
	Status      string    `bson:"status"`
	CreatedBy   string    `bson:"created_by,omitempty"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

type agentGrantDoc struct {
	ID      string `bson:"_id"`
	AppID   string `bson:"app_id"`
	AgentID string `bson:"agent_id"`
	UserID  string `bson:"user_id"`
	// OrgID has no omitempty for the same reason as agentDoc.OrgID: it is
	// queried by equality in GetActiveGrant, RevokeGrantsByUserOrg and
	// RevokeGrantsByOrg, including the zero-org case.
	OrgID     string    `bson:"org_id"`
	Scopes    []string  `bson:"scopes,omitempty"`
	ConsentID string    `bson:"consent_id,omitempty"`
	ExpiresAt time.Time `bson:"expires_at"`
	// LastUsedAt and RevokedAt omit on nil, which mongo matches with an
	// equality-nil filter the same way it matches an explicit null — see
	// RevokeAgentGrant / GetActiveGrant.
	LastUsedAt *time.Time `bson:"last_used_at,omitempty"`
	RevokedAt  *time.Time `bson:"revoked_at,omitempty"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
}

type orgPolicyDoc struct {
	ID            string   `bson:"_id"` // org id
	Mode          string   `bson:"mode"`
	MaxGrantTTLNs int64    `bson:"max_grant_ttl_ns"`
	AllowedScopes []string `bson:"allowed_scopes,omitempty"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func agentDocToModel(d *agentDoc) (*Agent, error) {
	agentID, err := id.ParseAgentID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}
	a := &Agent{
		ID:          agentID,
		AppID:       appID,
		ClientID:    d.ClientID,
		Name:        d.Name,
		Description: d.Description,
		LogoURI:     d.LogoURI,
		Origin:      AgentOrigin(d.Origin),
		Status:      AgentStatus(d.Status),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if d.OrgID != "" {
		orgID, err := id.ParseOrgID(d.OrgID)
		if err != nil {
			return nil, err
		}
		a.OrgID = orgID
	}
	if d.CreatedBy != "" {
		createdBy, err := id.ParseUserID(d.CreatedBy)
		if err != nil {
			return nil, err
		}
		a.CreatedBy = createdBy
	}
	return a, nil
}

func agentToDoc(a *Agent) *agentDoc {
	d := &agentDoc{
		ID:          a.ID.String(),
		AppID:       a.AppID.String(),
		OrgID:       a.OrgID.String(),
		ClientID:    a.ClientID,
		Name:        a.Name,
		Description: a.Description,
		LogoURI:     a.LogoURI,
		Origin:      string(a.Origin),
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
	if !a.CreatedBy.IsNil() {
		d.CreatedBy = a.CreatedBy.String()
	}
	return d
}

func agentGrantDocToModel(d *agentGrantDoc) (*AgentGrant, error) {
	grantID, err := id.ParseAgentGrantID(d.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(d.AppID)
	if err != nil {
		return nil, err
	}
	agentID, err := id.ParseAgentID(d.AgentID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	g := &AgentGrant{
		ID:         grantID,
		AppID:      appID,
		AgentID:    agentID,
		UserID:     userID,
		Scopes:     d.Scopes,
		ExpiresAt:  d.ExpiresAt,
		LastUsedAt: d.LastUsedAt,
		RevokedAt:  d.RevokedAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
	if d.OrgID != "" {
		orgID, err := id.ParseOrgID(d.OrgID)
		if err != nil {
			return nil, err
		}
		g.OrgID = orgID
	}
	if d.ConsentID != "" {
		consentID, err := id.ParseConsentID(d.ConsentID)
		if err != nil {
			return nil, err
		}
		g.ConsentID = consentID
	}
	return g, nil
}

func agentGrantToDoc(g *AgentGrant) *agentGrantDoc {
	d := &agentGrantDoc{
		ID:        g.ID.String(),
		AppID:     g.AppID.String(),
		AgentID:   g.AgentID.String(),
		UserID:    g.UserID.String(),
		OrgID:     g.OrgID.String(),
		Scopes:    append([]string(nil), g.Scopes...),
		ExpiresAt: g.ExpiresAt,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
	if !g.ConsentID.IsNil() {
		d.ConsentID = g.ConsentID.String()
	}
	if g.LastUsedAt != nil {
		t := *g.LastUsedAt
		d.LastUsedAt = &t
	}
	if g.RevokedAt != nil {
		t := *g.RevokedAt
		d.RevokedAt = &t
	}
	return d
}

func orgPolicyDocToModel(d *orgPolicyDoc) (*OrgAgentPolicy, error) {
	p := &OrgAgentPolicy{
		Mode:          PolicyMode(d.Mode),
		MaxGrantTTL:   time.Duration(d.MaxGrantTTLNs),
		AllowedScopes: d.AllowedScopes,
	}
	// Guarded like agentDocToModel/agentGrantDocToModel's OrgID parses: the
	// zero-org policy's _id is "", a real and meaningful key (the
	// single-tenant / app-scoped case), not an absent value.
	// id.ParseOrgID("") errors, which made every read of a zero-org policy
	// 500 until this was guarded.
	if d.ID != "" {
		orgID, err := id.ParseOrgID(d.ID)
		if err != nil {
			return nil, err
		}
		p.OrgID = orgID
	}
	return p, nil
}

func orgPolicyToDoc(p *OrgAgentPolicy) *orgPolicyDoc {
	return &orgPolicyDoc{
		ID:            p.OrgID.String(),
		Mode:          string(p.Mode),
		MaxGrantTTLNs: int64(p.MaxGrantTTL),
		AllowedScopes: append([]string(nil), p.AllowedScopes...),
	}
}

// ──────────────────────────────────────────────────
// Agent methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateAgent(ctx context.Context, a *Agent) error {
	doc := agentToDoc(a)
	_, err := s.mdb.Collection(agentsColl).InsertOne(ctx, doc)
	return agentauthMongoError(err)
}

func (s *MongoStore) GetAgent(ctx context.Context, agentID id.AgentID) (*Agent, error) {
	doc := new(agentDoc)
	err := s.mdb.Collection(agentsColl).FindOne(ctx, bson.M{"_id": agentID.String()}).Decode(doc)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	return agentDocToModel(doc)
}

func (s *MongoStore) GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error) {
	doc := new(agentDoc)
	err := s.mdb.Collection(agentsColl).FindOne(ctx, bson.M{"client_id": clientID}).Decode(doc)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	return agentDocToModel(doc)
}

func (s *MongoStore) UpdateAgent(ctx context.Context, a *Agent) error {
	doc := agentToDoc(a)
	res, err := s.mdb.Collection(agentsColl).ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
	if err != nil {
		return agentauthMongoError(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) ListAgents(ctx context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error) {
	filter := bson.M{"app_id": appID.String()}
	if !orgID.IsNil() {
		filter["org_id"] = orgID.String()
	}
	cursor, err := s.mdb.Collection(agentsColl).Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []agentDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, agentauthMongoError(err)
	}
	out := make([]*Agent, 0, len(docs))
	for i := range docs {
		a, err := agentDocToModel(&docs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// ──────────────────────────────────────────────────
// Agent grant methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateAgentGrant(ctx context.Context, g *AgentGrant) error {
	doc := agentGrantToDoc(g)
	_, err := s.mdb.Collection(agentGrantsColl).InsertOne(ctx, doc)
	return agentauthMongoError(err)
}

func (s *MongoStore) GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	doc := new(agentGrantDoc)
	err := s.mdb.Collection(agentGrantsColl).FindOne(ctx, bson.M{"_id": grantID.String()}).Decode(doc)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	return agentGrantDocToModel(doc)
}

// GetActiveGrant returns the most recently created matching grant when more
// than one exists. See the identical comment on PostgresStore.GetActiveGrant
// for why duplicates are the normal state and why an explicit order is
// required for the four backends to agree on which one "the" active grant
// is.
func (s *MongoStore) GetActiveGrant(ctx context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error) {
	doc := new(agentGrantDoc)
	err := s.mdb.Collection(agentGrantsColl).FindOne(ctx, bson.M{
		"agent_id":   agentID.String(),
		"user_id":    userID.String(),
		"org_id":     orgID.String(),
		"revoked_at": nil,
		"expires_at": bson.M{"$gt": time.Now()},
	}, options.FindOne().SetSort(bson.M{"created_at": -1})).Decode(doc)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	return agentGrantDocToModel(doc)
}

func (s *MongoStore) ListGrantsByUser(ctx context.Context, userID id.UserID) ([]*AgentGrant, error) {
	cursor, err := s.mdb.Collection(agentGrantsColl).Find(ctx,
		bson.M{"user_id": userID.String()},
		options.Find().SetSort(bson.M{"created_at": 1}),
	)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []agentGrantDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, agentauthMongoError(err)
	}
	out := make([]*AgentGrant, 0, len(docs))
	for i := range docs {
		g, err := agentGrantDocToModel(&docs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (s *MongoStore) UpdateAgentGrant(ctx context.Context, g *AgentGrant) error {
	doc := agentGrantToDoc(g)
	res, err := s.mdb.Collection(agentGrantsColl).ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
	if err != nil {
		return agentauthMongoError(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) RevokeAgentGrant(ctx context.Context, grantID id.AgentGrantID) error {
	doc := new(agentGrantDoc)
	err := s.mdb.Collection(agentGrantsColl).FindOne(ctx, bson.M{"_id": grantID.String()}).Decode(doc)
	if err != nil {
		return agentauthMongoError(err)
	}
	if doc.RevokedAt != nil {
		// Already revoked: a no-op, not an error — matches MemoryStore.
		return nil
	}
	now := time.Now().UTC()
	_, err = s.mdb.Collection(agentGrantsColl).UpdateOne(ctx,
		bson.M{"_id": grantID.String(), "revoked_at": nil},
		bson.M{"$set": bson.M{"revoked_at": now, "updated_at": now}},
	)
	return agentauthMongoError(err)
}

func (s *MongoStore) RevokeGrantsByUser(ctx context.Context, userID id.UserID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, bson.M{"user_id": userID.String()})
}

func (s *MongoStore) RevokeGrantsByUserOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, bson.M{"user_id": userID.String(), "org_id": orgID.String()})
}

func (s *MongoStore) RevokeGrantsByOrg(ctx context.Context, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, bson.M{"org_id": orgID.String()})
}

// RevokeGrantsByAgent revokes the agent's grants. A nil orgID means "every
// org", not "the org whose id is empty" — the org_id key is omitted from
// the filter entirely rather than matched against "", matching MemoryStore.
func (s *MongoStore) RevokeGrantsByAgent(ctx context.Context, agentID id.AgentID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	filter := bson.M{"agent_id": agentID.String()}
	if !orgID.IsNil() {
		filter["org_id"] = orgID.String()
	}
	return s.revokeGrantsWhere(ctx, filter)
}

// revokeGrantsWhere returns the ids of every grant matching filter — whether
// or not it was already revoked, per the Store interface contract documented
// on Store.RevokeGrantsByUser — and revokes any of them that weren't
// revoked already.
// revokeGrantsWhere returns the ids of every grant matching filter — whether
// or not it was already revoked, per the Store interface contract
// documented on Store.RevokeGrantsByUser — and revokes any of them that
// weren't revoked already.
//
// The write runs BEFORE the read, not after: a find-then-update (the
// original shape here) leaves a window where a grant matching filter,
// inserted between the two calls, gets caught and revoked by UpdateMany but
// was never in the Find's result — so its id is missing from the return
// value and sweepSessions never deletes its sessions, exactly the
// under-sweeping the interface comment warns against. Updating first closes
// that: any document this call revokes has revoked_at set before the Find
// ever runs, so filtering on "revoked_at: {$ne: nil}" afterward is
// guaranteed to include it. A document that starts existing only after
// UpdateMany has run is, by construction, not one this call touched — it's
// still unrevoked, so the $ne filter correctly excludes it.
func (s *MongoStore) revokeGrantsWhere(ctx context.Context, filter bson.M) ([]id.AgentGrantID, error) {
	coll := s.mdb.Collection(agentGrantsColl)
	now := time.Now().UTC()

	updateFilter := bson.M{}
	for k, v := range filter {
		updateFilter[k] = v
	}
	updateFilter["revoked_at"] = nil
	if _, err := coll.UpdateMany(ctx, updateFilter,
		bson.M{"$set": bson.M{"revoked_at": now, "updated_at": now}},
	); err != nil {
		return nil, agentauthMongoError(err)
	}

	selectFilter := bson.M{}
	for k, v := range filter {
		selectFilter[k] = v
	}
	selectFilter["revoked_at"] = bson.M{"$ne": nil}
	cursor, err := coll.Find(ctx, selectFilter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, agentauthMongoError(err)
	}
	ids := make([]id.AgentGrantID, 0, len(rows))
	for _, r := range rows {
		gid, err := id.ParseAgentGrantID(r.ID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, gid)
	}
	return ids, nil
}

// ──────────────────────────────────────────────────
// Org policy methods
// ──────────────────────────────────────────────────

func (s *MongoStore) GetOrgPolicy(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	doc := new(orgPolicyDoc)
	err := s.mdb.Collection(agentPoliciesColl).FindOne(ctx, bson.M{"_id": orgID.String()}).Decode(doc)
	if err != nil {
		return nil, agentauthMongoError(err)
	}
	return orgPolicyDocToModel(doc)
}

func (s *MongoStore) PutOrgPolicy(ctx context.Context, p *OrgAgentPolicy) error {
	if err := validatePolicyMode(p.Mode); err != nil {
		return err
	}
	doc := orgPolicyToDoc(p)
	_, err := s.mdb.Collection(agentPoliciesColl).UpdateOne(ctx,
		bson.M{"_id": doc.ID},
		bson.M{"$set": bson.M{
			"mode":             doc.Mode,
			"max_grant_ttl_ns": doc.MaxGrantTTLNs,
			"allowed_scopes":   doc.AllowedScopes,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return agentauthMongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func agentauthMongoIsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

// agentauthMongoError maps low-level mongo failures to this package's
// sentinels. A duplicate-key error on the agents.client_id unique index
// becomes ErrConflict.
func agentauthMongoError(err error) error {
	if err == nil {
		return nil
	}
	if agentauthMongoIsNoDocuments(err) {
		return ErrNotFound
	}
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "E11000") {
		return ErrConflict
	}
	return err
}
