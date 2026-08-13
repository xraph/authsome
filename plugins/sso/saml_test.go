package sso

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/crewjam/saml"
)

// testIDPCertPEM returns a self-signed certificate PEM usable as a stand-in
// IdP signing certificate. Reuses the SP keypair generator since a cert is a
// cert — the private key is discarded.
func testIDPCertPEM(t *testing.T) string {
	t.Helper()
	certPEM, _, err := generateSPKeypair("https://idp.example.com")
	if err != nil {
		t.Fatalf("generate test cert: %v", err)
	}
	return certPEM
}

// baseSAMLConfig is a minimal valid config (IdP by SSO URL + cert).
func baseSAMLConfig(t *testing.T) SAMLConfig {
	t.Helper()
	return SAMLConfig{
		Name:              "okta",
		IDPSSOURL:         "https://idp.example.com/sso",
		IDPCertificatePEM: testIDPCertPEM(t),
		EntityID:          "https://sp.example.com/v1/sso/okta/metadata",
		ACSURL:            "https://sp.example.com/v1/sso/okta/acs",
	}
}

func TestGenerateSPKeypair(t *testing.T) {
	certPEM, keyPEM, err := generateSPKeypair("https://sp.example.com")
	if err != nil {
		t.Fatalf("generateSPKeypair: %v", err)
	}
	if !strings.Contains(certPEM, "BEGIN CERTIFICATE") {
		t.Errorf("cert PEM missing certificate block:\n%s", certPEM)
	}
	if !strings.Contains(keyPEM, "PRIVATE KEY") {
		t.Errorf("key PEM missing private key block")
	}
	// The pair must load as a valid X509 keypair.
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		t.Errorf("X509KeyPair: %v", err)
	}
}

func TestNewSAMLProvider_validation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*SAMLConfig)
		wantErr string
	}{
		{
			name:    "missing entity_id",
			mutate:  func(c *SAMLConfig) { c.EntityID = "" },
			wantErr: "entity_id and acs_url are required",
		},
		{
			name:    "missing acs_url",
			mutate:  func(c *SAMLConfig) { c.ACSURL = "" },
			wantErr: "entity_id and acs_url are required",
		},
		{
			name: "no IdP source",
			mutate: func(c *SAMLConfig) {
				c.IDPSSOURL = ""
				c.IDPCertificatePEM = ""
			},
			wantErr: "IdP config required",
		},
		{
			name:    "sso url without certificate",
			mutate:  func(c *SAMLConfig) { c.IDPCertificatePEM = "" },
			wantErr: "IdP config required",
		},
		{
			name:    "bad idp certificate",
			mutate:  func(c *SAMLConfig) { c.IDPCertificatePEM = "not-a-pem" },
			wantErr: "invalid certificate PEM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseSAMLConfig(t)
			tt.mutate(&cfg)
			_, err := NewSAMLProvider(cfg)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewSAMLProvider_ok(t *testing.T) {
	p, err := NewSAMLProvider(baseSAMLConfig(t))
	if err != nil {
		t.Fatalf("NewSAMLProvider: %v", err)
	}
	if p.Name() != "okta" {
		t.Errorf("Name() = %q, want okta", p.Name())
	}
	if p.Protocol() != "saml" {
		t.Errorf("Protocol() = %q, want saml", p.Protocol())
	}
	if _, ok := p.(SAMLMetadataProvider); !ok {
		t.Errorf("provider does not implement SAMLMetadataProvider")
	}
}

func TestSAMLProvider_LoginURL(t *testing.T) {
	tests := []struct {
		name        string
		sign        bool
		wantSignYes bool
	}{
		{name: "unsigned", sign: false, wantSignYes: false},
		{name: "signed", sign: true, wantSignYes: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseSAMLConfig(t)
			if tt.sign {
				certPEM, keyPEM, err := generateSPKeypair(cfg.EntityID)
				if err != nil {
					t.Fatalf("keypair: %v", err)
				}
				cfg.SPCertificatePEM = certPEM
				cfg.SPPrivateKeyPEM = keyPEM
				cfg.SignRequests = true
			}
			p, err := NewSAMLProvider(cfg)
			if err != nil {
				t.Fatalf("NewSAMLProvider: %v", err)
			}

			raw, err := p.LoginURL("state-xyz")
			if err != nil {
				t.Fatalf("LoginURL: %v", err)
			}
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse login URL: %v", err)
			}
			if u.Host != "idp.example.com" {
				t.Errorf("redirect host = %q, want idp.example.com", u.Host)
			}
			q := u.Query()
			if q.Get("SAMLRequest") == "" {
				t.Errorf("login URL missing SAMLRequest: %s", raw)
			}
			if q.Get("RelayState") != "state-xyz" {
				t.Errorf("RelayState = %q, want state-xyz", q.Get("RelayState"))
			}
			if got := q.Get("Signature") != ""; got != tt.wantSignYes {
				t.Errorf("has Signature = %v, want %v", got, tt.wantSignYes)
			}
		})
	}
}

func TestSAMLProvider_Metadata(t *testing.T) {
	p, err := NewSAMLProvider(baseSAMLConfig(t))
	if err != nil {
		t.Fatalf("NewSAMLProvider: %v", err)
	}
	mp, ok := p.(SAMLMetadataProvider)
	if !ok {
		t.Fatalf("provider is not a SAMLMetadataProvider")
	}
	xmlBytes, contentType, err := mp.Metadata()
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if contentType != "application/samlmetadata+xml" {
		t.Errorf("content type = %q", contentType)
	}
	body := string(xmlBytes)
	for _, want := range []string{
		"https://sp.example.com/v1/sso/okta/metadata", // EntityID
		"https://sp.example.com/v1/sso/okta/acs",      // ACS URL
		"EntityDescriptor",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metadata missing %q\n%s", want, body)
		}
	}
}

func TestAssertionToUser(t *testing.T) {
	tests := []struct {
		name      string
		assertion *saml.Assertion
		attrMap   map[string]string
		want      User
		wantErr   bool
	}{
		{
			name: "friendly attribute names",
			assertion: buildAssertion("user@corp.com", map[string][]string{
				"email":     {"user@corp.com"},
				"givenName": {"Ada"},
				"surname":   {"Lovelace"},
				"groups":    {"admins", "eng"},
			}),
			want: User{
				ProviderUserID: "user@corp.com",
				Email:          "user@corp.com",
				FirstName:      "Ada",
				LastName:       "Lovelace",
				Groups:         []string{"admins", "eng"},
			},
		},
		{
			name: "urn oid attribute names",
			assertion: buildAssertion("id-123", map[string][]string{
				"urn:oid:0.9.2342.19200300.100.1.3": {"person@corp.com"},
				"urn:oid:2.5.4.42":                  {"Grace"},
				"urn:oid:2.5.4.4":                   {"Hopper"},
			}),
			want: User{
				ProviderUserID: "id-123",
				Email:          "person@corp.com",
				FirstName:      "Grace",
				LastName:       "Hopper",
			},
		},
		{
			name:      "nameid email fallback",
			assertion: buildAssertion("fallback@corp.com", nil),
			want: User{
				ProviderUserID: "fallback@corp.com",
				Email:          "fallback@corp.com",
			},
		},
		{
			name:      "opaque nameid no fallback",
			assertion: buildAssertion("opaque-nameid", nil),
			want: User{
				ProviderUserID: "opaque-nameid",
			},
		},
		{
			name: "custom attribute mapping",
			assertion: buildAssertion("u1", map[string][]string{
				"mailPrimary": {"mapped@corp.com"},
			}),
			attrMap: map[string]string{"mailPrimary": "email"},
			want: User{
				ProviderUserID: "u1",
				Email:          "mapped@corp.com",
			},
		},
		{
			name:      "nil subject errors",
			assertion: &saml.Assertion{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := assertionToUser(tt.assertion, tt.attrMap)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("assertionToUser: %v", err)
			}
			if got.ProviderUserID != tt.want.ProviderUserID {
				t.Errorf("ProviderUserID = %q, want %q", got.ProviderUserID, tt.want.ProviderUserID)
			}
			if got.Email != tt.want.Email {
				t.Errorf("Email = %q, want %q", got.Email, tt.want.Email)
			}
			if got.FirstName != tt.want.FirstName {
				t.Errorf("FirstName = %q, want %q", got.FirstName, tt.want.FirstName)
			}
			if got.LastName != tt.want.LastName {
				t.Errorf("LastName = %q, want %q", got.LastName, tt.want.LastName)
			}
			if strings.Join(got.Groups, ",") != strings.Join(tt.want.Groups, ",") {
				t.Errorf("Groups = %v, want %v", got.Groups, tt.want.Groups)
			}
		})
	}
}

func TestResolveIDPMetadata(t *testing.T) {
	certPEM := testIDPCertPEM(t)

	t.Run("from sso url and cert", func(t *testing.T) {
		md, err := resolveIDPMetadata(SAMLConfig{
			IDPSSOURL:         "https://idp.example.com/sso",
			IDPCertificatePEM: certPEM,
		})
		if err != nil {
			t.Fatalf("resolveIDPMetadata: %v", err)
		}
		if len(md.IDPSSODescriptors) != 1 {
			t.Fatalf("expected 1 IDPSSODescriptor, got %d", len(md.IDPSSODescriptors))
		}
		svcs := md.IDPSSODescriptors[0].SingleSignOnServices
		if len(svcs) != 1 || svcs[0].Location != "https://idp.example.com/sso" {
			t.Errorf("unexpected SSO services: %+v", svcs)
		}
	})

	t.Run("from pasted metadata xml", func(t *testing.T) {
		// Round-trip: build metadata, marshal, re-parse.
		md, err := buildIDPMetadata("https://idp.example.com/sso", certPEM)
		if err != nil {
			t.Fatalf("buildIDPMetadata: %v", err)
		}
		xmlBytes, err := xml.Marshal(md)
		if err != nil {
			t.Fatalf("marshal metadata: %v", err)
		}
		got, err := resolveIDPMetadata(SAMLConfig{IDPMetadataXML: string(xmlBytes)})
		if err != nil {
			t.Fatalf("resolveIDPMetadata(xml): %v", err)
		}
		if got.EntityID == "" {
			t.Errorf("parsed metadata has empty EntityID")
		}
	})

	t.Run("no source errors", func(t *testing.T) {
		if _, err := resolveIDPMetadata(SAMLConfig{}); err == nil {
			t.Errorf("expected error for empty config")
		}
	})
}

// samlE2EFixture wires a real crewjam IdP (with its own signing keypair) to an
// SP built by NewSAMLProvider and pinned to that IdP's certificate, so a minted
// assertion round-trips through HandleCallback → ParseXMLResponse for real:
// signature, audience, recipient, and timestamps are all genuinely validated.
type samlE2EFixture struct {
	sp  *samlProvider
	idp *saml.IdentityProvider
}

// spProviderShim adapts the fixture's SP metadata to saml.ServiceProviderProvider.
type spProviderShim struct{ md *saml.EntityDescriptor }

func (s spProviderShim) GetServiceProvider(_ *http.Request, _ string) (*saml.EntityDescriptor, error) {
	return s.md, nil
}

func newSAMLE2EFixture(t *testing.T, allowIDPInitiated bool) *samlE2EFixture {
	t.Helper()

	// IdP signing keypair. The same cert is pinned into the SP so signature
	// validation exercises the real path.
	idpCertPEM, idpKeyPEM, err := generateSPKeypair("https://idp.example.com")
	if err != nil {
		t.Fatalf("generate idp keypair: %v", err)
	}
	keypair, err := tls.X509KeyPair([]byte(idpCertPEM), []byte(idpKeyPEM))
	if err != nil {
		t.Fatalf("load idp keypair: %v", err)
	}
	idpCert, err := x509.ParseCertificate(keypair.Certificate[0])
	if err != nil {
		t.Fatalf("parse idp cert: %v", err)
	}
	idpKey, ok := keypair.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("idp key is not RSA")
	}

	cfg := SAMLConfig{
		Name:              "okta",
		IDPSSOURL:         "https://idp.example.com/sso",
		IDPCertificatePEM: idpCertPEM,
		EntityID:          "https://sp.example.com/v1/sso/okta/metadata",
		ACSURL:            "https://sp.example.com/v1/sso/okta/acs",
		AllowIDPInitiated: allowIDPInitiated,
	}
	prov, err := NewSAMLProvider(cfg)
	if err != nil {
		t.Fatalf("NewSAMLProvider: %v", err)
	}
	sp := prov.(*samlProvider)

	// buildIDPMetadata (used inside NewSAMLProvider from IDPSSOURL) sets the
	// pinned IdP EntityID to the SSO URL, and crewjam sets a response's Issuer to
	// the IdP's MetadataURL. They must match or ParseXMLResponse rejects the
	// Issuer — so point MetadataURL at the SSO URL too.
	ssoURL, _ := url.Parse("https://idp.example.com/sso")
	idp := &saml.IdentityProvider{
		Key:                     idpKey,
		Certificate:             idpCert,
		MetadataURL:             *ssoURL,
		SSOURL:                  *ssoURL,
		ServiceProviderProvider: spProviderShim{md: sp.sp.Metadata()},
	}

	return &samlE2EFixture{sp: sp, idp: idp}
}

// mintResponse drives the SP's AuthnRequest through the test IdP and returns the
// base64 SAMLResponse the IdP would POST to the ACS, plus the AuthnRequest ID.
func (f *samlE2EFixture) mintResponse(t *testing.T, state string) (samlResponse, requestID string) {
	t.Helper()

	loginURL, reqID, err := f.sp.LoginURLWithRequestID(state)
	if err != nil {
		t.Fatalf("LoginURLWithRequestID: %v", err)
	}

	// Replay the redirect at the IdP as an inbound AuthnRequest.
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, loginURL, nil)
	authnReq, err := saml.NewIdpAuthnRequest(f.idp, r)
	if err != nil {
		t.Fatalf("NewIdpAuthnRequest: %v", err)
	}
	if err = authnReq.Validate(); err != nil {
		t.Fatalf("authn request validate: %v", err)
	}

	session := &saml.Session{
		ID:            "sess-1",
		NameID:        "user@corp.com",
		UserEmail:     "user@corp.com",
		UserGivenName: "Ada",
		UserSurname:   "Lovelace",
	}
	if err = (saml.DefaultAssertionMaker{}).MakeAssertion(authnReq, session); err != nil {
		t.Fatalf("MakeAssertion: %v", err)
	}
	form, err := authnReq.PostBinding()
	if err != nil {
		t.Fatalf("PostBinding: %v", err)
	}
	return form.SAMLResponse, reqID
}

// TestSAMLProvider_HandleCallback_e2e exercises the full assertion round-trip
// that the request-ID threading fix targets. Without a matching request ID a
// solicited (SP-initiated) assertion must be rejected; threading the ID minted
// at login through must let it validate. This is the regression guard for the
// bug where HandleCallback passed an empty possibleRequestIDs and every SAML
// login failed.
func TestSAMLProvider_HandleCallback_e2e(t *testing.T) {
	t.Run("valid request_id succeeds (SP-initiated)", func(t *testing.T) {
		f := newSAMLE2EFixture(t, false)
		resp, reqID := f.mintResponse(t, "state-abc")

		u, err := f.sp.HandleCallback(context.Background(), map[string]string{
			"SAMLResponse": resp,
			"request_id":   reqID,
		})
		if err != nil {
			t.Fatalf("HandleCallback with matching request_id: %v", err)
		}
		if u.Email != "user@corp.com" {
			t.Errorf("Email = %q, want user@corp.com", u.Email)
		}
		if u.FirstName != "Ada" || u.LastName != "Lovelace" {
			t.Errorf("name = %q %q, want Ada Lovelace", u.FirstName, u.LastName)
		}
	})

	t.Run("missing request_id rejected (no IdP-initiated)", func(t *testing.T) {
		f := newSAMLE2EFixture(t, false)
		resp, _ := f.mintResponse(t, "state-def")

		if _, err := f.sp.HandleCallback(context.Background(), map[string]string{
			"SAMLResponse": resp,
		}); err == nil {
			t.Fatal("expected rejection when request_id is absent and IdP-initiated is off")
		}
	})

	t.Run("wrong request_id rejected", func(t *testing.T) {
		f := newSAMLE2EFixture(t, false)
		resp, _ := f.mintResponse(t, "state-ghi")

		if _, err := f.sp.HandleCallback(context.Background(), map[string]string{
			"SAMLResponse": resp,
			"request_id":   "id-does-not-match",
		}); err == nil {
			t.Fatal("expected rejection when request_id does not match InResponseTo")
		}
	})

	t.Run("IdP-initiated accepted when allowed", func(t *testing.T) {
		f := newSAMLE2EFixture(t, true)
		resp, _ := f.mintResponse(t, "state-jkl")

		u, err := f.sp.HandleCallback(context.Background(), map[string]string{
			"SAMLResponse": resp,
		})
		if err != nil {
			t.Fatalf("HandleCallback with AllowIDPInitiated: %v", err)
		}
		if u.Email != "user@corp.com" {
			t.Errorf("Email = %q, want user@corp.com", u.Email)
		}
	})
}

// buildAssertion constructs a validated-shape assertion for mapping tests.
// attrs maps attribute Name → values.
func buildAssertion(nameID string, attrs map[string][]string) *saml.Assertion {
	a := &saml.Assertion{
		Subject: &saml.Subject{NameID: &saml.NameID{Value: nameID}},
	}
	if len(attrs) == 0 {
		return a
	}
	stmt := saml.AttributeStatement{}
	for name, vals := range attrs {
		attr := saml.Attribute{Name: name}
		for _, v := range vals {
			attr.Values = append(attr.Values, saml.AttributeValue{Value: v})
		}
		stmt.Attributes = append(stmt.Attributes, attr)
	}
	a.AttributeStatements = []saml.AttributeStatement{stmt}
	return a
}
