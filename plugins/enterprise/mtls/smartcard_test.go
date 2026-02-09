package mtls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/xraph/authsome/internal/errs"
)

func generateTestPIVCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) *x509.Certificate {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// PIV Authentication OID
	pivAuthOID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 2, 1, 3, 7}

	// Manually create certificate policies extension
	type policyInformation struct {
		PolicyIdentifier asn1.ObjectIdentifier
	}

	policies := []policyInformation{
		{PolicyIdentifier: pivAuthOID},
	}

	policiesBytes, err := asn1.Marshal(policies)
	if err != nil {
		t.Fatalf("failed to marshal policies: %v", err)
	}

	// Add PIV-specific extensions
	cardUUID := "12345678-1234-1234-1234-123456789abc"
	uri, _ := url.Parse("urn:uuid:" + cardUUID)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"US Government"},
			CommonName:   "John Doe",
			SerialNumber: "1234567890",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		URIs:        []*url.URL{uri},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    asn1.ObjectIdentifier{2, 5, 29, 32}, // certificatePolicies OID
				Value: policiesBytes,
			},
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert
}

func generateTestCACCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) *x509.Certificate {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// CAC Authentication OID
	cacAuthOID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 2, 1, 11, 42}

	// Manually create certificate policies extension
	type policyInformation struct {
		PolicyIdentifier asn1.ObjectIdentifier
	}

	policies := []policyInformation{
		{PolicyIdentifier: cacAuthOID},
	}

	policiesBytes, err := asn1.Marshal(policies)
	if err != nil {
		t.Fatalf("failed to marshal policies: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"DoD"},
			CommonName:   "Jane Smith",
			SerialNumber: "9876543210", // EDIPI
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(3 * 365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    asn1.ObjectIdentifier{2, 5, 29, 32}, // certificatePolicies OID
				Value: policiesBytes,
			},
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert
}

func TestSmartCardProvider_ValidatePIVCard(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	pivCert := generateTestPIVCert(t, caCert, caKey)

	config := DefaultConfig()
	config.SmartCard.EnablePIV = true

	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	cardInfo, err := provider.ValidatePIVCard(context.Background(), pivCert)
	if err != nil {
		t.Fatalf("failed to validate PIV card: %v", err)
	}

	if cardInfo == nil {
		t.Fatal("expected PIV card info, got nil")
	}

	if cardInfo.CardholderUUID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("expected cardholder UUID, got %s", cardInfo.CardholderUUID)
	}

	if len(cardInfo.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cardInfo.Certificates))
	}

	if cardInfo.Certificates[0].SlotID != "9A" {
		t.Errorf("expected slot 9A, got %s", cardInfo.Certificates[0].SlotID)
	}

	if cardInfo.Certificates[0].SlotName != "PIV Authentication" {
		t.Errorf("expected PIV Authentication, got %s", cardInfo.Certificates[0].SlotName)
	}
}

func TestSmartCardProvider_ValidatePIVCard_Disabled(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	pivCert := generateTestPIVCert(t, caCert, caKey)

	config := DefaultConfig()
	config.SmartCard.EnablePIV = false

	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	_, err := provider.ValidatePIVCard(context.Background(), pivCert)
	if err == nil {
		t.Fatal("expected error when PIV is disabled")
	}
}

func TestSmartCardProvider_ValidatePIVCard_NotPIV(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	// Parse regular certificate
	block, _ := pem.Decode(certPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	config := DefaultConfig()
	config.SmartCard.EnablePIV = true

	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	_, err := provider.ValidatePIVCard(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error for non-PIV certificate")
	}

	if !errs.Is(err, ErrNotPIVCertificate) {
		t.Errorf("expected ErrNotPIVCertificate, got %v", err)
	}
}

func TestSmartCardProvider_ValidateCACCard(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	cacCert := generateTestCACCert(t, caCert, caKey)

	config := DefaultConfig()
	config.SmartCard.EnableCAC = true

	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	cardInfo, err := provider.ValidateCACCard(context.Background(), cacCert)
	if err != nil {
		t.Fatalf("failed to validate CAC card: %v", err)
	}

	if cardInfo == nil {
		t.Fatal("expected CAC card info, got nil")
	}

	if cardInfo.CACNumber != "9876543210" {
		t.Errorf("expected CAC number 9876543210, got %s", cardInfo.CACNumber)
	}

	if len(cardInfo.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cardInfo.Certificates))
	}
}

func TestSmartCardProvider_ValidateCACCard_NotCAC(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	// Parse regular certificate
	block, _ := pem.Decode(certPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	config := DefaultConfig()
	config.SmartCard.EnableCAC = true

	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	_, err := provider.ValidateCACCard(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error for non-CAC certificate")
	}

	if !errs.Is(err, ErrNotCACCertificate) {
		t.Errorf("expected ErrNotCACCertificate, got %v", err)
	}
}

func TestIsPIVCertificate(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	// Test PIV certificate
	pivCert := generateTestPIVCert(t, caCert, caKey)
	if !isPIVCertificate(pivCert) {
		t.Error("expected isPIVCertificate to return true for PIV cert")
	}

	// Test non-PIV certificate
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)
	block, _ := pem.Decode(certPEM)

	regularCert, _ := x509.ParseCertificate(block.Bytes)
	if isPIVCertificate(regularCert) {
		t.Error("expected isPIVCertificate to return false for regular cert")
	}
}

func TestIsCACCertificate(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	// Test CAC certificate
	cacCert := generateTestCACCert(t, caCert, caKey)
	if !isCACCertificate(cacCert) {
		t.Error("expected isCACCertificate to return true for CAC cert")
	}

	// Test non-CAC certificate
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)
	block, _ := pem.Decode(certPEM)

	regularCert, _ := x509.ParseCertificate(block.Bytes)
	if isCACCertificate(regularCert) {
		t.Error("expected isCACCertificate to return false for regular cert")
	}
}

func TestExtractPIVCardUUID(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	pivCert := generateTestPIVCert(t, caCert, caKey)

	uuid := extractPIVCardUUID(pivCert)
	if uuid != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("expected UUID 12345678-1234-1234-1234-123456789abc, got %s", uuid)
	}
}

func TestExtractCACNumber(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	cacCert := generateTestCACCert(t, caCert, caKey)

	cacNumber := extractCACNumber(cacCert)
	if cacNumber != "9876543210" {
		t.Errorf("expected CAC number 9876543210, got %s", cacNumber)
	}
}

func TestDeterminePIVSlot(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	pivCert := generateTestPIVCert(t, caCert, caKey)

	slotID, slotName := determinePIVSlot(pivCert)
	if slotID != "9A" {
		t.Errorf("expected slot 9A, got %s", slotID)
	}

	if slotName != "PIV Authentication" {
		t.Errorf("expected PIV Authentication, got %s", slotName)
	}
}

func TestGetPIVSlotNames(t *testing.T) {
	slotNames := GetPIVSlotNames()

	if len(slotNames) == 0 {
		t.Error("expected slot names, got empty map")
	}

	if slotNames["9A"] != "PIV Authentication" {
		t.Errorf("expected 9A to be PIV Authentication, got %s", slotNames["9A"])
	}

	if slotNames["9C"] != "Digital Signature" {
		t.Errorf("expected 9C to be Digital Signature, got %s", slotNames["9C"])
	}
}

func TestGetCACCertificateTypes(t *testing.T) {
	certTypes := GetCACCertificateTypes()

	if len(certTypes) == 0 {
		t.Error("expected certificate types, got empty slice")
	}

	found := slices.Contains(certTypes, "ID Certificate")

	if !found {
		t.Error("expected to find ID Certificate in types")
	}
}

func BenchmarkValidatePIVCard(b *testing.B) {
	caCert, caKey := generateTestCA(&testing.T{})
	pivCert := generateTestPIVCert(&testing.T{}, caCert, caKey)

	config := DefaultConfig()
	config.SmartCard.EnablePIV = true
	repo := newMockRepository()
	provider := NewSmartCardProvider(config, repo)

	for b.Loop() {
		_, _ = provider.ValidatePIVCard(context.Background(), pivCert)
	}
}
