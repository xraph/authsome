package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// Test helper to generate test certificates
func generateTestCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

func generateTestClientCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "Test Client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Convert to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return cert, privateKey, certPEM
}

func TestCertificateValidator_parseCertificate(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	cert, err := validator.parseCertificate(certPEM)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	if cert == nil {
		t.Fatal("expected certificate, got nil")
	}

	if cert.Subject.CommonName != "Test Client" {
		t.Errorf("expected CN=Test Client, got %s", cert.Subject.CommonName)
	}
}

func TestCertificateValidator_parseCertificate_InvalidPEM(t *testing.T) {
	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	_, err := validator.parseCertificate([]byte("invalid pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}

	if err != ErrInvalidPEM {
		t.Errorf("expected ErrInvalidPEM, got %v", err)
	}
}

func TestCertificateValidator_validateBasic(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	cert, _ := validator.parseCertificate(certPEM)
	result := &ValidationResult{
		ValidationSteps: make(map[string]interface{}),
	}

	err := validator.validateBasic(cert, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ValidationSteps["expiration"] != "passed" {
		t.Error("expected expiration validation to pass")
	}

	if result.ValidationSteps["notBefore"] != "passed" {
		t.Error("expected notBefore validation to pass")
	}
}

func TestCertificateValidator_validateBasic_Expired(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	// Generate expired certificate
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Expired Cert",
		},
		NotBefore: time.Now().Add(-2 * 365 * 24 * time.Hour),
		NotAfter:  time.Now().Add(-365 * 24 * time.Hour), // Expired
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	cert, _ := x509.ParseCertificate(certDER)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	result := &ValidationResult{
		ValidationSteps: make(map[string]interface{}),
	}

	err := validator.validateBasic(cert, result)
	if err == nil {
		t.Fatal("expected error for expired certificate")
	}

	if err != ErrCertificateExpired {
		t.Errorf("expected ErrCertificateExpired, got %v", err)
	}
}

func TestCertificateValidator_validateKey(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	cert, _ := validator.parseCertificate(certPEM)
	result := &ValidationResult{
		ValidationSteps: make(map[string]interface{}),
	}

	err := validator.validateKey(cert, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ValidationSteps["keyValidation"] != "passed" {
		t.Error("expected key validation to pass")
	}
}

func TestCertificateValidator_validateKey_WeakKey(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	// Generate certificate with weak key (1024 bits)
	privateKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Weak Key",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	cert, _ := x509.ParseCertificate(certDER)

	config := DefaultConfig()
	config.Validation.MinKeySize = 2048
	validator := NewCertificateValidator(config, nil, nil)

	result := &ValidationResult{
		ValidationSteps: make(map[string]interface{}),
	}

	err := validator.validateKey(cert, result)
	if err == nil {
		t.Fatal("expected error for weak key")
	}

	if err != ErrKeyTooWeak {
		t.Errorf("expected ErrKeyTooWeak, got %v", err)
	}
}

func TestCertificateValidator_validateKeyUsage(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	cert, _ := validator.parseCertificate(certPEM)
	result := &ValidationResult{
		ValidationSteps: make(map[string]interface{}),
	}

	err := validator.validateKeyUsage(cert, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ValidationSteps["keyUsage"] != "passed" {
		t.Error("expected key usage validation to pass")
	}
}

func TestGetKeySize(t *testing.T) {
	tests := []struct {
		name     string
		keyGen   func() interface{}
		expected int
	}{
		{
			name: "RSA 2048",
			keyGen: func() interface{} {
				key, _ := rsa.GenerateKey(rand.Reader, 2048)
				return &key.PublicKey
			},
			expected: 2048,
		},
		{
			name: "RSA 4096",
			keyGen: func() interface{} {
				key, _ := rsa.GenerateKey(rand.Reader, 4096)
				return &key.PublicKey
			},
			expected: 4096,
		},
		{
			name: "ECDSA P256",
			keyGen: func() interface{} {
				key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				return &key.PublicKey
			},
			expected: 256,
		},
		{
			name: "ECDSA P384",
			keyGen: func() interface{} {
				key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
				return &key.PublicKey
			},
			expected: 384,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyGen()
			size := getKeySize(key)
			if size != tt.expected {
				t.Errorf("expected key size %d, got %d", tt.expected, size)
			}
		})
	}
}

func TestCalculateFingerprint(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	block, _ := pem.Decode(certPEM)
	fingerprint1 := calculateFingerprint(block.Bytes)
	fingerprint2 := calculateFingerprint(block.Bytes)

	// Same certificate should produce same fingerprint
	if fingerprint1 != fingerprint2 {
		t.Error("fingerprints should be identical for same certificate")
	}

	// Fingerprint should be 64 hex characters (SHA-256)
	if len(fingerprint1) != 64 {
		t.Errorf("expected fingerprint length 64, got %d", len(fingerprint1))
	}
}

func TestHasKeyUsage(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Test",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER, _ := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	cert, _ := x509.ParseCertificate(certDER)

	if !hasKeyUsage(cert, "digitalSignature") {
		t.Error("expected digitalSignature to be present")
	}

	if !hasKeyUsage(cert, "keyEncipherment") {
		t.Error("expected keyEncipherment to be present")
	}

	if hasKeyUsage(cert, "dataEncipherment") {
		t.Error("dataEncipherment should not be present")
	}
}

func TestHasExtendedKeyUsage(t *testing.T) {
	caCert, caKey := generateTestCA(t)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Test",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER, _ := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, caKey)
	cert, _ := x509.ParseCertificate(certDER)

	if !hasExtendedKeyUsage(cert, "clientAuth") {
		t.Error("expected clientAuth to be present")
	}

	if !hasExtendedKeyUsage(cert, "serverAuth") {
		t.Error("expected serverAuth to be present")
	}
}

// Benchmark tests
func BenchmarkCertificateParsing(b *testing.B) {
	caCert, caKey := generateTestCA(&testing.T{})
	_, _, certPEM := generateTestClientCert(&testing.T{}, caCert, caKey)

	config := DefaultConfig()
	validator := NewCertificateValidator(config, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.parseCertificate(certPEM)
	}
}

func BenchmarkFingerprintCalculation(b *testing.B) {
	caCert, caKey := generateTestCA(&testing.T{})
	_, _, certPEM := generateTestClientCert(&testing.T{}, caCert, caKey)

	block, _ := pem.Decode(certPEM)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateFingerprint(block.Bytes)
	}
}

