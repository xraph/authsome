# Enterprise Compliance & Audit Plugin

**Comprehensive compliance management for SOC 2, HIPAA, PCI-DSS, GDPR, ISO 27001, and CCPA**

## Overview

The Compliance plugin provides enterprise-grade compliance management, automated policy enforcement, audit trails, and reporting capabilities for AuthSome. It enables apps (platform-level tenants) to meet regulatory requirements and maintain compliance with various industry standards.

## Features

### 🏛️ Compliance Standards Support

- **SOC 2 Type II** - Service Organization Control 2
- **HIPAA** - Healthcare data protection (7-year retention)
- **PCI-DSS** - Payment card data security  
- **GDPR** - EU data protection and privacy
- **ISO/IEC 27001** - Information security management
- **CCPA** - California consumer privacy

### 📋 Compliance Profiles

- **Per-App Configuration** - Each app (tenant) gets its own compliance profile
- **Template-Based Setup** - Quick start with predefined templates
- **Custom Policies** - Define app-specific requirements
- **Multi-Standard Support** - Comply with multiple standards simultaneously

## App vs Organization Scoping

**Important:** Compliance profiles are scoped to **Apps** (platform-level tenants), not user-created Organizations.

- **App** = Your platform tenant (like customer-a, customer-b) - top-level multi-tenant boundary
- **Organization** = User-created workspaces within an app (optional feature, like Clerk's organizations)
- **Compliance Scope** = App-level (each app/customer has its own compliance requirements)

This means if you have multiple customers on your platform, each customer (app) has their own compliance profile. Organizations within an app share that app's compliance profile.

### ✅ Automated Compliance Checks

- **MFA Coverage** - Monitor MFA adoption rates
- **Password Policy** - Validate password strength and expiration
- **Session Policy** - Enforce session timeout and security
- **Access Review** - Regular permission audits
- **Inactive Users** - Identify dormant accounts
- **Data Retention** - Verify audit log retention compliance

### 🔒 Runtime Policy Enforcement

- **Password Validation** - Enforce complexity requirements at signup/change
- **MFA Enforcement** - Block login if MFA required but not enabled
- **Session Validation** - Enforce max age, idle timeout, IP binding
- **Training Requirements** - Require completion of compliance training
- **Data Residency** - Enforce geographic data restrictions

### 📊 Compliance Reports

- **SOC 2 Reports** - Audit-ready compliance documentation
- **HIPAA Audit Trails** - 7-year retention with export
- **Custom Reports** - Generate reports for any time period
- **Multiple Formats** - PDF, JSON, CSV export
- **Evidence Collection** - Attach supporting documentation

### 🎓 Compliance Training

- **Required Training Tracking** - Security awareness, HIPAA basics, etc.
- **Completion Monitoring** - Track training status per user
- **Expiration Alerts** - Notify when training needs renewal
- **Per-Standard Requirements** - Different training for different standards

### 🚨 Violation Management

- **Automatic Detection** - Identify policy violations in real-time
- **Severity Levels** - Critical, high, medium, low
- **Resolution Tracking** - Track who resolved violations and when
- **Notifications** - Alert compliance contacts immediately

### 📈 Compliance Dashboard

- **Overall Compliance Score** - 0-100% compliance rating
- **Real-Time Status** - Compliant, non-compliant, in-progress
- **Recent Checks** - View latest automated check results
- **Open Violations** - Track unresolved issues
- **Audit History** - Complete compliance timeline

## Installation

The plugin is located in `plugins/enterprise/compliance/` and integrates automatically with AuthSome.

### 1. Enable in Configuration

```yaml
# config.yaml
plugins:
  compliance:
    enabled: true
    defaultStandard: "SOC2"
    
    automatedChecks:
      enabled: true
      checkInterval: 24h
      
    audit:
      minRetentionDays: 90
      maxRetentionDays: 2555
      detailedTrail: true
      immutable: true
      
    notifications:
      enabled: true
      violations: true
      failedChecks: true
```

### 2. Run Database Migrations

```bash
# Migrations are automatically applied when plugin is enabled
authsome migrate up
```

### 3. Verify Installation

```bash
curl http://localhost:8080/auth/compliance/templates
```

## Quick Start

### Create Compliance Profile from Template

```bash
# Using SOC 2 template
POST /auth/compliance/profiles/from-template
{
  "appId": "app_abc123",
  "standard": "SOC2"
}

# Response
{
  "id": "prof_abc",
  "appId": "app_abc123",
  "name": "SOC 2 Type II",
  "standards": ["SOC2"],
  "mfaRequired": true,
  "passwordMinLength": 12,
  "sessionMaxAge": 86400,
  "retentionDays": 90,
  "status": "active"
}
```

### Check Compliance Status

```bash
GET /auth/compliance/apps/app_abc123/status

# Response
{
  "profileId": "prof_abc",
  "appId": "app_abc123",
  "overallStatus": "compliant",
  "score": 95,
  "checksPassed": 19,
  "checksFailed": 1,
  "violations": 2,
  "lastChecked": "2025-11-01T10:00:00Z"
}
```

### Generate Compliance Report

```bash
POST /auth/compliance/apps/app_abc123/reports
{
  "reportType": "soc2",
  "standard": "SOC2",
  "period": "2025-Q3",
  "format": "pdf"
}

# Response (202 Accepted - generating asynchronously)
{
  "id": "rep_xyz",
  "status": "generating",
  "appId": "app_abc123"
}
```

## API Reference

### Compliance Profiles

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/profiles` | POST | Create compliance profile |
| `/profiles/from-template` | POST | Create from template |
| `/profiles/:id` | GET | Get profile |
| `/apps/:appId/profile` | GET | Get app profile |
| `/profiles/:id` | PUT | Update profile |

### Compliance Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/apps/:appId/status` | GET | Get compliance status |
| `/apps/:appId/dashboard` | GET | Get dashboard data |

### Checks & Violations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/profiles/:profileId/checks` | POST | Run compliance check |
| `/profiles/:profileId/checks` | GET | List checks |
| `/apps/:appId/violations` | GET | List violations |
| `/violations/:id/resolve` | PUT | Resolve violation |

### Reports

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/apps/:appId/reports` | POST | Generate report |
| `/apps/:appId/reports` | GET | List reports |
| `/reports/:id/download` | GET | Download report |

### Training

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/apps/:appId/training` | POST | Create training record |
| `/users/:userId/training` | GET | Get user training status |
| `/training/:id/complete` | PUT | Mark training complete |

## Compliance Templates

### SOC 2 Type II

```yaml
Standard: SOC 2
MFA Required: Yes
Password Min Length: 12
Session Max Age: 24 hours
Retention Days: 90
Required Policies:
  - Access Control
  - Password Policy
  - Data Classification
  - Incident Response
  - Change Management
```

### HIPAA

```yaml
Standard: HIPAA
MFA Required: Yes
Password Min Length: 14
Session Max Age: 1 hour
Retention Days: 2555 (7 years)
Data Residency: US
Required Policies:
  - Access Control
  - Audit Controls
  - Breach Notification
  - Business Associate Agreement
  - Minimum Necessary
```

### PCI-DSS

```yaml
Standard: PCI-DSS
MFA Required: Yes
Password Min Length: 15
Session Max Age: 15 minutes
Retention Days: 365
Required Policies:
  - Firewall Configuration
  - Cardholder Data Protection
  - Encryption Transmission
  - Access Control
```

## Usage Examples

### Password Policy Enforcement

```go
// Hook automatically enforces password policy based on app's compliance profile
// user.password_changed hook

func (p *Plugin) onPasswordChanged(ctx context.Context, data HookData) error {
    appID := data.GetString("app_id")
    password := data.GetString("new_password")
    
    // Enforces: min length, uppercase, lowercase, number, symbol
    if err := p.policyEngine.EnforcePasswordPolicy(ctx, appID, password); err != nil {
        return err // Blocks password change
    }
    
    return nil
}
```

### MFA Enforcement

```go
// Hook automatically enforces MFA requirement based on app's compliance profile
// user.login hook

func (p *Plugin) onUserLogin(ctx context.Context, data HookData) error {
    appID := data.GetString("app_id")
    userID := data.GetString("user_id")
    mfaEnabled := data.GetBool("mfa_enabled")
    
    // If app requires MFA but user doesn't have it
    if err := p.policyEngine.EnforceMFA(ctx, appID, userID, mfaEnabled); err != nil {
        return err // Blocks login
    }
    
    return nil
}
```

### Session Policy Enforcement

```go
// Hook validates session on each request based on app's compliance profile
// session.validated hook

func (p *Plugin) onSessionValidated(ctx context.Context, data HookData) error {
    appID := data.GetString("app_id")
    
    session := &Session{
        ID:             data.GetString("session_id"),
        CreatedAt:      data.GetTime("created_at"),
        LastActivityAt: data.GetTime("last_activity"),
        CreatedIP:      data.GetString("created_ip"),
        CurrentIP:      data.GetString("current_ip"),
    }
    
    // Enforces: max age, idle timeout, IP binding
    if err := p.policyEngine.EnforceSessionPolicy(ctx, appID, session); err != nil {
        return err // Expires session
    }
    
    return nil
}
```

## Configuration Options

### Automated Checks

```yaml
automatedChecks:
  enabled: true
  checkInterval: 24h
  mfaCoverage: true
  passwordPolicy: true
  sessionPolicy: true
  accessReview: true
  inactiveUsers: true
  dataRetention: true
```

### Audit Settings

```yaml
audit:
  minRetentionDays: 90      # SOC 2 minimum
  maxRetentionDays: 2555    # HIPAA 7 years
  detailedTrail: true       # Log field changes
  immutable: true           # Cannot delete logs
  exportFormat: json
  signLogs: true            # Tamper detection
```

### Notifications

```yaml
notifications:
  enabled: true
  violations: true
  failedChecks: true
  auditReminders: true
  notifyComplianceContact: true
  channels:
    email: true
    slack: false
    webhook: false
```

## Compliance Workflow

### 1. Initial Setup

1. Create app in AuthSome (or use existing app/tenant)
2. Create compliance profile for the app via API
3. Automated checks run immediately
4. Review compliance status

### 2. Ongoing Compliance

1. Automated checks run every 24 hours per app
2. Policy enforcement on all auth actions within the app
3. Violations logged automatically
4. Compliance contact notified

### 3. Audit Preparation

1. Run full compliance check for the app
2. Generate audit report for period
3. Collect evidence (audit logs, policies)
4. Export for auditors

### 4. Remediation

1. Review open violations for the app
2. Resolve issues (enable MFA, update policies)
3. Mark violations as resolved
4. Re-run checks to verify

## Best Practices

### 1. Choose the Right Standard

- **SaaS Product**: Start with SOC 2
- **Healthcare**: HIPAA required
- **Payment Processing**: PCI-DSS required
- **EU Customers**: GDPR required
- **California**: CCPA may apply

### 2. Enable from Day One

- Create compliance profile when app/tenant is onboarded
- Enforce policies from the start (easier than retrofitting)
- Run initial checks immediately

### 3. Monitor Regularly

- Review compliance dashboard weekly per app
- Address violations within 48 hours
- Run manual checks before audits

### 4. Train Users

- Require compliance training for all users in the app
- Set expiration (annual renewal)
- Track completion in dashboard

### 5. Document Everything

- Store policies as documents per app
- Collect evidence continuously
- Generate reports quarterly per app

## Troubleshooting

### Compliance Score is Low

1. Check which checks are failing
2. Review open violations
3. Run manual checks for specific areas
4. Address violations and re-check

### MFA Enforcement Blocking Users

1. Verify `mfaRequired` setting in profile
2. Give users grace period to enable MFA
3. Send notification before enforcement
4. Provide self-service MFA setup

### Reports Not Generating

1. Check report status (generating, failed)
2. Review error logs
3. Verify storage configuration
4. Ensure sufficient permissions

### Training Not Showing

1. Verify compliance profile has standards
2. Check template for required training
3. Run check to create training records
4. Assign training manually if needed

## Performance

### Database Indexes

All tables have optimized indexes for:
- App lookups (app_id)
- Status filtering
- Date-based queries
- User-specific queries

### Caching

- Compliance profiles cached
- Check results cached (1 hour)
- Report metadata cached

### Async Operations

- Report generation
- Automated checks
- Violation notifications
- Evidence collection

## Security

### Data Protection

- Audit logs are immutable
- Evidence files use SHA256 hashing
- Sensitive data encrypted at rest
- Access logs for compliance data

### Access Control

- Only app admins can manage compliance profiles
- Compliance contacts get read-only access
- Auditors get time-limited export access

## Roadmap

- [ ] Automated policy document generation
- [ ] Integration with security scanning tools
- [ ] Real-time compliance monitoring dashboard
- [ ] AI-powered compliance recommendations
- [ ] Third-party auditor portal
- [ ] Compliance certifications marketplace

## Support

For enterprise support and compliance consulting:
- Email: compliance@authsome.dev
- Docs: https://docs.authsome.dev/plugins/compliance
- Issues: https://github.com/xraph/authsome/issues

---

**Built with ❤️ for enterprise security and compliance**

