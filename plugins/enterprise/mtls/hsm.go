package mtls

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"time"
)

// HSMProvider defines the interface for HSM providers
type HSMProvider interface {
	// Connect establishes connection to HSM
	Connect(ctx context.Context) error

	// Disconnect closes HSM connection
	Disconnect() error

	// GetKey retrieves a key from HSM
	GetKey(ctx context.Context, keyID string) (crypto.PrivateKey, error)

	// Sign performs a signing operation using HSM key
	Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error)

	// GetCertificate retrieves a certificate from HSM
	GetCertificate(ctx context.Context, keyID string) (*x509.Certificate, error)

	// ListKeys lists available keys in HSM
	ListKeys(ctx context.Context) ([]HSMKeyInfo, error)

	// ValidateKey validates that a key exists and is accessible
	ValidateKey(ctx context.Context, keyID string) error

	// GetProviderInfo returns HSM provider information
	GetProviderInfo() *HSMProviderInfo
}

// HSMKeyInfo contains information about an HSM key
type HSMKeyInfo struct {
	KeyID       string                 `json:"keyId"`
	Label       string                 `json:"label"`
	Algorithm   string                 `json:"algorithm"`
	KeySize     int                    `json:"keySize"`
	Certificate *x509.Certificate      `json:"-"`
	CreatedAt   time.Time              `json:"createdAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HSMProviderInfo contains HSM provider information
type HSMProviderInfo struct {
	Provider     string            `json:"provider"`
	Version      string            `json:"version"`
	Model        string            `json:"model,omitempty"`
	SerialNumber string            `json:"serialNumber,omitempty"`
	Capabilities []string          `json:"capabilities"`
	Connected    bool              `json:"connected"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// HSMManager manages HSM connections and operations
type HSMManager struct {
	config    *Config
	repo      Repository
	providers map[string]HSMProvider
}

// NewHSMManager creates a new HSM manager
func NewHSMManager(config *Config, repo Repository) *HSMManager {
	return &HSMManager{
		config:    config,
		repo:      repo,
		providers: make(map[string]HSMProvider),
	}
}

// Init initializes HSM providers based on configuration
func (m *HSMManager) Init(ctx context.Context) error {
	if !m.config.HSM.Enabled {
		return nil
	}

	// Initialize provider based on configuration
	switch m.config.HSM.Provider {
	case "pkcs11":
		provider := NewPKCS11Provider(m.config)
		if err := provider.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to PKCS#11 HSM: %w", err)
		}
		m.providers["pkcs11"] = provider

	case "cloudhsm":
		provider := NewCloudHSMProvider(m.config)
		if err := provider.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to AWS CloudHSM: %w", err)
		}
		m.providers["cloudhsm"] = provider

	case "azure":
		provider := NewAzureKeyVaultProvider(m.config)
		if err := provider.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to Azure Key Vault: %w", err)
		}
		m.providers["azure"] = provider

	case "gcp":
		provider := NewGCPCloudHSMProvider(m.config)
		if err := provider.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to GCP Cloud HSM: %w", err)
		}
		m.providers["gcp"] = provider

	default:
		return fmt.Errorf("%w: %s", ErrHSMProviderUnsupported, m.config.HSM.Provider)
	}

	return nil
}

// GetProvider returns an HSM provider by name
func (m *HSMManager) GetProvider(name string) (HSMProvider, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("HSM provider not found: %s", name)
	}
	return provider, nil
}

// ValidateCertificateHSMBinding validates that a certificate is backed by HSM key
func (m *HSMManager) ValidateCertificateHSMBinding(ctx context.Context, cert *Certificate) error {
	if cert.HSMKeyID == "" {
		return fmt.Errorf("certificate does not have HSM key binding")
	}

	if cert.HSMProvider == "" {
		return fmt.Errorf("certificate does not specify HSM provider")
	}

	provider, err := m.GetProvider(cert.HSMProvider)
	if err != nil {
		return err
	}

	// Validate that the key exists and is accessible
	if err := provider.ValidateKey(ctx, cert.HSMKeyID); err != nil {
		return fmt.Errorf("HSM key validation failed: %w", err)
	}

	return nil
}

// Shutdown closes all HSM connections
func (m *HSMManager) Shutdown() error {
	var lastErr error
	for _, provider := range m.providers {
		if err := provider.Disconnect(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// PKCS11Provider implements HSM provider for PKCS#11 devices
type PKCS11Provider struct {
	config    *Config
	connected bool
	// In a real implementation, this would contain pkcs11 context
}

// NewPKCS11Provider creates a new PKCS#11 provider
func NewPKCS11Provider(config *Config) *PKCS11Provider {
	return &PKCS11Provider{
		config: config,
	}
}

func (p *PKCS11Provider) Connect(ctx context.Context) error {
	// In production, initialize PKCS#11 library
	// Example: p11, err := crypto11.Configure(&crypto11.Config{
	//     Path:       p.config.HSM.PKCS11Library,
	//     TokenLabel: "token-label",
	//     Pin:        p.config.HSM.PKCS11PIN,
	// })

	p.connected = true
	return nil
}

func (p *PKCS11Provider) Disconnect() error {
	p.connected = false
	return nil
}

func (p *PKCS11Provider) GetKey(ctx context.Context, keyID string) (crypto.PrivateKey, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	// In production, retrieve key from PKCS#11 device
	return nil, ErrHSMKeyNotFound
}

func (p *PKCS11Provider) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	// In production, perform signing operation via PKCS#11
	return nil, ErrHSMOperationFailed
}

func (p *PKCS11Provider) GetCertificate(ctx context.Context, keyID string) (*x509.Certificate, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	// In production, retrieve certificate from PKCS#11 device
	return nil, ErrHSMKeyNotFound
}

func (p *PKCS11Provider) ListKeys(ctx context.Context) ([]HSMKeyInfo, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	// In production, enumerate keys from PKCS#11 device
	return []HSMKeyInfo{}, nil
}

func (p *PKCS11Provider) ValidateKey(ctx context.Context, keyID string) error {
	if !p.connected {
		return ErrHSMConnectionFailed
	}
	// In production, check if key exists and is accessible
	return nil
}

func (p *PKCS11Provider) GetProviderInfo() *HSMProviderInfo {
	return &HSMProviderInfo{
		Provider:     "PKCS#11",
		Version:      "2.40",
		Connected:    p.connected,
		Capabilities: []string{"signing", "key_storage", "certificate_storage"},
	}
}

// CloudHSMProvider implements HSM provider for AWS CloudHSM
type CloudHSMProvider struct {
	config    *Config
	connected bool
	// In real implementation, contains AWS CloudHSM client
}

// NewCloudHSMProvider creates a new AWS CloudHSM provider
func NewCloudHSMProvider(config *Config) *CloudHSMProvider {
	return &CloudHSMProvider{
		config: config,
	}
}

func (p *CloudHSMProvider) Connect(ctx context.Context) error {
	// In production, initialize AWS CloudHSM SDK
	// Example: session, err := session.NewSession(&aws.Config{
	//     Region: aws.String(p.config.HSM.CloudHSMRegion),
	// })
	// p.client = cloudhsmv2.New(session)

	p.connected = true
	return nil
}

func (p *CloudHSMProvider) Disconnect() error {
	p.connected = false
	return nil
}

func (p *CloudHSMProvider) GetKey(ctx context.Context, keyID string) (crypto.PrivateKey, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *CloudHSMProvider) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMOperationFailed
}

func (p *CloudHSMProvider) GetCertificate(ctx context.Context, keyID string) (*x509.Certificate, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *CloudHSMProvider) ListKeys(ctx context.Context) ([]HSMKeyInfo, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return []HSMKeyInfo{}, nil
}

func (p *CloudHSMProvider) ValidateKey(ctx context.Context, keyID string) error {
	if !p.connected {
		return ErrHSMConnectionFailed
	}
	return nil
}

func (p *CloudHSMProvider) GetProviderInfo() *HSMProviderInfo {
	return &HSMProviderInfo{
		Provider:     "AWS CloudHSM",
		Version:      "2.0",
		Connected:    p.connected,
		Capabilities: []string{"signing", "encryption", "key_generation"},
		Metadata: map[string]string{
			"region":    p.config.HSM.CloudHSMRegion,
			"clusterId": p.config.HSM.CloudHSMClusterID,
		},
	}
}

// AzureKeyVaultProvider implements HSM provider for Azure Key Vault
type AzureKeyVaultProvider struct {
	config    *Config
	connected bool
	// In real implementation, contains Azure Key Vault client
}

// NewAzureKeyVaultProvider creates a new Azure Key Vault provider
func NewAzureKeyVaultProvider(config *Config) *AzureKeyVaultProvider {
	return &AzureKeyVaultProvider{
		config: config,
	}
}

func (p *AzureKeyVaultProvider) Connect(ctx context.Context) error {
	// In production, initialize Azure SDK
	// Example: cred, err := azidentity.NewDefaultAzureCredential(nil)
	// p.client, err = azkeys.NewClient(p.config.HSM.AzureVaultURL, cred, nil)

	p.connected = true
	return nil
}

func (p *AzureKeyVaultProvider) Disconnect() error {
	p.connected = false
	return nil
}

func (p *AzureKeyVaultProvider) GetKey(ctx context.Context, keyID string) (crypto.PrivateKey, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *AzureKeyVaultProvider) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMOperationFailed
}

func (p *AzureKeyVaultProvider) GetCertificate(ctx context.Context, keyID string) (*x509.Certificate, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *AzureKeyVaultProvider) ListKeys(ctx context.Context) ([]HSMKeyInfo, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return []HSMKeyInfo{}, nil
}

func (p *AzureKeyVaultProvider) ValidateKey(ctx context.Context, keyID string) error {
	if !p.connected {
		return ErrHSMConnectionFailed
	}
	return nil
}

func (p *AzureKeyVaultProvider) GetProviderInfo() *HSMProviderInfo {
	return &HSMProviderInfo{
		Provider:     "Azure Key Vault",
		Version:      "1.0",
		Connected:    p.connected,
		Capabilities: []string{"signing", "encryption", "certificate_management"},
		Metadata: map[string]string{
			"vaultUrl": p.config.HSM.AzureVaultURL,
			"tenantId": p.config.HSM.AzureTenantID,
		},
	}
}

// GCPCloudHSMProvider implements HSM provider for GCP Cloud HSM
type GCPCloudHSMProvider struct {
	config    *Config
	connected bool
	// In real implementation, contains GCP KMS client
}

// NewGCPCloudHSMProvider creates a new GCP Cloud HSM provider
func NewGCPCloudHSMProvider(config *Config) *GCPCloudHSMProvider {
	return &GCPCloudHSMProvider{
		config: config,
	}
}

func (p *GCPCloudHSMProvider) Connect(ctx context.Context) error {
	// In production, initialize GCP KMS client
	// Example: client, err := kms.NewKeyManagementClient(ctx)
	// p.client = client

	p.connected = true
	return nil
}

func (p *GCPCloudHSMProvider) Disconnect() error {
	p.connected = false
	return nil
}

func (p *GCPCloudHSMProvider) GetKey(ctx context.Context, keyID string) (crypto.PrivateKey, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *GCPCloudHSMProvider) Sign(ctx context.Context, keyID string, digest []byte) ([]byte, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMOperationFailed
}

func (p *GCPCloudHSMProvider) GetCertificate(ctx context.Context, keyID string) (*x509.Certificate, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return nil, ErrHSMKeyNotFound
}

func (p *GCPCloudHSMProvider) ListKeys(ctx context.Context) ([]HSMKeyInfo, error) {
	if !p.connected {
		return nil, ErrHSMConnectionFailed
	}
	return []HSMKeyInfo{}, nil
}

func (p *GCPCloudHSMProvider) ValidateKey(ctx context.Context, keyID string) error {
	if !p.connected {
		return ErrHSMConnectionFailed
	}
	return nil
}

func (p *GCPCloudHSMProvider) GetProviderInfo() *HSMProviderInfo {
	return &HSMProviderInfo{
		Provider:     "GCP Cloud HSM",
		Version:      "1.0",
		Connected:    p.connected,
		Capabilities: []string{"signing", "encryption", "key_management"},
		Metadata: map[string]string{
			"projectId": p.config.HSM.GCPProjectID,
			"location":  p.config.HSM.GCPLocation,
			"keyRing":   p.config.HSM.GCPKeyRing,
		},
	}
}
