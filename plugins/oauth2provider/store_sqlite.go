package oauth2provider

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements oauth2provider.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed OAuth2 store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{
		db:  db,
		sdb: sqlitedriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*SqliteStore)(nil)

// ──────────────────────────────────────────────────
// Client methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateClient(ctx context.Context, c *OAuth2Client) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromOAuth2Client(c)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return oauth2SqliteError(err)
}

func (s *SqliteStore) GetClient(ctx context.Context, clientID string) (*OAuth2Client, error) {
	m := new(oauth2ClientModel)
	err := s.sdb.NewSelect(m).
		Where("client_id = ?", clientID).
		Scan(ctx)
	if err != nil {
		return nil, oauth2SqliteError(err)
	}
	return toOAuth2Client(m)
}

func (s *SqliteStore) GetClientByID(ctx context.Context, clientID id.OAuth2ClientID) (*OAuth2Client, error) {
	m := new(oauth2ClientModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", clientID.String()).
		Scan(ctx)
	if err != nil {
		return nil, oauth2SqliteError(err)
	}
	return toOAuth2Client(m)
}

func (s *SqliteStore) ListClients(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	var models []oauth2ClientModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, oauth2SqliteError(err)
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

func (s *SqliteStore) DeleteClient(ctx context.Context, clientID id.OAuth2ClientID) error {
	_, err := s.sdb.NewDelete((*oauth2ClientModel)(nil)).
		Where("id = ?", clientID.String()).
		Exec(ctx)
	return oauth2SqliteError(err)
}

// ──────────────────────────────────────────────────
// Auth code methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateAuthCode(ctx context.Context, code *AuthorizationCode) error {
	now := time.Now()
	if code.CreatedAt.IsZero() {
		code.CreatedAt = now
	}
	m := fromAuthCode(code)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return oauth2SqliteError(err)
}

func (s *SqliteStore) GetAuthCode(ctx context.Context, code string) (*AuthorizationCode, error) {
	m := new(authCodeModel)
	err := s.sdb.NewSelect(m).
		Where("code = ?", code).
		Scan(ctx)
	if err != nil {
		return nil, oauth2SqliteError(err)
	}
	return toAuthCode(m)
}

func (s *SqliteStore) ConsumeAuthCode(ctx context.Context, code string) error {
	_, err := s.sdb.NewUpdate((*authCodeModel)(nil)).
		Set("consumed = ?", true).
		Where("code = ?", code).
		Exec(ctx)
	return oauth2SqliteError(err)
}

// ──────────────────────────────────────────────────
// Device code methods (RFC 8628)
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	if dc.CreatedAt.IsZero() {
		dc.CreatedAt = time.Now()
	}
	m := fromDeviceCode(dc)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return oauth2SqliteError(err)
}

func (s *SqliteStore) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCode, error) {
	m := new(deviceCodeModel)
	err := s.sdb.NewSelect(m).
		Where("device_code = ?", deviceCode).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2SqliteError(err)
	}
	return toDeviceCode(m)
}

func (s *SqliteStore) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCode, error) {
	m := new(deviceCodeModel)
	err := s.sdb.NewSelect(m).
		Where("user_code = ?", userCode).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDeviceCodeNotFound
		}
		return nil, oauth2SqliteError(err)
	}
	return toDeviceCode(m)
}

func (s *SqliteStore) UpdateDeviceCode(ctx context.Context, dc *DeviceCode) error {
	m := fromDeviceCode(dc)
	_, err := s.sdb.NewUpdate(m).
		WherePK().
		Exec(ctx)
	return oauth2SqliteError(err)
}

func (s *SqliteStore) DeleteExpiredDeviceCodes(ctx context.Context) error {
	_, err := s.sdb.NewDelete((*deviceCodeModel)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	return oauth2SqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func oauth2SqliteError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrClientNotFound
	}
	return err
}
