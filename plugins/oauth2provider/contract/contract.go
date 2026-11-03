// Package contract wires the oauth2provider plugin into the forge-dashboard
// contract registry: the nine intents that manage OAuth2 clients, the issuer
// an app's id_tokens name, and that issuer's signing keys.
//
// The plugin package imports this one, so this one cannot import the plugin.
// It declares the service it needs as OAuth2Service, in its own types, and
// the plugin passes an adapter over itself.
package contract

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"time"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// Client is one OAuth2 client as the dashboard sees it. It carries no secret.
type Client struct {
	ID           string
	ClientID     string
	Name         string
	Public       bool
	FirstParty   bool
	RedirectURIs []string
	Scopes       []string
	GrantTypes   []string
	Resources    []string
	AuthMethod   string
	DPoPMode     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateParams is what a new client is registered with.
type CreateParams struct {
	Name         string
	Public       bool
	FirstParty   bool
	RedirectURIs []string
	Scopes       []string
	GrantTypes   []string
	Resources    []string
}

// UpdateParams carries the fields to change. A nil field is left alone.
type UpdateParams struct {
	Name         *string
	RedirectURIs *[]string
	Scopes       *[]string
	GrantTypes   *[]string
	Resources    *[]string
}

// Created is a freshly registered client. ClientSecret is the plaintext, set
// only for a confidential client.
type Created struct {
	ID           string
	ClientID     string
	ClientSecret string
}

// RotatedClient is a client whose secret was just replaced.
type RotatedClient struct {
	ID           string
	ClientID     string
	ClientSecret string
}

// IssuerDetails describes the issuer an app's id_tokens name.
type IssuerDetails struct {
	Issuer       string
	DiscoveryURL string
	JWKSURI      string
	AppIssuer    bool
	Algs         []string
}

// KeyInfo is one id_token signing key.
type KeyInfo struct {
	KID       string
	Alg       string
	Source    string
	Status    string
	CreatedAt time.Time
	RetiresAt *time.Time
}

// ErrClientNotFound, ErrOperatorKey, ErrSharedKey, ErrKeyTooNew and
// ErrNoSigningKey are what an OAuth2Service answers for the failures the
// handlers name. A request the service refused is a *ValidationError.
var (
	ErrClientNotFound = fmt.Errorf("oauth2 client not found")
	ErrOperatorKey    = fmt.Errorf("signing key is managed by configuration")
	ErrSharedKey      = fmt.Errorf("signing key is shared by every app")
	ErrKeyTooNew      = fmt.Errorf("signing key is too new to rotate")
	ErrNoSigningKey   = fmt.Errorf("no id_token signing key available")
)

// ValidationError is a request the service refused. Its message is safe to
// show the operator.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// OAuth2Service is the surface of the oauth2provider plugin the handlers need.
// Every method acts for one app and treats another app's client as not found.
type OAuth2Service interface {
	CreateClientFor(ctx context.Context, appID id.AppID, in CreateParams) (Created, error)
	ListClientsFor(ctx context.Context, appID id.AppID) ([]Client, error)
	GetClientFor(ctx context.Context, appID id.AppID, clientID string) (Client, error)
	UpdateClientFor(ctx context.Context, appID id.AppID, clientID string, in UpdateParams) (Client, error)
	DeleteClientFor(ctx context.Context, appID id.AppID, clientID string) error
	RotateClientSecretFor(ctx context.Context, appID id.AppID, clientID string) (RotatedClient, error)
	SigningKeys(ctx context.Context, appID id.AppID) (issuer string, keys []KeyInfo, err error)
	RotateSigningKey(ctx context.Context, appID id.AppID) (KeyInfo, error)
	IssuerInfo(ctx context.Context, appID id.AppID) (IssuerDetails, error)
}

// Deps carries the engine, which resolves the caller's app, and the plugin.
type Deps struct {
	Engine *authsome.Engine
	Plugin OAuth2Service
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("oauth2provider/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "oauth2provider/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("oauth2provider/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("oauth2provider/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("oauth2provider/contract: register manifest: %w", err)
	}

	const c = "oauth2provider"
	regs := []struct {
		name string
		fn   func() error
	}{
		{"oauth2.clients.list", func() error {
			return dispatcher.RegisterQuery(d, c, "oauth2.clients.list", 1, clientsListHandler(deps))
		}},
		{"oauth2.clients.detail", func() error {
			return dispatcher.RegisterQuery(d, c, "oauth2.clients.detail", 1, clientsDetailHandler(deps))
		}},
		{"oauth2.clients.create", func() error {
			return dispatcher.RegisterCommand(d, c, "oauth2.clients.create", 1, clientsCreateHandler(deps), dispatcher.SecretResponse())
		}},
		{"oauth2.clients.update", func() error {
			return dispatcher.RegisterCommand(d, c, "oauth2.clients.update", 1, clientsUpdateHandler(deps))
		}},
		{"oauth2.clients.delete", func() error {
			return dispatcher.RegisterCommand(d, c, "oauth2.clients.delete", 1, clientsDeleteHandler(deps))
		}},
		{"oauth2.clients.rotateSecret", func() error {
			return dispatcher.RegisterCommand(d, c, "oauth2.clients.rotateSecret", 1, clientsRotateSecretHandler(deps), dispatcher.SecretResponse())
		}},
		{"oauth2.issuer.info", func() error {
			return dispatcher.RegisterQuery(d, c, "oauth2.issuer.info", 1, issuerInfoHandler(deps))
		}},
		{"oauth2.keys.list", func() error {
			return dispatcher.RegisterQuery(d, c, "oauth2.keys.list", 1, keysListHandler(deps))
		}},
		{"oauth2.keys.rotate", func() error {
			return dispatcher.RegisterCommand(d, c, "oauth2.keys.rotate", 1, keysRotateHandler(deps))
		}},
	}
	for _, r := range regs {
		if err := r.fn(); err != nil {
			return fmt.Errorf("oauth2provider/contract: register %s: %w", r.name, err)
		}
	}
	return nil
}
