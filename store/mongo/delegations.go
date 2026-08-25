package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/store"
)

// findLimit turns a query limit into driver options. Zero means unlimited,
// which is how both the memory and postgres backends read an unset Limit.
func findLimit(n int) []options.Lister[options.FindOptions] {
	if n <= 0 {
		return nil
	}
	return []options.Lister[options.FindOptions]{options.Find().SetLimit(int64(n))}
}

// ──────────────────────────────────────────────────
// Principal read side
// ──────────────────────────────────────────────────

// GetPrincipal resolves any principal by ref. A user ref is assembled from the
// users collection; every other kind comes from the service accounts one,
// looked up by id alone. The ref's Kind only routes which collection to read,
// matching how the memory and postgres backends resolve a principal.
func (s *Store) GetPrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error) {
	if ref.Kind == principal.KindUser {
		uid, err := id.ParseUserID(ref.ID)
		if err != nil {
			return nil, principal.ErrNotFound
		}
		u, err := s.GetUser(ctx, uid)
		if err != nil {
			return nil, principal.ErrNotFound
		}
		return &principal.Principal{
			Ref:      ref,
			AppID:    u.AppID,
			EnvID:    u.EnvID,
			Name:     u.Name(),
			Disabled: false,
		}, nil
	}

	var m serviceAccountModel
	err := s.mdb.Collection(colServiceAccounts).
		FindOne(ctx, bson.M{"_id": ref.ID}).Decode(&m)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, principal.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get principal: %w", err)
	}
	svc, err := fromServiceAccountModel(&m)
	if err != nil {
		return nil, err
	}
	return svc.ToPrincipal(), nil
}

// ListPrincipals returns principals matching q. Only non-human principals live
// in the service accounts collection, so this never returns users.
func (s *Store) ListPrincipals(ctx context.Context, q *principal.Query) ([]*principal.Principal, error) {
	filter := bson.M{}
	// Clauses that can each constrain "kind" go through $and rather than
	// straight into the map: q.Kind and q.Parent.Kind are separate filters
	// that both land on the same field, and a plain map assignment would let
	// the second silently replace the first. Under $and two disagreeing kinds
	// match nothing, which is the honest answer, since a principal has one.
	var and []bson.M
	if !q.AppID.IsNil() {
		filter["app_id"] = q.AppID.String()
	}
	if q.Kind != "" {
		and = append(and, bson.M{"kind": string(q.Kind)})
	}
	if q.OwnerUser != nil {
		filter["owner_user_id"] = q.OwnerUser.String()
	}
	if q.Parent != nil {
		and = append(and, bson.M{"parent_id": q.Parent.ID})
		// ToPrincipal stamps a child's Parent ref with the child's own kind,
		// so the kind half of that ref constrains the child, not the parent.
		// Matching on the id alone would return a child of the right parent
		// under a kind the caller did not ask for.
		if q.Parent.Kind != "" {
			and = append(and, bson.M{"kind": string(q.Parent.Kind)})
		}
	}
	if len(and) > 0 {
		filter["$and"] = and
	}
	if q.ActiveOnly {
		filter["active"] = true
		// expires_at is null for a principal that never lapses. grove writes
		// the key whatever the omitempty tag says, so the null branch is the
		// common one rather than the absent branch.
		filter["$or"] = bson.A{
			bson.M{"expires_at": nil},
			bson.M{"expires_at": bson.M{"$gte": q.ActiveAsOf}},
		}
	}

	cur, err := s.mdb.Collection(colServiceAccounts).Find(ctx, filter, findLimit(q.Limit)...)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list principals: %w", err)
	}
	defer cur.Close(ctx)

	out := make([]*principal.Principal, 0)
	for cur.Next(ctx) {
		var m serviceAccountModel
		if err := cur.Decode(&m); err != nil {
			return nil, fmt.Errorf("authsome/mongo: decode principal: %w", err)
		}
		svc, err := fromServiceAccountModel(&m)
		if err != nil {
			return nil, err
		}
		out = append(out, svc.ToPrincipal())
	}
	return out, cur.Err()
}

// ──────────────────────────────────────────────────
// Delegation grants
// ──────────────────────────────────────────────────

// CreateDelegation stores a new grant, refusing one that would duplicate a
// grant already live for the same (app, actor, subject, kind).
//
// The liveness check runs here rather than in the index. Live means
// principal.Delegation.IsActive: not revoked AND not past its expiry, and a
// partialFilterExpression has no way to name the current time, so the unique
// index (see delegation_live_index_excludes_expired) now covers only grants
// that never expire and this covers the rest.
//
// Read-then-write, not atomic, so two concurrent creates for the same tuple
// can both land. That trade is deliberate: filtering on revoked_at alone meant
// an impersonation grant that merely lapsed blocked every later impersonation
// of that user by that admin forever, because nothing but StopImpersonation
// ever writes revoked_at.
func (s *Store) CreateDelegation(ctx context.Context, d *principal.Delegation) error {
	existing, findErr := s.FindActiveDelegation(ctx, d.AppID, d.Actor, d.Subject, d.GrantKind)
	switch {
	case findErr != nil && !errors.Is(findErr, principal.ErrNotFound):
		return findErr
	case findErr == nil && existing != nil && existing.ID != d.ID:
		return store.ErrConflict
	}

	m := toDelegationModel(d)
	// NewInsert, not a raw driver InsertOne, matching CreateServiceAccount in
	// this same package. Grove's insert path writes every mapped field
	// regardless of a bson omitempty tag, so a nil RevokedAt still reaches
	// mongo as an explicit revoked_at: null. The delegation partial unique
	// index (create_authsome_delegations) filters on {"revoked_at": {"$type":
	// "null"}} for exactly that reason: a raw InsertOne goes through the
	// standard driver encoder instead, which DOES honour omitempty, so a nil
	// RevokedAt would be omitted from the document entirely and $type: "null"
	// would never match, leaving the unique index unable to catch a second
	// live grant for the same (app, actor, subject, kind). NewInsert also
	// runs grove's model and store hook pipeline, which the sibling
	// Create* methods all get.
	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return store.ErrConflict
		}
		return fmt.Errorf("authsome/mongo: create delegation: %w", err)
	}
	return nil
}

func (s *Store) GetDelegation(ctx context.Context, delID id.DelegationID) (*principal.Delegation, error) {
	var m delegationModel
	err := s.mdb.Collection(colDelegations).
		FindOne(ctx, bson.M{"_id": delID.String()}).Decode(&m)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, principal.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get delegation: %w", err)
	}
	return fromDelegationModel(&m)
}

// FindActiveDelegation resolves the live grant letting actor act for subject.
//
// Liveness is evaluated in the query rather than by loading and filtering in
// Go: this runs on the authentication path for every delegated request, and
// the partial unique index means at most one document can match.
func (s *Store) FindActiveDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref, grantKind principal.GrantKind,
) (*principal.Delegation, error) {
	now := time.Now()
	filter := bson.M{
		"app_id":       appID.String(),
		"actor_kind":   string(actor.Kind),
		"actor_id":     actor.ID,
		"subject_kind": string(subject.Kind),
		"subject_id":   subject.ID,
		"grant_kind":   string(grantKind),
		"revoked_at":   nil,
		"$or": bson.A{
			bson.M{"expires_at": nil},
			bson.M{"expires_at": bson.M{"$gte": now}},
		},
	}

	var m delegationModel
	err := s.mdb.Collection(colDelegations).FindOne(ctx, filter).Decode(&m)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, principal.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: find active delegation: %w", err)
	}
	return fromDelegationModel(&m)
}

func (s *Store) ListDelegations(ctx context.Context, q *principal.DelegationQuery) ([]*principal.Delegation, error) {
	filter := bson.M{}
	if !q.AppID.IsNil() {
		filter["app_id"] = q.AppID.String()
	}
	if !q.OrgID.IsNil() {
		filter["org_id"] = q.OrgID.String()
	}
	if q.Actor != nil {
		filter["actor_kind"] = string(q.Actor.Kind)
		filter["actor_id"] = q.Actor.ID
	}
	if q.Subject != nil {
		filter["subject_kind"] = string(q.Subject.Kind)
		filter["subject_id"] = q.Subject.ID
	}
	if q.GrantKind != "" {
		filter["grant_kind"] = string(q.GrantKind)
	}
	if q.ActiveOnly {
		filter["revoked_at"] = nil
		filter["$or"] = bson.A{
			bson.M{"expires_at": nil},
			bson.M{"expires_at": bson.M{"$gte": q.ActiveAsOf}},
		}
	}

	cur, err := s.mdb.Collection(colDelegations).Find(ctx, filter, findLimit(q.Limit)...)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list delegations: %w", err)
	}
	defer cur.Close(ctx)

	out := make([]*principal.Delegation, 0)
	for cur.Next(ctx) {
		var m delegationModel
		if err := cur.Decode(&m); err != nil {
			return nil, fmt.Errorf("authsome/mongo: decode delegation: %w", err)
		}
		d, err := fromDelegationModel(&m)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, cur.Err()
}

// RevokeDelegation marks a grant revoked. The revoked_at guard makes a repeat
// call a no-op rather than moving the timestamp, so a retried revocation
// cannot rewrite when the revocation actually happened.
func (s *Store) RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error {
	res, err := s.mdb.Collection(colDelegations).UpdateOne(ctx,
		bson.M{"_id": delID.String(), "revoked_at": nil},
		bson.M{"$set": bson.M{"revoked_at": at, "updated_at": at}},
	)
	if err != nil {
		return fmt.Errorf("authsome/mongo: revoke delegation: %w", err)
	}
	if res.MatchedCount == 0 {
		// Either already revoked or absent. Distinguish, because a caller
		// revoking a grant that never existed has a different problem from one
		// retrying.
		if _, getErr := s.GetDelegation(ctx, delID); getErr != nil {
			return getErr
		}
	}
	return nil
}
