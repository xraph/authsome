package sso

import (
	"encoding/json"
	"fmt"
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
	DisplayName  string `grove:"display_name,notnull"`
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
	// AllowedDomains is a JSON array of domains; TrustedFederation adopts
	// the IdP subject as the local id on first creation.
	AllowedDomains    string `grove:"allowed_domains,notnull"`
	TrustedFederation bool   `grove:"trusted_federation,notnull"`

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
		DisplayName:  m.DisplayName,
		MetadataURL:  m.MetadataURL,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Issuer:       m.Issuer,
		Active:       m.Active,
		Enforced:     m.Enforced,

		TrustedFederation: m.TrustedFederation,

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

	if m.AllowedDomains != "" {
		if err := json.Unmarshal([]byte(m.AllowedDomains), &c.AllowedDomains); err != nil {
			return nil, fmt.Errorf("sso: decode allowed_domains: %w", err)
		}
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
		DisplayName:  c.DisplayName,
		MetadataURL:  c.MetadataURL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Issuer:       c.Issuer,
		Active:       c.Active,
		Enforced:     c.Enforced,

		TrustedFederation: c.TrustedFederation,

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
	if len(c.AllowedDomains) > 0 {
		if b, err := json.Marshal(c.AllowedDomains); err == nil {
			m.AllowedDomains = string(b)
		}
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

// ssoIdentityModel is one subject-to-user binding. The primary key is the
// connection id and subject joined, which is also the unique pair.
type ssoIdentityModel struct {
	grove.BaseModel `grove:"table:authsome_sso_identities,alias:si"`

	ID           string    `grove:"id,pk"`
	ConnectionID string    `grove:"connection_id,notnull"`
	Subject      string    `grove:"subject,notnull"`
	UserID       string    `grove:"user_id,notnull"`
	CreatedAt    time.Time `grove:"created_at,notnull,default:now()"`
}

func identityRowID(connID id.SSOConnectionID, subject string) string {
	return connID.String() + ":" + subject
}

func fromIdentity(i *Identity) *ssoIdentityModel {
	return &ssoIdentityModel{
		ID:           identityRowID(i.ConnectionID, i.Subject),
		ConnectionID: i.ConnectionID.String(),
		Subject:      i.Subject,
		UserID:       i.UserID.String(),
		CreatedAt:    i.CreatedAt,
	}
}

func toIdentity(m *ssoIdentityModel) (*Identity, error) {
	connID, err := id.ParseSSOConnectionID(m.ConnectionID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &Identity{ConnectionID: connID, Subject: m.Subject, UserID: userID, CreatedAt: m.CreatedAt}, nil
}
