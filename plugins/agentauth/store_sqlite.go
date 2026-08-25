package agentauth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements agentauth.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed agentauth store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{
		db:  db,
		sdb: sqlitedriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*SqliteStore)(nil)

// ──────────────────────────────────────────────────
// Agent methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateAgent(ctx context.Context, a *Agent) error {
	m := fromAgent(a)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return agentauthSqliteError(err)
}

func (s *SqliteStore) GetAgent(ctx context.Context, agentID id.AgentID) (*Agent, error) {
	m := new(agentModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", agentID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	return toAgent(m)
}

func (s *SqliteStore) GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error) {
	m := new(agentModel)
	err := s.sdb.NewSelect(m).
		Where("client_id = ?", clientID).
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	return toAgent(m)
}

func (s *SqliteStore) UpdateAgent(ctx context.Context, a *Agent) error {
	m := fromAgent(a)
	res, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return agentauthSqliteError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) ListAgents(ctx context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error) {
	var models []agentModel
	q := s.sdb.NewSelect(&models).Where("app_id = ?", appID.String())
	if !orgID.IsNil() {
		q = q.Where("org_id = ?", orgID.String())
	}
	if err := q.OrderExpr("created_at ASC").Scan(ctx); err != nil {
		return nil, agentauthSqliteError(err)
	}
	out := make([]*Agent, 0, len(models))
	for i := range models {
		a, err := toAgent(&models[i])
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

func (s *SqliteStore) CreateAgentGrant(ctx context.Context, g *AgentGrant) error {
	m, err := fromAgentGrant(g)
	if err != nil {
		return err
	}
	_, err = s.sdb.NewInsert(m).Exec(ctx)
	return agentauthSqliteError(err)
}

func (s *SqliteStore) GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	m := new(agentGrantModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", grantID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	return toAgentGrant(m)
}

// GetActiveGrant returns the most recently created matching grant when more
// than one exists. See the identical comment on PostgresStore.GetActiveGrant
// for why duplicates are the normal state and why an explicit order is
// required for the four backends to agree on which one "the" active grant
// is.
func (s *SqliteStore) GetActiveGrant(ctx context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error) {
	m := new(agentGrantModel)
	err := s.sdb.NewSelect(m).
		Where("agent_id = ?", agentID.String()).
		Where("user_id = ?", userID.String()).
		Where("org_id = ?", orgID.String()).
		Where("revoked_at IS NULL").
		Where("expires_at > ?", time.Now().UTC()).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	return toAgentGrant(m)
}

func (s *SqliteStore) ListGrantsByUser(ctx context.Context, userID id.UserID) ([]*AgentGrant, error) {
	var models []agentGrantModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	out := make([]*AgentGrant, 0, len(models))
	for i := range models {
		g, err := toAgentGrant(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (s *SqliteStore) UpdateAgentGrant(ctx context.Context, g *AgentGrant) error {
	m, err := fromAgentGrant(g)
	if err != nil {
		return err
	}
	res, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return agentauthSqliteError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) RevokeAgentGrant(ctx context.Context, grantID id.AgentGrantID) error {
	m := new(agentGrantModel)
	err := s.sdb.NewSelect(m).Where("id = ?", grantID.String()).Scan(ctx)
	if err != nil {
		return agentauthSqliteError(err)
	}
	if m.RevokedAt != nil {
		// Already revoked: a no-op, not an error — matches MemoryStore.
		return nil
	}
	now := time.Now().UTC()
	_, err = s.sdb.NewUpdate((*agentGrantModel)(nil)).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", grantID.String()).
		Where("revoked_at IS NULL").
		Exec(ctx)
	return agentauthSqliteError(err)
}

func (s *SqliteStore) RevokeGrantsByUser(ctx context.Context, userID id.UserID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "user_id = ?", []any{userID.String()})
}

func (s *SqliteStore) RevokeGrantsByUserOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "user_id = ? AND org_id = ?", []any{userID.String(), orgID.String()})
}

func (s *SqliteStore) RevokeGrantsByOrg(ctx context.Context, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "org_id = ?", []any{orgID.String()})
}

// RevokeGrantsByAgent revokes the agent's grants. A nil orgID means "every
// org", not "the org whose id is empty" — the org filter is only applied
// when orgID carries a real value, matching MemoryStore.
func (s *SqliteStore) RevokeGrantsByAgent(ctx context.Context, agentID id.AgentID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	if orgID.IsNil() {
		return s.revokeGrantsWhere(ctx, "agent_id = ?", []any{agentID.String()})
	}
	return s.revokeGrantsWhere(ctx, "agent_id = ? AND org_id = ?", []any{agentID.String(), orgID.String()})
}

// revokeGrantsWhere returns the ids of every grant matching the given
// filter — whether or not it was already revoked, per the Store interface
// contract documented on Store.RevokeGrantsByUser — and revokes any of them
// that weren't revoked already.
//
// The write runs BEFORE the read, not after: a select-then-update (the
// original shape here) leaves a window where a grant matching the filter,
// inserted between the two statements, gets caught and revoked by the
// UPDATE but was never in the SELECT's result — so its id is missing from
// the return value and sweepSessions never deletes its sessions, which is
// exactly the under-sweeping the interface comment warns against. Updating
// first closes that: any row this call revokes has revoked_at set before
// the SELECT ever runs, so "WHERE <filter> AND revoked_at IS NOT NULL"
// afterward is guaranteed to include it. A row that starts existing only
// after the UPDATE has run is, by construction, not one this call touched —
// it's still unrevoked, so the revoked_at IS NOT NULL filter correctly
// excludes it rather than falsely claiming it was swept.
func (s *SqliteStore) revokeGrantsWhere(ctx context.Context, where string, args []any) ([]id.AgentGrantID, error) {
	now := time.Now().UTC()
	upd := s.sdb.NewUpdate((*agentGrantModel)(nil)).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now)
	upd = upd.Where(where, args...)
	upd = upd.Where("revoked_at IS NULL")
	if _, err := upd.Exec(ctx); err != nil {
		return nil, agentauthSqliteError(err)
	}

	var models []agentGrantModel
	sel := s.sdb.NewSelect(&models)
	sel = sel.Where(where, args...)
	sel = sel.Where("revoked_at IS NOT NULL")
	if err := sel.Scan(ctx); err != nil {
		return nil, agentauthSqliteError(err)
	}
	ids := make([]id.AgentGrantID, 0, len(models))
	for i := range models {
		gid, err := id.ParseAgentGrantID(models[i].ID)
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

func (s *SqliteStore) GetOrgPolicy(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	m := new(orgPolicyModel)
	err := s.sdb.NewSelect(m).
		Where("org_id = ?", orgID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthSqliteError(err)
	}
	return toOrgPolicy(m)
}

func (s *SqliteStore) PutOrgPolicy(ctx context.Context, p *OrgAgentPolicy) error {
	if err := validatePolicyMode(p.Mode); err != nil {
		return err
	}
	m, err := fromOrgPolicy(p)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	res, err := s.sdb.NewUpdate(m).
		Set("mode = ?", m.Mode).
		Set("max_grant_ttl = ?", m.MaxGrantTTL).
		Set("allowed_scopes = ?", m.AllowedScopes).
		Set("updated_at = ?", now).
		Where("org_id = ?", m.OrgID).
		Exec(ctx)
	if err != nil {
		return agentauthSqliteError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows > 0 {
		return nil
	}

	m.CreatedAt = now
	m.UpdatedAt = now
	_, err = s.sdb.NewInsert(m).Exec(ctx)
	return agentauthSqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// agentauthSqliteError maps low-level sqlite failures to this package's
// sentinels. A unique-constraint violation on the agents.client_id index
// becomes ErrConflict.
func agentauthSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrConflict
	}
	return err
}
