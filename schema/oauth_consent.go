package schema

import (
	"slices"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// OAuthConsent stores persistent user consent decisions for OAuth clients.
type OAuthConsent struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:oauth_consents,alias:oc"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"                  json:"id"`
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)"         json:"appID"`
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"        json:"organizationID,omitempty"` // Optional org context
	UserID         xid.ID  `bun:"user_id,notnull,type:varchar(20)"        json:"userID"`
	ClientID       string  `bun:"client_id,notnull"                       json:"clientID"`

	// Consent details
	Scopes    []string   `bun:"scopes,array,type:text[],notnull" json:"scopes"`              // Granted scopes
	ExpiresAt *time.Time `bun:"expires_at"                       json:"expiresAt,omitempty"` // Optional consent expiration

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	User         *User         `bun:"rel:belongs-to,join:user_id=id"`
}

// IsExpired checks if the consent has expired.
func (oc *OAuthConsent) IsExpired() bool {
	if oc.ExpiresAt == nil {
		return false
	}

	return time.Now().After(*oc.ExpiresAt)
}

// IsValid checks if the consent is still valid.
func (oc *OAuthConsent) IsValid() bool {
	return !oc.IsExpired()
}

// HasScope checks if a specific scope was granted.
func (oc *OAuthConsent) HasScope(scope string) bool {

	return slices.Contains(oc.Scopes, scope)
}
