# Consent & Privacy Management Plugin

Enterprise-grade consent and privacy management plugin for AuthSome with full GDPR/CCPA compliance, cookie consent management, data portability (GDPR Article 20), and right to be forgotten (GDPR Article 17) implementation.

## Features

### Core Consent Management
- ✅ Granular consent tracking per user and organization
- ✅ Consent versioning and policy management
- ✅ Consent expiry and renewal workflows
- ✅ Immutable audit trail for all consent changes
- ✅ Multi-organization support (SaaS mode)
- ✅ Standalone mode support

### Cookie Consent Management
- ✅ Cookie consent banner integration
- ✅ Granular cookie categories (essential, functional, analytics, marketing, personalization, third-party)
- ✅ Anonymous user consent tracking
- ✅ Consent validity period management
- ✅ Cookie consent versioning

### GDPR Compliance

#### Article 20 - Right to Data Portability
- ✅ User-initiated data export requests
- ✅ Multiple export formats (JSON, CSV, XML, PDF)
- ✅ Configurable data sections (profile, sessions, consents, audit logs)
- ✅ Rate limiting for export requests
- ✅ Secure download URLs with expiry
- ✅ Automatic cleanup of expired exports

#### Article 17 - Right to be Forgotten
- ✅ User-initiated deletion requests
- ✅ Admin approval workflow for deletions
- ✅ Grace period before deletion
- ✅ Data archiving before deletion
- ✅ Partial or full data deletion
- ✅ Legal retention exemptions
- ✅ Audit trail preservation

### Data Processing Agreements (DPA)
- ✅ DPA/BAA creation and management
- ✅ Digital signature generation
- ✅ Multiple agreement types (DPA, BAA, GDPR, CCPA)
- ✅ Version tracking and expiry management

### Privacy Settings
- ✅ Per-organization privacy configuration
- ✅ GDPR mode and CCPA mode
- ✅ Configurable data retention periods
- ✅ Consent requirement toggles
- ✅ Data Protection Officer (DPO) contact management

## Installation

### 1. Install Plugin

```go
import (
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/enterprise/consent"
)

func main() {
	auth := authsome.New(/* ... */)
	
	// Register consent plugin
	consentPlugin := consent.NewPlugin()
	auth.RegisterPlugin(consentPlugin)
	
	// Initialize
	if err := auth.Init(); err != nil {
		log.Fatal(err)
	}
}
```

### 2. Configuration

Add to your `config.yaml`:

```yaml
auth:
  consent:
    enabled: true
    gdprEnabled: true
    ccpaEnabled: false
    
    cookieConsent:
      enabled: true
      defaultStyle: "banner"  # banner, modal, popup
      requireExplicit: true   # No implied consent
      validityPeriod: "8760h" # 1 year
      allowAnonymous: true
      bannerVersion: "1.0"
      categories:
        - essential
        - functional
        - analytics
        - marketing
        - personalization
        - third_party
    
    dataExport:
      enabled: true
      allowedFormats:
        - json
        - csv
      defaultFormat: json
      maxRequests: 5          # Max exports per period
      requestPeriod: "720h"   # 30 days
      expiryHours: 72         # Download URL valid for 3 days
      storagePath: "/var/lib/authsome/consent/exports"
      includeSections:
        - profile
        - sessions
        - consents
        - audit
      autoCleanup: true
      cleanupInterval: "24h"
      maxExportSize: 104857600  # 100MB
    
    dataDeletion:
      enabled: true
      requireAdminApproval: true
      gracePeriodDays: 30      # GDPR allows up to 30 days
      archiveBeforeDeletion: true
      archivePath: "/var/lib/authsome/consent/archives"
      retentionExemptions:
        - legal_hold
        - active_investigation
        - contractual_obligation
      notifyBeforeDeletion: true
      allowPartialDeletion: true
      preserveLegalData: true
      autoProcessAfterGrace: false
    
    audit:
      enabled: true
      retentionDays: 2555      # 7 years (common legal requirement)
      immutable: true
      logAllChanges: true
      logIpAddress: true
      logUserAgent: true
      signLogs: true
      exportFormat: json
      archiveOldLogs: true
      archiveInterval: "2160h"  # 90 days
    
    expiry:
      enabled: true
      defaultValidityDays: 365
      renewalReminderDays: 30
      autoExpireCheck: true
      expireCheckInterval: "24h"
      allowRenewal: true
      requireReConsent: false
    
    dashboard:
      enabled: true
      path: "/auth/consent"
      showConsentHistory: true
      showCookiePreferences: true
      showDataExport: true
      showDataDeletion: true
      showAuditLog: true
      showPolicies: true
    
    notifications:
      enabled: true
      notifyOnGrant: false
      notifyOnRevoke: true
      notifyOnExpiry: true
      notifyExportReady: true
      notifyDeletionApproved: true
      notifyDeletionComplete: true
      notifyDpoEmail: "dpo@example.com"
      channels:
        - email
```

### 3. Database Migration

The plugin automatically creates the following tables:
- `consent_records` - User consent records
- `consent_policies` - Consent policies and terms
- `consent_audit_logs` - Immutable audit trail
- `cookie_consents` - Cookie consent preferences
- `data_export_requests` - Data export requests
- `data_deletion_requests` - Deletion requests
- `data_processing_agreements` - DPAs and BAAs
- `privacy_settings` - Per-organization privacy settings

## API Reference

### Consent Records

#### Create Consent
```http
POST /auth/consent/records
```

```json
{
  "userId": "user_123",
  "consentType": "marketing",
  "purpose": "email_campaigns",
  "granted": true,
  "version": "1.0",
  "expiresIn": 365,
  "metadata": {
    "source": "signup_form"
  }
}
```

#### List User Consents
```http
GET /auth/consent/records
```

#### Get Consent Summary
```http
GET /auth/consent/summary
```

Response:
```json
{
  "userId": "user_123",
  "organizationId": "org_456",
  "totalConsents": 5,
  "grantedConsents": 4,
  "revokedConsents": 1,
  "expiredConsents": 0,
  "pendingRenewals": 1,
  "consentsByType": {
    "marketing": {
      "type": "marketing",
      "granted": true,
      "version": "1.0",
      "grantedAt": "2024-01-01T00:00:00Z",
      "expiresAt": "2025-01-01T00:00:00Z",
      "needsRenewal": false
    }
  },
  "hasPendingDeletion": false,
  "hasPendingExport": false
}
```

#### Revoke Consent
```http
POST /auth/consent/revoke
```

```json
{
  "consentType": "marketing",
  "purpose": "email_campaigns"
}
```

### Consent Policies

#### Create Policy
```http
POST /auth/consent/policies
```

```json
{
  "consentType": "terms",
  "name": "Terms of Service",
  "description": "Standard terms of service",
  "version": "2.0",
  "content": "Full policy text here...",
  "required": true,
  "renewable": true,
  "validityPeriod": 365,
  "metadata": {}
}
```

#### Get Latest Policy
```http
GET /auth/consent/policies/latest/:type
```

### Cookie Consent

#### Record Cookie Consent
```http
POST /auth/consent/cookies
```

```json
{
  "essential": true,
  "functional": true,
  "analytics": false,
  "marketing": false,
  "personalization": true,
  "thirdParty": false,
  "sessionId": "anonymous_session_id",
  "bannerVersion": "1.0"
}
```

#### Get Cookie Consent
```http
GET /auth/consent/cookies
```

### Data Export (GDPR Article 20)

#### Request Data Export
```http
POST /auth/consent/export
```

```json
{
  "format": "json",
  "includeSections": ["profile", "consents", "audit"]
}
```

Response:
```json
{
  "id": "export_123",
  "userId": "user_123",
  "organizationId": "org_456",
  "status": "pending",
  "format": "json",
  "includeSections": ["profile", "consents", "audit"],
  "createdAt": "2024-01-01T00:00:00Z"
}
```

#### Get Export Status
```http
GET /auth/consent/export/:id
```

Response:
```json
{
  "id": "export_123",
  "status": "completed",
  "exportUrl": "/auth/consent/export/export_123/download",
  "exportSize": 1024000,
  "expiresAt": "2024-01-04T00:00:00Z",
  "completedAt": "2024-01-01T00:05:00Z"
}
```

#### Download Export
```http
GET /auth/consent/export/:id/download
```

Returns the export file.

### Data Deletion (GDPR Article 17)

#### Request Data Deletion
```http
POST /auth/consent/deletion
```

```json
{
  "reason": "I want my data deleted per GDPR Article 17",
  "deleteSections": ["all"]
}
```

Response:
```json
{
  "id": "deletion_123",
  "userId": "user_123",
  "organizationId": "org_456",
  "status": "pending",
  "requestReason": "I want my data deleted per GDPR Article 17",
  "deleteSections": ["all"],
  "createdAt": "2024-01-01T00:00:00Z"
}
```

#### Approve Deletion (Admin)
```http
POST /auth/consent/deletion/:id/approve
```

#### Process Deletion (Admin)
```http
POST /auth/consent/deletion/:id/process
```

### Privacy Settings

#### Get Privacy Settings
```http
GET /auth/consent/settings
```

#### Update Privacy Settings (Admin)
```http
PUT /auth/consent/settings
```

```json
{
  "consentRequired": true,
  "cookieConsentEnabled": true,
  "gdprMode": true,
  "ccpaMode": false,
  "dataRetentionDays": 2555,
  "requireExplicitConsent": true,
  "allowDataPortability": true,
  "requireAdminApprovalForDeletion": true,
  "deletionGracePeriodDays": 30,
  "contactEmail": "privacy@example.com",
  "dpoEmail": "dpo@example.com"
}
```

### Data Processing Agreements

#### Create DPA
```http
POST /auth/consent/dpa
```

```json
{
  "agreementType": "dpa",
  "version": "1.0",
  "content": "Full DPA text...",
  "signedByName": "John Doe",
  "signedByTitle": "CTO",
  "signedByEmail": "john@example.com",
  "effectiveDate": "2024-01-01T00:00:00Z",
  "expiryDate": "2025-01-01T00:00:00Z"
}
```

## Usage Examples

### Programmatic Consent Check

```go
// Get plugin instance
consentPlugin := auth.GetPlugin("consent").(*consent.Plugin)

// Check if user has granted consent
granted, err := consentPlugin.GetUserConsentStatus(
    ctx,
    "user_123",
    "org_456",
    "marketing",
    "email_campaigns",
)

if !granted {
    // User hasn't granted consent
    return errors.New("consent required")
}
```

### Require Consent Middleware

```go
// Protect routes with consent requirement
router.POST("/marketing/subscribe", 
    consentPlugin.RequireConsent("marketing", "email_campaigns")(
        handler.Subscribe,
    ),
)
```

### Create Consent Policy

```go
policy, err := consentPlugin.Service().CreatePolicy(ctx, orgID, adminUserID, &consent.CreatePolicyRequest{
    ConsentType: "privacy",
    Name:        "Privacy Policy",
    Version:     "2.0",
    Content:     policyContent,
    Required:    true,
    Renewable:   false,
})
```

### Request Data Export

```go
export, err := consentPlugin.Service().RequestDataExport(ctx, userID, orgID, &consent.DataExportRequestInput{
    Format:          "json",
    IncludeSections: []string{"profile", "consents", "audit"},
})
```

### Request Data Deletion

```go
deletion, err := consentPlugin.Service().RequestDataDeletion(ctx, userID, orgID, &consent.DataDeletionRequestInput{
    Reason:         "GDPR Article 17 request",
    DeleteSections: []string{"all"},
})
```

## Multi-Tenancy Support

The consent plugin fully supports AuthSome's multi-tenancy:

### Standalone Mode
- Single default organization
- Organization-agnostic consent management
- Shared privacy settings

### SaaS Mode
- Per-organization privacy settings
- Organization-scoped consents
- Organization-specific consent policies
- Isolated data export and deletion requests

## GDPR Compliance Checklist

✅ **Article 6 (Lawfulness of processing)**
- Consent records with explicit grant/revoke
- Purpose specification
- Audit trail

✅ **Article 7 (Conditions for consent)**
- Explicit consent collection
- Easy consent withdrawal
- Versioned consent policies

✅ **Article 13/14 (Information to be provided)**
- Privacy policy management
- Terms of service versioning
- Contact information (DPO)

✅ **Article 17 (Right to erasure)**
- User-initiated deletion requests
- Admin approval workflow
- Grace period
- Data archiving
- Retention exemptions

✅ **Article 20 (Right to data portability)**
- User data export
- Multiple formats (JSON, CSV, XML)
- Secure download mechanism

✅ **Article 30 (Records of processing activities)**
- Immutable audit logs
- 7-year retention
- Cryptographic signing

## CCPA Compliance

✅ **Right to Know**
- Data export functionality

✅ **Right to Delete**
- Deletion request workflow

✅ **Right to Opt-Out**
- Consent revocation
- Cookie consent management

✅ **Do Not Sell**
- Third-party consent tracking
- Marketing consent management

## Security Considerations

### Data Protection
- Immutable audit logs
- Cryptographic log signing
- Digital signatures for DPAs
- Secure file storage

### Access Control
- User-scoped consent access
- Admin-only operations (approval, processing)
- Organization isolation in SaaS mode

### Privacy by Design
- Default privacy settings
- Explicit consent required
- No implied consent in GDPR mode
- Automatic expiry checking

## Performance

### Optimizations
- Indexed database queries
- Efficient consent lookups
- Background processing for exports
- Automatic cleanup of expired data

### Scalability
- Async data export processing
- Batch consent expiry checks
- Configurable cleanup intervals

## Troubleshooting

### Export Fails
- Check storage path permissions
- Verify max export size limits
- Check rate limits

### Deletion Not Processing
- Verify grace period has passed
- Check retention exemptions
- Ensure admin approval if required

### Consent Not Found
- Check organization ID
- Verify consent type and purpose match
- Check if consent has expired

## Contributing

See [CONTRIBUTING.md](../../../CONTRIBUTING.md) for guidelines.

## License

Part of the AuthSome project. See [LICENSE](../../../LICENSE).

## Support

For issues and questions:
- GitHub Issues: https://github.com/xraph/authsome/issues
- Documentation: https://authsome.dev/plugins/consent
- Email: support@authsome.dev

