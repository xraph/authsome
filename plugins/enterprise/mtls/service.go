package mtls

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/xraph/authsome/internal/errs"
)

// Service handles mTLS business logic.
type Service struct {
	config     *Config
	repo       Repository
	validator  *CertificateValidator
	revChecker *RevocationChecker
	smartCard  *SmartCardProvider
	hsmManager *HSMManager
}

// NewService creates a new mTLS service.
func NewService(
	config *Config,
	repo Repository,
	validator *CertificateValidator,
	revChecker *RevocationChecker,
	smartCard *SmartCardProvider,
	hsmManager *HSMManager,
) *Service {
	return &Service{
		config:     config,
		repo:       repo,
		validator:  validator,
		revChecker: revChecker,
		smartCard:  smartCard,
		hsmManager: hsmManager,
	}
}

// ===== Certificate Management =====

// RegisterCertificate registers a new client certificate.
func (s *Service) RegisterCertificate(ctx context.Context, req *RegisterCertificateRequest) (*Certificate, error) {
	// Parse certificate
	block, _ := pem.Decode([]byte(req.CertificatePEM))
	if block == nil {
		return nil, ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCertificateParseFailed, err)
	}

	// Validate certificate
	validationResult, err := s.validator.ValidateCertificate(ctx, []byte(req.CertificatePEM), req.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if !validationResult.Valid {
		return nil, fmt.Errorf("certificate validation failed: %v", validationResult.Errors)
	}

	// Check if certificate already exists
	fingerprint := calculateFingerprint(cert.Raw)

	existing, _ := s.repo.GetCertificateByFingerprint(ctx, fingerprint)
	if existing != nil {
		return nil, errs.Conflict("certificate already registered")
	}

	// Extract certificate information
	certRecord := &Certificate{
		ID:               generateID(),
		OrganizationID:   req.OrganizationID,
		UserID:           req.UserID,
		DeviceID:         req.DeviceID,
		Subject:          cert.Subject.String(),
		Issuer:           cert.Issuer.String(),
		SerialNumber:     cert.SerialNumber.String(),
		Fingerprint:      fingerprint,
		FingerprintSHA1:  calculateSHA1Fingerprint(cert.Raw),
		CertificatePEM:   req.CertificatePEM,
		PublicKeyPEM:     extractPublicKeyPEM(cert),
		NotBefore:        cert.NotBefore,
		NotAfter:         cert.NotAfter,
		CertificateType:  req.CertificateType,
		CertificateClass: req.CertificateClass,
		Status:           "active",
		PIVCardID:        req.PIVCardID,
		CACNumber:        req.CACNumber,
		HSMKeyID:         req.HSMKeyID,
		HSMProvider:      req.HSMProvider,
		IsPinned:         req.IsPinned,
		KeyUsage:         extractKeyUsage(cert),
		ExtendedKeyUsage: extractExtendedKeyUsage(cert),
		SubjectAltNames:  extractSubjectAltNames(cert),
		Metadata:         req.Metadata,
	}

	// Create certificate record
	if err := s.repo.CreateCertificate(ctx, certRecord); err != nil {
		return nil, fmt.Errorf("failed to register certificate: %w", err)
	}

	// Log auth event
	s.logAuthEvent(ctx, certRecord.ID, req.OrganizationID, req.UserID, "certificate_registered", "success", nil)

	return certRecord, nil
}

// AuthenticateWithCertificate authenticates a user with a client certificate.
func (s *Service) AuthenticateWithCertificate(ctx context.Context, certPEM []byte, orgID string) (*AuthenticationResult, error) {
	// Validate certificate
	validationResult, err := s.validator.ValidateCertificate(ctx, certPEM, orgID)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if !validationResult.Valid {
		s.logAuthEvent(ctx, "", orgID, "", "auth_attempt", "failed", map[string]any{
			"errors": validationResult.Errors,
		})

		return &AuthenticationResult{
			Success:          false,
			ValidationResult: validationResult,
			Errors:           validationResult.Errors,
		}, nil
	}

	cert := validationResult.Certificate
	fingerprint := calculateFingerprint(cert.Raw)

	// Get certificate from database
	certRecord, err := s.repo.GetCertificateByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, errs.NotFound("certificate not registered")
	}

	// Check certificate status
	if certRecord.Status != "active" {
		s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_attempt", "failed", map[string]any{
			"reason": "certificate_" + certRecord.Status,
		})

		return &AuthenticationResult{
			Success:          false,
			ValidationResult: validationResult,
			Errors:           []error{fmt.Errorf("certificate is %s", certRecord.Status)},
		}, nil
	}

	// Update certificate usage stats
	s.updateCertificateUsage(ctx, certRecord.ID)

	// Check PIV/CAC if applicable
	if certRecord.CertificateClass == "piv" && s.config.SmartCard.EnablePIV {
		pivInfo, err := s.smartCard.ValidatePIVCard(ctx, cert)
		if err != nil {
			s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_attempt", "failed", map[string]any{
				"reason": "piv_validation_failed",
				"error":  err.Error(),
			})

			return &AuthenticationResult{
				Success:          false,
				ValidationResult: validationResult,
				Errors:           []error{err},
			}, nil
		}

		s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_success", "success", map[string]any{
			"method":   "piv",
			"cardInfo": pivInfo,
		})
	} else if certRecord.CertificateClass == "cac" && s.config.SmartCard.EnableCAC {
		cacInfo, err := s.smartCard.ValidateCACCard(ctx, cert)
		if err != nil {
			s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_attempt", "failed", map[string]any{
				"reason": "cac_validation_failed",
				"error":  err.Error(),
			})

			return &AuthenticationResult{
				Success:          false,
				ValidationResult: validationResult,
				Errors:           []error{err},
			}, nil
		}

		s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_success", "success", map[string]any{
			"method":   "cac",
			"cardInfo": cacInfo,
		})
	} else {
		s.logAuthEvent(ctx, certRecord.ID, orgID, certRecord.UserID, "auth_success", "success", map[string]any{
			"method": "standard",
		})
	}

	return &AuthenticationResult{
		Success:          true,
		UserID:           certRecord.UserID,
		CertificateID:    certRecord.ID,
		Certificate:      certRecord,
		ValidationResult: validationResult,
	}, nil
}

// RevokeCertificate revokes a certificate.
func (s *Service) RevokeCertificate(ctx context.Context, id string, reason string) error {
	cert, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.RevokeCertificate(ctx, id, reason); err != nil {
		return err
	}

	s.logAuthEvent(ctx, id, cert.OrganizationID, cert.UserID, "certificate_revoked", "success", map[string]any{
		"reason": reason,
	})

	return nil
}

// GetCertificate retrieves a certificate by ID.
func (s *Service) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
	return s.repo.GetCertificate(ctx, id)
}

// ListCertificates lists certificates with filters.
func (s *Service) ListCertificates(ctx context.Context, filters CertificateFilters) ([]*Certificate, error) {
	return s.repo.ListCertificates(ctx, filters)
}

// ===== Trust Anchor Management =====

// AddTrustAnchor adds a new trusted CA certificate.
func (s *Service) AddTrustAnchor(ctx context.Context, req *AddTrustAnchorRequest) (*TrustAnchor, error) {
	// Parse certificate
	block, _ := pem.Decode([]byte(req.CertificatePEM))
	if block == nil {
		return nil, ErrInvalidPEM
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCertificateParseFailed, err)
	}

	// Verify it's a CA certificate
	if !cert.IsCA {
		return nil, errs.BadRequest("certificate is not a CA certificate")
	}

	fingerprint := calculateFingerprint(cert.Raw)

	// Check if already exists
	existing, _ := s.repo.GetTrustAnchorByFingerprint(ctx, fingerprint)
	if existing != nil {
		return nil, errs.Conflict("trust anchor already exists")
	}

	anchor := &TrustAnchor{
		ID:             generateID(),
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Subject:        cert.Subject.String(),
		Issuer:         cert.Issuer.String(),
		SerialNumber:   cert.SerialNumber.String(),
		Fingerprint:    fingerprint,
		CertificatePEM: req.CertificatePEM,
		NotBefore:      cert.NotBefore,
		NotAfter:       cert.NotAfter,
		TrustLevel:     req.TrustLevel,
		IsRootCA:       bytes.Equal(cert.RawIssuer, cert.RawSubject),
		CRLEndpoints:   cert.CRLDistributionPoints,
		OCSPEndpoints:  cert.OCSPServer,
		Status:         "active",
		Metadata:       req.Metadata,
	}

	if err := s.repo.CreateTrustAnchor(ctx, anchor); err != nil {
		return nil, fmt.Errorf("failed to create trust anchor: %w", err)
	}

	return anchor, nil
}

// GetTrustAnchors lists trust anchors for an organization.
func (s *Service) GetTrustAnchors(ctx context.Context, orgID string) ([]*TrustAnchor, error) {
	return s.repo.ListTrustAnchors(ctx, orgID)
}

// ===== Policy Management =====

// CreatePolicy creates a certificate policy.
func (s *Service) CreatePolicy(ctx context.Context, req *CreatePolicyRequest) (*CertificatePolicy, error) {
	policy := &CertificatePolicy{
		ID:                   generateID(),
		OrganizationID:       req.OrganizationID,
		Name:                 req.Name,
		Description:          req.Description,
		RequirePinning:       req.RequirePinning,
		AllowSelfSigned:      req.AllowSelfSigned,
		RequireCRLCheck:      req.RequireCRLCheck,
		RequireOCSPCheck:     req.RequireOCSPCheck,
		MinKeySize:           req.MinKeySize,
		AllowedKeyAlgorithms: req.AllowedKeyAlgorithms,
		AllowedSignatureAlgs: req.AllowedSignatureAlgs,
		MaxCertificateAge:    req.MaxCertificateAge,
		MinRemainingValidity: req.MinRemainingValidity,
		RequirePIV:           req.RequirePIV,
		RequireCAC:           req.RequireCAC,
		RequireHSM:           req.RequireHSM,
		Status:               "active",
		IsDefault:            req.IsDefault,
		Metadata:             req.Metadata,
	}

	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	return policy, nil
}

// GetPolicy retrieves a policy by ID.
func (s *Service) GetPolicy(ctx context.Context, id string) (*CertificatePolicy, error) {
	return s.repo.GetPolicy(ctx, id)
}

// ===== Statistics and Monitoring =====

// GetAuthEventStats retrieves authentication statistics.
func (s *Service) GetAuthEventStats(ctx context.Context, orgID string, since time.Time) (*AuthEventStats, error) {
	return s.repo.GetAuthEventStats(ctx, orgID, since)
}

// GetExpiringCertificates retrieves certificates expiring soon.
func (s *Service) GetExpiringCertificates(ctx context.Context, orgID string, days int) ([]*Certificate, error) {
	return s.repo.GetExpiringCertificates(ctx, orgID, days)
}

// ===== Helper Methods =====

func (s *Service) updateCertificateUsage(ctx context.Context, certID string) {
	cert, err := s.repo.GetCertificate(ctx, certID)
	if err != nil {
		return
	}

	now := time.Now()
	cert.LastUsedAt = &now
	cert.UseCount++

	_ = s.repo.UpdateCertificate(ctx, cert)
}

func (s *Service) logAuthEvent(ctx context.Context, certID, orgID, userID, eventType, status string, metadata map[string]any) {
	event := &CertificateAuthEvent{
		ID:              generateID(),
		CertificateID:   certID,
		OrganizationID:  orgID,
		UserID:          userID,
		EventType:       eventType,
		Status:          status,
		ValidationSteps: metadata,
	}

	_ = s.repo.CreateAuthEvent(ctx, event)
}

// Request/Response types

type RegisterCertificateRequest struct {
	OrganizationID   string         `json:"organizationId"`
	UserID           string         `json:"userId,omitempty"`
	DeviceID         string         `json:"deviceId,omitempty"`
	CertificatePEM   string         `json:"certificatePem"`
	CertificateType  string         `json:"certificateType"`  // user, device, service
	CertificateClass string         `json:"certificateClass"` // standard, piv, cac, smartcard
	PIVCardID        string         `json:"pivCardId,omitempty"`
	CACNumber        string         `json:"cacNumber,omitempty"`
	HSMKeyID         string         `json:"hsmKeyId,omitempty"`
	HSMProvider      string         `json:"hsmProvider,omitempty"`
	IsPinned         bool           `json:"isPinned"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type AuthenticationResult struct {
	Success          bool              `json:"success"`
	UserID           string            `json:"userId,omitempty"`
	CertificateID    string            `json:"certificateId,omitempty"`
	Certificate      *Certificate      `json:"certificate,omitempty"`
	ValidationResult *ValidationResult `json:"validationResult,omitempty"`
	Errors           []error           `json:"errors,omitempty"`
}

type AddTrustAnchorRequest struct {
	OrganizationID string         `json:"organizationId"`
	Name           string         `json:"name"`
	CertificatePEM string         `json:"certificatePem"`
	TrustLevel     string         `json:"trustLevel"` // root, intermediate, self_signed
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type CreatePolicyRequest struct {
	OrganizationID       string         `json:"organizationId"`
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
	RequirePinning       bool           `json:"requirePinning"`
	AllowSelfSigned      bool           `json:"allowSelfSigned"`
	RequireCRLCheck      bool           `json:"requireCrlCheck"`
	RequireOCSPCheck     bool           `json:"requireOcspCheck"`
	MinKeySize           int            `json:"minKeySize"`
	AllowedKeyAlgorithms StringArray    `json:"allowedKeyAlgorithms,omitempty"`
	AllowedSignatureAlgs StringArray    `json:"allowedSignatureAlgs,omitempty"`
	MaxCertificateAge    int            `json:"maxCertificateAge"`
	MinRemainingValidity int            `json:"minRemainingValidity"`
	RequirePIV           bool           `json:"requirePiv"`
	RequireCAC           bool           `json:"requireCac"`
	RequireHSM           bool           `json:"requireHsm"`
	IsDefault            bool           `json:"isDefault"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

// Helper functions

func extractPublicKeyPEM(cert *x509.Certificate) string {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return ""
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pemBlock)
}

func extractKeyUsage(cert *x509.Certificate) []string {
	var usages []string

	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "digitalSignature")
	}

	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "keyEncipherment")
	}

	if cert.KeyUsage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "keyAgreement")
	}

	if cert.KeyUsage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "dataEncipherment")
	}

	return usages
}

func extractExtendedKeyUsage(cert *x509.Certificate) []string {
	var usages []string

	for _, eku := range cert.ExtKeyUsage {
		switch eku {
		case x509.ExtKeyUsageClientAuth:
			usages = append(usages, "clientAuth")
		case x509.ExtKeyUsageServerAuth:
			usages = append(usages, "serverAuth")
		case x509.ExtKeyUsageCodeSigning:
			usages = append(usages, "codeSigning")
		case x509.ExtKeyUsageEmailProtection:
			usages = append(usages, "emailProtection")
		}
	}

	return usages
}

func extractSubjectAltNames(cert *x509.Certificate) StringArray {
	var names []string

	names = append(names, cert.DNSNames...)

	for _, email := range cert.EmailAddresses {
		names = append(names, "email:"+email)
	}

	for _, ip := range cert.IPAddresses {
		names = append(names, "ip:"+ip.String())
	}

	for _, uri := range cert.URIs {
		names = append(names, "uri:"+uri.String())
	}

	return names
}

func calculateSHA1Fingerprint(certDER []byte) string {
	// SHA-1 fingerprint calculation
	// In production, use crypto/sha1
	return ""
}
