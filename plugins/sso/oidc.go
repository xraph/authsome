package sso

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
)

// OIDCConfig configures an OpenID Connect SSO provider.
type OIDCConfig struct {
	// Name is the provider's stable name, used in routes and state.
	Name string
	// Issuer is the OpenID provider's issuer URL. Discovery is fetched from
	// Issuer + "/.well-known/openid-configuration".
	Issuer string
	// ClientID and ClientSecret are the relying party credentials.
	ClientID     string
	ClientSecret string
	// RedirectURL is the callback registered with the provider.
	RedirectURL string
	// Scopes defaults to openid, profile and email.
	Scopes []string
	// HTTPClient overrides the client used for discovery, the token exchange
	// and userinfo. Nil selects the hardened client, which refuses private
	// addresses and plain http; tests point it at a local server.
	HTTPClient *http.Client
}

// discoveryTTL is how long a fetched discovery document is reused. The
// endpoints in it change about as often as the provider does.
const discoveryTTL = time.Hour

// discoveryMaxBytes bounds a discovery document. Real ones are a few
// kilobytes; a megabyte is the ceiling before a misbehaving issuer costs
// memory.
const discoveryMaxBytes = 1 << 20

// discoveryCache holds discovery documents per issuer across the providers
// the plugin constructs per request, so a login does not refetch what the
// previous login already learned.
var discoveryCache sync.Map // issuer -> cachedDiscovery

type cachedDiscovery struct {
	doc     *oidcDiscovery
	expires time.Time
}

type oidcProvider struct {
	name       string
	issuer     string
	config     *oauth2.Config
	httpClient *http.Client
	// customClient records that the caller supplied the client, in which
	// case the issuer URL is not validated: tests reach a local server.
	customClient bool
}

// NewOIDCProvider creates an OIDC SSO provider. It uses OIDC discovery to
// find the authorization, token and userinfo endpoints and falls back to
// the conventional paths under the issuer when discovery is unavailable.
func NewOIDCProvider(cfg OIDCConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	issuer := strings.TrimRight(cfg.Issuer, "/")
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuer + "/authorize",
			TokenURL: issuer + "/oauth/token",
		},
	}
	client := cfg.HTTPClient
	if client == nil {
		client = jwksclient.NewHTTPClient(10 * time.Second)
	}
	return &oidcProvider{name: cfg.Name, issuer: issuer, config: oauthCfg, httpClient: client, customClient: cfg.HTTPClient != nil}
}

func (p *oidcProvider) Name() string     { return p.name }
func (p *oidcProvider) Protocol() string { return "oidc" }

// requestConfig returns a copy of the OAuth2 configuration with discovered
// endpoints applied. The copy is what keeps concurrent logins from writing
// endpoints into one another's configuration.
func (p *oidcProvider) requestConfig() (oauth2.Config, *oidcDiscovery) {
	cfg := *p.config
	discovered, err := p.discover()
	if err == nil && discovered != nil {
		if discovered.AuthorizationEndpoint != "" {
			cfg.Endpoint.AuthURL = discovered.AuthorizationEndpoint
		}
		if discovered.TokenEndpoint != "" {
			cfg.Endpoint.TokenURL = discovered.TokenEndpoint
		}
	}
	return cfg, discovered
}

// LoginURL builds the authorization URL without a nonce or PKCE. The plugin
// prefers LoginURLWithPKCE; this remains for callers of the Provider
// interface alone.
func (p *oidcProvider) LoginURL(state string) (string, error) {
	cfg, _ := p.requestConfig()
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// LoginURLWithPKCE builds the authorization URL carrying a nonce, which the
// id_token must echo, and an S256 challenge for verifier, which the token
// exchange must present. Together they bind the code to this login: a code
// injected from another session fails the verifier, and an id_token minted
// for another login fails the nonce.
func (p *oidcProvider) LoginURLWithPKCE(state, nonce, verifier string) (string, error) {
	if nonce == "" || verifier == "" {
		return "", errors.New("sso/oidc: nonce and verifier are required")
	}
	cfg, _ := p.requestConfig()
	return cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// HandleCallback exchanges the code and reads the user from userinfo. When
// the login minted a nonce and verifier (params "nonce" and "code_verifier")
// the exchange presents the verifier and the id_token must carry the nonce.
func (p *oidcProvider) HandleCallback(ctx context.Context, params map[string]string) (*User, error) {
	code := params["code"]
	if code == "" {
		return nil, errors.New("sso/oidc: missing authorization code")
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	cfg, discovered := p.requestConfig()

	var exchangeOpts []oauth2.AuthCodeOption
	if v := params["code_verifier"]; v != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(v))
	}
	token, err := cfg.Exchange(ctx, code, exchangeOpts...)
	if err != nil {
		return nil, fmt.Errorf("sso/oidc: token exchange failed: %w", err)
	}
	if nonce := params["nonce"]; nonce != "" {
		if nonceErr := checkIDTokenNonce(token, nonce); nonceErr != nil {
			return nil, nonceErr
		}
	}

	userinfoURL := p.issuer + "/userinfo"
	if discovered != nil && discovered.UserinfoEndpoint != "" {
		userinfoURL = discovered.UserinfoEndpoint
	}
	client := cfg.Client(ctx, token)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, discoveryMaxBytes))
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
		EmailVerified:  parseEmailVerified(claims.EmailVerified),
	}
	if user.FirstName == "" && user.LastName == "" && claims.Name != "" {
		user.FirstName = claims.Name
	}
	return user, nil
}

// checkIDTokenNonce requires the id_token in the token response to carry
// the nonce this login minted. The claims are read without signature
// verification: the token arrived over TLS from the provider's own token
// endpoint, and the identity itself comes from userinfo, so the nonce here
// binds the response to the login rather than proving who signed it.
func checkIDTokenNonce(token *oauth2.Token, nonce string) error {
	raw, _ := token.Extra("id_token").(string) //nolint:errcheck // absent id_token handled below
	if raw == "" {
		return errors.New("sso/oidc: token response carries no id_token to check the nonce against")
	}
	claims, err := decodeUnverifiedClaims(raw)
	if err != nil {
		return fmt.Errorf("sso/oidc: id_token: %w", err)
	}
	got, _ := claims["nonce"].(string) //nolint:errcheck // a non-string nonce is a mismatch
	if subtle.ConstantTimeCompare([]byte(got), []byte(nonce)) != 1 {
		return errors.New("sso/oidc: id_token nonce does not match this login")
	}
	return nil
}

// decodeUnverifiedClaims reads a JWT payload without checking the signature.
func decodeUnverifiedClaims(jwt string) (map[string]any, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	return claims, nil
}

// oidcDiscovery represents the OIDC .well-known/openid-configuration response.
type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// discover returns the issuer's discovery document, fetched through the
// hardened client and cached for discoveryTTL. The issuer URL is validated
// first unless a custom client was supplied, which is how tests reach a
// local server.
func (p *oidcProvider) discover() (*oidcDiscovery, error) {
	if c, ok := discoveryCache.Load(p.issuer); ok {
		if cached, ok := c.(cachedDiscovery); ok && time.Now().Before(cached.expires) {
			return cached.doc, nil
		}
	}
	url := p.issuer + "/.well-known/openid-configuration"
	if !p.customClient {
		if err := jwksclient.ValidateURI(url); err != nil {
			return nil, fmt.Errorf("sso/oidc: discovery: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sso/oidc: discovery returned status %d", resp.StatusCode)
	}
	var doc oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, discoveryMaxBytes)).Decode(&doc); err != nil {
		return nil, err
	}
	discoveryCache.Store(p.issuer, cachedDiscovery{doc: &doc, expires: time.Now().Add(discoveryTTL)})
	return &doc, nil
}

type oidcClaims struct {
	Sub        string   `json:"sub"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	GivenName  string   `json:"given_name"`
	FamilyName string   `json:"family_name"`
	Groups     []string `json:"groups"`
	// EmailVerified is a bool in the specification and a string at some
	// providers, so it is decoded by hand.
	EmailVerified json.RawMessage `json:"email_verified"`
}

// parseEmailVerified reads an email_verified claim as OIDC defines it (a
// JSON boolean) and as some providers send it (the strings "true" and
// "false"). Anything else, including absence, is nil: no statement.
func parseEmailVerified(raw json.RawMessage) *bool {
	t := strings.TrimSpace(string(raw))
	switch t {
	case "true", `"true"`:
		v := true
		return &v
	case "false", `"false"`:
		v := false
		return &v
	}
	return nil
}
