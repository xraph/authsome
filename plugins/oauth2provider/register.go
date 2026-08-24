package oauth2provider

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
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

// issuerURL returns the configured issuer, or the localhost default the
// discovery handler already falls back to.
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
	for _, u := range req.RedirectURIs {
		if uriErr := validateRedirectURI(u); uriErr != nil {
			return nil, uriErr
		}
	}

	grantTypes, err := clampGrantTypes(req.GrantTypes)
	if err != nil {
		return nil, err
	}
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
	// top of this one.
	return nil, ctx.JSON(http.StatusCreated, resp)
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
