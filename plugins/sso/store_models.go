package sso

import (
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// SSO connection model (shared across SQL stores)
// ──────────────────────────────────────────────────

type ssoConnectionModel struct {
	grove.BaseModel `grove:"table:authsome_sso_connections,alias:sc"`

	ID           string `grove:"id,pk"`
	AppID        string `grove:"app_id,notnull"`
	EnvID        string `grove:"env_id,notnull"`
	OrgID        string `grove:"org_id,notnull"`
	Provider     string `grove:"provider,notnull"`
	Protocol     string `grove:"protocol,notnull"`
	Domain       string `grove:"domain,notnull"`
	MetadataURL  string `grove:"metadata_url,notnull"`
	ClientID     string `grove:"client_id,notnull"`
	ClientSecret string `grove:"client_secret,notnull"`
	Issuer       string `grove:"issuer,notnull"`
	Active       bool   `grove:"active,notnull"`
	Enforced     bool   `grove:"enforced,notnull"`

	// SAML fields. attribute_mappings is stored as a JSON object.
	IDPMetadataXML    string `grove:"idp_metadata_xml,notnull"`
	IDPSSOURL         string `grove:"idp_sso_url,notnull"`
	IDPCertificate    string `grove:"idp_certificate,notnull"`
	EntityID          string `grove:"entity_id,notnull"`
	ACSURL            string `grove:"acs_url,notnull"`
	SPCertificate     string `grove:"sp_certificate,notnull"`
	SPPrivateKey      string `grove:"sp_private_key,notnull"`
	SignRequests      bool   `grove:"sign_requests,notnull"`
	AttributeMappings string `grove:"attribute_mappings,notnull"`

	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `grove:"updated_at,notnull,default:now()"`
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
		EnvID:        m.EnvID,
		Provider:     m.Provider,
		Protocol:     m.Protocol,
		Domain:       m.Domain,
		MetadataURL:  m.MetadataURL,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Issuer:       m.Issuer,
		Active:       m.Active,
		Enforced:     m.Enforced,

		IDPMetadataXML: m.IDPMetadataXML,
		IDPSSOURL:      m.IDPSSOURL,
		IDPCertificate: m.IDPCertificate,
		EntityID:       m.EntityID,
		ACSURL:         m.ACSURL,
		SPCertificate:  m.SPCertificate,
		SPPrivateKey:   m.SPPrivateKey,
		SignRequests:   m.SignRequests,

		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.AttributeMappings != "" {
		if err := json.Unmarshal([]byte(m.AttributeMappings), &c.AttributeMappings); err != nil {
			return nil, err
		}
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
		EnvID:        c.EnvID,
		Provider:     c.Provider,
		Protocol:     c.Protocol,
		Domain:       c.Domain,
		MetadataURL:  c.MetadataURL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Issuer:       c.Issuer,
		Active:       c.Active,
		Enforced:     c.Enforced,

		IDPMetadataXML: c.IDPMetadataXML,
		IDPSSOURL:      c.IDPSSOURL,
		IDPCertificate: c.IDPCertificate,
		EntityID:       c.EntityID,
		ACSURL:         c.ACSURL,
		SPCertificate:  c.SPCertificate,
		SPPrivateKey:   c.SPPrivateKey,
		SignRequests:   c.SignRequests,

		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	if len(c.AttributeMappings) > 0 {
		if b, err := json.Marshal(c.AttributeMappings); err == nil {
			m.AttributeMappings = string(b)
		}
	}
	if c.OrgID.Prefix() != "" {
		m.OrgID = c.OrgID.String()
	}
	return m
}
