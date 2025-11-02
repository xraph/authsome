# Enterprise Plugins

This directory contains enterprise-grade plugins for AuthSome.

## Available Plugins

### 1. Identity Verification (KYC)
**Path**: `plugins/enterprise/idverification`

Complete identity verification solution with multi-provider support.

**Features:**
- ✅ Stripe Identity integration (production-ready)
- ✅ Document verification
- ✅ Liveness detection
- ✅ Age verification
- ✅ AML/sanctions screening
- ✅ Webhook support
- ✅ Mock mode for testing
- ⚠️ Onfido & Jumio (placeholder, needs SDK integration)

**Use Cases:**
- Fintech applications
- Healthcare platforms
- Age-restricted content
- Regulated industries
- KYC/AML compliance

**Status**: ✅ Production Ready (Stripe)

**Documentation**:
- [Complete README](./idverification/README.md)
- [Stripe Integration Guide](./idverification/STRIPE_INTEGRATION.md)
- [SDK Integration Guide](./idverification/SDK_INTEGRATION_GUIDE.md)
- [Usage Examples](./idverification/EXAMPLE.md)

**Quick Start:**
```go
import "github.com/xraph/authsome/plugins/enterprise/idverification"

// Create plugin
plugin := idverification.NewPlugin()

// Register with AuthSome
auth.RegisterPlugin(plugin)

// Use in your app
middleware := plugin.GetMiddleware()
router.GET("/protected", middleware.RequireVerified(), handler)
```

---

### 2. Compliance & Audit
**Path**: `plugins/enterprise/compliance`

Comprehensive compliance and audit logging solution.

**Features:**
- ✅ GDPR compliance tools
- ✅ Audit trail system
- ✅ Data retention policies
- ✅ Consent management
- ✅ Privacy controls

**Status**: ✅ Complete

---

## Why Enterprise Plugins?

Enterprise plugins provide advanced features for:

1. **Regulatory Compliance**
   - KYC/AML requirements
   - GDPR/CCPA compliance
   - Industry regulations
   - Audit requirements

2. **Security & Trust**
   - Identity verification
   - Risk assessment
   - Fraud prevention
   - User validation

3. **Business Needs**
   - Age verification
   - Geographic restrictions
   - Premium features
   - Enterprise authentication

## Plugin Architecture

All enterprise plugins follow the same architecture:

```
plugins/enterprise/{plugin-name}/
├── plugin.go          # Plugin registration
├── config.go          # Configuration
├── service.go         # Business logic
├── handler.go         # HTTP handlers
├── middleware.go      # Route middleware
├── repository.go      # Data access interface
├── types.go           # Data types
├── errors.go          # Error definitions
├── README.md          # Documentation
└── *_test.go          # Tests
```

## Installation

Enterprise plugins are included with AuthSome. No additional installation needed.

```bash
go get github.com/xraph/authsome
```

## Usage

### Register Plugins

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/idverification"
    "github.com/xraph/authsome/plugins/enterprise/compliance"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New(config)
    
    // Register enterprise plugins
    auth.RegisterPlugin(idverification.NewPlugin())
    auth.RegisterPlugin(compliance.NewPlugin())
    
    // Mount to your app
    auth.Mount(app)
}
```

### Configuration

```yaml
# config.yaml
auth:
  plugins:
    # Identity Verification
    idverification:
      enabled: true
      default_provider: "stripe_identity"
      stripe_identity:
        api_key: "${STRIPE_SECRET_KEY}"
        webhook_secret: "${STRIPE_WEBHOOK_SECRET}"
        require_live_capture: true
        use_mock: false
    
    # Compliance
    compliance:
      enabled: true
      gdpr_mode: true
      audit_retention_days: 365
```

## Testing

All enterprise plugins include comprehensive tests:

```bash
# Test all enterprise plugins
go test ./plugins/enterprise/...

# Test specific plugin
go test ./plugins/enterprise/idverification -v
go test ./plugins/enterprise/compliance -v
```

## Documentation

Each plugin includes extensive documentation:

- **README.md** - Overview and quick start
- **EXAMPLE.md** - Real-world usage examples
- **Integration guides** - Provider-specific guides
- **API reference** - Complete API documentation
- **Testing guides** - Testing strategies

## Support

For issues or questions:
1. Check plugin-specific README
2. Review example code
3. Check integration guides
4. Open GitHub issue

## License

Same as AuthSome main project.

---

## Roadmap

### Planned Enterprise Plugins

1. **Advanced MFA** (`plugins/enterprise/advancedmfa`)
   - Hardware tokens
   - Biometric authentication
   - Risk-based authentication

2. **Session Intelligence** (`plugins/enterprise/sessionintel`)
   - Advanced device fingerprinting
   - Behavioral analytics
   - Anomaly detection

3. **Geographic Restrictions** (`plugins/enterprise/geofencing`)
   - IP-based restrictions
   - Regional compliance
   - Content geo-blocking

4. **Enterprise SSO** (Already planning in idverification)
   - SAML 2.0
   - Custom OIDC providers
   - Enterprise directory integration

---

*Enterprise plugins are designed for production use in regulated industries and high-security applications.*

