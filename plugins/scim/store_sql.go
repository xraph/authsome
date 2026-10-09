package scim

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"

	"golang.org/x/crypto/bcrypt"
)

// The SQL stores share models and queries; the only difference between
// postgres and sqlite is the driver handle the queries run on, so both are
// one type over a small query interface.

type scimConfigModel struct {
	grove.BaseModel `grove:"table:authsome_scim_configs,alias:sc"`

	ID          string          `grove:"id,pk"`
	AppID       string          `grove:"app_id,notnull"`
	OrgID       string          `grove:"org_id"`
	Name        string          `grove:"name,notnull"`
	Enabled     bool            `grove:"enabled,notnull"`
	AutoCreate  bool            `grove:"auto_create,notnull"`
	AutoSuspend bool            `grove:"auto_suspend,notnull"`
	GroupSync   bool            `grove:"group_sync,notnull"`
	DefaultRole string          `grove:"default_role,notnull"`
	Metadata    json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedAt   time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time       `grove:"updated_at,notnull,default:now()"`
}

type scimTokenModel struct {
	grove.BaseModel `grove:"table:authsome_scim_tokens,alias:st"`

	ID          string         `grove:"id,pk"`
	ConfigID    string         `grove:"config_id,notnull"`
	Name        string         `grove:"name,notnull"`
	TokenHash   string         `grove:"token_hash,notnull"`
	TokenLookup sql.NullString `grove:"token_lookup"`
	LastUsedAt  *time.Time     `grove:"last_used_at"`
	ExpiresAt   *time.Time     `grove:"expires_at"`
	CreatedAt   time.Time      `grove:"created_at,notnull,default:now()"`
}

type scimLogModel struct {
	grove.BaseModel `grove:"table:authsome_scim_provision_logs,alias:sl"`

	ID           string    `grove:"id,pk"`
	ConfigID     string    `grove:"config_id,notnull"`
	Action       string    `grove:"action,notnull"`
	ResourceType string    `grove:"resource_type,notnull"`
	ExternalID   string    `grove:"external_id"`
	InternalID   string    `grove:"internal_id"`
	Status       string    `grove:"status,notnull"`
	Detail       string    `grove:"detail"`
	CreatedAt    time.Time `grove:"created_at,notnull,default:now()"`
}

func fromConfig(c *SCIMConfig) *scimConfigModel {
	meta, _ := json.Marshal(c.Metadata) //nolint:errcheck // marshaling a string map
	if len(c.Metadata) == 0 {
		meta = []byte("{}")
	}
	orgID := ""
	if !c.OrgID.IsNil() {
		orgID = c.OrgID.String()
	}
	return &scimConfigModel{
		ID: c.ID.String(), AppID: c.AppID.String(), OrgID: orgID, Name: c.Name, Enabled: c.Enabled,
		AutoCreate: c.AutoCreate, AutoSuspend: c.AutoSuspend, GroupSync: c.GroupSync, DefaultRole: c.DefaultRole,
		Metadata: meta, CreatedAt: c.CreatedAt.UTC().Round(0), UpdatedAt: c.UpdatedAt.UTC().Round(0),
	}
}

func toConfig(m *scimConfigModel) (*SCIMConfig, error) {
	cfgID, err := id.ParseSCIMConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	c := &SCIMConfig{
		ID: cfgID, AppID: appID, Name: m.Name, Enabled: m.Enabled, AutoCreate: m.AutoCreate,
		AutoSuspend: m.AutoSuspend, GroupSync: m.GroupSync, DefaultRole: m.DefaultRole,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	if m.OrgID != "" {
		if orgID, perr := id.ParseOrgID(m.OrgID); perr == nil {
			c.OrgID = orgID
		}
	}
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &c.Metadata) //nolint:errcheck // best-effort decode
	}
	return c, nil
}

func fromToken(t *Token) *scimTokenModel {
	m := &scimTokenModel{
		ID: t.ID.String(), ConfigID: t.ConfigID.String(), Name: t.Name, TokenHash: t.TokenHash,
		TokenLookup: sql.NullString{String: t.TokenLookup, Valid: t.TokenLookup != ""},
		CreatedAt:   t.CreatedAt.UTC().Round(0),
	}
	if t.LastUsedAt != nil {
		v := t.LastUsedAt.UTC().Round(0)
		m.LastUsedAt = &v
	}
	if t.ExpiresAt != nil {
		v := t.ExpiresAt.UTC().Round(0)
		m.ExpiresAt = &v
	}
	return m
}

func toToken(m *scimTokenModel) (*Token, error) {
	tokID, err := id.ParseSCIMTokenID(m.ID)
	if err != nil {
		return nil, err
	}
	cfgID, err := id.ParseSCIMConfigID(m.ConfigID)
	if err != nil {
		return nil, err
	}
	return &Token{
		ID: tokID, ConfigID: cfgID, Name: m.Name, TokenHash: m.TokenHash, TokenLookup: m.TokenLookup.String,
		LastUsedAt: m.LastUsedAt, ExpiresAt: m.ExpiresAt, CreatedAt: m.CreatedAt,
	}, nil
}

func fromLog(l *ProvisionLog) *scimLogModel {
	return &scimLogModel{
		ID: l.ID.String(), ConfigID: l.ConfigID.String(), Action: l.Action, ResourceType: l.ResourceType,
		ExternalID: l.ExternalID, InternalID: l.InternalID, Status: l.Status, Detail: l.Detail,
		CreatedAt: l.CreatedAt.UTC().Round(0),
	}
}

func toLog(m *scimLogModel) (*ProvisionLog, error) {
	logID, err := id.ParseSCIMLogID(m.ID)
	if err != nil {
		return nil, err
	}
	cfgID, err := id.ParseSCIMConfigID(m.ConfigID)
	if err != nil {
		return nil, err
	}
	return &ProvisionLog{
		ID: logID, ConfigID: cfgID, Action: m.Action, ResourceType: m.ResourceType, ExternalID: m.ExternalID,
		InternalID: m.InternalID, Status: m.Status, Detail: m.Detail, CreatedAt: m.CreatedAt,
	}, nil
}

// sqlQuerier is the slice of the grove driver handles the SQL store uses.
// Both pgdriver.PgDB and sqlitedriver.SqliteDB satisfy it.
type sqlQuerier interface {
	NewSelect(model ...any) sqlSelect
	NewInsert(model any) sqlExec
	NewUpdate(model any) sqlUpdate
	NewDelete(model any) sqlDelete
}

// The query shapes below name only what this store calls, so one store body
// serves both drivers through small adapters.
type sqlSelect interface {
	Where(query string, args ...any) sqlSelect
	OrderExpr(expr string) sqlSelect
	Limit(n int) sqlSelect
	Scan(ctx context.Context) error
}

// sqlResult is the slice of a driver result this store reads.
type sqlResult interface {
	RowsAffected() (int64, error)
}

type sqlExec interface {
	Exec(ctx context.Context) (sqlResult, error)
}

type sqlUpdate interface {
	Set(query string, args ...any) sqlUpdate
	Where(query string, args ...any) sqlUpdate
	Exec(ctx context.Context) (sqlResult, error)
}

type sqlDelete interface {
	Where(query string, args ...any) sqlDelete
	Exec(ctx context.Context) (sqlResult, error)
}

// SQLStore implements Store over postgres or sqlite.
type SQLStore struct {
	q    sqlQuerier
	name string
}

// NewPostgresStore creates a SCIM store on the postgres driver.
func NewPostgresStore(db *grove.DB) *SQLStore {
	return &SQLStore{q: pgQuerier{pgdriver.Unwrap(db)}, name: "postgres"}
}

// NewSqliteStore creates a SCIM store on the sqlite driver.
func NewSqliteStore(db *grove.DB) *SQLStore {
	return &SQLStore{q: sqliteQuerier{sqlitedriver.Unwrap(db)}, name: "sqlite"}
}

var _ Store = (*SQLStore)(nil)

func (s *SQLStore) wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("scim/%s: %s: %w", s.name, op, err)
}

// ── configs ──

func (s *SQLStore) CreateConfig(ctx context.Context, c *SCIMConfig) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	_, err := s.q.NewInsert(fromConfig(c)).Exec(ctx)
	return s.wrap("create config", err)
}

func (s *SQLStore) GetConfig(ctx context.Context, configID id.SCIMConfigID) (*SCIMConfig, error) {
	m := new(scimConfigModel)
	if err := s.q.NewSelect(m).Where("id = ?", configID.String()).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configID)
		}
		return nil, s.wrap("get config", err)
	}
	return toConfig(m)
}

func (s *SQLStore) UpdateConfig(ctx context.Context, c *SCIMConfig) error {
	c.UpdatedAt = time.Now()
	m := fromConfig(c)
	res, err := s.q.NewUpdate((*scimConfigModel)(nil)).
		Set("name = ?", m.Name).
		Set("enabled = ?", m.Enabled).
		Set("auto_create = ?", m.AutoCreate).
		Set("auto_suspend = ?", m.AutoSuspend).
		Set("group_sync = ?", m.GroupSync).
		Set("default_role = ?", m.DefaultRole).
		Set("metadata = ?", m.Metadata).
		Set("updated_at = ?", m.UpdatedAt).
		Where("id = ?", m.ID).
		Exec(ctx)
	if err != nil {
		return s.wrap("update config", err)
	}
	if n, _ := res.RowsAffected(); n == 0 { //nolint:errcheck // both drivers report rows affected
		return fmt.Errorf("%w: %s", ErrConfigNotFound, c.ID)
	}
	return nil
}

func (s *SQLStore) DeleteConfig(ctx context.Context, configID id.SCIMConfigID) error {
	_, err := s.q.NewDelete((*scimConfigModel)(nil)).Where("id = ?", configID.String()).Exec(ctx)
	return s.wrap("delete config", err)
}

func (s *SQLStore) listConfigs(ctx context.Context, col, val string) ([]*SCIMConfig, error) {
	var models []scimConfigModel
	err := s.q.NewSelect(&models).Where(col+" = ?", val).OrderExpr("created_at DESC").Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, s.wrap("list configs", err)
	}
	out := make([]*SCIMConfig, 0, len(models))
	for i := range models {
		c, convErr := toConfig(&models[i])
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *SQLStore) ListConfigs(ctx context.Context, appID string) ([]*SCIMConfig, error) {
	return s.listConfigs(ctx, "app_id", appID)
}

func (s *SQLStore) ListConfigsByOrg(ctx context.Context, orgID id.OrgID) ([]*SCIMConfig, error) {
	return s.listConfigs(ctx, "org_id", orgID.String())
}

// ── tokens ──

func (s *SQLStore) CreateToken(ctx context.Context, t *Token) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.q.NewInsert(fromToken(t)).Exec(ctx)
	return s.wrap("create token", err)
}

func (s *SQLStore) GetToken(ctx context.Context, tokenID id.SCIMTokenID) (*Token, error) {
	m := new(scimTokenModel)
	if err := s.q.NewSelect(m).Where("id = ?", tokenID.String()).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrTokenNotFound, tokenID)
		}
		return nil, s.wrap("get token", err)
	}
	return toToken(m)
}

func (s *SQLStore) UpdateToken(ctx context.Context, t *Token) error {
	m := fromToken(t)
	res, err := s.q.NewUpdate((*scimTokenModel)(nil)).
		Set("name = ?", m.Name).
		Set("token_lookup = ?", m.TokenLookup).
		Set("last_used_at = ?", m.LastUsedAt).
		Set("expires_at = ?", m.ExpiresAt).
		Where("id = ?", m.ID).
		Exec(ctx)
	if err != nil {
		return s.wrap("update token", err)
	}
	if n, _ := res.RowsAffected(); n == 0 { //nolint:errcheck // both drivers report rows affected
		return fmt.Errorf("%w: %s", ErrTokenNotFound, t.ID)
	}
	return nil
}

func (s *SQLStore) ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*Token, error) {
	var models []scimTokenModel
	err := s.q.NewSelect(&models).Where("config_id = ?", configID.String()).OrderExpr("created_at DESC").Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, s.wrap("list tokens", err)
	}
	out := make([]*Token, 0, len(models))
	for i := range models {
		t, convErr := toToken(&models[i])
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *SQLStore) DeleteToken(ctx context.Context, tokenID id.SCIMTokenID) error {
	_, err := s.q.NewDelete((*scimTokenModel)(nil)).Where("id = ?", tokenID.String()).Exec(ctx)
	return s.wrap("delete token", err)
}

// FindTokenByPlaintext resolves the presented token by its SHA-256 lookup
// column and bcrypt-compares that one row. Rows from before the column
// existed carry no lookup and are found the old way, by comparing each; a
// match there is rewritten with its lookup so the scan is paid once.
func (s *SQLStore) FindTokenByPlaintext(ctx context.Context, plaintext string) (*Token, *SCIMConfig, error) {
	lookup := store.HashToken(plaintext)
	m := new(scimTokenModel)
	err := s.q.NewSelect(m).Where("token_lookup = ?", lookup).Scan(ctx)
	switch {
	case err == nil:
		if bcrypt.CompareHashAndPassword([]byte(m.TokenHash), []byte(plaintext)) != nil {
			return nil, nil, ErrTokenNotFound
		}
	case errors.Is(err, sql.ErrNoRows):
		var legacy []scimTokenModel
		if lerr := s.q.NewSelect(&legacy).Where("token_lookup IS NULL").Scan(ctx); lerr != nil && !errors.Is(lerr, sql.ErrNoRows) {
			return nil, nil, s.wrap("scan legacy tokens", lerr)
		}
		m = nil
		for i := range legacy {
			if bcrypt.CompareHashAndPassword([]byte(legacy[i].TokenHash), []byte(plaintext)) == nil {
				m = &legacy[i]
				break
			}
		}
		if m == nil {
			return nil, nil, ErrTokenNotFound
		}
		_, _ = s.q.NewUpdate((*scimTokenModel)(nil)). //nolint:errcheck // best-effort upgrade
								Set("token_lookup = ?", lookup).
								Where("id = ?", m.ID).
								Where("token_lookup IS NULL").
								Exec(ctx)
		m.TokenLookup = sql.NullString{String: lookup, Valid: true}
	default:
		return nil, nil, s.wrap("find token", err)
	}
	t, err := toToken(m)
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

func (s *SQLStore) CreateLog(ctx context.Context, l *ProvisionLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	_, err := s.q.NewInsert(fromLog(l)).Exec(ctx)
	return s.wrap("create log", err)
}

func (s *SQLStore) listLogs(ctx context.Context, where, arg string, limit int) ([]*ProvisionLog, error) {
	var models []scimLogModel
	q := s.q.NewSelect(&models).Where(where, arg).OrderExpr("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, s.wrap("list logs", err)
	}
	out := make([]*ProvisionLog, 0, len(models))
	for i := range models {
		l, convErr := toLog(&models[i])
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, l)
	}
	return out, nil
}

func (s *SQLStore) ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*ProvisionLog, error) {
	return s.listLogs(ctx, "config_id = ?", configID.String(), limit)
}

func (s *SQLStore) ListAllLogs(ctx context.Context, appID string, limit int) ([]*ProvisionLog, error) {
	return s.listLogs(ctx, "config_id IN (SELECT id FROM authsome_scim_configs WHERE app_id = ?)", appID, limit)
}

func (s *SQLStore) countByStatus(ctx context.Context, where, arg string) (success, errs, skipped int, err error) {
	logs, err := s.listLogs(ctx, where, arg, 0)
	if err != nil {
		return 0, 0, 0, err
	}
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
	return success, errs, skipped, nil
}

func (s *SQLStore) CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, errs, skipped int, err error) {
	return s.countByStatus(ctx, "config_id = ?", configID.String())
}

func (s *SQLStore) CountAllLogsByStatus(ctx context.Context, appID string) (success, errs, skipped int, err error) {
	return s.countByStatus(ctx, "config_id IN (SELECT id FROM authsome_scim_configs WHERE app_id = ?)", appID)
}

// ── driver adapters ──

type pgQuerier struct{ db *pgdriver.PgDB }

func (q pgQuerier) NewSelect(model ...any) sqlSelect { return pgSelect{q.db.NewSelect(model...)} }
func (q pgQuerier) NewInsert(model any) sqlExec      { return pgInsert{q.db.NewInsert(model)} }
func (q pgQuerier) NewUpdate(model any) sqlUpdate    { return pgUpdate{q.db.NewUpdate(model)} }
func (q pgQuerier) NewDelete(model any) sqlDelete    { return pgDelete{q.db.NewDelete(model)} }

type pgInsert struct{ q *pgdriver.InsertQuery }

func (i pgInsert) Exec(ctx context.Context) (sqlResult, error) { return i.q.Exec(ctx) }

type pgSelect struct{ q *pgdriver.SelectQuery }

func (s pgSelect) Where(query string, args ...any) sqlSelect {
	return pgSelect{s.q.Where(query, args...)}
}
func (s pgSelect) OrderExpr(expr string) sqlSelect { return pgSelect{s.q.OrderExpr(expr)} }
func (s pgSelect) Limit(n int) sqlSelect           { return pgSelect{s.q.Limit(n)} }
func (s pgSelect) Scan(ctx context.Context) error  { return s.q.Scan(ctx) }

type pgUpdate struct{ q *pgdriver.UpdateQuery }

func (u pgUpdate) Set(query string, args ...any) sqlUpdate { return pgUpdate{u.q.Set(query, args...)} }
func (u pgUpdate) Where(query string, args ...any) sqlUpdate {
	return pgUpdate{u.q.Where(query, args...)}
}
func (u pgUpdate) Exec(ctx context.Context) (sqlResult, error) { return u.q.Exec(ctx) }

type pgDelete struct{ q *pgdriver.DeleteQuery }

func (d pgDelete) Where(query string, args ...any) sqlDelete {
	return pgDelete{d.q.Where(query, args...)}
}
func (d pgDelete) Exec(ctx context.Context) (sqlResult, error) { return d.q.Exec(ctx) }

type sqliteQuerier struct{ db *sqlitedriver.SqliteDB }

func (q sqliteQuerier) NewSelect(model ...any) sqlSelect {
	return sqliteSelect{q.db.NewSelect(model...)}
}
func (q sqliteQuerier) NewInsert(model any) sqlExec { return sqliteInsert{q.db.NewInsert(model)} }

type sqliteInsert struct{ q *sqlitedriver.InsertQuery }

func (i sqliteInsert) Exec(ctx context.Context) (sqlResult, error) { return i.q.Exec(ctx) }
func (q sqliteQuerier) NewUpdate(model any) sqlUpdate              { return sqliteUpdate{q.db.NewUpdate(model)} }
func (q sqliteQuerier) NewDelete(model any) sqlDelete              { return sqliteDelete{q.db.NewDelete(model)} }

type sqliteSelect struct{ q *sqlitedriver.SelectQuery }

func (s sqliteSelect) Where(query string, args ...any) sqlSelect {
	return sqliteSelect{s.q.Where(query, args...)}
}
func (s sqliteSelect) OrderExpr(expr string) sqlSelect { return sqliteSelect{s.q.OrderExpr(expr)} }
func (s sqliteSelect) Limit(n int) sqlSelect           { return sqliteSelect{s.q.Limit(n)} }
func (s sqliteSelect) Scan(ctx context.Context) error  { return s.q.Scan(ctx) }

type sqliteUpdate struct{ q *sqlitedriver.UpdateQuery }

func (u sqliteUpdate) Set(query string, args ...any) sqlUpdate {
	return sqliteUpdate{u.q.Set(query, args...)}
}
func (u sqliteUpdate) Where(query string, args ...any) sqlUpdate {
	return sqliteUpdate{u.q.Where(query, args...)}
}
func (u sqliteUpdate) Exec(ctx context.Context) (sqlResult, error) { return u.q.Exec(ctx) }

type sqliteDelete struct{ q *sqlitedriver.DeleteQuery }

func (d sqliteDelete) Where(query string, args ...any) sqlDelete {
	return sqliteDelete{d.q.Where(query, args...)}
}
func (d sqliteDelete) Exec(ctx context.Context) (sqlResult, error) { return d.q.Exec(ctx) }
