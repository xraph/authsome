package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"time"
)

// OAuthClient stores registered OAuth/OIDC clients
type OAuthClient struct {
	bun.BaseModel `bun:"table:oauth_clients"`

	ID        xid.ID `bun:",pk"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string // simplified; in practice support multiple URIs
}
