# Notification Plugin

The Notification plugin provides comprehensive template-based notification management for AuthSome, supporting both SaaS and standalone modes with multi-language templates.

## Features

- **Template Management**: Create, update, and manage notification templates via API
- **Multi-Language Support**: Templates can be created in multiple languages
- **Organization-Scoped Templates**: SaaS mode allows org-specific template overrides
- **Default Templates**: Includes 9 pre-configured templates for common auth flows
- **Variable Substitution**: Dynamic template rendering with Go template syntax
- **Multiple Channels**: Email and SMS notifications (extensible for push)
- **Template Versioning**: Track and audit template changes
- **Preview Mode**: Test templates before deployment

## Configuration

```yaml
auth:
  notification:
    add_default_templates: true
    default_language: "en"
    allow_org_overrides: false  # true for SaaS mode
    auto_send_welcome: true
    retry_attempts: 3
    retry_delay: "5m"
    cleanup_after: "720h"  # 30 days
    
    rate_limits:
      email:
        max_requests: 100
        window: "1h"
      sms:
        max_requests: 50
        window: "1h"
    
    providers:
      email:
        provider: "smtp"
        from: "noreply@example.com"
        from_name: "AuthSome"
        config:
          host: "smtp.example.com"
          port: 587
          username: "smtp-user"
          password: "smtp-pass"
      sms:
        provider: "twilio"
        from: "+1234567890"
        config:
          account_sid: "your-account-sid"
          auth_token: "your-auth-token"
```

## Default Templates

The plugin includes these default templates:

1. **auth.welcome** - Welcome email for new users
2. **auth.verify_email** - Email verification
3. **auth.password_reset** - Password reset email
4. **auth.mfa_code** - MFA verification code (email/SMS)
5. **auth.magic_link** - Magic link sign-in
6. **auth.email_otp** - Email OTP code
7. **auth.phone_otp** - Phone OTP SMS
8. **auth.security_alert** - Security event notifications

## Usage in Other Plugins

### Simple Adapter Usage

```go
package myplugin

import (
	"context"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
)

type MyPlugin struct {
	notifAdapter *notificationPlugin.Adapter
}

func (p *MyPlugin) Init(dep interface{}) error {
	// Get notification service from registry
	registry := dep.(*registry.ServiceRegistry)
	templateSvc := registry.Get("notification.template").(*notificationPlugin.TemplateService)
	
	p.notifAdapter = notificationPlugin.NewAdapter(templateSvc)
	return nil
}

func (p *MyPlugin) SendVerificationCode(ctx context.Context, email, code string) error {
	return p.notifAdapter.SendEmailOTP(ctx, "default", email, code, 10)
}
```

### Direct Service Usage

```go
import (
	"github.com/xraph/authsome/core/notification"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
)

// Send with template
func sendWithTemplate(ctx context.Context, templateSvc *notificationPlugin.TemplateService) error {
	_, err := templateSvc.SendWithTemplate(ctx, &notificationPlugin.SendWithTemplateRequest{
		OrganizationID: "org_123",
		TemplateKey:    "auth.mfa_code",
		Type:           notification.NotificationTypeEmail,
		Recipient:      "user@example.com",
		Variables: map[string]interface{}{
			"user_name":      "John Doe",
			"code":           "123456",
			"expiry_minutes": 10,
			"app_name":       "MyApp",
		},
	})
	return err
}

// Send direct without template
func sendDirect(ctx context.Context, templateSvc *notificationPlugin.TemplateService) error {
	_, err := templateSvc.SendDirect(
		ctx,
		"org_123",
		notification.NotificationTypeEmail,
		"user@example.com",
		"Subject",
		"Email body",
		nil,
	)
	return err
}
```

## API Endpoints

### Template Management

```bash
# Create template
POST /auth/templates
{
  "organization_id": "org_123",
  "template_key": "custom.welcome",
  "name": "Custom Welcome Email",
  "type": "email",
  "language": "en",
  "subject": "Welcome {{.user_name}}!",
  "body": "Hi {{.user_name}}, welcome to {{.app_name}}!",
  "variables": ["user_name", "app_name"]
}

# List templates
GET /auth/templates?organization_id=org_123&type=email&language=en

# Get template
GET /auth/templates/:id

# Update template
PUT /auth/templates/:id
{
  "subject": "Updated subject",
  "active": true
}

# Delete template
DELETE /auth/templates/:id

# Preview template
POST /auth/templates/:id/preview
{
  "variables": {
    "user_name": "Test User",
    "app_name": "TestApp"
  }
}
```

### Sending Notifications

```bash
# Send notification with template
POST /auth/notifications/send
{
  "organization_id": "org_123",
  "template_key": "auth.welcome",
  "type": "email",
  "recipient": "user@example.com",
  "variables": {
    "user_name": "John Doe",
    "user_email": "user@example.com",
    "app_name": "MyApp",
    "login_url": "https://app.example.com/login"
  }
}

# List notifications
GET /auth/notifications?organization_id=org_123&status=sent

# Get notification
GET /auth/notifications/:id

# Resend failed notification
POST /auth/notifications/:id/resend
```

## Template Syntax

Templates use Go's `text/template` syntax:

```
Subject: Welcome to {{.app_name}}, {{.user_name}}!

Body:
Hi {{.user_name}},

Your verification code is: {{.code}}

This code expires in {{.expiry_minutes}} minutes.

{{if .support_email}}
Need help? Contact us at {{.support_email}}
{{end}}

Best regards,
The {{.app_name}} Team
```

### Available Functions

- `{{upper .text}}` - Convert to uppercase
- `{{lower .text}}` - Convert to lowercase
- `{{title .text}}` - Title case
- `{{trim .text}}` - Trim whitespace
- `{{truncate .text 100}}` - Truncate to length
- `{{default "fallback" .value}}` - Default value if empty

## SaaS Mode Features

### Organization-Specific Templates

In SaaS mode with `allow_org_overrides: true`:

```yaml
# Override default template for org_123
POST /auth/templates
{
  "organization_id": "org_123",
  "template_key": "auth.welcome",  # Same key as default
  "name": "Org 123 Welcome Email",
  "type": "email",
  "language": "en",
  "subject": "Welcome to Org 123!",
  "body": "Custom welcome message..."
}
```

The plugin will automatically:
1. Try org-specific template first (`org_123`)
2. Fall back to default template if not found
3. Support language fallback (requested lang → "en")

### Multi-Language Support

```bash
# Create Spanish version
POST /auth/templates
{
  "organization_id": "default",
  "template_key": "auth.welcome",
  "name": "Bienvenida",
  "type": "email",
  "language": "es",
  "subject": "¡Bienvenido a {{.app_name}}!",
  "body": "Hola {{.user_name}}..."
}

# Request Spanish template
POST /auth/notifications/send
{
  "template_key": "auth.welcome",
  "type": "email",
  "language": "es",
  ...
}
```

## Migration

The plugin automatically runs migrations to create:

- `notification_templates` table
- `notifications` table
- Indexes for performance
- Default templates (if enabled)

## Observability

### Audit Logging

All template operations are logged:
- Template creation/updates/deletions
- Notification sending
- Template rendering failures

### Metrics to Monitor

- Notification send rate
- Template rendering time
- Delivery success rate
- Failed notifications count

### Cleanup

Old notifications are automatically cleaned up based on `cleanup_after` configuration.

## Security Considerations

1. **Template Injection**: Templates are validated before saving
2. **Rate Limiting**: Configurable per channel
3. **PII Protection**: Notifications shouldn't log sensitive data
4. **Organization Isolation**: Templates are org-scoped in SaaS mode

## Production Best Practices

1. **Test Templates**: Use preview endpoint before activating
2. **Version Templates**: Keep metadata for tracking changes
3. **Monitor Delivery**: Track notification status
4. **Set up Webhooks**: Provider callbacks for delivery status
5. **Configure Retries**: Handle temporary failures
6. **Cache Templates**: Templates are cached after first load

## Extending

### Custom Providers

Implement the `notification.Provider` interface:

```go
type CustomProvider struct {}

func (p *CustomProvider) ID() string { return "custom" }
func (p *CustomProvider) Type() notification.NotificationType { return "email" }
func (p *CustomProvider) Send(ctx context.Context, n *notification.Notification) error { ... }
func (p *CustomProvider) GetStatus(ctx context.Context, id string) (notification.NotificationStatus, error) { ... }
func (p *CustomProvider) ValidateConfig() error { ... }

// Register
notificationSvc.RegisterProvider(&CustomProvider{})
```

### Custom Templates

Add your own templates programmatically:

```go
templateSvc.service.CreateTemplate(ctx, &notification.CreateTemplateRequest{
	OrganizationID: "default",
	TemplateKey:    "my.custom",
	Name:           "My Custom Template",
	Type:           notification.NotificationTypeEmail,
	Language:       "en",
	Subject:        "...",
	Body:           "...",
	Variables:      []string{"var1", "var2"},
})
```

## Troubleshooting

### Template Not Found
- Check organization_id matches
- Verify template_key is correct
- Ensure template is active
- Check language fallback

### Rendering Failures
- Validate template syntax
- Ensure all variables are provided
- Check for template function errors

### Send Failures
- Verify provider configuration
- Check rate limits
- Review provider credentials
- Check network connectivity

## See Also

- [Core Notification Service](/core/notification/)
- [Email Providers](/providers/email/)
- [SMS Providers](/providers/sms/)

