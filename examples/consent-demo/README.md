# Consent Plugin Demo Application

Interactive demo application showcasing the Consent & Privacy Management plugin features.

## Features Demonstrated

This demo application demonstrates:

1. **User Authentication** - Sign up/Sign in
2. **Consent Management** - Grant and revoke consent for different purposes
3. **Cookie Consent** - Manage cookie preferences with granular categories
4. **GDPR Data Export (Article 20)** - Request and download personal data
5. **GDPR Right to be Forgotten (Article 17)** - Request account deletion
6. **Consent-Protected Routes** - Routes that require specific consent
7. **Consent Summary** - View all user consent status

## Prerequisites

- Go 1.21 or higher
- SQLite (for demo database)

## Quick Start

```bash
# From the authsome root directory
cd examples/consent-demo

# Run the demo
go run main.go
```

The server will start on `http://localhost:8080`

## Usage

### 1. Open the Demo UI

Navigate to http://localhost:8080/demo in your browser.

### 2. Create an Account

1. Enter an email and password
2. Click "Sign Up"
3. The response will include an authentication token (automatically saved)

### 3. Test Consent Management

**Grant Marketing Consent:**
- Click "Grant Marketing Consent"
- This allows the application to send marketing emails

**Revoke Consent:**
- Click "Revoke Marketing Consent"
- This withdraws your consent for marketing

### 4. Manage Cookie Preferences

1. Select your cookie preferences:
   - Essential (always on)
   - Functional
   - Analytics
   - Marketing

2. Click "Save Cookie Preferences"

### 5. Request Data Export (GDPR Article 20)

1. Click "Request Data Export"
2. Wait for processing (shows as "pending")
3. Click "List My Exports" to see status
4. When completed, you can download your data

### 6. Request Account Deletion (GDPR Article 17)

1. Enter a reason for deletion
2. Click "Request Account Deletion"
3. Status shows as "pending" (requires admin approval)
4. Click "List Deletion Requests" to view status

### 7. Test Protected Endpoints

**Marketing Endpoint:**
- Click "Test Marketing Endpoint"
- If you've granted marketing consent: ✅ Success
- If not granted: ❌ 403 Forbidden with consent required error

**Analytics Endpoint:**
- Click "Test Analytics Endpoint"
- Requires analytics consent to access

### 8. View Consent Summary

Click "Get My Consent Summary" to see:
- Total consents
- Granted vs revoked consents
- Consent by type
- Pending deletions/exports

## API Examples

### Grant Consent
```bash
curl -X POST http://localhost:8080/api/auth/consent/records \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "consentType": "marketing",
    "purpose": "email_campaigns",
    "granted": true,
    "version": "1.0"
  }'
```

### Revoke Consent
```bash
curl -X POST http://localhost:8080/api/auth/consent/revoke \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "consentType": "marketing",
    "purpose": "email_campaigns"
  }'
```

### Request Data Export
```bash
curl -X POST http://localhost:8080/api/auth/consent/export \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "includeSections": ["profile", "consents", "audit"]
  }'
```

### Request Data Deletion
```bash
curl -X POST http://localhost:8080/api/auth/consent/deletion \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "GDPR Article 17 request",
    "deleteSections": ["all"]
  }'
```

### Get Consent Summary
```bash
curl http://localhost:8080/api/auth/consent/summary \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Configuration

The demo uses default configuration. To customize, create a `config.yaml`:

```yaml
auth:
  consent:
    enabled: true
    gdprEnabled: true
    ccpaEnabled: false
    
    cookieConsent:
      enabled: true
      defaultStyle: "banner"
      requireExplicit: true
      validityPeriod: "8760h" # 1 year
    
    dataExport:
      enabled: true
      maxRequests: 5
      requestPeriod: "720h" # 30 days
      expiryHours: 72
    
    dataDeletion:
      enabled: true
      requireAdminApproval: true
      gracePeriodDays: 30
```

## Testing Consent-Protected Routes

The demo includes two protected routes:

1. **Marketing Endpoint** (`/marketing/subscribe`)
   - Requires marketing consent for email_campaigns
   - Returns success if consent granted
   - Returns 403 if consent not granted

2. **Analytics Endpoint** (`/analytics/track`)
   - Requires analytics consent for usage_tracking
   - Returns success if consent granted
   - Returns 403 if consent not granted

## Database

The demo uses an in-memory SQLite database. Data is lost when the server stops.

For persistent data, modify the configuration to use a file-based database:

```go
// In main.go, add database configuration
config := authsome.Config{
    DatabaseURL: "sqlite://consent_demo.db",
    // ... other config
}
```

## Troubleshooting

### "unauthorized" error
- Make sure you're signed in
- Check that the auth token is saved in localStorage
- Try signing in again

### "consent required" error
- You need to grant the specific consent type
- Click the appropriate "Grant Consent" button
- Try the protected endpoint again

### Export/Deletion not processing
- Background processing is simulated in this demo
- In production, these would be handled by job queues
- Refresh the list to see updated status

## Next Steps

After trying the demo:

1. Review the plugin code in `plugins/enterprise/consent/`
2. Read the full documentation in `README.md`
3. Check integration examples in `EXAMPLES.md`
4. Review GDPR compliance details in `IMPLEMENTATION_SUMMARY.md`

## Production Deployment

This is a demo application. For production:

1. Use a production database (PostgreSQL recommended)
2. Enable TLS/HTTPS
3. Configure proper secret keys
4. Set up job queues for export/deletion processing
5. Enable rate limiting
6. Configure email notifications
7. Set up monitoring and alerting
8. Review security settings

## Support

For questions or issues:
- Documentation: https://authsome.dev/plugins/consent
- GitHub Issues: https://github.com/xraph/authsome/issues
- Email: support@authsome.dev

