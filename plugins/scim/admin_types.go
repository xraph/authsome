package scim

import "time"

// ---------------------------------------------------------------------------
// Admin API requests and responses
//
// These are the routes a dashboard calls to connect an IdP: create a config,
// mint a token to paste into the IdP, rotate it, revoke it. They are a
// separate surface from the SCIM 2.0 protocol in routes.go, which an IdP
// calls with one of these tokens, and they are mounted under /v1 rather than
// the SCIM base path so an IdP's bearer token can never reach them.
// ---------------------------------------------------------------------------

// ListSCIMConfigsRequest binds GET /orgs/:orgId/scim/configs.
type ListSCIMConfigsRequest struct {
	OrgID string `path:"orgId" description:"Organization ID"`
}

// SCIMConfigListResponse is a page of SCIM configurations.
type SCIMConfigListResponse struct {
	Configs []*SCIMConfig `json:"configs" description:"SCIM configurations for the organization"`
}

// CreateSCIMConfigRequest binds POST /orgs/:orgId/scim/configs.
type CreateSCIMConfigRequest struct {
	OrgID       string            `path:"orgId" description:"Organization ID"`
	Name        string            `json:"name" description:"A name for this connection, e.g. the IdP's name"`
	Enabled     *bool             `json:"enabled,omitempty" description:"Whether provisioning is accepted. Defaults to true"`
	AutoCreate  *bool             `json:"auto_create,omitempty" description:"Create users on first push"`
	AutoSuspend *bool             `json:"auto_suspend,omitempty" description:"Suspend users the IdP deactivates"`
	GroupSync   *bool             `json:"group_sync,omitempty" description:"Sync SCIM groups to teams"`
	DefaultRole string            `json:"default_role,omitempty" description:"Org role given to provisioned users"`
	Metadata    map[string]string `json:"metadata,omitempty" description:"Free-form metadata"`
}

// GetSCIMConfigRequest binds GET and DELETE /scim/configs/:configId.
type GetSCIMConfigRequest struct {
	ConfigID string `path:"configId" description:"SCIM configuration ID"`
}

// UpdateSCIMConfigRequest binds PATCH /scim/configs/:configId. Every field is
// a pointer so that leaving one out means "do not change it", which is what
// separates a PATCH from a PUT.
type UpdateSCIMConfigRequest struct {
	ConfigID    string            `path:"configId" description:"SCIM configuration ID"`
	Name        *string           `json:"name,omitempty" description:"New name"`
	Enabled     *bool             `json:"enabled,omitempty" description:"Turn provisioning on or off"`
	AutoCreate  *bool             `json:"auto_create,omitempty" description:"Create users on first push"`
	AutoSuspend *bool             `json:"auto_suspend,omitempty" description:"Suspend users the IdP deactivates"`
	GroupSync   *bool             `json:"group_sync,omitempty" description:"Sync SCIM groups to teams"`
	DefaultRole *string           `json:"default_role,omitempty" description:"Org role given to provisioned users"`
	Metadata    map[string]string `json:"metadata,omitempty" description:"Replaces the stored metadata"`
}

// ListSCIMTokensRequest binds GET /scim/configs/:configId/tokens.
type ListSCIMTokensRequest struct {
	ConfigID string `path:"configId" description:"SCIM configuration ID"`
}

// SCIMTokenListResponse is a config's tokens. The plaintext is never here:
// it exists once, in the response that created it.
type SCIMTokenListResponse struct {
	Tokens []*Token `json:"tokens" description:"Tokens for this configuration"`
}

// CreateSCIMTokenRequest binds POST /scim/configs/:configId/tokens.
type CreateSCIMTokenRequest struct {
	ConfigID  string     `path:"configId" description:"SCIM configuration ID"`
	Name      string     `json:"name" description:"A name for this token, e.g. which IdP holds it"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" description:"When it stops working. Omit for no expiry"`
}

// RotateSCIMTokenRequest binds POST /scim/tokens/:tokenId/rotate.
type RotateSCIMTokenRequest struct {
	TokenID string `path:"tokenId" description:"Token being replaced"`
	Name    string `json:"name,omitempty" description:"Name for the replacement"`
	// GraceSeconds keeps the old token working for a while after the new one
	// is minted, so an IdP can be updated without a window where provisioning
	// fails. Zero revokes the old one at once.
	GraceSeconds int `json:"grace_seconds,omitempty" description:"How long the replaced token keeps working, in seconds"`
}

// RevokeSCIMTokenRequest binds DELETE /scim/tokens/:tokenId.
type RevokeSCIMTokenRequest struct {
	TokenID string `path:"tokenId" description:"Token to revoke"`
}

// SCIMTokenCreatedResponse carries a freshly minted token.
//
// Plaintext appears here and nowhere else, ever: the store keeps a hash. A
// caller that loses it has to rotate. The base URL is included because it is
// the other half of what gets pasted into an IdP, and having to assemble it
// from documentation is how people paste the wrong one.
type SCIMTokenCreatedResponse struct {
	Token     *Token `json:"token" description:"The token record"`
	Plaintext string `json:"plaintext" description:"The bearer token. Shown once and never again"`
	BaseURL   string `json:"base_url" description:"SCIM 2.0 base URL to configure in the identity provider"`
}

// SCIMStatusResponse is the answer to a delete.
type SCIMStatusResponse struct {
	Status string `json:"status" description:"Outcome"`
}
