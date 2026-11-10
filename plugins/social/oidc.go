package social

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Generic OpenID Connect providers. Unlike the built-in providers, these
// have no code of their own: an app gives an issuer URL, a client ID and a
// secret, and the endpoints and signing keys come from the issuer's
// discovery document. That is what lets one authsome deployment offer
// "Sign in with <another authsome deployment>" without a release.

// ProviderTypeOIDC marks a ProviderSetting as a generic OpenID Connect
// provider.
const ProviderTypeOIDC = "oidc"

// OIDCConfig configures a generic OpenID Connect provider.
type OIDCConfig struct {
	ProviderConfig

	// Name is the provider's ID: lowercase letters, digits and dashes. It
	// appears in the sign-in URLs (/v1/social/{name}) and on connections.
	Name string

	// Issuer is the issuer URL. Discovery reads
	// {Issuer}/.well-known/openid-configuration.
	Issuer string

	// HTTPClient makes the discovery, key and token calls. Nil uses a
	// client that refuses private and loopback addresses.
	HTTPClient *http.Client
}

var oidcNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// ValidateOIDCName checks a generic provider's ID. It cannot reuse a
// built-in provider's ID, since the built-in would answer for it.
func ValidateOIDCName(name string) error {
	if !oidcNamePattern.MatchString(name) {
		return errors.New("name must be 1 to 32 lowercase letters, digits or dashes, starting with a letter or digit")
	}
	if catalogContains(name) || name == ProviderTypeOIDC {
		return fmt.Errorf("name %q is taken by a built-in provider", name)
	}
	return nil
}

// ValidateOIDCIssuer checks an issuer URL: absolute https, with a host,
// and no query or fragment (OpenID Connect Discovery 1.0 section 2).
func ValidateOIDCIssuer(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("issuer: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" {
		return errors.New("issuer must be an absolute https URL")
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return errors.New("issuer must not carry a query, fragment or credentials")
	}
	return nil
}

type oidcProvider struct {
	name     string
	config   *oauth2.Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	client   *http.Client
}

// NewOIDCProvider runs discovery against cfg.Issuer and returns a provider
// that signs users in with it. The id_token is the source of the profile;
// the userinfo endpoint fills in an email the id_token left out.
func NewOIDCProvider(ctx context.Context, cfg OIDCConfig) (Provider, error) {
	if err := ValidateOIDCName(cfg.Name); err != nil {
		return nil, fmt.Errorf("oidc: %w", err)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = publicHTTPClient()
	}
	disc, err := oidc.NewProvider(oidc.ClientContext(ctx, client), cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc: discover %s: %w", cfg.Issuer, err)
	}
	return newOIDCProviderFrom(disc, cfg, client), nil
}

func newOIDCProviderFrom(disc *oidc.Provider, cfg OIDCConfig, client *http.Client) *oidcProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	} else if !containsString(scopes, oidc.ScopeOpenID) {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}
	return &oidcProvider{
		name: cfg.Name,
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     disc.Endpoint(),
		},
		provider: disc,
		verifier: disc.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		client:   client,
	}
}

func (p *oidcProvider) Name() string { return p.name }

func (p *oidcProvider) OAuth2Config() *oauth2.Config { return p.config }

// HTTPClient is the client every call to this issuer must use, the token
// exchange included: the endpoints came from the issuer's own discovery
// document, so they are no more trusted than the issuer URL.
func (p *oidcProvider) HTTPClient() *http.Client { return p.client }

// httpClientProvider is implemented by providers whose calls must go
// through a particular client.
type httpClientProvider interface {
	HTTPClient() *http.Client
}

// FetchUser verifies the id_token (signature, issuer, audience, expiry)
// and reads the profile from its claims. The nonce is checked by the
// callback before this runs.
func (p *oidcProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	raw, _ := token.Extra("id_token").(string) //nolint:errcheck // empty handled below
	if raw == "" {
		return nil, fmt.Errorf("oidc %s: token response carried no id_token", p.name)
	}
	// The callback hands us its instrumented copy of HTTPClient(); a bare
	// context gets the provider's own client. Either way the key and
	// userinfo calls never use an unguarded default.
	client := contextHTTPClient(ctx)
	if client == nil {
		client = p.client
	}
	ctx = oidc.ClientContext(ctx, client)
	idt, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("oidc %s: verify id_token: %w", p.name, err)
	}
	var c oidcClaims
	if err := idt.Claims(&c); err != nil {
		return nil, fmt.Errorf("oidc %s: read id_token claims: %w", p.name, err)
	}

	if c.Email == "" && p.provider.UserInfoEndpoint() != "" {
		info, uerr := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if uerr == nil && info.Subject == idt.Subject {
			var extra oidcClaims
			if info.Claims(&extra) == nil {
				c.Email, c.EmailVerified = extra.Email, extra.EmailVerified
				if c.GivenName == "" && c.FamilyName == "" && c.Name == "" {
					c.GivenName, c.FamilyName, c.Name = extra.GivenName, extra.FamilyName, extra.Name
				}
				if c.Picture == "" {
					c.Picture = extra.Picture
				}
			}
		}
	}

	first, last := c.GivenName, c.FamilyName
	if first == "" && last == "" && c.Name != "" {
		first, last, _ = strings.Cut(c.Name, " ")
	}
	return &ProviderUser{
		ProviderUserID: idt.Subject,
		Email:          c.Email,
		EmailVerified:  bool(c.EmailVerified),
		FirstName:      first,
		LastName:       last,
		AvatarURL:      c.Picture,
	}, nil
}

type oidcClaims struct {
	Email         string       `json:"email"`
	EmailVerified flexibleBool `json:"email_verified"`
	GivenName     string       `json:"given_name"`
	FamilyName    string       `json:"family_name"`
	Name          string       `json:"name"`
	Picture       string       `json:"picture"`
}

// flexibleBool reads a JSON boolean, or the strings "true" and "false"
// that some issuers send for email_verified. Anything else is false.
type flexibleBool bool

func (b *flexibleBool) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case bool:
		*b = flexibleBool(t)
	case string:
		*b = flexibleBool(strings.EqualFold(t, "true"))
	default:
		*b = false
	}
	return nil
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────
// Discovery cache
// ──────────────────────────────────────────────────

// oidcDiscoveryTTL is how long a discovery document is reused. Keys are
// refreshed by go-oidc on an unknown kid regardless.
const oidcDiscoveryTTL = time.Hour

type oidcDiscoveryEntry struct {
	provider *oidc.Provider
	fetched  time.Time
}

// oidcDiscoveryCache keeps one discovery result per issuer so sign-in does
// not refetch the document on every request.
type oidcDiscoveryCache struct {
	mu      sync.Mutex
	entries map[string]oidcDiscoveryEntry
	now     func() time.Time
}

func newOIDCDiscoveryCache() *oidcDiscoveryCache {
	return &oidcDiscoveryCache{entries: make(map[string]oidcDiscoveryEntry), now: time.Now}
}

func (c *oidcDiscoveryCache) get(ctx context.Context, client *http.Client, issuer string) (*oidc.Provider, error) {
	c.mu.Lock()
	e, ok := c.entries[issuer]
	c.mu.Unlock()
	if ok && c.now().Sub(e.fetched) < oidcDiscoveryTTL {
		return e.provider, nil
	}
	disc, err := oidc.NewProvider(oidc.ClientContext(ctx, client), issuer)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.entries[issuer] = oidcDiscoveryEntry{provider: disc, fetched: c.now()}
	c.mu.Unlock()
	return disc, nil
}

// ──────────────────────────────────────────────────
// Outbound client
// ──────────────────────────────────────────────────

// errPrivateAddress is returned when an issuer resolves to an address the
// server should not call on an app admin's behalf.
var errPrivateAddress = errors.New("refusing to connect to a private, loopback or link-local address")

// publicHTTPClient is the shared default client for issuer calls. The issuer URL
// comes from an app's settings, so the address is checked when the
// connection is made (after DNS, which also covers rebinding) and anything
// inside the network is refused.
var publicHTTPClient = sync.OnceValue(func() *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return errPrivateAddress
			}
			return nil
		},
	}
	transport := &http.Transport{}
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = base.Clone()
	}
	transport.DialContext = dialer.DialContext
	transport.Proxy = nil
	return &http.Client{Transport: transport, Timeout: 15 * time.Second}
})

func isPublicIP(ip net.IP) bool {
	switch {
	case ip.IsLoopback(), ip.IsPrivate(), ip.IsUnspecified(), isSharedAddressSpace(ip),
		ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsInterfaceLocalMulticast(), ip.IsMulticast():
		return false
	}
	return true
}

// isSharedAddressSpace reports 100.64.0.0/10 (RFC 6598), which IsPrivate
// leaves out but which carrier and cloud networks use internally.
func isSharedAddressSpace(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64
}
