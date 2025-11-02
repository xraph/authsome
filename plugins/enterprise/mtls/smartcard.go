package mtls

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"time"
)

// SmartCardProvider handles PIV/CAC smart card authentication
type SmartCardProvider struct {
	config *Config
	repo   Repository
}

// NewSmartCardProvider creates a new smart card provider
func NewSmartCardProvider(config *Config, repo Repository) *SmartCardProvider {
	return &SmartCardProvider{
		config: config,
		repo:   repo,
	}
}

// PIVCardInfo contains PIV card information
type PIVCardInfo struct {
	CardID           string            `json:"cardId"`
	CardholderUUID   string            `json:"cardholderUuid,omitempty"`
	ExpirationDate   *time.Time        `json:"expirationDate,omitempty"`
	Certificates     []PIVCertificate  `json:"certificates"`
	PINPolicy        PIVPINPolicy      `json:"pinPolicy"`
	ReaderName       string            `json:"readerName,omitempty"`
}

// PIVCertificate represents a certificate slot on a PIV card
type PIVCertificate struct {
	SlotID      string `json:"slotId"` // 9A, 9C, 9D, 9E
	SlotName    string `json:"slotName"` // Authentication, Digital Signature, Key Management, Card Authentication
	Certificate *x509.Certificate `json:"-"`
	Fingerprint string `json:"fingerprint"`
}

// PIVPINPolicy defines PIN requirements
type PIVPINPolicy struct {
	PINRequired    bool `json:"pinRequired"`
	PINMinLength   int  `json:"pinMinLength"`
	PINMaxLength   int  `json:"pinMaxLength"`
	Retries        int  `json:"retries"`
	PINVerified    bool `json:"pinVerified"`
}

// CACCardInfo contains CAC card information
type CACCardInfo struct {
	CardID         string            `json:"cardId"`
	CACNumber      string            `json:"cacNumber"`
	PersonDN       string            `json:"personDn,omitempty"`
	Certificates   []CACCertificate  `json:"certificates"`
	IssueDate      *time.Time        `json:"issueDate,omitempty"`
	ExpirationDate *time.Time        `json:"expirationDate,omitempty"`
	ReaderName     string            `json:"readerName,omitempty"`
}

// CACCertificate represents a certificate on a CAC
type CACCertificate struct {
	CertificateType string `json:"certificateType"` // ID, Email, Signature, Encryption
	Certificate     *x509.Certificate `json:"-"`
	Fingerprint     string `json:"fingerprint"`
}

// SmartCardAuthRequest contains authentication request data
type SmartCardAuthRequest struct {
	OrganizationID string                 `json:"organizationId"`
	CardType       string                 `json:"cardType"` // piv, cac
	ReaderName     string                 `json:"readerName,omitempty"`
	PIN            string                 `json:"pin,omitempty"`
	CertificateSlot string                `json:"certificateSlot,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SmartCardAuthResponse contains authentication response
type SmartCardAuthResponse struct {
	Success        bool                   `json:"success"`
	UserID         string                 `json:"userId,omitempty"`
	CertificateID  string                 `json:"certificateId,omitempty"`
	CardInfo       interface{}            `json:"cardInfo,omitempty"`
	ValidationResult *ValidationResult    `json:"validationResult,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// ValidatePIVCard validates a PIV card and extracts certificate
func (s *SmartCardProvider) ValidatePIVCard(ctx context.Context, cert *x509.Certificate) (*PIVCardInfo, error) {
	if !s.config.SmartCard.EnablePIV {
		return nil, fmt.Errorf("PIV support is not enabled")
	}
	
	// Verify this is a PIV certificate
	if !isPIVCertificate(cert) {
		return nil, ErrNotPIVCertificate
	}
	
	// Extract PIV-specific information from certificate
	cardInfo := &PIVCardInfo{
		Certificates: []PIVCertificate{},
		PINPolicy: PIVPINPolicy{
			PINRequired:  s.config.SmartCard.RequirePIN,
			PINMinLength: s.config.SmartCard.PINMinLength,
			PINMaxLength: s.config.SmartCard.PINMaxLength,
			Retries:      s.config.SmartCard.MaxPINAttempts,
		},
	}
	
	// Extract card UUID from certificate (if present)
	cardInfo.CardholderUUID = extractPIVCardUUID(cert)
	
	// Extract card ID from certificate subject or extensions
	cardInfo.CardID = extractPIVCardID(cert)
	
	// Determine certificate slot
	slotID, slotName := determinePIVSlot(cert)
	
	pivCert := PIVCertificate{
		SlotID:      slotID,
		SlotName:    slotName,
		Certificate: cert,
		Fingerprint: calculateFingerprint(cert.Raw),
	}
	
	cardInfo.Certificates = append(cardInfo.Certificates, pivCert)
	
	// Validate PIV-specific requirements
	if s.config.SmartCard.PIVAuthCertOnly && slotID != "9A" {
		return nil, fmt.Errorf("only PIV authentication certificate (slot 9A) is allowed")
	}
	
	// Check required OIDs if configured
	if len(s.config.SmartCard.PIVRequiredOIDs) > 0 {
		if err := validateRequiredOIDs(cert, s.config.SmartCard.PIVRequiredOIDs); err != nil {
			return nil, err
		}
	}
	
	return cardInfo, nil
}

// ValidateCACCard validates a CAC card and extracts certificate
func (s *SmartCardProvider) ValidateCACCard(ctx context.Context, cert *x509.Certificate) (*CACCardInfo, error) {
	if !s.config.SmartCard.EnableCAC {
		return nil, fmt.Errorf("CAC support is not enabled")
	}
	
	// Verify this is a CAC certificate
	if !isCACCertificate(cert) {
		return nil, ErrNotCACCertificate
	}
	
	cardInfo := &CACCardInfo{
		Certificates: []CACCertificate{},
	}
	
	// Extract CAC number from certificate
	cardInfo.CACNumber = extractCACNumber(cert)
	cardInfo.CardID = cardInfo.CACNumber
	
	// Extract person DN
	cardInfo.PersonDN = cert.Subject.String()
	
	// Determine certificate type
	certType := determineCACCertificateType(cert)
	
	cacCert := CACCertificate{
		CertificateType: certType,
		Certificate:     cert,
		Fingerprint:     calculateFingerprint(cert.Raw),
	}
	
	cardInfo.Certificates = append(cardInfo.Certificates, cacCert)
	
	// Check required OIDs if configured
	if len(s.config.SmartCard.CACRequiredOIDs) > 0 {
		if err := validateRequiredOIDs(cert, s.config.SmartCard.CACRequiredOIDs); err != nil {
			return nil, err
		}
	}
	
	return cardInfo, nil
}

// AuthenticateWithPIV authenticates a user with PIV certificate
func (s *SmartCardProvider) AuthenticateWithPIV(ctx context.Context, cert *x509.Certificate, orgID string) (*SmartCardAuthResponse, error) {
	// Validate PIV card
	cardInfo, err := s.ValidatePIVCard(ctx, cert)
	if err != nil {
		return &SmartCardAuthResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	
	// Find or create user based on certificate
	userID, certificateID, err := s.findOrCreateUserFromCertificate(ctx, cert, orgID)
	if err != nil {
		return &SmartCardAuthResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get user: %v", err),
		}, err
	}
	
	return &SmartCardAuthResponse{
		Success:       true,
		UserID:        userID,
		CertificateID: certificateID,
		CardInfo:      cardInfo,
	}, nil
}

// AuthenticateWithCAC authenticates a user with CAC certificate
func (s *SmartCardProvider) AuthenticateWithCAC(ctx context.Context, cert *x509.Certificate, orgID string) (*SmartCardAuthResponse, error) {
	// Validate CAC card
	cardInfo, err := s.ValidateCACCard(ctx, cert)
	if err != nil {
		return &SmartCardAuthResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	
	// Find or create user based on certificate
	userID, certificateID, err := s.findOrCreateUserFromCertificate(ctx, cert, orgID)
	if err != nil {
		return &SmartCardAuthResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get user: %v", err),
		}, err
	}
	
	return &SmartCardAuthResponse{
		Success:       true,
		UserID:        userID,
		CertificateID: certificateID,
		CardInfo:      cardInfo,
	}, nil
}

// findOrCreateUserFromCertificate finds or creates a user based on certificate
func (s *SmartCardProvider) findOrCreateUserFromCertificate(ctx context.Context, cert *x509.Certificate, orgID string) (string, string, error) {
	fingerprint := calculateFingerprint(cert.Raw)
	
	// Try to find existing certificate
	storedCert, err := s.repo.GetCertificateByFingerprint(ctx, fingerprint)
	if err == nil && storedCert != nil {
		return storedCert.UserID, storedCert.ID, nil
	}
	
	// Certificate not found - would need to create user and certificate
	// This is a placeholder - real implementation would integrate with user service
	return "", "", fmt.Errorf("certificate not registered")
}

// Helper functions for PIV/CAC certificate parsing

// extractPIVCardUUID extracts the card UUID from PIV certificate
func extractPIVCardUUID(cert *x509.Certificate) string {
	// PIV card UUID is typically in the subject alternative name
	// as a URI: urn:uuid:XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	for _, uri := range cert.URIs {
		if uri.Scheme == "urn" && len(uri.Opaque) > 5 && uri.Opaque[:5] == "uuid:" {
			return uri.Opaque[5:]
		}
	}
	return ""
}

// extractPIVCardID extracts card ID from certificate
func extractPIVCardID(cert *x509.Certificate) string {
	// Try to extract from subject serial number
	if cert.Subject.SerialNumber != "" {
		return cert.Subject.SerialNumber
	}
	
	// Try to extract from certificate serial number
	return cert.SerialNumber.String()
}

// determinePIVSlot determines the PIV slot based on certificate attributes
func determinePIVSlot(cert *x509.Certificate) (string, string) {
	// Check certificate policies and key usage to determine slot
	
	// Slot 9A - PIV Authentication
	for _, policy := range cert.PolicyIdentifiers {
		if policy.String() == "2.16.840.1.101.3.2.1.3.7" {
			return "9A", "PIV Authentication"
		}
		if policy.String() == "2.16.840.1.101.3.2.1.3.13" {
			return "9E", "Card Authentication"
		}
		if policy.String() == "2.16.840.1.101.3.2.1.3.2" {
			return "9C", "Digital Signature"
		}
		if policy.String() == "2.16.840.1.101.3.2.1.3.4" {
			return "9D", "Key Management"
		}
	}
	
	// Default to authentication slot
	return "9A", "PIV Authentication"
}

// extractCACNumber extracts CAC number from certificate
func extractCACNumber(cert *x509.Certificate) string {
	// CAC number is typically encoded in the subject
	// It may be in the serial number field or a custom extension
	
	// Try subject serial number
	if cert.Subject.SerialNumber != "" {
		return cert.Subject.SerialNumber
	}
	
	// Try certificate serial number
	return cert.SerialNumber.String()
}

// determineCACCertificateType determines the CAC certificate type
func determineCACCertificateType(cert *x509.Certificate) string {
	// Check certificate policies to determine type
	for _, policy := range cert.PolicyIdentifiers {
		policyStr := policy.String()
		switch {
		case policyStr == "2.16.840.1.101.2.1.11.42":
			return "ID Certificate"
		case policyStr == "2.16.840.1.101.2.1.11.39":
			return "PKI Certificate"
		case cert.KeyUsage&x509.KeyUsageDigitalSignature != 0:
			return "Signature Certificate"
		case cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0:
			return "Encryption Certificate"
		}
	}
	
	return "Unknown"
}

// validateRequiredOIDs validates that certificate contains required OIDs
func validateRequiredOIDs(cert *x509.Certificate, requiredOIDs []string) error {
	certOIDs := make(map[string]bool)
	
	// Collect all OIDs from certificate policies
	for _, policy := range cert.PolicyIdentifiers {
		certOIDs[policy.String()] = true
	}
	
	// Check each required OID
	for _, required := range requiredOIDs {
		if !certOIDs[required] {
			return fmt.Errorf("certificate missing required OID: %s", required)
		}
	}
	
	return nil
}

// PIV/CAC OID Constants
var (
	// PIV OIDs
	OID_PIV_Authentication    = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 2, 1, 3, 7}
	OID_PIV_CardAuth          = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 2, 1, 3, 13}
	OID_PIV_DigitalSignature  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 2, 1, 3, 2}
	OID_PIV_KeyManagement     = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 2, 1, 3, 4}
	
	// CAC OIDs
	OID_CAC_PKI               = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 2, 1, 11, 39}
	OID_CAC_Authentication    = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 2, 1, 11, 42}
	OID_CAC_Email             = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 2, 1, 11, 17}
)

// GetPIVSlotNames returns human-readable PIV slot names
func GetPIVSlotNames() map[string]string {
	return map[string]string{
		"9A": "PIV Authentication",
		"9B": "PIV Card Application Administration",
		"9C": "Digital Signature",
		"9D": "Key Management",
		"9E": "Card Authentication",
		"82": "Retired Key Management 1",
		"83": "Retired Key Management 2",
		"84": "Retired Key Management 3",
		"85": "Retired Key Management 4",
	}
}

// GetCACCertificateTypes returns CAC certificate types
func GetCACCertificateTypes() []string {
	return []string{
		"ID Certificate",
		"PKI Certificate",
		"Email Signature Certificate",
		"Email Encryption Certificate",
	}
}

