package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// OIDCConfig configures an OIDC identity provider.
type OIDCConfig struct {
	// Name is the unique provider identifier (e.g. "okta", "azure-ad").
	Name string

	// Issuer is the OIDC issuer URL (e.g. "https://mycompany.okta.com").
	Issuer string

	// ClientID is the OAuth2 client ID.
	ClientID string

	// ClientSecret is the OAuth2 client secret.
	ClientSecret string

	// RedirectURL is the callback URL registered with the provider.
	RedirectURL string

	// Scopes is the set of OAuth2 scopes. Defaults to ["openid", "profile", "email"].
	Scopes []string
}

// oidcProvider implements the Provider interface for OpenID Connect.
type oidcProvider struct {
	name   string
	issuer string
	config *oauth2.Config
}

// NewOIDCProvider creates an OIDC SSO provider. It uses OIDC discovery
// to resolve the authorization and token endpoints from the issuer URL.
func NewOIDCProvider(cfg OIDCConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	// Build the OAuth2 config using OIDC discovery convention.
	// The actual endpoints will be resolved at login time via .well-known.
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  strings.TrimRight(cfg.Issuer, "/") + "/authorize",
			TokenURL: strings.TrimRight(cfg.Issuer, "/") + "/oauth/token",
		},
	}

	return &oidcProvider{
		name:   cfg.Name,
		issuer: strings.TrimRight(cfg.Issuer, "/"),
		config: oauthCfg,
	}
}

func (p *oidcProvider) Name() string     { return p.name }
func (p *oidcProvider) Protocol() string { return "oidc" }

// LoginURL returns the authorization URL with the given state parameter.
func (p *oidcProvider) LoginURL(state string) (string, error) {
	// Try to discover the actual auth endpoint.
	discovered, err := p.discover()
	if err == nil && discovered.AuthorizationEndpoint != "" {
		p.config.Endpoint.AuthURL = discovered.AuthorizationEndpoint
		p.config.Endpoint.TokenURL = discovered.TokenEndpoint
	}

	url := p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, nil
}

// HandleCallback exchanges the authorization code for tokens and fetches user info.
func (p *oidcProvider) HandleCallback(ctx context.Context, params map[string]string) (*User, error) {
	code := params["code"]
	if code == "" {
		return nil, fmt.Errorf("sso/oidc: missing authorization code")
	}

	// Try to discover endpoints.
	discovered, err := p.discover()
	if err == nil && discovered.TokenEndpoint != "" {
		p.config.Endpoint.TokenURL = discovered.TokenEndpoint
	}

	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("sso/oidc: token exchange failed: %w", err)
	}

	// Determine the userinfo endpoint.
	userinfoURL := p.issuer + "/userinfo"
	if discovered != nil && discovered.UserinfoEndpoint != "" {
		userinfoURL = discovered.UserinfoEndpoint
	}

	// Fetch user info.
	client := p.config.Client(ctx, token)
	userinfoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("sso/oidc: create userinfo request failed: %w", err)
	}
	resp, err := client.Do(userinfoReq)
	if err != nil {
		return nil, fmt.Errorf("sso/oidc: userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sso/oidc: userinfo returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sso/oidc: failed to read userinfo response: %w", err)
	}

	var claims oidcClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("sso/oidc: failed to parse userinfo: %w", err)
	}

	user := &User{
		ProviderUserID: claims.Sub,
		Email:          claims.Email,
		FirstName:      claims.GivenName,
		LastName:       claims.FamilyName,
		Groups:         claims.Groups,
		Attributes:     make(map[string]string),
	}

	// If given_name/family_name are empty, fall back to the full name claim.
	if user.FirstName == "" && user.LastName == "" && claims.Name != "" {
		user.FirstName = claims.Name
	}

	return user, nil
}

// oidcClaims represents standard OIDC userinfo claims.
type oidcClaims struct {
	Sub        string   `json:"sub"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	GivenName  string   `json:"given_name"`
	FamilyName string   `json:"family_name"`
	Groups     []string `json:"groups"`
}

// oidcDiscovery represents the OIDC .well-known/openid-configuration response.
type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// discover fetches the OIDC discovery document from the issuer.
func (p *oidcProvider) discover() (*oidcDiscovery, error) {
	url := p.issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sso/oidc: discovery returned status %d", resp.StatusCode)
	}

	var doc oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
