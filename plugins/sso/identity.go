package sso

import (
	"strings"
	"time"

	"github.com/xraph/authsome/id"
)

// Identity binds an identity provider subject to a local user for one
// connection. A later login by the same subject lands on this user whatever
// email the provider asserts, and a login by a subject the connection has
// never seen falls back to email matching inside the connection's domains.
type Identity struct {
	ConnectionID id.SSOConnectionID `json:"connection_id"`
	Subject      string             `json:"subject"`
	UserID       id.UserID          `json:"user_id"`
	CreatedAt    time.Time          `json:"created_at"`
}

// AllowsEmailDomain reports whether an email the provider asserted may sign
// in through this connection: its domain must be the connection's own or one
// of the listed allowed domains. A connection with neither configured
// accepts any domain, which is the shape code-configured providers have.
func (c *Connection) AllowsEmailDomain(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	if c.Domain == "" && len(c.AllowedDomains) == 0 {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(c.Domain), domain) {
		return true
	}
	for _, d := range c.AllowedDomains {
		if strings.EqualFold(strings.TrimSpace(d), domain) {
			return true
		}
	}
	return false
}

// Connection represents a stored SSO connection for a tenant.
