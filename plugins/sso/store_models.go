package sso

import (
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// SSO connection model (shared across SQL stores)
// ──────────────────────────────────────────────────

type ssoConnectionModel struct {
	grove.BaseModel `grove:"table:authsome_sso_connections,alias:sc"`

	ID           string    `grove:"id,pk"`
	AppID        string    `grove:"app_id,notnull"`
	OrgID        string    `grove:"org_id,notnull"`
	Provider     string    `grove:"provider,notnull"`
	Protocol     string    `grove:"protocol,notnull"`
	Domain       string    `grove:"domain,notnull"`
	MetadataURL  string    `grove:"metadata_url,notnull"`
	ClientID     string    `grove:"client_id,notnull"`
	ClientSecret string    `grove:"client_secret,notnull"`
	Issuer       string    `grove:"issuer,notnull"`
	Active       bool      `grove:"active,notnull"`
	CreatedAt    time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt    time.Time `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// SSO connection converters
// ──────────────────────────────────────────────────

func toConnection(m *ssoConnectionModel) (*Connection, error) {
	connID, err := id.ParseSSOConnectionID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	c := &Connection{
		ID:           connID,
		AppID:        appID,
		Provider:     m.Provider,
		Protocol:     m.Protocol,
		Domain:       m.Domain,
		MetadataURL:  m.MetadataURL,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Issuer:       m.Issuer,
		Active:       m.Active,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	if m.OrgID != "" {
		orgID, err := id.ParseOrgID(m.OrgID)
		if err != nil {
			return nil, err
		}
		c.OrgID = orgID
	}

	return c, nil
}

func fromConnection(c *Connection) *ssoConnectionModel {
	m := &ssoConnectionModel{
		ID:           c.ID.String(),
		AppID:        c.AppID.String(),
		Provider:     c.Provider,
		Protocol:     c.Protocol,
		Domain:       c.Domain,
		MetadataURL:  c.MetadataURL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Issuer:       c.Issuer,
		Active:       c.Active,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
	if c.OrgID.Prefix() != "" {
		m.OrgID = c.OrgID.String()
	}
	return m
}
