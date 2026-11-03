// handlers.go: OAuth2 client, issuer and signing key intent handlers.
//
// Each handler resolves the caller's app, calls the plugin and projects the
// answer onto a wire type. No wire type carries a secret hash. Only create and
// rotateSecret carry a plaintext secret, once, and both register with
// dispatcher.SecretResponse so the dispatcher never replays them.
package contract

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xraph/forge/extensions/dashboard/contract"

	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/authsome/middleware"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

// ClientSummary is one client in a list. ID addresses the client in inputs;
// ClientID is the public OAuth client_id a person sees.
type ClientSummary struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"clientId"`
	Name         string   `json:"name"`
	Public       bool     `json:"public"`
	FirstParty   bool     `json:"firstParty"`
	RedirectURIs []string `json:"redirectUris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grantTypes"`
	CreatedAt    string   `json:"createdAt"`
}

type ClientListResponse struct {
	Clients []ClientSummary `json:"clients"`
}

type ClientDetail struct {
	ClientSummary
	Resources  []string `json:"resources"`
	AuthMethod string   `json:"authMethod"`
	DPoPMode   string   `json:"dpopMode"`
	UpdatedAt  string   `json:"updatedAt"`
}

type ClientIDInput struct {
	ID string `json:"id"`
}

type CreateClientInput struct {
	Name         string   `json:"name"`
	Public       bool     `json:"public"`
	FirstParty   bool     `json:"firstParty"`
	RedirectURIs []string `json:"redirectUris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grantTypes"`
	Resources    []string `json:"resources"`
}

// UpdateClientInput changes the fields it carries. A nil field is left alone.
type UpdateClientInput struct {
	ID           string    `json:"id"`
	Name         *string   `json:"name,omitempty"`
	RedirectURIs *[]string `json:"redirectUris,omitempty"`
	Scopes       *[]string `json:"scopes,omitempty"`
	GrantTypes   *[]string `json:"grantTypes,omitempty"`
	Resources    *[]string `json:"resources,omitempty"`
}

// CreatedClient carries the plaintext secret of a confidential client, once.
type CreatedClient struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

type RotatedSecret struct {
	ID           string `json:"id"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type IssuerInfoResponse struct {
	Issuer       string   `json:"issuer"`
	DiscoveryURL string   `json:"discoveryUrl"`
	JWKSURI      string   `json:"jwksUri"`
	AppIssuer    bool     `json:"appIssuer"`
	Algs         []string `json:"algs"`
}

type KeySummary struct {
	KID       string `json:"kid"`
	Alg       string `json:"alg"`
	Source    string `json:"source"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	RetiresAt string `json:"retiresAt,omitempty"`
}

type KeyListResponse struct {
	Issuer string       `json:"issuer"`
	Keys   []KeySummary `json:"keys"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func clientsListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (ClientListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (ClientListResponse, error) {
		if deps.Plugin == nil {
			return ClientListResponse{}, unavailable()
		}
		list, err := deps.Plugin.ListClientsFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return ClientListResponse{}, mapErr(err)
		}
		out := ClientListResponse{Clients: make([]ClientSummary, 0, len(list))}
		for _, c := range list {
			out.Clients = append(out.Clients, projectSummary(c))
		}
		return out, nil
	}
}

func clientsDetailHandler(deps Deps) func(context.Context, ClientIDInput, contract.Principal) (ClientDetail, error) {
	return func(ctx context.Context, in ClientIDInput, p contract.Principal) (ClientDetail, error) {
		if deps.Plugin == nil {
			return ClientDetail{}, unavailable()
		}
		c, err := deps.Plugin.GetClientFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine), in.ID)
		if err != nil {
			return ClientDetail{}, mapErr(err)
		}
		return projectDetail(c), nil
	}
}

func clientsCreateHandler(deps Deps) func(context.Context, CreateClientInput, contract.Principal) (CreatedClient, error) {
	return func(ctx context.Context, in CreateClientInput, p contract.Principal) (CreatedClient, error) {
		if refuseDelegated(ctx) {
			return CreatedClient{}, delegatedRefusal()
		}
		if deps.Plugin == nil {
			return CreatedClient{}, unavailable()
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return CreatedClient{}, badReq("name is required")
		}
		res, err := deps.Plugin.CreateClientFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine), CreateParams{
			Name: name, Public: in.Public, FirstParty: in.FirstParty,
			RedirectURIs: in.RedirectURIs, Scopes: in.Scopes,
			GrantTypes: in.GrantTypes, Resources: in.Resources,
		})
		if err != nil {
			return CreatedClient{}, mapErr(err)
		}
		return CreatedClient(res), nil
	}
}

func clientsUpdateHandler(deps Deps) func(context.Context, UpdateClientInput, contract.Principal) (ClientDetail, error) {
	return func(ctx context.Context, in UpdateClientInput, p contract.Principal) (ClientDetail, error) {
		if refuseDelegated(ctx) {
			return ClientDetail{}, delegatedRefusal()
		}
		if deps.Plugin == nil {
			return ClientDetail{}, unavailable()
		}
		c, err := deps.Plugin.UpdateClientFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine), in.ID, UpdateParams{
			Name: in.Name, RedirectURIs: in.RedirectURIs, Scopes: in.Scopes,
			GrantTypes: in.GrantTypes, Resources: in.Resources,
		})
		if err != nil {
			return ClientDetail{}, mapErr(err)
		}
		return projectDetail(c), nil
	}
}

func clientsDeleteHandler(deps Deps) func(context.Context, ClientIDInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in ClientIDInput, p contract.Principal) (ackResponse, error) {
		if refuseDelegated(ctx) {
			return ackResponse{}, delegatedRefusal()
		}
		if deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		if err := deps.Plugin.DeleteClientFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine), in.ID); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: in.ID}, nil
	}
}

func clientsRotateSecretHandler(deps Deps) func(context.Context, ClientIDInput, contract.Principal) (RotatedSecret, error) {
	return func(ctx context.Context, in ClientIDInput, p contract.Principal) (RotatedSecret, error) {
		if refuseDelegated(ctx) {
			return RotatedSecret{}, delegatedRefusal()
		}
		if deps.Plugin == nil {
			return RotatedSecret{}, unavailable()
		}
		r, err := deps.Plugin.RotateClientSecretFor(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine), in.ID)
		if err != nil {
			return RotatedSecret{}, mapErr(err)
		}
		return RotatedSecret(r), nil
	}
}

func issuerInfoHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (IssuerInfoResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (IssuerInfoResponse, error) {
		if deps.Plugin == nil {
			return IssuerInfoResponse{}, unavailable()
		}
		i, err := deps.Plugin.IssuerInfo(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return IssuerInfoResponse{}, mapErr(err)
		}
		algs := i.Algs
		if algs == nil {
			algs = []string{}
		}
		return IssuerInfoResponse{
			Issuer: i.Issuer, DiscoveryURL: i.DiscoveryURL, JWKSURI: i.JWKSURI,
			AppIssuer: i.AppIssuer, Algs: algs,
		}, nil
	}
}

func keysListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (KeyListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (KeyListResponse, error) {
		if deps.Plugin == nil {
			return KeyListResponse{}, unavailable()
		}
		issuer, keys, err := deps.Plugin.SigningKeys(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return KeyListResponse{}, mapErr(err)
		}
		return projectKeys(issuer, keys), nil
	}
}

// keysRotateHandler rotates the signing key and answers with the key list as
// it stands afterwards, so the dashboard refreshes in one round trip. If the
// rotation succeeded but the list could not be read, it answers with the new
// key alone (with the issuer when that can be read) rather than an error,
// because the rotation cannot be retried harmlessly.
func keysRotateHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (KeyListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (KeyListResponse, error) {
		if refuseDelegated(ctx) {
			return KeyListResponse{}, delegatedRefusal()
		}
		if deps.Plugin == nil {
			return KeyListResponse{}, unavailable()
		}
		appID := authcontract.AppIDFromPrincipal(p, deps.Engine)
		rotated, err := deps.Plugin.RotateSigningKey(ctx, appID)
		if err != nil {
			return KeyListResponse{}, mapErr(err)
		}
		issuer, keys, err := deps.Plugin.SigningKeys(ctx, appID)
		if err != nil {
			// Best effort: the issuer is a nicety, the rotation already happened.
			var iss string
			if info, infoErr := deps.Plugin.IssuerInfo(ctx, appID); infoErr == nil {
				iss = info.Issuer
			}
			return projectKeys(iss, []KeyInfo{rotated}), nil
		}
		return projectKeys(issuer, keys), nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func projectSummary(c Client) ClientSummary {
	return ClientSummary{
		ID: c.ID, ClientID: c.ClientID, Name: c.Name,
		Public: c.Public, FirstParty: c.FirstParty,
		RedirectURIs: nonNil(c.RedirectURIs), Scopes: nonNil(c.Scopes), GrantTypes: nonNil(c.GrantTypes),
		CreatedAt: rfc3339(c.CreatedAt),
	}
}

func projectDetail(c Client) ClientDetail {
	return ClientDetail{
		ClientSummary: projectSummary(c),
		Resources:     nonNil(c.Resources),
		AuthMethod:    c.AuthMethod,
		DPoPMode:      c.DPoPMode,
		UpdatedAt:     rfc3339(c.UpdatedAt),
	}
}

func projectKeys(issuer string, keys []KeyInfo) KeyListResponse {
	out := KeyListResponse{Issuer: issuer, Keys: make([]KeySummary, 0, len(keys))}
	for _, k := range keys {
		s := KeySummary{KID: k.KID, Alg: k.Alg, Source: k.Source, Status: k.Status, CreatedAt: rfc3339(k.CreatedAt)}
		if k.RetiresAt != nil {
			s.RetiresAt = rfc3339(*k.RetiresAt)
		}
		out.Keys = append(out.Keys, s)
	}
	return out
}

// refuseDelegated is true for an agent or delegated session. The dashboard's
// auth check turns those away already; this keeps a dashboard wired to another
// checker from letting one change clients or keys.
func refuseDelegated(ctx context.Context) bool { return middleware.RefuseDelegated(ctx) != nil }

func delegatedRefusal() error {
	return &contract.Error{Code: contract.CodePermissionDenied, Message: middleware.DelegatedSessionMessage}
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "oauth2 provider not configured"}
}

// mapErr turns a service error into a contract error. Anything it does not
// recognise becomes a generic internal error: the cause can name a store or a
// key, and none of that belongs on the wire.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var ce *contract.Error
	if errors.As(err, &ce) {
		return ce
	}
	var v *ValidationError
	switch {
	case errors.As(err, &v):
		return badReq(v.Msg)
	case errors.Is(err, ErrClientNotFound):
		return &contract.Error{Code: contract.CodeNotFound, Message: "oauth2 client not found"}
	case errors.Is(err, ErrOperatorKey):
		return badReq("the signing key for this issuer is managed by configuration")
	case errors.Is(err, ErrSharedKey):
		return badReq("the deployment issuer's signing key is shared by every app and rotates on its own schedule")
	case errors.Is(err, ErrKeyTooNew):
		return badReq("the current signing key is less than an hour old; try again later")
	case errors.Is(err, ErrNoSigningKey):
		return &contract.Error{Code: contract.CodeUnavailable, Message: "no id_token signing key is available; configure an encryption key or an operator key"}
	default:
		return &contract.Error{Code: contract.CodeInternal, Message: "internal error"}
	}
}
