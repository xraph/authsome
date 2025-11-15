package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// OAuthClient stores registered OAuth/OIDC clients
type OAuthClient struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:oauth_clients,alias:oc"`

	ID           xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID        xid.ID `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	Name         string `bun:"name,notnull" json:"name"`
	ClientID     string `bun:"client_id,notnull,unique" json:"clientID"`
	ClientSecret string `bun:"client_secret,notnull" json:"-"`
	RedirectURI  string `bun:"redirect_uri,notnull" json:"redirectURI"` // simplified; in practice support multiple URIs

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
