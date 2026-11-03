// admin_clients.go: the admin surface for editing an OAuth2 client that
// already exists. Creation, listing and deletion live in plugin.go; this file
// holds the update path so plugin.go does not grow another handler.
//
// This is deliberately separate from the RFC 7592 management routes in
// register.go. Those authenticate the *client* with its registration access
// token and refuse anything that is not DynamicallyRegistered, which is every
// admin-created client. The two surfaces have disjoint callers and neither can
// stand in for the other.
package oauth2provider

import (
	"errors"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugin"
)

// UpdateClientRequest is the admin request to edit an existing OAuth2 client.
//
// Every mutable field is a pointer so the handler can distinguish "field
// absent, leave it alone" from "field present but empty, clear it". A plain
// []string cannot carry that difference: an omitted key and an explicit []
// both unmarshal to nil.
//
// The path-bound client id carries json:"-" so a "client_id" key in the body
// can never land on it, mirroring UpdateRegistrationRequest. The handler also
// reads ctx.Param directly rather than trusting this field, so there are two
// independent protections against the identifier being editable.
type UpdateClientRequest struct {
	ClientID string `path:"clientId" json:"-"`

	Name         *string   `json:"name,omitempty"`
	RedirectURIs *[]string `json:"redirect_uris,omitempty"`
	Scopes       *[]string `json:"scopes,omitempty"`
	GrantTypes   *[]string `json:"grant_types,omitempty"`
	Resources    *[]string `json:"resources,omitempty"`

	// These three are bound only so they can be refused. Leaving them out of
	// the struct would make encoding/json discard them silently, and a
	// caller who sends {"public": true} would get a 200 describing a change
	// that never happened. app_id is immutable because re-parenting a client
	// moves it across a tenancy boundary; public is immutable because
	// flipping it mints or destroys a secret, which belongs on the rotation
	// route; client_id is the value this endpoint exists to preserve.
	BodyClientID *string `json:"client_id,omitempty"`
	BodyAppID    *string `json:"app_id,omitempty"`
	BodyPublic   *bool   `json:"public,omitempty"`

	// Public and TokenEndpointAuthMethod encode one fact between them, and
	// models.go is explicit that nothing reads the latter for behaviour:
	// every write site derives one from the other instead. Accepting this
	// field on its own would let the two disagree, which is the bug that
	// comment exists to prevent.
	BodyTokenEndpointAuthMethod *string `json:"token_endpoint_auth_method,omitempty"`

	// FirstParty marks a client the operator owns; authorization skips the
	// consent page for it.
	FirstParty *bool `json:"first_party,omitempty"`
}

// UpdateClientResponse echoes the client as it stands after the edit. It
// carries no client_secret: this route never issues or rotates one.
type UpdateClientResponse struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"client_id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
	Resources    []string `json:"resources"`
	Public       bool     `json:"public"`
}

// adminGrantTypes is what an operator holding manage:oauth2_client may put on
// a client. It is deliberately wider than dynamicGrantTypes, which guards the
// open registration endpoint against anonymous callers helping themselves to
// client_credentials.
//
// The first three are what handleToken actually implements. refresh_token is
// included because clampGrantTypes already issues it to dynamically registered
// clients, so refusing it here would make an admin unable to edit grant_types
// on such a client without silently stripping a value the server itself
// granted. Note that handleToken has no refresh_token case, so the grant is
// registerable but not redeemable at the token endpoint; that predates this
// route and is not changed here.
var adminGrantTypes = map[string]struct{}{
	"authorization_code":                           {},
	"client_credentials":                           {},
	"urn:ietf:params:oauth:grant-type:device_code": {},
	"refresh_token":                                {},
}

// RotateClientSecretRequest identifies the client whose secret to replace.
// There is no body: the new secret is server-generated, the way it is at
// creation. Letting a caller supply one would invite weak or reused values.
type RotateClientSecretRequest struct {
	ClientID string `path:"clientId"`
}

// RotateClientSecretResponse carries the new secret in the clear. This is the
// only time it is ever readable, matching CreateClientResponse.
type RotateClientSecretResponse struct {
	ID           string `json:"id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// handleRotateClientSecret replaces a confidential client's secret in place.
//
// Separate from handleUpdateClient on purpose. This is the one admin route
// that emits a credential, so keeping it alone makes it straightforward to
// audit, rate-limit or gate behind a stronger permission later without
// dragging ordinary metadata edits along with it.
func (p *Plugin) handleRotateClientSecret(ctx forge.Context, _ *RotateClientSecretRequest) (*RotateClientSecretResponse, error) {
	appID, ok := plugin.CallerAppID(ctx)
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	client, secret, err := p.rotateClientSecret(ctx.Context(), appID, ctx.Param("clientId"))
	if err != nil {
		return nil, clientHTTPError(ctx, err)
	}
	return &RotateClientSecretResponse{
		ID:           client.ID.String(),
		ClientID:     client.ClientID,
		ClientSecret: secret,
	}, nil
}

// immutableClientFieldError names why req tries to change one of the fields
// the update path deliberately does not own, or returns "" when it does not.
func immutableClientFieldError(req *UpdateClientRequest) string {
	switch {
	case req.BodyClientID != nil:
		return "client_id is immutable; it cannot be changed on an existing client"
	case req.BodyAppID != nil:
		return "app_id is immutable; a client cannot be moved between apps"
	case req.BodyPublic != nil:
		return "public is immutable here; changing the client type issues or destroys a secret, so use the client secret route"
	case req.BodyTokenEndpointAuthMethod != nil:
		return "token_endpoint_auth_method is immutable here; it is derived from the client type, which this route does not change"
	}
	return ""
}

// registrationErrorDescription unwraps the RFC 7591 error shape used by the
// shared validators into a plain description, so the admin surface keeps the
// house error envelope instead of leaking the dynamic-registration body shape
// that MCP clients expect on /v1/oauth/register.
func registrationErrorDescription(err error) string {
	var regErr *oauthRegError
	if errors.As(err, &regErr) && regErr.Desc != "" {
		return regErr.Desc
	}
	return err.Error()
}

func (p *Plugin) handleUpdateClient(ctx forge.Context, req *UpdateClientRequest) (*UpdateClientResponse, error) {
	appID, ok := plugin.CallerAppID(ctx)
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	// Read the identifier from the path rather than the bound struct. See
	// UpdateClientRequest's doc comment: this is the second of two
	// independent protections against a body key redirecting the write at
	// some other client.
	client, err := p.UpdateClientFor(ctx.Context(), appID, ctx.Param("clientId"), *req)
	if err != nil {
		return nil, clientHTTPError(ctx, err)
	}
	return &UpdateClientResponse{
		ID:           client.ID.String(),
		ClientID:     client.ClientID,
		Name:         client.Name,
		RedirectURIs: client.RedirectURIs,
		Scopes:       client.Scopes,
		GrantTypes:   client.GrantTypes,
		Resources:    client.Resources,
		Public:       client.Public,
	}, nil
}
