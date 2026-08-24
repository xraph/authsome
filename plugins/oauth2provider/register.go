package oauth2provider

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// Cheap per-field caps enforced on an unauthenticated registration request.
// See the checks in handleRegisterClient for why these exist.
const (
	maxRegisterRedirectURIs  = 20
	maxRegisterClientNameLen = 256
	maxRegisterContacts      = 10
)

// RegisterClientRequest is an RFC 7591 client registration request.
type RegisterClientRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// RegisterClientResponse is an RFC 7591 client information response. It is
// also what the RFC 7592 read and update endpoints return.
type RegisterClientResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RegistrationAccessToken string   `json:"registration_access_token,omitempty"`
	RegistrationClientURI   string   `json:"registration_client_uri,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// resolveRegistrationAppID picks the app a new dynamic client belongs to.
//
// RFC 7591 has no field for it and a body field would let any caller name
// any tenant, so it comes off the transport: the publishable key middleware
// resolves X-Publishable-Key (or ?publishable_key=) onto the context. That
// middleware never aborts, so an unresolvable key looks exactly like a
// missing one here, and the config fallback is what a single-tenant
// deployment sets so a stock MCP client can register with no key at all.
func (p *Plugin) resolveRegistrationAppID(ctx forge.Context) (id.AppID, error) {
	if appID, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return appID, nil
	}
	if p.config.RegistrationAppID != "" {
		appID, err := id.ParseAppID(p.config.RegistrationAppID)
		if err != nil {
			return id.AppID{}, forge.InternalError(
				fmt.Errorf("oauth2: RegistrationAppID is not a valid app id: %w", err))
		}
		return appID, nil
	}
	return id.AppID{}, regError(http.StatusForbidden, errAccessDenied,
		"registration requires a publishable key on this deployment")
}

// issuerURL returns the configured issuer, or a localhost default when none
// is set. Shared by the discovery document, the device flow's verification
// URI, and dynamic registration's registration_client_uri, so the fallback
// only lives in one place.
func (p *Plugin) issuerURL() string {
	if p.config.Issuer != "" {
		return p.config.Issuer
	}
	return "https://localhost"
}

func (p *Plugin) registrationClientURI(clientID string) string {
	return p.issuerURL() + "/v1/oauth/register/" + clientID
}

// metadataFromRequest collects the RFC 7591 fields that carry no behaviour.
// They only have to round-trip on a 7592 read, so they live in one blob
// instead of eight columns across four backends.
func metadataFromRequest(req *RegisterClientRequest) map[string]any {
	m := map[string]any{}
	for k, v := range map[string]string{
		"client_uri":       req.ClientURI,
		"logo_uri":         req.LogoURI,
		"tos_uri":          req.TOSURI,
		"policy_uri":       req.PolicyURI,
		"software_id":      req.SoftwareID,
		"software_version": req.SoftwareVersion,
	} {
		if v != "" {
			m[k] = v
		}
	}
	if len(req.Contacts) > 0 {
		m["contacts"] = req.Contacts
	}
	return m
}

func (p *Plugin) handleRegisterClient(ctx forge.Context, req *RegisterClientRequest) (*RegisterClientResponse, error) {
	if !p.config.DynamicRegistration {
		return nil, forge.NotFound("dynamic client registration is not enabled")
	}

	appID, err := p.resolveRegistrationAppID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.RedirectURIs) == 0 {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"redirect_uris is required")
	}
	// Cheap caps on an unauthenticated write path. None of these are RFC
	// 7591 requirements; they exist so a single request cannot make the
	// stored client record, or the work done building it, unboundedly
	// large.
	if len(req.RedirectURIs) > maxRegisterRedirectURIs {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("redirect_uris must not exceed %d entries", maxRegisterRedirectURIs))
	}
	if len(req.ClientName) > maxRegisterClientNameLen {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("client_name must not exceed %d characters", maxRegisterClientNameLen))
	}
	if len(req.Contacts) > maxRegisterContacts {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("contacts must not exceed %d entries", maxRegisterContacts))
	}
	for _, u := range req.RedirectURIs {
		if uriErr := validateRedirectURI(u); uriErr != nil {
			return nil, uriErr
		}
	}

	grantTypes, err := clampGrantTypes(req.GrantTypes)
	if err != nil {
		return nil, err
	}

	// response_types is accepted but neither validated nor stored: this
	// server only ever runs the authorization_code response type, "code",
	// and RFC 7591 section 2 lets the server substitute its own values the
	// same way it does for scope. Unlike grant_types just above, there is
	// nothing here to clamp against or reject — the response always
	// reports ["code"] in clientInfoResponse regardless of what was asked
	// for.
	scopes := clampScopes(strings.Fields(req.Scope), p.config.DynamicRegistrationScopes)

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_basic"
	}
	switch authMethod {
	case "none", "client_secret_basic", "client_secret_post":
	default:
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("token_endpoint_auth_method %q is not supported", authMethod))
	}
	isPublic := authMethod == "none"

	clientIDStr, err := generateSecureToken(16)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_id: %w", err))
	}

	var rawSecret, hashedSecret string
	if !isPublic {
		rawSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_secret: %w", err))
		}
		h, hashErr := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: hash client_secret: %w", hashErr))
		}
		hashedSecret = string(h)
	}

	rawRegToken, err := generateSecureToken(32)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate registration token: %w", err))
	}
	regHash, err := bcrypt.GenerateFromPassword([]byte(rawRegToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: hash registration token: %w", err))
	}

	now := time.Now()
	client := &OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		Name:                    req.ClientName,
		ClientID:                clientIDStr,
		ClientSecret:            hashedSecret,
		RedirectURIs:            req.RedirectURIs,
		Scopes:                  scopes,
		GrantTypes:              grantTypes,
		Public:                  isPublic,
		TokenEndpointAuthMethod: authMethod,
		RegistrationTokenHash:   string(regHash),
		DynamicallyRegistered:   true,
		Metadata:                metadataFromRequest(req),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := p.oauth2Store.CreateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create dynamic client: %w", err))
	}

	p.logger.Info("oauth2: dynamic client registered",
		log.String("client_id", client.ClientID),
		log.String("app_id", appID.String()),
		log.String("client_name", client.Name))

	resp := p.clientInfoResponse(client)
	resp.ClientSecret = rawSecret
	resp.RegistrationAccessToken = rawRegToken

	// forge's opinionated handler always answers 200 for a non-nil response
	// value, so the 201 RFC 7591 requires has to be written directly; the
	// nil return here (mirroring handleRevoke/handleAuthorize elsewhere in
	// this package) tells the framework not to write a second response on
	// top of this one. Returning ctx.JSON's error as the handler error
	// would send it back through handleError onto a response that is
	// already committed — a superfluous WriteHeader plus a mangled body if
	// the client disconnected mid-encode — so log it and swallow it
	// instead.
	if jsonErr := ctx.JSON(http.StatusCreated, resp); jsonErr != nil {
		p.logger.Warn("oauth2: write registration response", log.Error(jsonErr))
	}
	return nil, nil
}

// clientInfoResponse renders the RFC 7591 client information response. It
// omits both credentials: the caller adds the raw secret and registration
// token on the one response that is allowed to carry them.
func (p *Plugin) clientInfoResponse(c *OAuth2Client) *RegisterClientResponse {
	var secretExpires int64
	// Nil means the secret never expires, which RFC 7591 represents as 0.
	// The field is a pointer because encoding/json never omits a struct, so
	// a bare time.Time would put a zero timestamp into every response.
	if c.ClientSecretExpiresAt != nil && !c.ClientSecretExpiresAt.IsZero() {
		secretExpires = c.ClientSecretExpiresAt.Unix()
	}
	str := func(k string) string {
		if v, ok := c.Metadata[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	var contacts []string
	switch raw := c.Metadata["contacts"].(type) {
	case []any:
		for _, v := range raw {
			if s, ok := v.(string); ok {
				contacts = append(contacts, s)
			}
		}
	case []string:
		contacts = raw
	}

	return &RegisterClientResponse{
		ClientID:                c.ClientID,
		ClientIDIssuedAt:        c.CreatedAt.Unix(),
		ClientSecretExpiresAt:   secretExpires,
		RegistrationClientURI:   p.registrationClientURI(c.ClientID),
		RedirectURIs:            c.RedirectURIs,
		GrantTypes:              c.GrantTypes,
		ResponseTypes:           []string{"code"},
		Scope:                   strings.Join(c.Scopes, " "),
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		ClientName:              c.Name,
		ClientURI:               str("client_uri"),
		LogoURI:                 str("logo_uri"),
		TOSURI:                  str("tos_uri"),
		PolicyURI:               str("policy_uri"),
		SoftwareID:              str("software_id"),
		SoftwareVersion:         str("software_version"),
		Contacts:                contacts,
	}
}

// RegistrationRequest addresses one registration by its client_id.
type RegistrationRequest struct {
	ClientID string `path:"clientId"`
}

// UpdateRegistrationRequest is an RFC 7592 client update. The body repeats
// the full registration; anything omitted is cleared, per RFC 7592
// section 2.2, which specifies a replacement and not a merge.
type UpdateRegistrationRequest struct {
	ClientID string `path:"clientId"`

	BodyClientID     string   `json:"client_id,omitempty"`
	BodyClientSecret string   `json:"client_secret,omitempty"`
	RedirectURIs     []string `json:"redirect_uris"`
	ClientName       string   `json:"client_name,omitempty"`
	GrantTypes       []string `json:"grant_types,omitempty"`
	Scope            string   `json:"scope,omitempty"`

	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// authenticateRegistration resolves the client named in the path and checks
// the bearer registration access token against its stored hash.
//
// Every failure returns the same 401 so the endpoint does not tell an
// unauthenticated caller which client_ids exist. Admin-created clients have
// an empty hash and are rejected before the compare, which is what keeps
// them off these routes entirely.
func (p *Plugin) authenticateRegistration(ctx forge.Context, clientID string) (*OAuth2Client, error) {
	unauthorized := func() error {
		ctx.SetHeader("WWW-Authenticate", `Bearer realm="registration"`)
		return regError(http.StatusUnauthorized, "invalid_token",
			"a valid registration access token is required")
	}

	raw := ctx.Request().Header.Get("Authorization")
	const prefix = "Bearer "
	if len(raw) <= len(prefix) || !strings.EqualFold(raw[:len(prefix)], prefix) {
		return nil, unauthorized()
	}
	token := strings.TrimSpace(raw[len(prefix):])
	if token == "" {
		return nil, unauthorized()
	}

	client, err := p.oauth2Store.GetClient(ctx.Context(), clientID)
	if err != nil {
		return nil, unauthorized()
	}
	if !client.DynamicallyRegistered || client.RegistrationTokenHash == "" {
		return nil, unauthorized()
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(client.RegistrationTokenHash), []byte(token)); err != nil {
		return nil, unauthorized()
	}
	return client, nil
}

func (p *Plugin) handleReadRegistration(ctx forge.Context, req *RegistrationRequest) (*RegisterClientResponse, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	// The registration access token is not rotated on read. RFC 7592
	// permits rotation, and it strands any client that does not persist
	// the new value, which in practice is most of them.
	return p.clientInfoResponse(client), nil
}

func (p *Plugin) handleUpdateRegistration(ctx forge.Context, req *UpdateRegistrationRequest) (*RegisterClientResponse, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}

	// RFC 7592 section 2.2: a client_id in the body must match the one
	// being updated, and a client_secret must match the current one.
	if req.BodyClientID != "" && req.BodyClientID != client.ClientID {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"client_id in the body does not match the registration being updated")
	}
	if req.BodyClientSecret != "" {
		if bcrypt.CompareHashAndPassword(
			[]byte(client.ClientSecret), []byte(req.BodyClientSecret)) != nil {
			return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
				"client_secret in the body does not match the current secret")
		}
	}

	if len(req.RedirectURIs) == 0 {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"redirect_uris is required")
	}
	for _, u := range req.RedirectURIs {
		if uriErr := validateRedirectURI(u); uriErr != nil {
			return nil, uriErr
		}
	}

	// Same clamps as registration. An update must not be able to buy a
	// capability that registration refused.
	grantTypes, err := clampGrantTypes(req.GrantTypes)
	if err != nil {
		return nil, err
	}
	scopes := clampScopes(strings.Fields(req.Scope), p.config.DynamicRegistrationScopes)

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = client.TokenEndpointAuthMethod
	}
	switch authMethod {
	case "none", "client_secret_basic", "client_secret_post":
	default:
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("token_endpoint_auth_method %q is not supported", authMethod))
	}

	client.Name = req.ClientName
	client.RedirectURIs = req.RedirectURIs
	client.GrantTypes = grantTypes
	client.Scopes = scopes
	client.TokenEndpointAuthMethod = authMethod
	client.Public = authMethod == "none"
	client.Metadata = metadataFromRequest(&RegisterClientRequest{
		ClientURI:       req.ClientURI,
		LogoURI:         req.LogoURI,
		TOSURI:          req.TOSURI,
		PolicyURI:       req.PolicyURI,
		SoftwareID:      req.SoftwareID,
		SoftwareVersion: req.SoftwareVersion,
		Contacts:        req.Contacts,
	})

	if err := p.oauth2Store.UpdateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: update registration: %w", err))
	}
	return p.clientInfoResponse(client), nil
}

// handleDeleteRegistration answers 204 on success. forge's typed-handler
// adapter only ever writes a response when the handler returns a non-nil
// value, and when it does it hardcodes 200 (see handleRegisterClient's 201
// for the same constraint on the create path) — WithNoContentResponse is
// OpenAPI documentation only and has no effect on what status is actually
// written. So the 204 has to be written here directly, and the nil, nil
// return tells the adapter not to write anything on top of it.
func (p *Plugin) handleDeleteRegistration(ctx forge.Context, req *RegistrationRequest) (*apitypes.Empty, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	if err := p.oauth2Store.DeleteClient(ctx.Context(), client.ID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: delete registration: %w", err))
	}
	p.logger.Info("oauth2: dynamic client deleted",
		log.String("client_id", client.ClientID))

	if noContentErr := ctx.NoContent(http.StatusNoContent); noContentErr != nil {
		p.logger.Warn("oauth2: write delete registration response", log.Error(noContentErr))
	}
	return nil, nil
}
