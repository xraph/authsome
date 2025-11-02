# AuthSome Notification System - Complete Example

This example demonstrates the **complete notification system** integrated with AuthSome.

## Features Demonstrated

✅ Notification plugin auto-initialization  
✅ Default template creation  
✅ Email OTP with templates  
✅ Magic link with templates  
✅ Phone OTP with templates  
✅ Mock providers for development  
✅ Template customization via API  
✅ Multi-language support  
✅ Organization-specific templates  
✅ Automatic welcome emails

## Quick Start

### 1. Run the Example

```bash
cd examples/notification-complete
go run main.go
```

### 2. Test Endpoints

#### Send Email OTP
```bash
curl -X POST http://localhost:8080/api/auth/email-otp/send \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

#### Send Magic Link
```bash
curl -X POST http://localhost:8080/api/auth/magic-link/send \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

#### Send Phone OTP
```bash
curl -X POST http://localhost:8080/api/auth/phone/send-code \
  -H "Content-Type: application/json" \
  -d '{"phone": "+1234567890"}'
```

### 3. Manage Templates

#### List All Templates
```bash
curl http://localhost:8080/api/auth/templates?organization_id=default
```

#### Get Specific Template
```bash
curl http://localhost:8080/api/auth/templates/{template_id}
```

#### Create Custom Template
```bash
curl -X POST http://localhost:8080/api/auth/templates \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": "default",
    "template_key": "custom.welcome",
    "name": "My Custom Welcome",
    "type": "email",
    "language": "en",
    "subject": "Welcome to {{.app_name}}",
    "body": "<h1>Hello {{.user_name}}</h1><p>Welcome!</p>",
    "variables": ["app_name", "user_name"]
  }'
```

#### Test Template Rendering
```bash
curl -X POST http://localhost:8080/api/auth/templates/{template_id}/test \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "user_name": "John Doe",
      "code": "123456",
      "expiry_minutes": 10
    }
  }'
```

### 4. View Notifications

#### List Sent Notifications
```bash
curl "http://localhost:8080/api/auth/notifications/list?organization_id=default"
```

#### Get Notification Details
```bash
curl http://localhost:8080/api/auth/notifications/{notification_id}
```

#### Check Delivery Status
```bash
curl http://localhost:8080/api/auth/notifications/{notification_id}/status
```

## Configuration

See `config.yaml` for complete configuration options.

### Switch to Real Providers

#### SMTP Email
```yaml
auth:
  notification:
    providers:
      email:
        provider: "smtp"
        from: "noreply@yourdomain.com"
        from_name: "Your App"
        config:
          host: "smtp.gmail.com"
          port: 587
          username: "your-email@gmail.com"
          password: "your-app-password"
          use_tls: true
```

#### SendGrid Email
```yaml
auth:
  notification:
    providers:
      email:
        provider: "sendgrid"
        from: "noreply@yourdomain.com"
        from_name: "Your App"
        config:
          api_key: "SG.your-api-key-here"
```

#### Twilio SMS
```yaml
auth:
  notification:
    providers:
      sms:
        provider: "twilio"
        from: "+1234567890"
        config:
          account_sid: "ACxxxxxxxxxxxxxxxxxxxxx"
          auth_token: "your-auth-token"
```

## Template Variables

### Common Variables
- `{{.app_name}}` - Application name
- `{{.user_name}}` - User's name
- `{{.user_email}}` - User's email

### Auth Flow Variables
- `{{.code}}` - OTP/verification code
- `{{.expiry_minutes}}` - Code expiry time
- `{{.magic_link}}` - Magic link URL
- `{{.verification_url}}` - Email verification URL
- `{{.reset_url}}` - Password reset URL

### Security Variables
- `{{.event_type}}` - Security event type
- `{{.event_time}}` - When event occurred
- `{{.location}}` - Event location
- `{{.device}}` - Device information

## Custom Template Example

```html
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background: #4CAF50; color: white; padding: 20px; }
        .content { padding: 20px; }
        .code { font-size: 32px; font-weight: bold; color: #4CAF50; }
        .footer { padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.app_name}}</h1>
    </div>
    <div class="content">
        <h2>Hello {{.user_name}},</h2>
        <p>Your verification code is:</p>
        <p class="code">{{.code}}</p>
        <p>This code will expire in {{.expiry_minutes}} minutes.</p>
    </div>
    <div class="footer">
        <p>© 2024 {{.app_name}}. All rights reserved.</p>
    </div>
</body>
</html>
```

## Multi-Language Support

Create language-specific templates:

```bash
# English template
curl -X POST http://localhost:8080/api/auth/templates \
  -d '{"template_key": "welcome", "language": "en", "subject": "Welcome", ...}'

# Spanish template
curl -X POST http://localhost:8080/api/auth/templates \
  -d '{"template_key": "welcome", "language": "es", "subject": "Bienvenido", ...}'

# French template
curl -X POST http://localhost:8080/api/auth/templates \
  -d '{"template_key": "welcome", "language": "fr", "subject": "Bienvenue", ...}'
```

The system automatically falls back to English if the requested language isn't found.

## SaaS Mode - Organization Templates

In SaaS mode, create org-specific templates:

```bash
# Default template for all orgs
curl -X POST http://localhost:8080/api/auth/templates \
  -d '{"organization_id": "default", "template_key": "welcome", ...}'

# Custom template for specific org
curl -X POST http://localhost:8080/api/auth/templates \
  -d '{"organization_id": "org_abc123", "template_key": "welcome", ...}'
```

## Webhooks

Receive delivery status updates:

```yaml
auth:
  notification:
    webhooks:
      - url: "https://your-app.com/webhooks/notification"
        events: ["sent", "delivered", "failed"]
```

Webhook payload:
```json
{
  "event": "delivered",
  "notification_id": "notif_123",
  "type": "email",
  "recipient": "user@example.com",
  "delivered_at": "2024-01-15T10:30:00Z"
}
```

## Testing

Run the included tests:

```bash
cd plugins/notification
go test -v
```

Tests cover:
- Plugin initialization
- Default template creation
- Email OTP sending
- Magic link sending
- Phone OTP sending
- Custom template creation
- Multi-language support
- Status tracking

## Troubleshooting

### Templates Not Sending

1. Check provider configuration
2. Verify templates exist: `curl http://localhost:8080/api/auth/templates`
3. Check notification status: `curl http://localhost:8080/api/auth/notifications/list`
4. Review logs for errors

### Mock Provider Not Working

- Ensure notification plugin is registered **first**
- Check database migrations ran successfully
- Verify `provider: "mock"` in config

### Real Provider Errors

- **SMTP**: Verify host, port, username, password, enable "Less secure apps" for Gmail
- **SendGrid**: Verify API key, check sender verification
- **Twilio**: Verify Account SID and Auth Token, check phone number format

## Production Checklist

✅ Configure real email provider (SMTP/SendGrid)  
✅ Configure real SMS provider (Twilio)  
✅ Customize default templates with your branding  
✅ Set up webhooks for delivery tracking  
✅ Configure rate limits appropriately  
✅ Enable audit logging  
✅ Set appropriate retry attempts  
✅ Configure cleanup schedule  
✅ Test all notification flows  
✅ Monitor notification delivery rates

## Support

For issues or questions:
- Check the main documentation: `/docs/notification/`
- Review the code: `/plugins/notification/`
- See integration examples: `/plugins/notification/INTEGRATION_EXAMPLE.md`

## License

MIT License - See LICENSE file for details

