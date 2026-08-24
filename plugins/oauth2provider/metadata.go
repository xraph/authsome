package oauth2provider

import (
	"github.com/xraph/forge"
)

// AuthServerMetadata is the RFC 8414 authorization server metadata document.
// It carries every field the OIDC discovery document does, plus the RFC 7591
// registration endpoint. handleDiscovery hand-maps buildAuthServerMetadata's
// output onto the OIDC DiscoveryResponse struct field by field; the two are
// not one shared type, so TestMetadata_OIDCAndAuthServerAgree compares the
// full set of JSON keys both documents actually advertise, to catch a field
// added to one and not the other.
type AuthServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// ProtectedResourceMetadata is the RFC 9728 protected resource metadata
// document. An MCP client fetches this first and follows
// authorization_servers to the RFC 8414 document.
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// ProtectedResourceRequest addresses one configured resource by the path
// suffix it is served under (RFC 9728 section 3.1).
type ProtectedResourceRequest struct {
	Path string `path:"resourcePath"`
}

// buildAuthServerMetadata is the single source of truth for the RFC 8414
// document. handleDiscovery maps its fields onto the OIDC discovery shape so
// the two documents cannot drift apart.
func (p *Plugin) buildAuthServerMetadata() *AuthServerMetadata {
	issuer := p.issuerURL()

	m := &AuthServerMetadata{
		Issuer:                      issuer,
		AuthorizationEndpoint:       issuer + "/v1/oauth/authorize",
		TokenEndpoint:               issuer + "/v1/oauth/token",
		UserinfoEndpoint:            issuer + "/v1/oauth/userinfo",
		RevocationEndpoint:          issuer + "/v1/oauth/revoke",
		DeviceAuthorizationEndpoint: issuer + "/v1/oauth/device/authorize",
		JWKSURI:                     issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:      []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code",
			"client_credentials",
			deviceCodeGrantType,
		},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256", "ES256"},
		// The configured DynamicRegistrationScopes allowlist, not a
		// separate hardcoded list: advertising a scope here that
		// registration would then silently drop (or vice versa) is
		// exactly the drift this shares a single source to avoid.
		ScopesSupported: p.config.DynamicRegistrationScopes,
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post", "client_secret_basic", "none",
		},
		CodeChallengeMethodsSupported: []string{"S256", "plain"},
	}

	// Only advertise registration when it will actually answer. Pointing a
	// client at an endpoint that 404s produces a worse failure than not
	// mentioning it at all.
	if p.config.DynamicRegistration {
		m.RegistrationEndpoint = issuer + "/v1/oauth/register"
	}
	return m
}

func (p *Plugin) handleAuthServerMetadata(_ forge.Context, _ *DiscoveryRequest) (*AuthServerMetadata, error) {
	return p.buildAuthServerMetadata(), nil
}

// handleProtectedResourceMetadata always describes AuthSome itself,
// regardless of what Config.ProtectedResources declares. Declared extras
// are served at the suffixed path by handleScopedProtectedResourceMetadata.
func (p *Plugin) handleProtectedResourceMetadata(_ forge.Context, _ *DiscoveryRequest) (*ProtectedResourceMetadata, error) {
	issuer := p.issuerURL()
	return &ProtectedResourceMetadata{
		Resource:               issuer,
		AuthorizationServers:   []string{issuer},
		ScopesSupported:        p.config.DynamicRegistrationScopes,
		BearerMethodsSupported: []string{"header"},
	}, nil
}

func (p *Plugin) handleScopedProtectedResourceMetadata(_ forge.Context, req *ProtectedResourceRequest) (*ProtectedResourceMetadata, error) {
	res, ok := p.config.ProtectedResources[req.Path]
	if !ok {
		return nil, forge.NotFound("no protected resource is declared at that path")
	}
	return &ProtectedResourceMetadata{
		Resource:               res.Resource,
		AuthorizationServers:   []string{p.issuerURL()},
		ScopesSupported:        res.ScopesSupported,
		BearerMethodsSupported: []string{"header"},
	}, nil
}
