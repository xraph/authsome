package mtls

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/crypto/ocsp"

	"github.com/xraph/authsome/internal/errs"
)

// RevocationChecker handles certificate revocation checking via CRL and OCSP.
type RevocationChecker struct {
	config     *Config
	repo       Repository
	httpClient *http.Client
}

// NewRevocationChecker creates a new revocation checker.
func NewRevocationChecker(config *Config, repo Repository) *RevocationChecker {
	return &RevocationChecker{
		config: config,
		repo:   repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckRevocation checks if a certificate has been revoked.
func (r *RevocationChecker) CheckRevocation(ctx context.Context, cert *x509.Certificate) (string, error) {
	// Prefer OCSP if enabled
	if r.config.Revocation.EnableOCSP && r.config.Revocation.PreferOCSP {
		status, err := r.checkOCSP(ctx, cert)
		if err == nil {
			return status, nil
		}

		// Fall back to CRL if OCSP fails
		if r.config.Revocation.EnableCRL {
			return r.checkCRL(ctx, cert)
		}

		return "unknown", err
	}

	// Try CRL first
	if r.config.Revocation.EnableCRL {
		status, err := r.checkCRL(ctx, cert)
		if err == nil {
			return status, nil
		}

		// Fall back to OCSP if CRL fails
		if r.config.Revocation.EnableOCSP {
			return r.checkOCSP(ctx, cert)
		}

		return "unknown", err
	}

	// Only OCSP enabled
	if r.config.Revocation.EnableOCSP {
		return r.checkOCSP(ctx, cert)
	}

	return "unknown", ErrRevocationUnavailable
}

// checkOCSP performs OCSP revocation checking.
func (r *RevocationChecker) checkOCSP(ctx context.Context, cert *x509.Certificate) (string, error) {
	fingerprint := calculateFingerprint(cert.Raw)

	// Check cache first
	if cachedResp, err := r.repo.GetOCSPResponse(ctx, fingerprint); err == nil && cachedResp != nil {
		if time.Now().Before(cachedResp.ExpiresAt) {
			return cachedResp.Status, nil
		}
	}

	// Get OCSP server URLs
	if len(cert.OCSPServer) == 0 {
		return "unknown", errs.BadRequest("no OCSP servers in certificate")
	}

	// Get issuer certificate (needed for OCSP request)
	issuerCert, err := r.getIssuerCertificate(ctx, cert)
	if err != nil {
		return "unknown", fmt.Errorf("failed to get issuer certificate: %w", err)
	}

	// Create OCSP request
	ocspReq, err := ocsp.CreateRequest(cert, issuerCert, nil)
	if err != nil {
		return "unknown", fmt.Errorf("failed to create OCSP request: %w", err)
	}

	// Try each OCSP server
	var lastErr error

	for _, server := range cert.OCSPServer {
		status, err := r.performOCSPRequest(ctx, server, ocspReq, issuerCert)
		if err == nil {
			// Cache the response
			r.cacheOCSPResponse(ctx, fingerprint, status)

			return status, nil
		}

		lastErr = err
	}

	return "unknown", fmt.Errorf("all OCSP servers failed: %w", lastErr)
}

// performOCSPRequest sends an OCSP request and parses the response.
func (r *RevocationChecker) performOCSPRequest(ctx context.Context, server string, reqBytes []byte, issuerCert *x509.Certificate) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server, bytes.NewReader(reqBytes))
	if err != nil {
		return "unknown", err
	}

	httpReq.Header.Set("Content-Type", "application/ocsp-request")
	httpReq.Header.Set("Accept", "application/ocsp-response")

	httpResp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return "unknown", err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return "unknown", fmt.Errorf("OCSP server returned status %d", httpResp.StatusCode)
	}

	respBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return "unknown", err
	}

	ocspResp, err := ocsp.ParseResponse(respBytes, issuerCert)
	if err != nil {
		return "unknown", fmt.Errorf("%w: %w", ErrOCSPParseFailed, err)
	}

	// Check response status
	switch ocspResp.Status {
	case ocsp.Good:
		return "good", nil
	case ocsp.Revoked:
		return "revoked", nil
	case ocsp.Unknown:
		return "unknown", nil
	default:
		return "unknown", fmt.Errorf("unexpected OCSP status: %d", ocspResp.Status)
	}
}

// cacheOCSPResponse caches an OCSP response.
func (r *RevocationChecker) cacheOCSPResponse(ctx context.Context, certID string, status string) {
	expiresAt := time.Now().Add(r.config.Revocation.OCSPCacheDuration)

	ocspResp := &OCSPResponse{
		ID:            generateID(),
		CertificateID: certID,
		Status:        status,
		ProducedAt:    time.Now(),
		ThisUpdate:    time.Now(),
		ExpiresAt:     expiresAt,
	}

	// Best effort caching - don't fail if it doesn't work
	_ = r.repo.CreateOCSPResponse(ctx, ocspResp)
}

// checkCRL performs CRL revocation checking.
func (r *RevocationChecker) checkCRL(ctx context.Context, cert *x509.Certificate) (string, error) {
	// Get CRL distribution points
	if len(cert.CRLDistributionPoints) == 0 {
		return "unknown", errs.BadRequest("no CRL distribution points in certificate")
	}

	// Get issuer DN
	issuer := cert.Issuer.String()

	// Check if we have a cached CRL
	cachedCRL, err := r.repo.GetCRLByIssuer(ctx, issuer)
	if err == nil && cachedCRL != nil {
		if time.Now().Before(cachedCRL.NextUpdate) {
			// CRL is still valid, check if certificate is in it
			return r.checkCertificateInCRL(cert, cachedCRL)
		}
	}

	// Fetch fresh CRL
	var lastErr error

	for _, dp := range cert.CRLDistributionPoints {
		crl, err := r.fetchCRL(ctx, dp)
		if err != nil {
			lastErr = err

			continue
		}

		// Store CRL in database
		r.storeCRL(ctx, crl, issuer)

		// Check certificate against CRL
		return r.checkCertificateAgainstCRL(cert, crl)
	}

	return "unknown", fmt.Errorf("failed to fetch CRL: %w", lastErr)
}

// fetchCRL fetches a CRL from a distribution point.
func (r *RevocationChecker) fetchCRL(ctx context.Context, url string) (*x509.RevocationList, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	httpResp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CRL server returned status %d", httpResp.StatusCode)
	}

	crlBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, r.config.Revocation.CRLMaxSize))
	if err != nil {
		return nil, err
	}

	// Try parsing as DER first
	crl, err := x509.ParseRevocationList(crlBytes)
	if err != nil {
		// Try PEM
		block, _ := pem.Decode(crlBytes)
		if block != nil {
			crl, err = x509.ParseRevocationList(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrCRLParseFailed, err)
			}
		} else {
			return nil, fmt.Errorf("%w: %w", ErrCRLParseFailed, err)
		}
	}

	return crl, nil
}

// checkCertificateAgainstCRL checks if a certificate is revoked according to CRL.
func (r *RevocationChecker) checkCertificateAgainstCRL(cert *x509.Certificate, crl *x509.RevocationList) (string, error) {
	// Check each revoked certificate
	for _, revoked := range crl.RevokedCertificateEntries {
		if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			return "revoked", nil
		}
	}

	return "good", nil
}

// checkCertificateInCRL checks if certificate is in cached CRL.
func (r *RevocationChecker) checkCertificateInCRL(cert *x509.Certificate, cachedCRL *CertificateRevocationList) (string, error) {
	// Parse the CRL
	block, _ := pem.Decode([]byte(cachedCRL.CRLPEM))
	if block == nil {
		return "unknown", errs.InternalServerErrorWithMessage("failed to decode cached CRL")
	}

	crl, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return "unknown", err
	}

	return r.checkCertificateAgainstCRL(cert, crl)
}

// storeCRL stores a CRL in the database.
func (r *RevocationChecker) storeCRL(ctx context.Context, crl *x509.RevocationList, issuer string) {
	// Encode CRL to PEM
	crlPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crl.Raw,
	})

	crlRecord := &CertificateRevocationList{
		ID:               generateID(),
		OrganizationID:   "", // Would need to determine from context
		Issuer:           issuer,
		ThisUpdate:       crl.ThisUpdate,
		NextUpdate:       crl.NextUpdate,
		CRLPEM:           string(crlPEM),
		Status:           "valid",
		RevokedCertCount: len(crl.RevokedCertificateEntries),
		LastFetchedAt:    time.Now(),
	}

	// Best effort storage - don't fail if it doesn't work
	_ = r.repo.CreateCRL(ctx, crlRecord)
}

// getIssuerCertificate gets the issuer certificate for OCSP requests.
func (r *RevocationChecker) getIssuerCertificate(ctx context.Context, cert *x509.Certificate) (*x509.Certificate, error) {
	issuerFingerprint := calculateFingerprint(cert.RawIssuer)

	// Try to find issuer in trust anchors
	anchor, err := r.repo.GetTrustAnchorByFingerprint(ctx, issuerFingerprint)
	if err == nil && anchor != nil {
		block, _ := pem.Decode([]byte(anchor.CertificatePEM))
		if block != nil {
			issuerCert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				return issuerCert, nil
			}
		}
	}

	// Try to get from AIA (Authority Information Access) extension
	if len(cert.IssuingCertificateURL) > 0 {
		for _, url := range cert.IssuingCertificateURL {
			issuerCert, err := r.fetchIssuerCertificate(ctx, url)
			if err == nil {
				return issuerCert, nil
			}
		}
	}

	return nil, errs.NotFound("issuer certificate not found")
}

// fetchIssuerCertificate fetches an issuer certificate from a URL.
func (r *RevocationChecker) fetchIssuerCertificate(ctx context.Context, url string) (*x509.Certificate, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	httpResp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch issuer certificate: status %d", httpResp.StatusCode)
	}

	certBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return nil, err
	}

	// Try parsing as DER
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		// Try PEM
		block, _ := pem.Decode(certBytes)
		if block != nil {
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return cert, nil
}

// CleanupExpiredCache removes expired OCSP responses from cache.
func (r *RevocationChecker) CleanupExpiredCache(ctx context.Context) error {
	return r.repo.DeleteExpiredOCSPResponses(ctx)
}
