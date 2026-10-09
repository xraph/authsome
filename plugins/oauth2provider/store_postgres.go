package oauth2provider

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// PostgresStore implements oauth2provider.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed OAuth2 store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
		pg: pgdriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*PostgresStore)(nil)

// ──────────────────────────────────────────────────
// Client methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateClient(ctx context.Context, c *OAuth2Client) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromOAuth2Client(c)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return oauth2PgError(err)
}

func (s *PostgresStore) GetClient(ctx context.Context, clientID string) (*OAuth2Client, error) {
	m := new(oauth2ClientModel)
	err := s.pg.NewSelect(m).
		Where("client_id = ?", clientID).
		Scan(ctx)
	if err != nil {
		return nil, oauth2PgError(err)
	}
	return toOAuth2Client(m)
}

func (s *PostgresStore) GetClientByID(ctx context.Context, clientID id.OAuth2ClientID) (*OAuth2Client, error) {
	m := new(oauth2ClientModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", clientID.String()).
		Scan(ctx)
	if err != nil {
		return nil, oauth2PgError(err)
	}
	return toOAuth2Client(m)
}

func (s *PostgresStore) UpdateClient(ctx context.Context, c *OAuth2Client) error {
	c.UpdatedAt = time.Now()
	m := fromOAuth2Client(c)
	res, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return oauth2PgError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return oauth2PgError(err)
	}
	if n == 0 {
		return ErrClientNotFound
	}
	return nil
}

func (s *PostgresStore) ListClients(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	var models []oauth2ClientModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, oauth2PgError(err)
	}
	result := make([]*OAuth2Client, 0, len(models))
	for i := range models {
		c, err := toOAuth2Client(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *PostgresStore) DeleteClient(ctx context.Context, clientID id.OAuth2ClientID) error {
	_, err := s.pg.NewDelete((*oauth2ClientModel)(nil)).
		Where("id = ?", clientID.String()).
		Exec(ctx)
	return oauth2PgError(err)
}

// ──────────────────────────────────────────────────
// Auth code methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateAuthCode(ctx context.Context, code *AuthorizationCode) error {
	now := time.Now()
	if code.CreatedAt.IsZero() {
		code.CreatedAt = now
	}
	m := fromAuthCode(code)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return oauth2PgError(err)
}

func (s *PostgresStore) GetAuthCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	m := new(authCodeModel)
	err := s.pg.NewSelect(m).
		Where("code = ?", store.HashToken(code)).
		Scan(ctx)
	if err != nil {
		return nil, oauth2PgError(err)
	}
	ac, err := toAuthCode(m)
	if err != nil {
		return nil, err
	}
	ac.Code = code
	return ac, nil
}

func (s *PostgresStore) ConsumeAuthCode(ctx context.Context, code string) (bool, error) {
	res, err := s.pg.NewUpdate((*authCodeModel)(nil)).
		Set("consumed = ?", true).
		Where("code = ?", store.HashToken(code)).
		Where("consumed = ?", false).
		Exec(ctx)
	if err != nil {
		return false, oauth2PgError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, oauth2PgError(err)
	}
	return n > 0, nil
}

// ──────────────────────────────────────────────────
// Device code methods (RFC 8628)
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	if dc.CreatedAt.IsZero() {
		dc.CreatedAt = time.Now()
	}
	m := fromDeviceCode(dc)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return oauth2PgError(err)
}

func (s *PostgresStore) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCode, error) {
	m := new(deviceCodeModel)
	err := s.pg.NewSelect(m).
		Where("device_code = ?", store.HashToken(deviceCode)).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2PgError(err)
	}
	dc, err := toDeviceCode(m)
	if err != nil {
		return nil, err
	}
	dc.DeviceCode, dc.UserCode = deviceCode, ""
	return dc, nil
}

func (s *PostgresStore) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCode, error) {
	m := new(deviceCodeModel)
	err := s.pg.NewSelect(m).
		Where("user_code = ?", store.HashToken(userCode)).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2PgError(err)
	}
	dc, err := toDeviceCode(m)
	if err != nil {
		return nil, err
	}
	dc.UserCode, dc.DeviceCode = userCode, ""
	return dc, nil
}

func (s *PostgresStore) UpdateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	// Only the fields the flow mutates. dc may carry a plaintext code from
	// the lookup that found it, and rewriting the code columns from it would
	// hash that plaintext a second time and orphan the row.
	_, err := s.pg.NewUpdate((*deviceCodeModel)(nil)).
		Set("status = ?", dc.Status).
		Set("user_id = ?", dc.UserID.String()).
		Set("last_polled_at = ?", dc.LastPolledAt).
		Where("id = ?", dc.ID.String()).
		Exec(ctx)
	return oauth2PgError(err)
}

func (s *PostgresStore) DeleteExpiredDeviceCodes(ctx context.Context) error {
	_, err := s.pg.NewDelete((*deviceCodeModel)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	return oauth2PgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func oauth2PgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrClientNotFound
	}
	return err
}

// ──────────────────────────────────────────────────
// Grants
// ──────────────────────────────────────────────────

func (s *PostgresStore) UpsertGrant(ctx context.Context, g *Grant) error {
	now := time.Now()
	existing, err := s.GetGrant(ctx, g.AppID, g.UserID, g.ClientID)
	switch {
	case err == nil:
		g.ID = existing.ID
		g.CreatedAt = existing.CreatedAt
		g.UpdatedAt = now
		m := fromGrant(g)
		_, err = s.pg.NewUpdate((*grantModel)(nil)).
			Set("scopes = ?", m.Scopes).
			Set("updated_at = ?", m.UpdatedAt).
			Where("id = ?", m.ID).
			Exec(ctx)
		return oauth2PgError(err)
	case errors.Is(err, ErrGrantNotFound):
		if g.ID.IsNil() {
			g.ID = id.NewOAuth2GrantID()
		}
		if g.CreatedAt.IsZero() {
			g.CreatedAt = now
		}
		g.UpdatedAt = now
		_, err = s.pg.NewInsert(fromGrant(g)).Exec(ctx)
		return oauth2PgError(err)
	default:
		return err
	}
}

func (s *PostgresStore) GetGrant(ctx context.Context, appID id.AppID, userID id.UserID, clientID string) (*Grant, error) {
	m := new(grantModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Where("client_id = ?", clientID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGrantNotFound
		}
		return nil, oauth2PgError(err)
	}
	return toGrant(m)
}

func (s *PostgresStore) ListGrantsByUser(ctx context.Context, appID id.AppID, userID id.UserID) ([]*Grant, error) {
	var models []grantModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, oauth2PgError(err)
	}
	out := make([]*Grant, 0, len(models))
	for i := range models {
		g, convErr := toGrant(&models[i])
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, g)
	}
	return out, nil
}

func (s *PostgresStore) DeleteGrant(ctx context.Context, appID id.AppID, userID id.UserID, clientID string) error {
	res, err := s.pg.NewDelete((*grantModel)(nil)).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Where("client_id = ?", clientID).
		Exec(ctx)
	if err != nil {
		return oauth2PgError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return oauth2PgError(err)
	}
	if n == 0 {
		return ErrGrantNotFound
	}
	return nil
}
