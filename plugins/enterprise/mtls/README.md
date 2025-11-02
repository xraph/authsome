# Certificate-Based Authentication (mTLS) Plugin

Enterprise-grade mutual TLS (mTLS) authentication plugin for AuthSome, providing X.509 certificate validation, PIV/CAC smart card support, Hardware Security Module (HSM) integration, and comprehensive certificate lifecycle management.

## 🎯 Overview

The mTLS plugin enables secure authentication using client certificates, designed for high-security environments including:

- **Government & Defense**: PIV/CAC card authentication for federal employees and military personnel
- **IoT & Machine-to-Machine**: Device authentication with certificate-based identity
- **Enterprise Applications**: Strong authentication for APIs and services
- **Banking & Finance**: Hardware-backed certificate authentication with HSM
- **Healthcare**: HIPAA-compliant authentication with audit trails

## ✨ Features

### Core Features

- ✅ **X.509 Certificate Validation**
  - Full chain validation with configurable trust anchors
  - Signature verification and cryptographic validation
  - Key usage and extended key usage validation
  - Certificate expiration and validity checks
  - Configurable key size and algorithm requirements

- ✅ **Revocation Checking**
  - CRL (Certificate Revocation List) support with caching
  - OCSP (Online Certificate Status Protocol) with stapling
  - Automatic revocation status caching
  - Configurable fail-open/fail-closed behavior

- ✅ **PIV/CAC Smart Card Support**
  - Personal Identity Verification (PIV) cards
  - Common Access Card (CAC) authentication
  - Multi-slot certificate support (9A, 9C, 9D, 9E)
  - Smart card PIN handling
  - Card reader integration

- ✅ **Hardware Security Module (HSM) Integration**
  - PKCS#11 standard support
  - AWS CloudHSM integration
  - Azure Key Vault integration
  - GCP Cloud HSM integration
  - Hardware-backed key operations

- ✅ **Certificate Lifecycle Management**
  - Certificate registration and tracking
  - Expiration monitoring and alerts
  - Certificate pinning support
  - Revocation management
  - Usage statistics and audit logs

- ✅ **Policy-Based Validation**
  - Organization-specific certificate policies
  - Configurable validation rules
  - Trust anchor management
  - Certificate type restrictions
  - Compliance enforcement

## 📦 Installation

### 1. Add Plugin to Your Application

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/mtls"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New()
    
    // Register mTLS plugin
    auth.Use(mtls.NewPlugin())
    
    // Mount on Forge app
    app := forge.New()
    auth.Mount(app)
    
    app.Run(":8080")
}
```

### 2. Configure in YAML

Create `config.yaml`:

```yaml
auth:
  mtls:
    enabled: true
    
    # Certificate Validation
    validation:
      checkExpiration: true
      checkNotBefore: true
      checkSignature: true
      checkKeyUsage: true
      validateChain: true
      maxChainLength: 5
      allowSelfSigned: false
      minKeySize: 2048
      allowedKeyAlgorithms:
        - RSA
        - ECDSA
        - Ed25519
      maxCertificateAge: 365
      minRemainingValidity: 30
      requiredKeyUsage:
        - digitalSignature
        - keyEncipherment
      requiredEku:
        - clientAuth
    
    # Revocation Checking
    revocation:
      enableCrl: true
      crlCacheDuration: 24h
      autoFetchCrl: true
      enableOcsp: true
      ocspCacheDuration: 1h
      ocspStapling: true
      failOpen: false  # Fail closed for security
      preferOcsp: true
    
    # Smart Card Support
    smartCard:
      enabled: true
      enablePiv: true
      pivAuthCertOnly: true
      enableCac: true
      requirePin: false
      maxPinAttempts: 3
    
    # HSM Integration (optional)
    hsm:
      enabled: false
      provider: pkcs11  # pkcs11, cloudhsm, azure, gcp
      # pkcs11Library: /usr/lib/softhsm/libsofthsm2.so
      # pkcs11SlotId: 0
      # pkcs11Pin: "1234"
    
    # API Configuration
    api:
      basePath: /auth/mtls
      enableManagement: true
      enableValidation: true
      enableMetrics: true
    
    # Security
    security:
      rateLimitEnabled: true
      maxAttemptsPerMinute: 10
      auditAllAttempts: true
      storeCertificates: true
      notifyOnExpiration: true
      expirationWarning: 30
```

## 🚀 Quick Start

### Standard Certificate Authentication

```go
package main

import (
    "crypto/tls"
    "log"
    
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/mtls"
    "github.com/xraph/forge"
)

func main() {
    // Create AuthSome with mTLS plugin
    auth := authsome.New(authsome.WithConfig("config.yaml"))
    auth.Use(mtls.NewPlugin())
    
    // Create Forge app with TLS
    app := forge.New()
    
    // Configure TLS with client certificate requirement
    tlsConfig := &tls.Config{
        ClientAuth: tls.RequireAndVerifyClientCert,
        MinVersion: tls.VersionTLS12,
    }
    
    // Mount AuthSome
    auth.Mount(app)
    
    // Protected route requiring certificate
    app.POST("/api/secure", func(c *forge.Context) error {
        // Certificate authentication happens automatically
        userID := c.GetString("userId")
        certID := c.GetString("certificateId")
        
        return c.JSON(200, forge.Map{
            "message": "Authenticated via certificate",
            "userId": userID,
            "certificateId": certID,
        })
    })
    
    // Start with TLS
    log.Fatal(app.RunTLS(":8443", "server.crt", "server.key", tlsConfig))
}
```

### Register a Certificate

```bash
curl -X POST https://api.example.com/auth/mtls/certificates \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "org_123",
    "userId": "user_456",
    "certificatePem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "certificateType": "user",
    "certificateClass": "standard"
  }'
```

### Authenticate with Certificate

```bash
curl -X POST https://api.example.com/auth/mtls/authenticate \
  --cert client.crt \
  --key client.key \
  --cacert ca.crt \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "org_123"
  }'
```

## 📚 Use Cases

### 1. PIV Card Authentication (Government)

```yaml
auth:
  mtls:
    enabled: true
    smartCard:
      enabled: true
      enablePiv: true
      pivAuthCertOnly: true  # Only accept PIV auth certificates (slot 9A)
      pivRequiredOids:
        - "2.16.840.1.101.3.2.1.3.7"  # PIV Authentication OID
    
    validation:
      minKeySize: 2048
      requiredEku:
        - clientAuth
```

**Usage:**

```go
// Register PIV certificate
req := &mtls.RegisterCertificateRequest{
    OrganizationID: "gov_agency",
    UserID: "john.doe",
    CertificatePEM: pivCertPEM,
    CertificateType: "user",
    CertificateClass: "piv",
    PIVCardID: "card_12345",
}

cert, err := mtlsPlugin.Service().RegisterCertificate(ctx, req)
```

### 2. CAC Authentication (Military/DoD)

```yaml
auth:
  mtls:
    smartCard:
      enabled: true
      enableCac: true
      cacRequiredOids:
        - "2.16.840.1.101.2.1.11.42"  # CAC Authentication OID
```

### 3. IoT Device Authentication

```yaml
auth:
  mtls:
    enabled: true
    validation:
      allowSelfSigned: false
      maxCertificateAge: 365
    
    revocation:
      enableOcsp: true
      enableCrl: true
      failOpen: false  # Strict revocation checking
```

**Register IoT Device:**

```bash
curl -X POST https://api.example.com/auth/mtls/certificates \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "org_123",
    "deviceId": "device_sensor_001",
    "certificatePem": "-----BEGIN CERTIFICATE-----\n...",
    "certificateType": "device",
    "certificateClass": "standard",
    "metadata": {
      "deviceType": "temperature_sensor",
      "location": "warehouse_a"
    }
  }'
```

### 4. HSM-Backed Authentication (Banking)

```yaml
auth:
  mtls:
    hsm:
      enabled: true
      provider: pkcs11
      pkcs11Library: /usr/lib/libpkcs11.so
      pkcs11SlotId: 0
      requireHsm: true  # Only accept HSM-backed certificates
```

**AWS CloudHSM Example:**

```yaml
auth:
  mtls:
    hsm:
      enabled: true
      provider: cloudhsm
      cloudHsmClusterId: cluster-abc123
      cloudHsmRegion: us-east-1
      requireHsm: true
```

### 5. API Machine-to-Machine Authentication

```go
// Service-to-service authentication
app.Use(func(c *forge.Context) error {
    // Verify client certificate
    if c.Request().TLS == nil || len(c.Request().TLS.PeerCertificates) == 0 {
        return c.JSON(401, forge.Map{"error": "certificate required"})
    }
    
    cert := c.Request().TLS.PeerCertificates[0]
    
    // Validate certificate through mTLS plugin
    result, err := mtlsPlugin.Service().AuthenticateWithCertificate(
        c.Context(),
        certToPEM(cert.Raw),
        "org_123",
    )
    
    if err != nil || !result.Success {
        return c.JSON(401, forge.Map{"error": "authentication failed"})
    }
    
    // Set context for downstream handlers
    c.Set("userId", result.UserID)
    c.Set("certificateId", result.CertificateID)
    
    return c.Next()
})
```

## 🔧 Advanced Configuration

### Certificate Policies

Create organization-specific certificate policies:

```bash
curl -X POST https://api.example.com/auth/mtls/policies \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "org_123",
    "name": "High Security Policy",
    "description": "Policy for sensitive operations",
    "requirePinning": true,
    "requireCrlCheck": true,
    "requireOcspCheck": true,
    "minKeySize": 4096,
    "allowedKeyAlgorithms": ["RSA", "ECDSA"],
    "maxCertificateAge": 180,
    "minRemainingValidity": 60,
    "requirePiv": true,
    "isDefault": true
  }'
```

### Trust Anchor Management

Add custom CA certificates:

```bash
curl -X POST https://api.example.com/auth/mtls/trust-anchors \
  -H "Content-Type: application/json" \
  -d '{
    "organizationId": "org_123",
    "name": "Corporate Root CA",
    "certificatePem": "-----BEGIN CERTIFICATE-----\n...",
    "trustLevel": "root"
  }'
```

### Certificate Pinning

```go
// Register certificate with pinning
req := &mtls.RegisterCertificateRequest{
    OrganizationID: "org_123",
    UserID: "user_456",
    CertificatePEM: certPEM,
    IsPinned: true,  // Enable pinning
}
```

## 📊 Monitoring & Statistics

### Get Authentication Statistics

```bash
curl -X GET "https://api.example.com/auth/mtls/stats/auth?organizationId=org_123&since=2025-01-01T00:00:00Z"
```

Response:
```json
{
  "totalAttempts": 1523,
  "successfulAuths": 1487,
  "failedAuths": 36,
  "validationErrors": 12,
  "uniqueUsers": 145,
  "uniqueCerts": 203
}
```

### Get Expiring Certificates

```bash
curl -X GET "https://api.example.com/auth/mtls/certificates/expiring?organizationId=org_123&days=30"
```

## 🔒 Security Best Practices

### 1. Always Use Strong Keys

```yaml
validation:
  minKeySize: 2048  # Minimum 2048 bits for RSA
  allowedKeyAlgorithms:
    - RSA
    - ECDSA  # Prefer ECDSA for better performance
```

### 2. Enable Revocation Checking

```yaml
revocation:
  enableCrl: true
  enableOcsp: true
  failOpen: false  # Fail closed - reject if revocation unavailable
```

### 3. Implement Certificate Rotation

- Set reasonable certificate lifetimes (e.g., 1 year)
- Monitor expiring certificates
- Implement automated renewal processes

### 4. Audit Everything

```yaml
security:
  auditAllAttempts: true
  auditFailures: true
  auditValidation: true
```

### 5. Use Certificate Pinning for High Security

```go
req.IsPinned = true  // Bind session to specific certificate
```

## 🧪 Testing

### Generate Test Certificates

```bash
# Generate CA
openssl req -x509 -new -nodes -keyout ca.key -sha256 -days 365 -out ca.crt -subj "/CN=Test CA"

# Generate client key and CSR
openssl req -new -nodes -keyout client.key -out client.csr -subj "/CN=Test Client"

# Sign client certificate
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -sha256
```

### Test Authentication

```go
package main

import (
    "testing"
    "crypto/x509"
    "encoding/pem"
)

func TestCertificateAuth(t *testing.T) {
    // Load test certificate
    certPEM := loadTestCertificate()
    
    // Authenticate
    result, err := service.AuthenticateWithCertificate(
        context.Background(),
        certPEM,
        "test_org",
    )
    
    if err != nil {
        t.Fatalf("Authentication failed: %v", err)
    }
    
    if !result.Success {
        t.Fatalf("Expected successful authentication")
    }
}
```

## 📖 API Reference

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/mtls/certificates` | Register a certificate |
| GET | `/auth/mtls/certificates` | List certificates |
| GET | `/auth/mtls/certificates/:id` | Get certificate details |
| POST | `/auth/mtls/certificates/:id/revoke` | Revoke a certificate |
| GET | `/auth/mtls/certificates/expiring` | Get expiring certificates |
| POST | `/auth/mtls/authenticate` | Authenticate with certificate |
| POST | `/auth/mtls/trust-anchors` | Add trust anchor |
| GET | `/auth/mtls/trust-anchors` | List trust anchors |
| POST | `/auth/mtls/policies` | Create certificate policy |
| GET | `/auth/mtls/policies/:id` | Get policy |
| POST | `/auth/mtls/validate` | Validate certificate |
| GET | `/auth/mtls/stats/auth` | Get authentication statistics |

## 🤝 Integration Examples

### With Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: client-cert
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-cert>
  tls.key: <base64-encoded-key>
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        volumeMounts:
        - name: client-cert
          mountPath: /etc/certs
          readOnly: true
      volumes:
      - name: client-cert
        secret:
          secretName: client-cert
```

### With NGINX

```nginx
server {
    listen 443 ssl;
    ssl_certificate /path/to/server.crt;
    ssl_certificate_key /path/to/server.key;
    
    # Client certificate verification
    ssl_client_certificate /path/to/ca.crt;
    ssl_verify_client on;
    ssl_verify_depth 2;
    
    location /api {
        proxy_pass http://backend;
        proxy_set_header X-SSL-Client-Cert $ssl_client_cert;
        proxy_set_header X-SSL-Client-Verify $ssl_client_verify;
    }
}
```

## 🐛 Troubleshooting

### Certificate Validation Fails

```bash
# Test certificate validation
curl -X POST https://api.example.com/auth/mtls/validate \
  -H "Content-Type: application/json" \
  -d '{
    "certificatePem": "-----BEGIN CERTIFICATE-----\n...",
    "organizationId": "org_123"
  }'
```

### Check Certificate Chain

```bash
openssl verify -CAfile ca.crt -untrusted intermediate.crt client.crt
```

### Debug OCSP

```bash
openssl ocsp -issuer ca.crt -cert client.crt -url http://ocsp.example.com -text
```

## 📝 License

Part of AuthSome - Enterprise Authentication Framework

## 🔗 Related Documentation

- [X.509 Certificate Standards](https://datatracker.ietf.org/doc/html/rfc5280)
- [PIV Specifications](https://csrc.nist.gov/publications/detail/fips/201/3/final)
- [PKCS#11 Standard](http://docs.oasis-open.org/pkcs11/pkcs11-base/v2.40/os/pkcs11-base-v2.40-os.html)
- [mTLS Best Practices](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/)

## 🤝 Support

For issues, questions, or contributions, please visit the [AuthSome repository](https://github.com/xraph/authsome).

