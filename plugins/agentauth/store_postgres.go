package agentauth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements agentauth.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed agentauth store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
		pg: pgdriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*PostgresStore)(nil)

// ──────────────────────────────────────────────────
// Agent methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateAgent(ctx context.Context, a *Agent) error {
	m := fromAgent(a)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return agentauthPgError(err)
}

func (s *PostgresStore) GetAgent(ctx context.Context, agentID id.AgentID) (*Agent, error) {
	m := new(agentModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", agentID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
	}
	return toAgent(m)
}

func (s *PostgresStore) GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error) {
	m := new(agentModel)
	err := s.pg.NewSelect(m).
		Where("client_id = ?", clientID).
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
	}
	return toAgent(m)
}

func (s *PostgresStore) UpdateAgent(ctx context.Context, a *Agent) error {
	m := fromAgent(a)
	res, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return agentauthPgError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListAgents(ctx context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error) {
	var models []agentModel
	q := s.pg.NewSelect(&models).Where("app_id = ?", appID.String())
	if !orgID.IsNil() {
		q = q.Where("org_id = ?", orgID.String())
	}
	if err := q.OrderExpr("created_at ASC").Scan(ctx); err != nil {
		return nil, agentauthPgError(err)
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

func (s *PostgresStore) CreateAgentGrant(ctx context.Context, g *AgentGrant) error {
	m, err := fromAgentGrant(g)
	if err != nil {
		return err
	}
	_, err = s.pg.NewInsert(m).Exec(ctx)
	return agentauthPgError(err)
}

func (s *PostgresStore) GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	m := new(agentGrantModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", grantID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
	}
	return toAgentGrant(m)
}

func (s *PostgresStore) GetActiveGrant(ctx context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error) {
	m := new(agentGrantModel)
	err := s.pg.NewSelect(m).
		Where("agent_id = ?", agentID.String()).
		Where("user_id = ?", userID.String()).
		Where("org_id = ?", orgID.String()).
		Where("revoked_at IS NULL").
		Where("expires_at > ?", time.Now().UTC()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
	}
	return toAgentGrant(m)
}

func (s *PostgresStore) ListGrantsByUser(ctx context.Context, userID id.UserID) ([]*AgentGrant, error) {
	var models []agentGrantModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
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

func (s *PostgresStore) UpdateAgentGrant(ctx context.Context, g *AgentGrant) error {
	m, err := fromAgentGrant(g)
	if err != nil {
		return err
	}
	res, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return agentauthPgError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) RevokeAgentGrant(ctx context.Context, grantID id.AgentGrantID) error {
	m := new(agentGrantModel)
	err := s.pg.NewSelect(m).Where("id = ?", grantID.String()).Scan(ctx)
	if err != nil {
		return agentauthPgError(err)
	}
	if m.RevokedAt != nil {
		// Already revoked: a no-op, not an error — matches MemoryStore.
		return nil
	}
	now := time.Now().UTC()
	_, err = s.pg.NewUpdate((*agentGrantModel)(nil)).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", grantID.String()).
		Exec(ctx)
	return agentauthPgError(err)
}

func (s *PostgresStore) RevokeGrantsByUser(ctx context.Context, userID id.UserID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "user_id = ?", []any{userID.String()})
}

func (s *PostgresStore) RevokeGrantsByUserOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "user_id = ? AND org_id = ?", []any{userID.String(), orgID.String()})
}

func (s *PostgresStore) RevokeGrantsByOrg(ctx context.Context, orgID id.OrgID) ([]id.AgentGrantID, error) {
	return s.revokeGrantsWhere(ctx, "org_id = ?", []any{orgID.String()})
}

// RevokeGrantsByAgent revokes the agent's grants. A nil orgID means "every
// org", not "the org whose id is empty" — the org filter is only applied
// when orgID carries a real value, matching MemoryStore.
func (s *PostgresStore) RevokeGrantsByAgent(ctx context.Context, agentID id.AgentID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	if orgID.IsNil() {
		return s.revokeGrantsWhere(ctx, "agent_id = ?", []any{agentID.String()})
	}
	return s.revokeGrantsWhere(ctx, "agent_id = ? AND org_id = ?", []any{agentID.String(), orgID.String()})
}

// revokeGrantsWhere returns the ids of every grant matching the given
// filter — whether or not it was already revoked, per the Store interface
// contract documented on Store.RevokeGrantsByUser — and revokes any of them
// that weren't revoked already.
func (s *PostgresStore) revokeGrantsWhere(ctx context.Context, where string, args []any) ([]id.AgentGrantID, error) {
	var models []agentGrantModel
	sel := s.pg.NewSelect(&models)
	sel = sel.Where(where, args...)
	if err := sel.Scan(ctx); err != nil {
		return nil, agentauthPgError(err)
	}
	ids := make([]id.AgentGrantID, 0, len(models))
	for i := range models {
		gid, err := id.ParseAgentGrantID(models[i].ID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, gid)
	}
	if len(ids) == 0 {
		return ids, nil
	}

	now := time.Now().UTC()
	upd := s.pg.NewUpdate((*agentGrantModel)(nil)).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now)
	upd = upd.Where(where, args...)
	upd = upd.Where("revoked_at IS NULL")
	if _, err := upd.Exec(ctx); err != nil {
		return nil, agentauthPgError(err)
	}
	return ids, nil
}

// ──────────────────────────────────────────────────
// Org policy methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) GetOrgPolicy(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	m := new(orgPolicyModel)
	err := s.pg.NewSelect(m).
		Where("org_id = ?", orgID.String()).
		Scan(ctx)
	if err != nil {
		return nil, agentauthPgError(err)
	}
	return toOrgPolicy(m)
}

func (s *PostgresStore) PutOrgPolicy(ctx context.Context, p *OrgAgentPolicy) error {
	if err := validatePolicyMode(p.Mode); err != nil {
		return err
	}
	m, err := fromOrgPolicy(p)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	res, err := s.pg.NewUpdate(m).
		Set("mode = ?", m.Mode).
		Set("max_grant_ttl = ?", m.MaxGrantTTL).
		Set("allowed_scopes = ?", m.AllowedScopes).
		Set("updated_at = ?", now).
		Where("org_id = ?", m.OrgID).
		Exec(ctx)
	if err != nil {
		return agentauthPgError(err)
	}
	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows > 0 {
		return nil
	}

	m.CreatedAt = now
	m.UpdatedAt = now
	_, err = s.pg.NewInsert(m).Exec(ctx)
	return agentauthPgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// agentauthPgError maps low-level pg failures to this package's sentinels.
// A unique-constraint violation on the agents.client_id index becomes
// ErrConflict — CreateAgent's atomicity requirement (the brief calls out
// that a check-then-insert races) is enforced by the partial unique index
// itself; this only translates the driver error the index produces.
func agentauthPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	msg := err.Error()
	if strings.Contains(msg, "SQLSTATE 23505") || strings.Contains(msg, "duplicate key value") {
		return ErrConflict
	}
	return err
}
