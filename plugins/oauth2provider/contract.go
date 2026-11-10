// contract.go: Wires the oauth2provider plugin into the forge-dashboard
// contract surface via plugin.ContractContributor.
package oauth2provider

import (
	"context"
	"errors"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	oauth2contract "github.com/xraph/authsome/plugins/oauth2provider/contract"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

var _ plugin.ContractContributor = (*Plugin)(nil)

func (p *Plugin) RegisterContract(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	engine plugin.Engine,
) error {
	eng, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("oauth2provider: contract registration requires *authsome.Engine, got %T", engine)
	}
	return oauth2contract.Register(d, reg, wreg, oauth2contract.Deps{Engine: eng, Plugin: contractService{p}})
}

// contractService adapts *Plugin to oauth2contract.OAuth2Service. The contract
// package cannot import this one, so it speaks its own types and sentinel
// errors, and this translates both ways.
type contractService struct{ p *Plugin }

var _ oauth2contract.OAuth2Service = contractService{}

func toContractErr(err error) error {
	var v *clientValidationError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &v):
		return &oauth2contract.ValidationError{Msg: v.msg}
	case errors.Is(err, ErrClientNotFound):
		return oauth2contract.ErrClientNotFound
	case errors.Is(err, ErrOperatorKey):
		return oauth2contract.ErrOperatorKey
	case errors.Is(err, ErrSharedKey):
		return oauth2contract.ErrSharedKey
	case errors.Is(err, ErrKeyTooNew):
		return oauth2contract.ErrKeyTooNew
	case errors.Is(err, errNoSigningKey):
		return oauth2contract.ErrNoSigningKey
	default:
		return err
	}
}

func toContractClient(c *OAuth2Client) oauth2contract.Client {
	return oauth2contract.Client{
		ID: c.ID.String(), ClientID: c.ClientID, Name: c.Name,
		Public: c.Public, FirstParty: c.FirstParty,
		RedirectURIs: c.RedirectURIs, Scopes: c.Scopes, GrantTypes: c.GrantTypes, Resources: c.Resources,
		AuthMethod: c.TokenEndpointAuthMethod, DPoPMode: c.DPoPMode,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		PostLogoutRedirectURIs: c.PostLogoutRedirectURIs,
	}
}

func toContractKey(k KeyInfo) oauth2contract.KeyInfo {
	return oauth2contract.KeyInfo{
		KID: k.KID, Alg: k.Alg, Source: k.Source, Status: k.Status,
		CreatedAt: k.CreatedAt, RetiresAt: k.RetiresAt,
	}
}

func (s contractService) CreateClientFor(ctx context.Context, appID id.AppID, in oauth2contract.CreateParams) (oauth2contract.Created, error) {
	r, err := s.p.CreateClientFor(ctx, appID, CreateClientRequest{
		Name: in.Name, Public: in.Public, FirstParty: in.FirstParty,
		RedirectURIs: in.RedirectURIs, Scopes: in.Scopes, GrantTypes: in.GrantTypes, Resources: in.Resources,
		PostLogoutRedirectURIs: in.PostLogoutRedirectURIs,
	})
	if err != nil {
		return oauth2contract.Created{}, toContractErr(err)
	}
	return oauth2contract.Created{ID: r.ID, ClientID: r.ClientID, ClientSecret: r.ClientSecret}, nil
}

func (s contractService) ListClientsFor(ctx context.Context, appID id.AppID) ([]oauth2contract.Client, error) {
	list, err := s.p.ListClientsFor(ctx, appID)
	if err != nil {
		return nil, toContractErr(err)
	}
	out := make([]oauth2contract.Client, 0, len(list))
	for _, c := range list {
		out = append(out, toContractClient(c))
	}
	return out, nil
}

func (s contractService) GetClientFor(ctx context.Context, appID id.AppID, clientID string) (oauth2contract.Client, error) {
	c, err := s.p.GetClientFor(ctx, appID, clientID)
	if err != nil {
		return oauth2contract.Client{}, toContractErr(err)
	}
	return toContractClient(c), nil
}

func (s contractService) UpdateClientFor(ctx context.Context, appID id.AppID, clientID string, in oauth2contract.UpdateParams) (oauth2contract.Client, error) {
	c, err := s.p.UpdateClientFor(ctx, appID, clientID, UpdateClientRequest{
		Name: in.Name, RedirectURIs: in.RedirectURIs, Scopes: in.Scopes,
		GrantTypes: in.GrantTypes, Resources: in.Resources,
		PostLogoutRedirectURIs: in.PostLogoutRedirectURIs,
	})
	if err != nil {
		return oauth2contract.Client{}, toContractErr(err)
	}
	return toContractClient(c), nil
}

func (s contractService) DeleteClientFor(ctx context.Context, appID id.AppID, clientID string) error {
	return toContractErr(s.p.DeleteClientFor(ctx, appID, clientID))
}

func (s contractService) RotateClientSecretFor(ctx context.Context, appID id.AppID, clientID string) (oauth2contract.RotatedClient, error) {
	c, secret, err := s.p.rotateClientSecret(ctx, appID, clientID)
	if err != nil {
		return oauth2contract.RotatedClient{}, toContractErr(err)
	}
	return oauth2contract.RotatedClient{ID: c.ID.String(), ClientID: c.ClientID, ClientSecret: secret}, nil
}

func (s contractService) SigningKeys(ctx context.Context, appID id.AppID) (string, []oauth2contract.KeyInfo, error) {
	issuer, keys, err := s.p.SigningKeys(ctx, appID)
	if err != nil {
		return "", nil, toContractErr(err)
	}
	out := make([]oauth2contract.KeyInfo, 0, len(keys))
	for _, k := range keys {
		out = append(out, toContractKey(k))
	}
	return issuer, out, nil
}

func (s contractService) RotateSigningKey(ctx context.Context, appID id.AppID) (oauth2contract.KeyInfo, error) {
	k, err := s.p.RotateSigningKey(ctx, appID)
	if err != nil {
		return oauth2contract.KeyInfo{}, toContractErr(err)
	}
	return toContractKey(k), nil
}

func (s contractService) IssuerInfo(ctx context.Context, appID id.AppID) (oauth2contract.IssuerDetails, error) {
	i, err := s.p.IssuerInfo(ctx, appID)
	if err != nil {
		return oauth2contract.IssuerDetails{}, toContractErr(err)
	}
	return oauth2contract.IssuerDetails{
		Issuer: i.Issuer, DiscoveryURL: i.DiscoveryURL, JWKSURI: i.JWKSURI, AppIssuer: i.AppIssuer, Algs: i.Algs,
	}, nil
}
