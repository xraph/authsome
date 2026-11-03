// clients_service.go: the client and signing key admin logic as plain methods.
//
// The HTTP handlers take a forge.Context, which the dashboard's contract
// handlers do not have. The logic lives here instead, on a plain
// context.Context, and the handlers are thin wrappers that resolve the caller's
// app and map the errors. Every method takes the app it acts for and treats a
// client of any other app as not found, so the tenancy rule is enforced here
// rather than left to each caller.
package oauth2provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// clientValidationError is a request the service refused. The HTTP handlers
// answer it with a 400 carrying the same message.
type clientValidationError struct{ msg string }

func (e *clientValidationError) Error() string { return e.msg }

func invalidClient(msg string) error { return &clientValidationError{msg: msg} }

// clientHTTPError maps a service error to the forge error the handlers have
// always returned for it.
func clientHTTPError(ctx forge.Context, err error) error {
	var v *clientValidationError
	switch {
	case errors.As(err, &v):
		return forge.BadRequest(v.msg)
	case errors.Is(err, ErrClientNotFound):
		return forge.NotFound("oauth2 client not found")
	default:
		return middleware.InternalError(ctx, err)
	}
}

// loadClientFor loads the client rawID names, and answers ErrClientNotFound
// for one that belongs to another app, so the answer never says whether the
// client exists elsewhere.
func (p *Plugin) loadClientFor(ctx context.Context, appID id.AppID, rawID string) (*OAuth2Client, error) {
	if rawID == "" {
		return nil, invalidClient("client ID required")
	}
	clientID, err := id.ParseOAuth2ClientID(rawID)
	if err != nil {
		return nil, invalidClient("invalid client ID")
	}
	client, err := p.oauth2Store.GetClientByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("oauth2: load client: %w", err)
	}
	if client.AppID.String() != appID.String() {
		return nil, ErrClientNotFound
	}
	return client, nil
}

// CreateClientFor registers a client under appID. req.AppID is ignored: the
// caller has already decided which app this is for.
func (p *Plugin) CreateClientFor(ctx context.Context, appID id.AppID, req CreateClientRequest) (*CreateClientResponse, error) {
	if req.Name == "" {
		return nil, invalidClient("name required")
	}
	if len(req.RedirectURIs) == 0 && !req.Public {
		return nil, invalidClient("redirect_uris required for confidential clients")
	}
	// Same rule the update path and dynamic registration apply. Leaving create
	// more permissive would mean a client could be registered with a URI that
	// can never be edited afterwards, since any later edit would be refused.
	for _, u := range req.RedirectURIs {
		if uriErr := validateRedirectURI(u); uriErr != nil {
			return nil, invalidClient(registrationErrorDescription(uriErr))
		}
	}
	// Validate the allowlist at registration so a malformed entry is caught
	// here rather than turning every later authorize request into an opaque
	// invalid_target. This shares the exact rule resolveResources applies at
	// request time, so the two can never disagree about what counts as valid.
	for _, raw := range req.Resources {
		if msg := resourceURISyntaxError(raw); msg != "" {
			return nil, invalidClient(msg)
		}
	}

	clientIDStr, err := generateSecureToken(16)
	if err != nil {
		return nil, fmt.Errorf("oauth2: generate client_id: %w", err)
	}

	var rawSecret, hashedSecret string
	if !req.Public {
		rawSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, fmt.Errorf("oauth2: generate client_secret: %w", err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("oauth2: hash client_secret: %w", err)
		}
		hashedSecret = string(hash)
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	// Normalise through ParseMode so an unrecognised value is stored as a
	// known one rather than sitting in the column waiting to be misread.
	// Empty stays empty, which means inherit.
	dpopMode := ""
	if req.DPoPMode != "" {
		dpopMode = string(dpop.ParseMode(req.DPoPMode))
	}

	client := &OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		Name:                    req.Name,
		ClientID:                clientIDStr,
		ClientSecret:            hashedSecret,
		RedirectURIs:            req.RedirectURIs,
		Scopes:                  scopes,
		Resources:               req.Resources,
		GrantTypes:              grantTypes,
		Public:                  req.Public,
		FirstParty:              req.FirstParty,
		TokenEndpointAuthMethod: authMethodForPublic(req.Public),
		DPoPMode:                dpopMode,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	if err := p.oauth2Store.CreateClient(ctx, client); err != nil {
		return nil, fmt.Errorf("oauth2: create client: %w", err)
	}

	resp := &CreateClientResponse{
		ID:           client.ID.String(),
		ClientID:     client.ClientID,
		Name:         client.Name,
		RedirectURIs: client.RedirectURIs,
		Scopes:       client.Scopes,
		Resources:    client.Resources,
		GrantTypes:   client.GrantTypes,
		Public:       client.Public,
	}
	// Only return the raw secret once.
	if rawSecret != "" {
		resp.ClientSecret = rawSecret
	}
	return resp, nil
}

// ListClientsFor lists appID's clients. The result is never nil.
func (p *Plugin) ListClientsFor(ctx context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	clients, err := p.oauth2Store.ListClients(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("oauth2: list clients: %w", err)
	}
	if clients == nil {
		clients = []*OAuth2Client{}
	}
	return clients, nil
}

// GetClientFor loads one of appID's clients by its internal id.
func (p *Plugin) GetClientFor(ctx context.Context, appID id.AppID, clientID string) (*OAuth2Client, error) {
	return p.loadClientFor(ctx, appID, clientID)
}

// UpdateClientFor applies the fields req carries to one of appID's clients.
// req.ClientID is not read: clientID is the only identifier that counts, so a
// body cannot redirect the write.
func (p *Plugin) UpdateClientFor(ctx context.Context, appID id.AppID, clientID string, req UpdateClientRequest) (*OAuth2Client, error) {
	if clientID == "" {
		return nil, invalidClient("client ID required")
	}
	if _, err := id.ParseOAuth2ClientID(clientID); err != nil {
		return nil, invalidClient("invalid client ID")
	}
	// Refuse a malformed request before touching the store. Nothing about
	// this check depends on the stored client, so there is no reason to spend
	// a read discovering that the body was never acceptable.
	if msg := immutableClientFieldError(&req); msg != "" {
		return nil, invalidClient(msg)
	}

	client, err := p.loadClientFor(ctx, appID, clientID)
	if err != nil {
		return nil, err
	}

	// Each field applies only when the caller sent it. Dereferencing an
	// explicitly-sent empty slice is what makes "clear this" expressible.
	if req.Name != nil {
		client.Name = *req.Name
	}
	if req.FirstParty != nil {
		client.FirstParty = *req.FirstParty
	}
	if req.RedirectURIs != nil {
		uris := *req.RedirectURIs
		// A confidential client with no redirect URI can never complete an
		// authorization code flow. Create refuses to make one, so the update
		// path must not be a back door into that state.
		if len(uris) == 0 && !client.Public {
			return nil, invalidClient("redirect_uris must not be empty for a confidential client")
		}
		for _, u := range uris {
			if uriErr := validateRedirectURI(u); uriErr != nil {
				return nil, invalidClient(registrationErrorDescription(uriErr))
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
				return nil, invalidClient(fmt.Sprintf("grant_type %q is not supported", g))
			}
		}
		client.GrantTypes = *req.GrantTypes
	}
	if req.Resources != nil {
		for _, r := range *req.Resources {
			if msg := resourceURISyntaxError(r); msg != "" {
				return nil, invalidClient(msg)
			}
		}
		client.Resources = *req.Resources
	}

	// UpdateClient is a full-record replace on every backend, so the write
	// has to carry the whole client, not just the edited fields.
	if err := p.oauth2Store.UpdateClient(ctx, client); err != nil {
		return nil, fmt.Errorf("oauth2: update client: %w", err)
	}
	return client, nil
}

// DeleteClientFor deletes one of appID's clients. The row is loaded first: the
// tenancy check needs its AppID and the store's delete takes only an id.
func (p *Plugin) DeleteClientFor(ctx context.Context, appID id.AppID, clientID string) error {
	client, err := p.loadClientFor(ctx, appID, clientID)
	if err != nil {
		return err
	}
	if err := p.oauth2Store.DeleteClient(ctx, client.ID); err != nil {
		return fmt.Errorf("oauth2: delete client: %w", err)
	}
	return nil
}

// RotateClientSecretFor replaces a confidential client's secret and returns
// the new one in the clear. It is the only time the secret is readable.
func (p *Plugin) RotateClientSecretFor(ctx context.Context, appID id.AppID, clientID string) (secret string, err error) {
	_, secret, err = p.rotateClientSecret(ctx, appID, clientID)
	return secret, err
}

// rotateClientSecret is RotateClientSecretFor for a caller that also wants the
// client, so the HTTP handler does not load it a second time.
func (p *Plugin) rotateClientSecret(ctx context.Context, appID id.AppID, clientID string) (*OAuth2Client, string, error) {
	client, err := p.loadClientFor(ctx, appID, clientID)
	if err != nil {
		return nil, "", err
	}
	// A public client authenticates with PKCE, not a secret. Minting one here
	// would contradict both Public and TokenEndpointAuthMethod "none", and
	// nothing would ever read it.
	if client.Public {
		return nil, "", invalidClient("a public client has no client_secret to rotate")
	}

	rawSecret, err := generateSecureToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("oauth2: generate client_secret: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("oauth2: hash client_secret: %w", err)
	}
	client.ClientSecret = string(hash)
	if err := p.oauth2Store.UpdateClient(ctx, client); err != nil {
		return nil, "", fmt.Errorf("oauth2: rotate client secret: %w", err)
	}
	return client, rawSecret, nil
}

// IssuerInfo is what an operator needs to point a relying party at the issuer
// an app's id_tokens name.
type IssuerInfo struct {
	Issuer       string
	DiscoveryURL string
	JWKSURI      string
	// AppIssuer is true when the app has its own issuer rather than the
	// deployment's.
	AppIssuer bool
	Algs      []string
}

// IssuerInfo describes the issuer appID's id_tokens are minted under.
func (p *Plugin) IssuerInfo(ctx context.Context, appID id.AppID) (IssuerInfo, error) {
	issuer, scope, err := p.issuerFor(ctx, appID)
	if err != nil {
		return IssuerInfo{}, err
	}
	return IssuerInfo{
		Issuer:       issuer,
		DiscoveryURL: issuer + "/.well-known/openid-configuration",
		JWKSURI:      issuer + "/v1/oauth/jwks",
		AppIssuer:    scope != "",
		Algs:         p.idTokenAlgs(ctx, scope),
	}, nil
}

// SigningKeys lists the id_token signing keys for appID's issuer, active key
// first, along with that issuer.
func (p *Plugin) SigningKeys(ctx context.Context, appID id.AppID) (issuer string, keys []KeyInfo, err error) {
	issuer, scope, err := p.issuerFor(ctx, appID)
	if err != nil {
		return "", nil, err
	}
	if p.keys == nil {
		return "", nil, errNoSigningKey
	}
	keys, err = p.keys.Describe(ctx, scope)
	if err != nil {
		return "", nil, err
	}
	return issuer, keys, nil
}

// RotateSigningKey generates the next managed signing key for appID's own
// issuer. It only rotates an app's own key: an app on the deployment issuer
// gets ErrOperatorKey when a configured key serves it and ErrSharedKey
// otherwise, since that key signs for every app and rotates on its own
// schedule. An app key younger than an hour gets ErrKeyTooNew.
func (p *Plugin) RotateSigningKey(ctx context.Context, appID id.AppID) (KeyInfo, error) {
	_, scope, err := p.issuerFor(ctx, appID)
	if err != nil {
		return KeyInfo{}, err
	}
	if p.keys == nil {
		return KeyInfo{}, errNoSigningKey
	}
	if scope == "" {
		if p.keys.usesOperatorKey(scope) {
			return KeyInfo{}, ErrOperatorKey
		}
		return KeyInfo{}, ErrSharedKey
	}
	row, err := p.keys.Rotate(ctx, scope, manualRotationMinAge)
	if err != nil {
		return KeyInfo{}, err
	}
	// The KID is the RFC 7638 thumbprint the JWKS publishes, not the row id.
	j, err := jwkFromStored(row)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("oauth2: read rotated signing key: %w", err)
	}
	return KeyInfo{
		KID: j.KID, Alg: row.Alg, Source: keySourceManaged, Status: keyStatusActive,
		CreatedAt: row.CreatedAt, RetiresAt: row.RetiresAt,
	}, nil
}
