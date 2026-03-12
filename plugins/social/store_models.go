package social

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// OAuth connection model (shared across SQL stores)
// ──────────────────────────────────────────────────

type oauthConnectionModel struct {
	grove.BaseModel `grove:"table:authsome_oauth_connections,alias:oc"`

	ID             string          `grove:"id,pk"`
	AppID          string          `grove:"app_id,notnull"`
	UserID         string          `grove:"user_id,notnull"`
	Provider       string          `grove:"provider,notnull"`
	ProviderUserID string          `grove:"provider_user_id,notnull"`
	Email          string          `grove:"email,notnull"`
	AccessToken    string          `grove:"access_token,notnull"`
	RefreshToken   string          `grove:"refresh_token,notnull"`
	ExpiresAt      sql.NullTime    `grove:"expires_at"`
	Metadata       json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedAt      time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt      time.Time       `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// OAuth connection converters
// ──────────────────────────────────────────────────

func toOAuthConnection(m *oauthConnectionModel) (*OAuthConnection, error) {
	connID, err := id.ParseOAuthConnectionID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}

	md := make(map[string]string)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}

	c := &OAuthConnection{
		ID:             connID,
		AppID:          appID,
		UserID:         userID,
		Provider:       m.Provider,
		ProviderUserID: m.ProviderUserID,
		Email:          m.Email,
		AccessToken:    m.AccessToken,
		RefreshToken:   m.RefreshToken,
		Metadata:       md,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	if m.ExpiresAt.Valid {
		c.ExpiresAt = m.ExpiresAt.Time
	}
	return c, nil
}

func fromOAuthConnection(c *OAuthConnection) *oauthConnectionModel {
	md, _ := json.Marshal(c.Metadata) //nolint:errcheck // marshaling known types
	if len(md) == 0 {
		md = []byte("{}")
	}

	m := &oauthConnectionModel{
		ID:             c.ID.String(),
		AppID:          c.AppID.String(),
		UserID:         c.UserID.String(),
		Provider:       c.Provider,
		ProviderUserID: c.ProviderUserID,
		Email:          c.Email,
		AccessToken:    c.AccessToken,
		RefreshToken:   c.RefreshToken,
		Metadata:       md,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
	if !c.ExpiresAt.IsZero() {
		m.ExpiresAt = sql.NullTime{Time: c.ExpiresAt, Valid: true}
	}
	return m
}
