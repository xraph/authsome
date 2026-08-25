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
	"fmt"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
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
	rawID := ctx.Param("clientId")
	if rawID == "" {
		return nil, forge.BadRequest("client ID required")
	}
	clientID, err := id.ParseOAuth2ClientID(rawID)
	if err != nil {
		return nil, forge.BadRequest("invalid client ID")
	}

	client, err := p.oauth2Store.GetClientByID(ctx.Context(), clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			return nil, forge.NotFound("oauth2 client not found")
		}
		return nil, forge.InternalError(fmt.Errorf("oauth2: load client: %w", err))
	}
	// Scope before doing anything else. Rotating another app's secret would
	// lock that app's client out, so this is a denial-of-service reachable by
	// guessing a primary key, not just an information leak.
	if scopeErr := plugin.AssertAppScope(ctx, client.AppID); scopeErr != nil {
		return nil, scopeErr
	}

	// A public client authenticates with PKCE, not a secret. Minting one here
	// would contradict both Public and TokenEndpointAuthMethod "none", and
	// nothing would ever read it.
	if client.Public {
		return nil, forge.BadRequest("a public client has no client_secret to rotate")
	}

	rawSecret, err := generateSecureToken(32)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_secret: %w", err))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: hash client_secret: %w", err))
	}
	client.ClientSecret = string(hash)

	if err := p.oauth2Store.UpdateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: rotate client secret: %w", err))
	}

	return &RotateClientSecretResponse{
		ID:           client.ID.String(),
		ClientID:     client.ClientID,
		ClientSecret: rawSecret,
	}, nil
}

// rejectImmutableClientFields refuses a request that tries to change one of
// the fields this endpoint deliberately does not own.
func rejectImmutableClientFields(req *UpdateClientRequest) error {
	switch {
	case req.BodyClientID != nil:
		return forge.BadRequest("client_id is immutable; it cannot be changed on an existing client")
	case req.BodyAppID != nil:
		return forge.BadRequest("app_id is immutable; a client cannot be moved between apps")
	case req.BodyPublic != nil:
		return forge.BadRequest("public is immutable here; changing the client type issues or destroys a secret, so use the client secret route")
	case req.BodyTokenEndpointAuthMethod != nil:
		return forge.BadRequest("token_endpoint_auth_method is immutable here; it is derived from the client type, which this route does not change")
	}
	return nil
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
	// Read the identifier from the path rather than the bound struct. See
	// UpdateClientRequest's doc comment: this is the second of two
	// independent protections against a body key redirecting the write at
	// some other client.
	rawID := ctx.Param("clientId")
	if rawID == "" {
		return nil, forge.BadRequest("client ID required")
	}
	clientID, err := id.ParseOAuth2ClientID(rawID)
	if err != nil {
		return nil, forge.BadRequest("invalid client ID")
	}

	// Refuse a malformed request before touching the store. Nothing about
	// this check depends on the stored client, so there is no reason to spend
	// a read discovering that the body was never acceptable.
	if fieldsErr := rejectImmutableClientFields(req); fieldsErr != nil {
		return nil, fieldsErr
	}

	client, err := p.oauth2Store.GetClientByID(ctx.Context(), clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			return nil, forge.NotFound("oauth2 client not found")
		}
		return nil, forge.InternalError(fmt.Errorf("oauth2: load client: %w", err))
	}
	// Same tenancy rule handleDeleteClient applies. A mismatch answers 404 so
	// the route cannot be used to probe for another app's clients.
	if err := plugin.AssertAppScope(ctx, client.AppID); err != nil {
		return nil, err
	}

	// Each field applies only when the caller sent it. Dereferencing an
	// explicitly-sent empty slice is what makes "clear this" expressible.
	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.RedirectURIs != nil {
		uris := *req.RedirectURIs
		// A confidential client with no redirect URI can never complete an
		// authorization code flow. handleCreateClient refuses to create one,
		// so the update path must not be a back door into that state.
		if len(uris) == 0 && !client.Public {
			return nil, forge.BadRequest("redirect_uris must not be empty for a confidential client")
		}
		for _, u := range uris {
			if uriErr := validateRedirectURI(u); uriErr != nil {
				return nil, forge.BadRequest(registrationErrorDescription(uriErr))
			}
		}
		client.RedirectURIs = uris
	}
	if req.Scopes != nil {
		// Deliberately not clampScopes. Silently dropping unknown scopes is
		// right for a client registering optimistically over RFC 7591; it is
		// wrong for an operator who typed a scope and expects to be told if
		// it did not take.
		client.Scopes = *req.Scopes
	}
	if req.GrantTypes != nil {
		for _, g := range *req.GrantTypes {
			if _, ok := adminGrantTypes[g]; !ok {
				return nil, forge.BadRequest(fmt.Sprintf("grant_type %q is not supported", g))
			}
		}
		client.GrantTypes = *req.GrantTypes
	}
	if req.Resources != nil {
		for _, r := range *req.Resources {
			if msg := resourceURISyntaxError(r); msg != "" {
				return nil, forge.BadRequest(msg)
			}
		}
		client.Resources = *req.Resources
	}

	// UpdateClient is a full-record replace on every backend, so the write
	// has to carry the whole client, not just the edited fields.
	if err := p.oauth2Store.UpdateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: update client: %w", err))
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
