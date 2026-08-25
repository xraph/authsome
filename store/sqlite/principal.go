package sqlite

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Service account store
// ──────────────────────────────────────────────────

func (s *Store) CreateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error {
	m := fromServiceAccount(svc)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetServiceAccount(ctx context.Context, svcID id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	m := new(ServiceAccountModel)
	err := s.sdb.NewSelect(m).Where("id = ?", svcID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toServiceAccount(m)
}

func (s *Store) ListServiceAccounts(ctx context.Context, q *serviceaccount.Query) (*serviceaccount.List, error) {
	var models []ServiceAccountModel
	query := s.sdb.NewSelect(&models).Where("app_id = ?", q.AppID.String())

	if q.Active != nil {
		query = query.Where("active = ?", *q.Active)
	}
	if q.Kind != "" {
		query = query.Where("kind = ?", string(q.Kind))
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	if q.Cursor != "" {
		query = query.Where("id < ?", q.Cursor)
	}

	query = query.OrderExpr("id DESC").Limit(limit + 1)

	err := query.Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}

	list := &serviceaccount.List{
		ServiceAccounts: make([]*serviceaccount.ServiceAccount, 0, len(models)),
		Total:           len(models),
	}

	// Adjust total if we fetched the extra sentinel row (limit+1).
	if len(models) > limit {
		list.Total = limit // at least `limit` results; exact total unknown
	}

	for i := range models {
		if i >= limit {
			list.NextCursor = models[i].ID
			break
		}
		svc, err := toServiceAccount(&models[i])
		if err != nil {
			return nil, err
		}
		list.ServiceAccounts = append(list.ServiceAccounts, svc)
	}
	return list, nil
}

func (s *Store) UpdateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error {
	m := fromServiceAccount(svc)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteServiceAccount(ctx context.Context, svcID id.ServiceAccountID) error {
	_, err := s.sdb.NewDelete((*ServiceAccountModel)(nil)).Where("id = ?", svcID.String()).Exec(ctx)
	return sqliteError(err)
}

// ──────────────────────────────────────────────────
// Principal store
// ──────────────────────────────────────────────────

// GetPrincipal resolves any principal by ref. A user ref is assembled from
// authsome_users; every other kind comes from authsome_service_accounts,
// looked up by ID alone: the ref's Kind only routes which table to read,
// matching how the memory backend resolves a principal.
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

	m := new(ServiceAccountModel)
	err := s.sdb.NewSelect(m).Where("id = ?", ref.ID).Scan(ctx)
	if err != nil {
		if errors.Is(sqliteError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, sqliteError(err)
	}
	svc, err := toServiceAccount(m)
	if err != nil {
		return nil, err
	}
	return svc.ToPrincipal(), nil
}

// ListPrincipals returns principals matching q. Only non-human principals are
// stored in authsome_service_accounts, so this never returns users.
func (s *Store) ListPrincipals(ctx context.Context, q *principal.Query) ([]*principal.Principal, error) {
	var models []ServiceAccountModel
	query := s.sdb.NewSelect(&models)

	if !q.AppID.IsNil() {
		query = query.Where("app_id = ?", q.AppID.String())
	}
	if q.Kind != "" {
		query = query.Where("kind = ?", string(q.Kind))
	}
	if q.OwnerUser != nil {
		query = query.Where("owner_user_id = ?", q.OwnerUser.String())
	}
	if q.Parent != nil {
		// ToPrincipal stamps the parent ref's Kind from the child row's own
		// kind (a child is always minted by a principal of its own kind), so
		// matching on parent_id alone would let a Ref naming a different kind
		// match a row it should not, and a Ref with an empty ID match every
		// row that merely has a parent. Comparing the whole ref, the way
		// store/memory does, is what makes those two cases behave the same
		// on both backends.
		query = query.
			Where("parent_id = ?", q.Parent.ID).
			Where("kind = ?", string(q.Parent.Kind))
	}
	if q.ActiveOnly {
		// Inclusive at the boundary: principal.Principal.IsActive defines
		// active as `!at.After(expiresAt)`, so a principal expiring at exactly
		// ActiveAsOf must still come back here. `>` would exclude it and
		// disagree with the domain method every non-store caller uses.
		query = query.
			Where("active = TRUE").
			Where("(expires_at IS NULL OR expires_at >= ?)", q.ActiveAsOf)
	}
	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}

	out := make([]*principal.Principal, 0, len(models))
	for i := range models {
		svc, err := toServiceAccount(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, svc.ToPrincipal())
	}
	return out, nil
}

// CreateDelegation stores a new grant, refusing one that would duplicate a
// grant already live for the same (app, actor, subject, kind).
//
// The liveness check runs here, in Go, and not in the index predicate. Live
// means principal.Delegation.IsActive: not revoked AND not past its expiry.
// A partial index cannot express the second half, because sqlite requires an index predicate to be deterministic
// and datetime('now') is not, so the index
// (see the delegation_live_index_excludes_expired migration) now covers only
// grants that never expire, and this is where the rest is enforced.
//
// This is a read followed by a write rather than one atomic statement, so two
// concurrent creates for the same tuple can both land. That is a deliberate
// trade against the bug it replaces: filtering on revoked_at alone meant an
// impersonation grant that merely lapsed blocked every later impersonation of
// that user by that admin forever, because nothing but StopImpersonation ever
// writes revoked_at.
func (s *Store) CreateDelegation(ctx context.Context, d *principal.Delegation) error {
	if err := s.refuseLiveDuplicateDelegation(ctx, d); err != nil {
		return err
	}
	m := fromDelegation(d)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

// refuseLiveDuplicateDelegation returns store.ErrConflict when a live grant
// already exists for d's (app, actor, subject, kind).
func (s *Store) refuseLiveDuplicateDelegation(ctx context.Context, d *principal.Delegation) error {
	existing, err := s.FindActiveDelegation(ctx, d.AppID, d.Actor, d.Subject, d.GrantKind)
	if err != nil {
		if errors.Is(err, principal.ErrNotFound) {
			return nil
		}
		return err
	}
	if existing != nil && existing.ID != d.ID {
		return store.ErrConflict
	}
	return nil
}

func (s *Store) GetDelegation(ctx context.Context, delID id.DelegationID) (*principal.Delegation, error) {
	m := new(DelegationModel)
	err := s.sdb.NewSelect(m).Where("id = ?", delID.String()).Scan(ctx)
	if err != nil {
		if errors.Is(sqliteError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, sqliteError(err)
	}
	return toDelegation(m)
}

// FindActiveDelegation resolves the live grant letting actor act for subject.
//
// Revocation is filtered in SQL; expiry is filtered in Go. That split is not
// a style choice, it is what this backend actually supports.
//
// A bare `expires_at >= ?` here matched every row whatever its expiry, and an
// expired grant went on authenticating. The column is declared TIMESTAMP,
// which sqlite gives NUMERIC affinity, but the driver writes an ISO-8601
// string that sqlite cannot coerce to a number, so the stored value settles as
// TEXT while the bound time.Time parameter arrives as a number. Sqlite then
// compares across storage classes by its own type ordering, where every TEXT
// value outranks every numeric one, and the predicate is constant-true.
//
// Wrapping both sides in julianday() does not rescue it: julianday() applied
// to a numeric argument reads that number as a Julian day count rather than as
// an instant, so the bound parameter comes back astronomically large and the
// predicate flips to constant-false. Constant-false is worse than
// constant-true, because it fails closed on grants that are perfectly live.
//
// Filtering in Go removes the guesswork. principal.Delegation.IsActive is the
// definition every other consumer already uses, boundary convention included:
// a grant expiring at exactly this instant is still active. The row count this
// walks is bounded by the live-grant uniqueness constraint on
// (app, actor, subject, kind), so this is a handful of rows at most even on
// the authentication path.
func (s *Store) FindActiveDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref, grantKind principal.GrantKind,
) (*principal.Delegation, error) {
	var models []DelegationModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("actor_kind = ?", string(actor.Kind)).
		Where("actor_id = ?", actor.ID).
		Where("subject_kind = ?", string(subject.Kind)).
		Where("subject_id = ?", subject.ID).
		Where("grant_kind = ?", string(grantKind)).
		Where("revoked_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(sqliteError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, sqliteError(err)
	}

	now := time.Now()
	for i := range models {
		d, convErr := toDelegation(&models[i])
		if convErr != nil {
			return nil, convErr
		}
		if d.IsActive(now) {
			return d, nil
		}
	}
	return nil, principal.ErrNotFound
}

func (s *Store) ListDelegations(ctx context.Context, q *principal.DelegationQuery) ([]*principal.Delegation, error) {
	var models []DelegationModel
	query := s.sdb.NewSelect(&models)

	if !q.AppID.IsNil() {
		query = query.Where("app_id = ?", q.AppID.String())
	}
	if q.Actor != nil {
		query = query.
			Where("actor_kind = ?", string(q.Actor.Kind)).
			Where("actor_id = ?", q.Actor.ID)
	}
	if q.Subject != nil {
		query = query.
			Where("subject_kind = ?", string(q.Subject.Kind)).
			Where("subject_id = ?", q.Subject.ID)
	}
	if q.GrantKind != "" {
		query = query.Where("grant_kind = ?", string(q.GrantKind))
	}
	if q.ActiveOnly {
		// Revocation in SQL, expiry in Go, for the reason spelled out on
		// FindActiveDelegation above: no comparison this backend can express
		// against this column answers "is it past its expiry" correctly.
		query = query.Where("revoked_at IS NULL")
	}
	if q.Limit > 0 && !q.ActiveOnly {
		// The limit only rides along in SQL when nothing is filtered out
		// afterwards. With ActiveOnly it is applied below instead, or a page
		// of mostly-expired grants would come back short.
		query = query.Limit(q.Limit)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}

	out := make([]*principal.Delegation, 0, len(models))
	for i := range models {
		d, err := toDelegation(&models[i])
		if err != nil {
			return nil, err
		}
		// Inclusive at the boundary, because IsActive is: a grant expiring at
		// exactly ActiveAsOf is still active.
		if q.ActiveOnly && !d.IsActive(q.ActiveAsOf) {
			continue
		}
		out = append(out, d)
		if q.Limit > 0 && len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

// RevokeDelegation marks a grant revoked. The revoked_at guard makes a repeat
// call a no-op rather than moving the timestamp, so a retried revocation
// cannot rewrite when the revocation actually happened.
func (s *Store) RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error {
	res, err := s.sdb.NewUpdate((*DelegationModel)(nil)).
		Set("revoked_at = ?", at).
		Set("updated_at = ?", at).
		Where("id = ?", delID.String()).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return sqliteError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 { //nolint:errcheck // RowsAffected always succeeds for sqlite
		// Either already revoked or absent. Distinguish, because a caller
		// revoking a grant that never existed has a different problem from one
		// retrying.
		if _, getErr := s.GetDelegation(ctx, delID); getErr != nil {
			return getErr
		}
	}
	return nil
}
