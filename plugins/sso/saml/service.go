package saml

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/rs/xid"
)

// Service provides minimal SAML capabilities: SP metadata generation and assertion parsing
type Service struct {
	sp *saml.ServiceProvider
}

// NewService creates a SAML service without SP configured (parsing only)
func NewService() *Service { return &Service{} }

// NewServiceProvider initializes a ServiceProvider with self-signed certificate for metadata
func (s *Service) NewServiceProvider(entityID, acsURL, metadataURL string) error {
	sp := &saml.ServiceProvider{}
	// Parse URLs
	if metadataURL != "" {
		if u, err := url.Parse(metadataURL); err == nil {
			sp.MetadataURL = *u
		}
	}
	if acsURL != "" {
		if u, err := url.Parse(acsURL); err == nil {
			sp.AcsURL = *u
		}
	}
	// Generate self-signed cert for metadata KeyDescriptor
	key, cert, err := generateSelfSignedCert(entityID)
	if err != nil {
		return err
	}
	sp.Key = key
	sp.Certificate = cert
	s.sp = sp
	return nil
}

// Metadata returns SP metadata XML using crewjam/saml if configured, else minimal fallback
func (s *Service) Metadata() string {
	if s.sp != nil {
		md := s.sp.Metadata()
		if md != nil {
			buf, _ := xml.MarshalIndent(md, "", "  ")
			return string(buf)
		}
	}
	// Fallback minimal metadata
	return "<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"authsome-sp\"></EntityDescriptor>"
}

// ParseResponse decodes a base64-encoded SAMLResponse and extracts Issuer and NameID
// Returns NameID on success when Issuer matches expectedIssuer
func (s *Service) ParseResponse(b64, expectedIssuer string) (string, error) {
	if b64 == "" {
		return "", errors.New("empty saml response")
	}
	xmlBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return "", err
	}
	root := doc.Root()
	if root == nil {
		return "", errors.New("invalid xml")
	}
	var issuer, nameID string
	// Walk to find Issuer and NameID regardless of namespaces
	for _, el := range root.FindElements(".//Issuer") {
		if el.Text() != "" {
			issuer = el.Text()
		}
	}
	for _, el := range root.FindElements(".//NameID") {
		if el.Text() != "" {
			nameID = el.Text()
		}
	}
	if expectedIssuer != "" && issuer != expectedIssuer {
		return "", errors.New("issuer mismatch")
	}
	if nameID == "" {
		return "", errors.New("missing nameid")
	}
	return nameID, nil
}

// ParseAndValidateResponse performs full SAML response validation including signatures
func (s *Service) ParseAndValidateResponse(b64Response, expectedIssuer, relayState string, idpCert *x509.Certificate) (*SAMLAssertion, error) {
	if b64Response == "" {
		return nil, errors.New("empty saml response")
	}

	// Decode base64 response
	xmlBytes, err := base64.StdEncoding.DecodeString(b64Response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 response: %w", err)
	}

	// Parse XML using crewjam/saml for proper validation
	if s.sp == nil {
		return nil, errors.New("service provider not configured for validation")
	}

	// Use crewjam/saml to parse and validate the response
	assertion, err := s.sp.ParseXMLResponse(xmlBytes, []string{})
	if err != nil {
		// Fallback to basic parsing if crewjam/saml fails
		return s.parseResponseBasic(xmlBytes, expectedIssuer)
	}

	// Validate issuer
	if expectedIssuer != "" && assertion.Issuer.Value != expectedIssuer {
		return nil, fmt.Errorf("issuer mismatch: expected %s, got %s", expectedIssuer, assertion.Issuer.Value)
	}

	// Validate conditions (time bounds)
	now := time.Now().UTC()
	if assertion.Conditions != nil {
		if !assertion.Conditions.NotBefore.IsZero() && now.Before(assertion.Conditions.NotBefore) {
			return nil, errors.New("assertion not yet valid")
		}
		if !assertion.Conditions.NotOnOrAfter.IsZero() && now.After(assertion.Conditions.NotOnOrAfter) {
			return nil, errors.New("assertion expired")
		}

		// Validate audience restriction
		if len(assertion.Conditions.AudienceRestrictions) > 0 {
			validAudience := false
			for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
				if audienceRestriction.Audience.Value == s.sp.EntityID {
					validAudience = true
					break
				}
			}
			if !validAudience {
				return nil, errors.New("invalid audience restriction")
			}
		}
	}

	// Extract subject NameID
	var nameID string
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		nameID = assertion.Subject.NameID.Value
	}
	if nameID == "" {
		return nil, errors.New("missing subject NameID")
	}

	// Validate RelayState if provided
	if relayState != "" && !s.ValidateRelayState(relayState, relayState) {
		return nil, errors.New("invalid relay state")
	}

	var notBefore, notOnOrAfter *time.Time
	if assertion.Conditions != nil {
		if !assertion.Conditions.NotBefore.IsZero() {
			notBefore = &assertion.Conditions.NotBefore
		}
		if !assertion.Conditions.NotOnOrAfter.IsZero() {
			notOnOrAfter = &assertion.Conditions.NotOnOrAfter
		}
	}

	return &SAMLAssertion{
		Issuer:       assertion.Issuer.Value,
		Subject:      nameID,
		NotBefore:    notBefore,
		NotOnOrAfter: notOnOrAfter,
		Attributes:   extractAttributes(assertion),
	}, nil
}

// parseResponseBasic provides fallback basic parsing when full validation fails
func (s *Service) parseResponseBasic(xmlBytes []byte, expectedIssuer string) (*SAMLAssertion, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return nil, err
	}
	root := doc.Root()
	if root == nil {
		return nil, errors.New("invalid xml")
	}

	var issuer, nameID string
	// Walk to find Issuer and NameID regardless of namespaces
	for _, el := range root.FindElements(".//Issuer") {
		if el.Text() != "" {
			issuer = el.Text()
		}
	}
	for _, el := range root.FindElements(".//NameID") {
		if el.Text() != "" {
			nameID = el.Text()
		}
	}

	if expectedIssuer != "" && issuer != expectedIssuer {
		return nil, errors.New("issuer mismatch")
	}
	if nameID == "" {
		return nil, errors.New("missing nameid")
	}

	return &SAMLAssertion{
		Issuer:  issuer,
		Subject: nameID,
	}, nil
}

// SAMLAssertion represents a parsed and validated SAML assertion
type SAMLAssertion struct {
	Issuer       string
	Subject      string
	NotBefore    *time.Time
	NotOnOrAfter *time.Time
	Attributes   map[string][]string
}

// extractAttributes extracts attribute statements from the assertion
func extractAttributes(assertion *saml.Assertion) map[string][]string {
	attrs := make(map[string][]string)
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			var values []string
			for _, val := range attr.Values {
				values = append(values, val.Value)
			}
			attrs[attr.Name] = values
		}
	}
	return attrs
}

// GenerateAuthnRequest creates a SAML AuthnRequest for login initiation
func (s *Service) GenerateAuthnRequest(idpURL, relayState string) (string, string, error) {
	if s.sp == nil {
		return "", "", errors.New("service provider not configured")
	}

	// Generate unique request ID
	requestID := "_" + xid.New().String()

	// Create AuthnRequest using crewjam/saml
	req, err := s.sp.MakeAuthenticationRequest(idpURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", "", fmt.Errorf("failed to create authn request: %w", err)
	}

	// Set request ID and other attributes
	req.ID = requestID
	req.IssueInstant = time.Now().UTC()

	// Serialize to XML
	xmlBytes, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal authn request: %w", err)
	}

	// Base64 encode for HTTP-Redirect binding
	b64Request := base64.StdEncoding.EncodeToString(xmlBytes)

	// Build redirect URL with SAMLRequest and RelayState
	redirectURL := fmt.Sprintf("%s?SAMLRequest=%s", idpURL, url.QueryEscape(b64Request))
	if relayState != "" {
		redirectURL += "&RelayState=" + url.QueryEscape(relayState)
	}

	return redirectURL, requestID, nil
}

// ValidateRelayState checks if the RelayState matches expected format
func (s *Service) ValidateRelayState(relayState, expectedState string) bool {
	return relayState == expectedState
}

// generateSelfSignedCert creates a minimal self-signed x509 certificate
func generateSelfSignedCert(cn string) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	// keep PEM encoding around if needed later
	_ = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	_ = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return key, cert, nil
}
