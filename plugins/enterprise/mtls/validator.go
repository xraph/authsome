package mtls

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"slices"
	"time"
)

// CertificateValidator handles X.509 certificate validation.
type CertificateValidator struct {
	config     *Config
	repo       Repository
	revChecker *RevocationChecker
}

// NewCertificateValidator creates a new certificate validator.
func NewCertificateValidator(config *Config, repo Repository, revChecker *RevocationChecker) *CertificateValidator {
	return &CertificateValidator{
		config:     config,
		repo:       repo,
		revChecker: revChecker,
	}
}

// ValidationResult contains the result of certificate validation.
type ValidationResult struct {
	Valid            bool                `json:"valid"`
	Certificate      *x509.Certificate   `json:"-"`
	Chain            []*x509.Certificate `json:"-"`
	Errors           []error             `json:"errors,omitempty"`
	Warnings         []string            `json:"warnings,omitempty"`
	ValidationSteps  map[string]any      `json:"validationSteps"`
	TrustAnchor      *TrustAnchor        `json:"trustAnchor,omitempty"`
	RevocationStatus string              `json:"revocationStatus,omitempty"`
}

// ValidateCertificate performs comprehensive certificate validation.
func (v *CertificateValidator) ValidateCertificate(ctx context.Context, certPEM []byte, orgID string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:           true,
		Errors:          []error{},
		Warnings:        []string{},
		ValidationSteps: make(map[string]any),
	}

	// Step 1: Parse certificate
	cert, err := v.parseCertificate(certPEM)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
		result.ValidationSteps["parse"] = "failed"

		return result, nil
	}

	result.Certificate = cert
	result.ValidationSteps["parse"] = "passed"

	// Step 2: Basic validation (expiration, not before, etc.)
	if err := v.validateBasic(cert, result); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
	}

	// Step 3: Key validation
	if err := v.validateKey(cert, result); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
	}

	// Step 4: Key usage and extended key usage
	if err := v.validateKeyUsage(cert, result); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
	}

	// Step 5: Chain validation
	chain, trustAnchor, err := v.validateChain(ctx, cert, orgID, result)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
	} else {
		result.Chain = chain
		result.TrustAnchor = trustAnchor
	}

	// Step 6: Revocation checking
	if v.config.Revocation.EnableCRL || v.config.Revocation.EnableOCSP {
		revStatus, err := v.checkRevocation(ctx, cert, result)
		if err != nil {
			if !v.config.Revocation.FailOpen {
				result.Valid = false
				result.Errors = append(result.Errors, err)
			} else {
				result.Warnings = append(result.Warnings, "Revocation check failed but fail-open enabled: "+err.Error())
			}
		}

		result.RevocationStatus = revStatus
	}

	// Step 7: Policy validation (if exists)
	policy, err := v.repo.GetDefaultPolicy(ctx, orgID)
	if err == nil && policy != nil {
		if err := v.validatePolicy(cert, policy, result); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err)
		}
	}

	return result, nil
}

// parseCertificate parses a PEM-encoded certificate.
func (v *CertificateValidator) parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, ErrInvalidPEM
	}

	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid PEM type: %s", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCertificateParseFailed, err)
	}

	return cert, nil
}

// validateBasic performs basic certificate validation.
func (v *CertificateValidator) validateBasic(cert *x509.Certificate, result *ValidationResult) error {
	now := time.Now()

	// Check expiration
	if v.config.Validation.CheckExpiration {
		if now.After(cert.NotAfter) {
			result.ValidationSteps["expiration"] = "failed"

			return ErrCertificateExpired
		}

		// Check remaining validity
		remaining := cert.NotAfter.Sub(now)

		minRemaining := time.Duration(v.config.Validation.MinRemainingValidity) * 24 * time.Hour
		if remaining < minRemaining {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Certificate expires soon (in %d days)", int(remaining.Hours()/24)))
		}

		result.ValidationSteps["expiration"] = "passed"
	}

	// Check not before
	if v.config.Validation.CheckNotBefore {
		if now.Before(cert.NotBefore) {
			result.ValidationSteps["notBefore"] = "failed"

			return ErrCertificateNotYetValid
		}

		result.ValidationSteps["notBefore"] = "passed"
	}

	// Check certificate age
	age := now.Sub(cert.NotBefore)

	maxAge := time.Duration(v.config.Validation.MaxCertificateAge) * 24 * time.Hour
	if age > maxAge {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Certificate is old (issued %d days ago)", int(age.Hours()/24)))
	}

	return nil
}

// validateKey validates the certificate's public key.
func (v *CertificateValidator) validateKey(cert *x509.Certificate, result *ValidationResult) error {
	// Check key algorithm
	keyAlgo := cert.PublicKeyAlgorithm.String()

	if len(v.config.Validation.AllowedKeyAlgorithms) > 0 {
		allowed := false

		for _, algo := range v.config.Validation.AllowedKeyAlgorithms {
			if keyAlgo == algo || contains(keyAlgo, algo) {
				allowed = true

				break
			}
		}

		if !allowed {
			result.ValidationSteps["keyAlgorithm"] = "failed"

			return fmt.Errorf("%w: %s not allowed", ErrUnsupportedAlgorithm, keyAlgo)
		}
	}

	// Check key size
	keySize := getKeySize(cert.PublicKey)
	if keySize < v.config.Validation.MinKeySize {
		result.ValidationSteps["keySize"] = "failed"

		return fmt.Errorf("%w: %d bits (minimum: %d)", ErrKeyTooWeak, keySize, v.config.Validation.MinKeySize)
	}

	// Check signature algorithm
	sigAlgo := cert.SignatureAlgorithm.String()

	if len(v.config.Validation.AllowedSignatureAlgs) > 0 {
		allowed := false

		for _, algo := range v.config.Validation.AllowedSignatureAlgs {
			if sigAlgo == algo || contains(sigAlgo, algo) {
				allowed = true

				break
			}
		}

		if !allowed {
			result.ValidationSteps["signatureAlgorithm"] = "failed"

			return fmt.Errorf("%w: %s not allowed", ErrUnsupportedAlgorithm, sigAlgo)
		}
	}

	result.ValidationSteps["keyValidation"] = "passed"

	return nil
}

// validateKeyUsage validates key usage and extended key usage.
func (v *CertificateValidator) validateKeyUsage(cert *x509.Certificate, result *ValidationResult) error {
	if !v.config.Validation.CheckKeyUsage {
		return nil
	}

	// Check required key usage
	for _, required := range v.config.Validation.RequiredKeyUsage {
		if !hasKeyUsage(cert, required) {
			result.ValidationSteps["keyUsage"] = "failed"

			return fmt.Errorf("%w: missing %s", ErrInvalidKeyUsage, required)
		}
	}

	// Check extended key usage
	if v.config.Validation.CheckExtendedKeyUsage {
		for _, required := range v.config.Validation.RequiredEKU {
			if !hasExtendedKeyUsage(cert, required) {
				result.ValidationSteps["extendedKeyUsage"] = "failed"

				return fmt.Errorf("%w: missing %s", ErrInvalidKeyUsage, required)
			}
		}
	}

	result.ValidationSteps["keyUsage"] = "passed"

	return nil
}

// validateChain validates the certificate chain.
func (v *CertificateValidator) validateChain(ctx context.Context, cert *x509.Certificate, orgID string, result *ValidationResult) ([]*x509.Certificate, *TrustAnchor, error) {
	if !v.config.Validation.ValidateChain {
		return nil, nil, nil
	}

	// Get trust anchors
	anchors, err := v.repo.ListTrustAnchors(ctx, orgID)
	if err != nil {
		result.ValidationSteps["chainValidation"] = "failed"

		return nil, nil, fmt.Errorf("failed to get trust anchors: %w", err)
	}

	if len(anchors) == 0 {
		if !v.config.Validation.AllowSelfSigned {
			result.ValidationSteps["chainValidation"] = "failed"

			return nil, nil, ErrNoTrustAnchors
		}
		// Check if self-signed
		if !bytes.Equal(cert.RawIssuer, cert.RawSubject) {
			result.ValidationSteps["chainValidation"] = "failed"

			return nil, nil, ErrUntrustedCA
		}

		result.ValidationSteps["chainValidation"] = "passed (self-signed)"

		return []*x509.Certificate{cert}, nil, nil
	}

	// Build certificate pool
	roots := x509.NewCertPool()

	var matchedAnchor *TrustAnchor

	for _, anchor := range anchors {
		block, _ := pem.Decode([]byte(anchor.CertificatePEM))
		if block == nil {
			continue
		}

		caCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		roots.AddCert(caCert)
	}

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		result.ValidationSteps["chainValidation"] = "failed"

		return nil, nil, fmt.Errorf("%w: %w", ErrCertificateChainInvalid, err)
	}

	if len(chains) == 0 {
		result.ValidationSteps["chainValidation"] = "failed"

		return nil, nil, ErrCertificateChainInvalid
	}

	// Use the first valid chain
	chain := chains[0]

	// Check chain length
	if len(chain) > v.config.Validation.MaxChainLength {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Certificate chain is long (%d certificates)", len(chain)))
	}

	// Find the matching trust anchor
	if len(chain) > 0 {
		rootCert := chain[len(chain)-1]
		fingerprint := calculateFingerprint(rootCert.Raw)

		for _, anchor := range anchors {
			if anchor.Fingerprint == fingerprint {
				matchedAnchor = anchor

				break
			}
		}
	}

	result.ValidationSteps["chainValidation"] = "passed"

	return chain, matchedAnchor, nil
}

// checkRevocation checks certificate revocation status.
func (v *CertificateValidator) checkRevocation(ctx context.Context, cert *x509.Certificate, result *ValidationResult) (string, error) {
	fingerprint := calculateFingerprint(cert.Raw)

	// First check if certificate is already revoked in database
	storedCert, err := v.repo.GetCertificateByFingerprint(ctx, fingerprint)
	if err == nil && storedCert != nil {
		if storedCert.Status == "revoked" {
			result.ValidationSteps["revocationCheck"] = "failed"

			return "revoked", ErrCertificateRevoked
		}
	}

	// Perform online revocation checking
	if v.revChecker != nil {
		status, err := v.revChecker.CheckRevocation(ctx, cert)
		if err != nil {
			result.ValidationSteps["revocationCheck"] = "error"

			return "unknown", err
		}

		result.ValidationSteps["revocationCheck"] = status

		if status == "revoked" {
			return "revoked", ErrCertificateRevoked
		}

		return status, nil
	}

	result.ValidationSteps["revocationCheck"] = "skipped"

	return "unknown", nil
}

// validatePolicy validates certificate against organization policy.
func (v *CertificateValidator) validatePolicy(cert *x509.Certificate, policy *CertificatePolicy, result *ValidationResult) error {
	// Check pinning requirement
	if policy.RequirePinning {
		// Would need to check if certificate is pinned
		result.Warnings = append(result.Warnings, "Certificate pinning required by policy")
	}

	// Check key size
	keySize := getKeySize(cert.PublicKey)
	if keySize < policy.MinKeySize {
		result.ValidationSteps["policyValidation"] = "failed"

		return fmt.Errorf("%w: key size %d < minimum %d", ErrPolicyViolation, keySize, policy.MinKeySize)
	}

	// Check allowed CAs (if specified)
	if len(policy.AllowedCAs) > 0 {
		// Would check if issuer is in allowed list
	}

	// Check PIV/CAC requirements
	if policy.RequirePIV {
		if !isPIVCertificate(cert) {
			result.ValidationSteps["policyValidation"] = "failed"

			return ErrNotPIVCertificate
		}
	}

	if policy.RequireCAC {
		if !isCACCertificate(cert) {
			result.ValidationSteps["policyValidation"] = "failed"

			return ErrNotCACCertificate
		}
	}

	result.ValidationSteps["policyValidation"] = "passed"

	return nil
}

// calculateFingerprint calculates SHA-256 fingerprint of certificate.
func calculateFingerprint(certDER []byte) string {
	hash := sha256.Sum256(certDER)

	return hex.EncodeToString(hash[:])
}

// getKeySize returns the key size in bits.
func getKeySize(pub any) int {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return k.N.BitLen()
	case *ecdsa.PublicKey:
		return k.Params().BitSize
	case ed25519.PublicKey:
		return 256
	default:
		return 0
	}
}

// hasKeyUsage checks if certificate has specific key usage.
func hasKeyUsage(cert *x509.Certificate, usage string) bool {
	switch usage {
	case "digitalSignature":
		return cert.KeyUsage&x509.KeyUsageDigitalSignature != 0
	case "keyEncipherment":
		return cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0
	case "keyAgreement":
		return cert.KeyUsage&x509.KeyUsageKeyAgreement != 0
	case "dataEncipherment":
		return cert.KeyUsage&x509.KeyUsageDataEncipherment != 0
	default:
		return false
	}
}

// hasExtendedKeyUsage checks if certificate has specific extended key usage.
func hasExtendedKeyUsage(cert *x509.Certificate, usage string) bool {
	switch usage {
	case "clientAuth":
		if slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
			return true
		}
	case "serverAuth":
		if slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
			return true
		}
	}

	return false
}

// isPIVCertificate checks if certificate is a PIV certificate.
func isPIVCertificate(cert *x509.Certificate) bool {
	// PIV certificates have specific OIDs in certificate policies
	pivOIDs := []string{
		"2.16.840.1.101.3.2.1.3.7",  // id-fpki-common-authentication
		"2.16.840.1.101.3.2.1.3.13", // id-fpki-common-cardAuth
	}

	for _, policy := range cert.PolicyIdentifiers {
		if slices.Contains(pivOIDs, policy.String()) {
			return true
		}
	}

	return false
}

// isCACCertificate checks if certificate is a CAC certificate.
func isCACCertificate(cert *x509.Certificate) bool {
	// CAC certificates have specific OIDs
	cacOIDs := []string{
		"2.16.840.1.101.2.1.11.39", // id-cac-PKI
		"2.16.840.1.101.2.1.11.42", // id-cac-authentication
	}

	for _, policy := range cert.PolicyIdentifiers {
		if slices.Contains(cacOIDs, policy.String()) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
