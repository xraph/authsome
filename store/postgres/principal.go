package postgres

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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetServiceAccount(ctx context.Context, svcID id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	m := new(ServiceAccountModel)
	err := s.pg.NewSelect(m).Where("id = ?", svcID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toServiceAccount(m)
}

func (s *Store) ListServiceAccounts(ctx context.Context, q *serviceaccount.Query) (*serviceaccount.List, error) {
	var models []ServiceAccountModel
	query := s.pg.NewSelect(&models).Where("app_id = ?", q.AppID.String())

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
		return nil, pgError(err)
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
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteServiceAccount(ctx context.Context, svcID id.ServiceAccountID) error {
	_, err := s.pg.NewDelete((*ServiceAccountModel)(nil)).Where("id = ?", svcID.String()).Exec(ctx)
	return pgError(err)
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
	err := s.pg.NewSelect(m).Where("id = ?", ref.ID).Scan(ctx)
	if err != nil {
		if errors.Is(pgError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, pgError(err)
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
	query := s.pg.NewSelect(&models)

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
		return nil, pgError(err)
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

func (s *Store) CreateDelegation(ctx context.Context, d *principal.Delegation) error {
	m := fromDelegation(d)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetDelegation(ctx context.Context, delID id.DelegationID) (*principal.Delegation, error) {
	m := new(DelegationModel)
	err := s.pg.NewSelect(m).Where("id = ?", delID.String()).Scan(ctx)
	if err != nil {
		if errors.Is(pgError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, pgError(err)
	}
	return toDelegation(m)
}

// FindActiveDelegation resolves the live grant letting actor act for subject.
//
// Active is evaluated in SQL rather than by loading and filtering in Go: this
// runs on the authentication path for every delegated request, and the partial
// unique index means at most one row can match.
//
// The interface takes no time argument (unlike ListDelegations, which is
// given q.ActiveAsOf as a bound parameter), so the expiry check has to name a
// SQL clock rather than pass one in. That has to be clock_timestamp(), not
// NOW(): NOW() is transaction_timestamp(), frozen for the life of the calling
// transaction, so inside a long-lived transaction a grant would stay usable
// past its expires_at for the transaction's whole life. clock_timestamp()
// reads the wall clock at the moment this statement runs instead.
//
// The comparison is `>=`, not `>`: principal.Delegation.IsActive defines a
// grant expiring at exactly the query instant as still active
// (`!at.After(expiresAt)`), and every store must agree with the domain
// method, not just with each other.
func (s *Store) FindActiveDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref, grantKind principal.GrantKind,
) (*principal.Delegation, error) {
	m := new(DelegationModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("actor_kind = ?", string(actor.Kind)).
		Where("actor_id = ?", actor.ID).
		Where("subject_kind = ?", string(subject.Kind)).
		Where("subject_id = ?", subject.ID).
		Where("grant_kind = ?", string(grantKind)).
		Where("revoked_at IS NULL").
		Where("(expires_at IS NULL OR expires_at >= clock_timestamp())").
		Scan(ctx)
	if err != nil {
		if errors.Is(pgError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, pgError(err)
	}
	return toDelegation(m)
}

func (s *Store) ListDelegations(ctx context.Context, q *principal.DelegationQuery) ([]*principal.Delegation, error) {
	var models []DelegationModel
	query := s.pg.NewSelect(&models)

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
		// Inclusive at the boundary, matching principal.Delegation.IsActive:
		// see the comment on ListPrincipals's ActiveOnly branch above.
		query = query.
			Where("revoked_at IS NULL").
			Where("(expires_at IS NULL OR expires_at >= ?)", q.ActiveAsOf)
	}
	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}

	out := make([]*principal.Delegation, 0, len(models))
	for i := range models {
		d, err := toDelegation(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// RevokeDelegation marks a grant revoked. The revoked_at guard makes a repeat
// call a no-op rather than moving the timestamp, so a retried revocation
// cannot rewrite when the revocation actually happened.
func (s *Store) RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error {
	res, err := s.pg.NewUpdate((*DelegationModel)(nil)).
		Set("revoked_at = ?", at).
		Set("updated_at = ?", at).
		Where("id = ?", delID.String()).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 { //nolint:errcheck // RowsAffected always succeeds for pgx
		// Either already revoked or absent. Distinguish, because a caller
		// revoking a grant that never existed has a different problem from one
		// retrying.
		if _, getErr := s.GetDelegation(ctx, delID); getErr != nil {
			return getErr
		}
	}
	return nil
}
