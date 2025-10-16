package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Code generation commands",
	Long:  `Commands for generating keys, configurations, and boilerplate code.`,
}

var generateKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Generate RSA key pair for JWT/OIDC",
	Long:  `Generate RSA private and public key pair for JWT signing and OIDC provider.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keySize, _ := cmd.Flags().GetInt("size")
		outputDir, _ := cmd.Flags().GetString("output")

		// Ensure output directory exists
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Generate private key
		privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
		if err != nil {
			return fmt.Errorf("failed to generate private key: %w", err)
		}

		// Encode private key to PEM
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}

		// Write private key
		privateKeyPath := filepath.Join(outputDir, "private.pem")
		privateKeyFile, err := os.Create(privateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to create private key file: %w", err)
		}
		defer privateKeyFile.Close()

		if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
			return fmt.Errorf("failed to encode private key: %w", err)
		}

		// Generate public key
		publicKey := &privateKey.PublicKey
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return fmt.Errorf("failed to marshal public key: %w", err)
		}

		// Encode public key to PEM
		publicKeyPEM := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}

		// Write public key
		publicKeyPath := filepath.Join(outputDir, "public.pem")
		publicKeyFile, err := os.Create(publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to create public key file: %w", err)
		}
		defer publicKeyFile.Close()

		if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
			return fmt.Errorf("failed to encode public key: %w", err)
		}

		fmt.Printf("Generated RSA key pair:\n")
		fmt.Printf("  Private key: %s\n", privateKeyPath)
		fmt.Printf("  Public key: %s\n", publicKeyPath)
		fmt.Printf("  Key size: %d bits\n", keySize)

		return nil
	},
}

var generateConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate sample configuration file",
	Long:  `Generate a sample configuration file with all available options.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, _ := cmd.Flags().GetString("output")
		mode, _ := cmd.Flags().GetString("mode")

		var config string
		if mode == "saas" {
			config = generateSaaSConfig()
		} else {
			config = generateStandaloneConfig()
		}

		if err := os.WriteFile(outputPath, []byte(config), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Generated %s mode configuration: %s\n", mode, outputPath)
		return nil
	},
}

var generateSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Generate cryptographically secure secret",
	Long:  `Generate a cryptographically secure secret for session signing or other purposes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		length, _ := cmd.Flags().GetInt("length")

		bytes := make([]byte, length)
		if _, err := rand.Read(bytes); err != nil {
			return fmt.Errorf("failed to generate secret: %w", err)
		}

		// Convert to hex string
		secret := fmt.Sprintf("%x", bytes)
		fmt.Printf("Generated secret (%d bytes): %s\n", length, secret)

		return nil
	},
}

func init() {
	generateCmd.AddCommand(generateKeysCmd)
	generateCmd.AddCommand(generateConfigCmd)
	generateCmd.AddCommand(generateSecretCmd)

	// Keys command flags
	generateKeysCmd.Flags().Int("size", 2048, "RSA key size in bits")
	generateKeysCmd.Flags().String("output", "./keys", "Output directory for keys")

	// Config command flags
	generateConfigCmd.Flags().String("output", "./authsome.yaml", "Output path for config file")
	generateConfigCmd.Flags().String("mode", "standalone", "Configuration mode (standalone|saas)")

	// Secret command flags
	generateSecretCmd.Flags().Int("length", 32, "Secret length in bytes")
}

func generateStandaloneConfig() string {
	return `# AuthSome Standalone Configuration
mode: standalone

# Database configuration
database:
  url: "authsome.db"
  # For PostgreSQL: "postgres://user:password@localhost/authsome?sslmode=disable"
  # For MySQL: "mysql://user:password@localhost/authsome"

# Server configuration
server:
  host: "localhost"
  port: 8080
  cors:
    enabled: true
    origins: ["http://localhost:3000"]

# Session configuration
session:
  secret: "your-session-secret-here"
  maxAge: 86400 # 24 hours
  secure: false # Set to true in production with HTTPS
  sameSite: "lax"

# Rate limiting
rateLimit:
  enabled: true
  requests: 100
  window: 3600 # 1 hour

# Email configuration
email:
  provider: "smtp"
  smtp:
    host: "smtp.gmail.com"
    port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "noreply@yourapp.com"

# Authentication plugins
plugins:
  username:
    enabled: true
  twofa:
    enabled: true
    issuer: "YourApp"
  magiclink:
    enabled: true
    expiresIn: 900 # 15 minutes
  oauth:
    google:
      enabled: false
      clientId: "your-google-client-id"
      clientSecret: "your-google-client-secret"
    github:
      enabled: false
      clientId: "your-github-client-id"
      clientSecret: "your-github-client-secret"

# JWT configuration
jwt:
  secret: "your-jwt-secret-here"
  expiresIn: 3600 # 1 hour
  issuer: "authsome"
  audience: "your-app"

# Webhooks
webhooks:
  enabled: false
  endpoints: []

# Audit logging
audit:
  enabled: true
  retention: 2160 # 90 days in hours
`
}

func generateSaaSConfig() string {
	return `# AuthSome SaaS Configuration
mode: saas

# Database configuration
database:
  url: "postgres://user:password@localhost/authsome?sslmode=disable"

# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  cors:
    enabled: true
    origins: ["https://yourapp.com", "https://app.yourapp.com"]

# Session configuration
session:
  secret: "your-session-secret-here"
  maxAge: 86400 # 24 hours
  secure: true
  sameSite: "strict"

# Rate limiting
rateLimit:
  enabled: true
  requests: 1000
  window: 3600 # 1 hour

# Redis for session storage and rate limiting
redis:
  url: "redis://localhost:6379"
  prefix: "authsome:"

# Email configuration
email:
  provider: "smtp"
  smtp:
    host: "smtp.sendgrid.net"
    port: 587
    username: "apikey"
    password: "your-sendgrid-api-key"
    from: "noreply@yourapp.com"

# Default organization settings
defaultOrg:
  name: "Default Organization"
  slug: "default"

# Multi-tenant configuration
multiTenant:
  enabled: true
  allowOrgCreation: true
  maxOrgsPerUser: 5

# Authentication plugins (default for all orgs)
plugins:
  username:
    enabled: true
  twofa:
    enabled: true
    issuer: "YourApp"
  magiclink:
    enabled: true
    expiresIn: 900 # 15 minutes
  oauth:
    google:
      enabled: true
      clientId: "your-google-client-id"
      clientSecret: "your-google-client-secret"
    github:
      enabled: true
      clientId: "your-github-client-id"
      clientSecret: "your-github-client-secret"
  sso:
    enabled: true
  oidcProvider:
    enabled: true
    privateKeyPath: "./keys/private.pem"
    publicKeyPath: "./keys/public.pem"

# JWT configuration
jwt:
  secret: "your-jwt-secret-here"
  expiresIn: 3600 # 1 hour
  issuer: "authsome"
  audience: "your-app"

# Webhooks
webhooks:
  enabled: true
  maxPerOrg: 10

# Audit logging
audit:
  enabled: true
  retention: 8760 # 1 year in hours

# Organization-specific overrides example
orgs:
  acme-corp:
    plugins:
      oauth:
        google:
          clientId: "acme-specific-google-client-id"
          clientSecret: "acme-specific-google-client-secret"
    webhooks:
      endpoints:
        - url: "https://acme-corp.com/webhooks/auth"
          events: ["user.created", "user.updated"]
          secret: "acme-webhook-secret"
`
}
