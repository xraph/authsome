package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	dsig "github.com/russellhaering/goxmldsig"
)

// metadataFetchTimeout bounds how long an IdP metadata fetch may take.
const metadataFetchTimeout = 10 * time.Second

// SAMLConfig configures a SAML 2.0 identity provider connection.
type SAMLConfig struct {
	// Name is the unique provider identifier (e.g. "okta-saml").
	Name string

	// IdP description — resolved in this order: pasted metadata XML, a
	// metadata URL to fetch, or an explicit SSO URL + signing certificate.
	IDPMetadataXML    string
	MetadataURL       string
	IDPSSOURL         string
	IDPCertificatePEM string

	// Service Provider identity.
	EntityID         string // SP entity ID (assertion audience)
	ACSURL           string // SP Assertion Consumer Service URL
	SPCertificatePEM string // SP signing cert (PEM); optional
	SPPrivateKeyPEM  string // SP signing key (PEM); optional, enables signed AuthnRequests
	SignRequests     bool

	// AllowIDPInitiated permits unsolicited assertions (no prior AuthnRequest).
	AllowIDPInitiated bool

	// AttributeMap maps SAML attribute names to user fields
	// ("email" | "first_name" | "last_name" | "groups"). Empty uses defaults.
	AttributeMap map[string]string
}

// samlProvider implements Provider (and SAMLMetadataProvider) for SAML 2.0
// backed by a configured crewjam ServiceProvider.
type samlProvider struct {
	name    string
	sp      *saml.ServiceProvider
	attrMap map[string]string
}

// SAMLMetadataProvider is implemented by SAML providers that can emit SP
// metadata XML for IdP configuration.
type SAMLMetadataProvider interface {
	Metadata() (xml []byte, contentType string, err error)
}

// NewSAMLProvider builds a SAML provider from cfg. It resolves the IdP
// metadata/cert up front so a misconfigured connection fails fast rather than
// at first login.
func NewSAMLProvider(cfg SAMLConfig) (Provider, error) {
	if strings.TrimSpace(cfg.EntityID) == "" || strings.TrimSpace(cfg.ACSURL) == "" {
		return nil, fmt.Errorf("sso/saml: entity_id and acs_url are required")
	}
	acs, err := url.Parse(cfg.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("sso/saml: invalid acs_url: %w", err)
	}

	idpMeta, err := resolveIDPMetadata(cfg)
	if err != nil {
		return nil, err
	}

	sp := &saml.ServiceProvider{
		EntityID:          cfg.EntityID,
		AcsURL:            *acs,
		IDPMetadata:       idpMeta,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
		AllowIDPInitiated: cfg.AllowIDPInitiated,
	}

	// The SP keypair is only needed to sign AuthnRequests / decrypt assertions.
	if cfg.SPCertificatePEM != "" && cfg.SPPrivateKeyPEM != "" {
		keypair, kerr := tls.X509KeyPair([]byte(cfg.SPCertificatePEM), []byte(cfg.SPPrivateKeyPEM))
		if kerr != nil {
			return nil, fmt.Errorf("sso/saml: invalid SP keypair: %w", kerr)
		}
		leaf, cerr := x509.ParseCertificate(keypair.Certificate[0])
		if cerr != nil {
			return nil, fmt.Errorf("sso/saml: invalid SP certificate: %w", cerr)
		}
		key, ok := keypair.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("sso/saml: SP private key must be RSA")
		}
		sp.Certificate = leaf
		sp.Key = key
		if cfg.SignRequests {
			sp.SignatureMethod = dsig.RSASHA256SignatureMethod
		}
	}

	return &samlProvider{name: cfg.Name, sp: sp, attrMap: cfg.AttributeMap}, nil
}

func (p *samlProvider) Name() string     { return p.name }
func (p *samlProvider) Protocol() string { return "saml" }

// LoginURL builds a signed (when configured) AuthnRequest and returns the IdP
// HTTP-Redirect URL carrying the request + RelayState (= state).
func (p *samlProvider) LoginURL(state string) (string, error) {
	u, _, err := p.LoginURLWithRequestID(state)
	return u, err
}

// LoginURLWithRequestID is LoginURL that also returns the generated AuthnRequest
// ID. The caller persists it (keyed by state) and passes it back at the ACS so
// ParseXMLResponse can match the assertion's InResponseTo — without this the SP
// has no request IDs to validate against and every solicited assertion is
// rejected. Satisfies the optional requestIDProvider interface in the plugin.
func (p *samlProvider) LoginURLWithRequestID(state string) (string, string, error) {
	loc := p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	if loc == "" {
		return "", "", fmt.Errorf("sso/saml: IdP exposes no HTTP-Redirect SSO endpoint")
	}
	req, err := p.sp.MakeAuthenticationRequest(loc, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", fmt.Errorf("sso/saml: build AuthnRequest: %w", err)
	}
	u, err := req.Redirect(state, p.sp)
	if err != nil {
		return "", "", fmt.Errorf("sso/saml: build redirect: %w", err)
	}
	return u.String(), req.ID, nil
}

// HandleCallback validates the SAMLResponse (signature against the pinned IdP
// cert, conditions, audience, recipient, timestamps) and maps it to a User.
func (p *samlProvider) HandleCallback(_ context.Context, params map[string]string) (*User, error) {
	raw := params["SAMLResponse"]
	if raw == "" {
		return nil, fmt.Errorf("sso/saml: missing SAMLResponse")
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("sso/saml: decode SAMLResponse: %w", err)
	}
	// Match the assertion's InResponseTo against the AuthnRequest ID we minted at
	// login (threaded back via params["request_id"]). Without a request ID this
	// is a no-match unless AllowIDPInitiated is set — crewjam enforces that. The
	// signature, conditions, audience, recipient, and timestamps are validated
	// regardless.
	var possibleRequestIDs []string
	if reqID := params["request_id"]; reqID != "" {
		possibleRequestIDs = []string{reqID}
	}
	assertion, err := p.sp.ParseXMLResponse(decoded, possibleRequestIDs, p.sp.AcsURL)
	if err != nil {
		return nil, fmt.Errorf("sso/saml: invalid assertion: %w", err)
	}
	return assertionToUser(assertion, p.attrMap)
}

// Metadata returns the SP metadata XML for the IdP to consume.
func (p *samlProvider) Metadata() ([]byte, string, error) {
	out, err := xml.MarshalIndent(p.sp.Metadata(), "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("sso/saml: marshal metadata: %w", err)
	}
	return out, "application/samlmetadata+xml", nil
}

// ──────────────────────────────────────────────────
// IdP metadata resolution
// ──────────────────────────────────────────────────

func resolveIDPMetadata(cfg SAMLConfig) (*saml.EntityDescriptor, error) {
	switch {
	case strings.TrimSpace(cfg.IDPMetadataXML) != "":
		return samlsp.ParseMetadata([]byte(cfg.IDPMetadataXML))
	case strings.TrimSpace(cfg.MetadataURL) != "":
		u, err := url.Parse(cfg.MetadataURL)
		if err != nil {
			return nil, fmt.Errorf("sso/saml: invalid metadata_url: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), metadataFetchTimeout)
		defer cancel()
		md, err := samlsp.FetchMetadata(ctx, http.DefaultClient, *u)
		if err != nil {
			return nil, fmt.Errorf("sso/saml: fetch metadata: %w", err)
		}
		return md, nil
	case strings.TrimSpace(cfg.IDPSSOURL) != "" && strings.TrimSpace(cfg.IDPCertificatePEM) != "":
		return buildIDPMetadata(cfg.IDPSSOURL, cfg.IDPCertificatePEM)
	default:
		return nil, fmt.Errorf("sso/saml: IdP config required (metadata_url, idp_metadata_xml, or idp_sso_url + idp_certificate)")
	}
}

// buildIDPMetadata synthesizes a minimal IdP EntityDescriptor from an SSO URL
// and a signing certificate — for IdPs configured by pasted cert rather than
// discoverable metadata.
func buildIDPMetadata(ssoURL, certPEM string) (*saml.EntityDescriptor, error) {
	certDER, err := certDERFromPEM(certPEM)
	if err != nil {
		return nil, err
	}
	return &saml.EntityDescriptor{
		EntityID: ssoURL,
		IDPSSODescriptors: []saml.IDPSSODescriptor{{
			SSODescriptor: saml.SSODescriptor{
				RoleDescriptor: saml.RoleDescriptor{
					KeyDescriptors: []saml.KeyDescriptor{{
						Use: "signing",
						KeyInfo: saml.KeyInfo{X509Data: saml.X509Data{
							X509Certificates: []saml.X509Certificate{{Data: certDER}},
						}},
					}},
				},
			},
			SingleSignOnServices: []saml.Endpoint{{
				Binding:  saml.HTTPRedirectBinding,
				Location: ssoURL,
			}},
		}},
	}, nil
}

// certDERFromPEM returns the base64-encoded DER (metadata's X509Certificate
// form) of a PEM certificate.
func certDERFromPEM(certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", fmt.Errorf("sso/saml: invalid certificate PEM")
	}
	return base64.StdEncoding.EncodeToString(block.Bytes), nil
}

// ──────────────────────────────────────────────────
// Assertion → User mapping
// ──────────────────────────────────────────────────

const (
	fieldEmail     = "email"
	fieldFirstName = "first_name"
	fieldLastName  = "last_name"
	fieldGroups    = "groups"
)

// defaultAttrAliases maps common IdP attribute names (lowercased) to user
// fields, covering the friendly, WS-Fed claim, and LDAP OID forms.
var defaultAttrAliases = map[string]string{
	"email":        fieldEmail,
	"emailaddress": fieldEmail,
	"mail":         fieldEmail,
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": fieldEmail,
	"urn:oid:0.9.2342.19200300.100.1.3":                                  fieldEmail,
	"firstname":                                                          fieldFirstName,
	"givenname":                                                          fieldFirstName,
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname": fieldFirstName,
	"urn:oid:2.5.4.42": fieldFirstName,
	"lastname":         fieldLastName,
	"surname":          fieldLastName,
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname": fieldLastName,
	"urn:oid:2.5.4.4": fieldLastName,
	"groups":          fieldGroups,
	"memberof":        fieldGroups,
	"http://schemas.xmlsoap.org/claims/group": fieldGroups,
}

// assertionToUser extracts identity from a validated assertion. Pure and
// side-effect-free so it can be unit-tested without SAML crypto.
func assertionToUser(a *saml.Assertion, attrMap map[string]string) (*User, error) {
	if a == nil || a.Subject == nil || a.Subject.NameID == nil {
		return nil, fmt.Errorf("sso/saml: assertion missing subject NameID")
	}
	nameID := strings.TrimSpace(a.Subject.NameID.Value)

	u := &User{ProviderUserID: nameID, Attributes: map[string]string{}}
	for _, stmt := range a.AttributeStatements {
		for _, attr := range stmt.Attributes {
			vals := attrValues(attr)
			if len(vals) == 0 {
				continue
			}
			if attr.Name != "" {
				u.Attributes[attr.Name] = vals[0]
			}
			switch mapSAMLAttr(attr, attrMap) {
			case fieldEmail:
				u.Email = vals[0]
			case fieldFirstName:
				u.FirstName = vals[0]
			case fieldLastName:
				u.LastName = vals[0]
			case fieldGroups:
				u.Groups = append(u.Groups, vals...)
			}
		}
	}

	if u.Email == "" && looksLikeEmail(nameID) {
		u.Email = nameID
	}
	return u, nil
}

// mapSAMLAttr resolves an attribute to a user field, preferring an explicit
// per-connection mapping over the built-in aliases.
func mapSAMLAttr(attr saml.Attribute, attrMap map[string]string) string {
	for _, name := range []string{attr.Name, attr.FriendlyName} {
		if name == "" {
			continue
		}
		if attrMap != nil {
			if f := normalizeField(attrMap[name]); f != "" {
				return f
			}
		}
		if f, ok := defaultAttrAliases[strings.ToLower(name)]; ok {
			return f
		}
	}
	return ""
}

func normalizeField(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "email", "email_address", "emailaddress":
		return fieldEmail
	case "first_name", "firstname", "given_name", "givenname":
		return fieldFirstName
	case "last_name", "lastname", "surname":
		return fieldLastName
	case "groups", "group":
		return fieldGroups
	default:
		return ""
	}
}

func attrValues(attr saml.Attribute) []string {
	out := make([]string, 0, len(attr.Values))
	for _, v := range attr.Values {
		if s := strings.TrimSpace(v.Value); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1
}

// ──────────────────────────────────────────────────
// SP keypair generation
// ──────────────────────────────────────────────────

// generateSPKeypair creates a self-signed RSA-2048 SP keypair, returned as PEM.
// Used at connection-create time when the admin supplies no SP keypair.
func generateSPKeypair(entityID string) (certPEM, keyPEM string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: entityID},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	return certPEM, keyPEM, nil
}
