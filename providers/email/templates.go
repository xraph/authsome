package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// Template represents an email template.
type Template struct {
	Name     string
	Subject  string
	HTMLBody string
	TextBody string
}

// TemplateData represents data passed to email templates.
type TemplateData struct {
	UserName         string
	UserEmail        string
	OrganizationName string
	VerificationURL  string
	ResetURL         string
	LoginURL         string
	IPAddress        string
	UserAgent        string
	DeviceName       string
	Location         string
	Timestamp        string
	ExpiryTime       string
	SupportEmail     string
	CompanyName      string
	AppName          string
}

// DefaultTemplates contains the default email templates.
var DefaultTemplates = map[string]*Template{
	"verification": {
		Name:    "verification",
		Subject: "Verify your email address",
		HTMLBody: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Verify your email address</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #fff; padding: 30px; border: 1px solid #e9ecef; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; font-size: 14px; color: #6c757d; }
        .button { display: inline-block; padding: 12px 24px; background: #007bff; color: #fff; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .button:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Verify your email address</h2>
            <p>Hello {{.UserName}},</p>
            <p>Thank you for signing up for {{.AppName}}! To complete your registration, please verify your email address by clicking the button below:</p>
            <p style="text-align: center;">
                <a href="{{.VerificationURL}}" class="button">Verify Email Address</a>
            </p>
            <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
            <p style="word-break: break-all; background: #f8f9fa; padding: 10px; border-radius: 4px;">{{.VerificationURL}}</p>
            <p>This verification link will expire in {{.ExpiryTime}}.</p>
            <p>If you didn't create an account with {{.AppName}}, you can safely ignore this email.</p>
        </div>
        <div class="footer">
            <p>Need help? Contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
            <p>&copy; {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
		TextBody: `
Verify your email address

Hello {{.UserName}},

Thank you for signing up for {{.AppName}}! To complete your registration, please verify your email address by visiting this link:

{{.VerificationURL}}

This verification link will expire in {{.ExpiryTime}}.

If you didn't create an account with {{.AppName}}, you can safely ignore this email.

Need help? Contact us at {{.SupportEmail}}

© {{.CompanyName}}. All rights reserved.`,
	},

	"password_reset": {
		Name:    "password_reset",
		Subject: "Reset your password",
		HTMLBody: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Reset your password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #fff; padding: 30px; border: 1px solid #e9ecef; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; font-size: 14px; color: #6c757d; }
        .button { display: inline-block; padding: 12px 24px; background: #dc3545; color: #fff; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .button:hover { background: #c82333; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Reset your password</h2>
            <p>Hello {{.UserName}},</p>
            <p>We received a request to reset the password for your {{.AppName}} account ({{.UserEmail}}).</p>
            <p style="text-align: center;">
                <a href="{{.ResetURL}}" class="button">Reset Password</a>
            </p>
            <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
            <p style="word-break: break-all; background: #f8f9fa; padding: 10px; border-radius: 4px;">{{.ResetURL}}</p>
            <div class="warning">
                <strong>Security Information:</strong>
                <ul>
                    <li>This reset link will expire in {{.ExpiryTime}}</li>
                    <li>Request made from IP: {{.IPAddress}}</li>
                    <li>Device: {{.UserAgent}}</li>
                    <li>Time: {{.Timestamp}}</li>
                </ul>
            </div>
            <p>If you didn't request a password reset, please ignore this email or contact support if you have concerns about your account security.</p>
        </div>
        <div class="footer">
            <p>Need help? Contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
            <p>&copy; {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
		TextBody: `
Reset your password

Hello {{.UserName}},

We received a request to reset the password for your {{.AppName}} account ({{.UserEmail}}).

To reset your password, visit this link:
{{.ResetURL}}

Security Information:
- This reset link will expire in {{.ExpiryTime}}
- Request made from IP: {{.IPAddress}}
- Device: {{.UserAgent}}
- Time: {{.Timestamp}}

If you didn't request a password reset, please ignore this email or contact support if you have concerns about your account security.

Need help? Contact us at {{.SupportEmail}}

© {{.CompanyName}}. All rights reserved.`,
	},

	"login_notification": {
		Name:    "login_notification",
		Subject: "New login to your account",
		HTMLBody: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>New login to your account</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #fff; padding: 30px; border: 1px solid #e9ecef; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; font-size: 14px; color: #6c757d; }
        .info-box { background: #e7f3ff; border: 1px solid #b3d9ff; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>New login to your account</h2>
            <p>Hello {{.UserName}},</p>
            <p>We detected a new login to your {{.AppName}} account. Here are the details:</p>
            <div class="info-box">
                <strong>Login Details:</strong>
                <ul>
                    <li><strong>Time:</strong> {{.Timestamp}}</li>
                    <li><strong>IP Address:</strong> {{.IPAddress}}</li>
                    <li><strong>Location:</strong> {{.Location}}</li>
                    <li><strong>Device:</strong> {{.DeviceName}}</li>
                    <li><strong>Browser:</strong> {{.UserAgent}}</li>
                </ul>
            </div>
            <p>If this was you, no action is needed.</p>
            <div class="warning">
                <strong>If this wasn't you:</strong>
                <ol>
                    <li>Change your password immediately</li>
                    <li>Review your account activity</li>
                    <li>Enable two-factor authentication if not already enabled</li>
                    <li>Contact support if you need assistance</li>
                </ol>
            </div>
        </div>
        <div class="footer">
            <p>Need help? Contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
            <p>&copy; {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
		TextBody: `
New login to your account

Hello {{.UserName}},

We detected a new login to your {{.AppName}} account. Here are the details:

Login Details:
- Time: {{.Timestamp}}
- IP Address: {{.IPAddress}}
- Location: {{.Location}}
- Device: {{.DeviceName}}
- Browser: {{.UserAgent}}

If this was you, no action is needed.

If this wasn't you:
1. Change your password immediately
2. Review your account activity
3. Enable two-factor authentication if not already enabled
4. Contact support if you need assistance

Need help? Contact us at {{.SupportEmail}}

© {{.CompanyName}}. All rights reserved.`,
	},

	"magic_link": {
		Name:    "magic_link",
		Subject: "Your magic link to sign in",
		HTMLBody: `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Your magic link to sign in</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #fff; padding: 30px; border: 1px solid #e9ecef; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 8px 8px; font-size: 14px; color: #6c757d; }
        .button { display: inline-block; padding: 12px 24px; background: #28a745; color: #fff; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .button:hover { background: #218838; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Your magic link to sign in</h2>
            <p>Hello {{.UserName}},</p>
            <p>Click the button below to sign in to your {{.AppName}} account:</p>
            <p style="text-align: center;">
                <a href="{{.LoginURL}}" class="button">Sign In</a>
            </p>
            <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
            <p style="word-break: break-all; background: #f8f9fa; padding: 10px; border-radius: 4px;">{{.LoginURL}}</p>
            <p>This magic link will expire in {{.ExpiryTime}} and can only be used once.</p>
            <p>If you didn't request this magic link, you can safely ignore this email.</p>
        </div>
        <div class="footer">
            <p>Need help? Contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
            <p>&copy; {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
		TextBody: `
Your magic link to sign in

Hello {{.UserName}},

Click this link to sign in to your {{.AppName}} account:
{{.LoginURL}}

This magic link will expire in {{.ExpiryTime}} and can only be used once.

If you didn't request this magic link, you can safely ignore this email.

Need help? Contact us at {{.SupportEmail}}

© {{.CompanyName}}. All rights reserved.`,
	},
}

// RenderTemplate renders an email template with the provided data.
func RenderTemplate(templateName string, data *TemplateData) (*RenderedTemplate, error) {
	tmpl, exists := DefaultTemplates[templateName]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	// Render subject
	subjectTmpl, err := template.New("subject").Parse(tmpl.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render subject: %w", err)
	}

	// Render HTML body
	htmlTmpl, err := template.New("html").Parse(tmpl.HTMLBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render HTML body: %w", err)
	}

	// Render text body
	textTmpl, err := template.New("text").Parse(tmpl.TextBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render text body: %w", err)
	}

	return &RenderedTemplate{
		Subject:  strings.TrimSpace(subjectBuf.String()),
		HTMLBody: strings.TrimSpace(htmlBuf.String()),
		TextBody: strings.TrimSpace(textBuf.String()),
	}, nil
}

// RenderedTemplate represents a rendered email template.
type RenderedTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}

// GetTemplate returns a template by name.
func GetTemplate(name string) (*Template, bool) {
	tmpl, exists := DefaultTemplates[name]

	return tmpl, exists
}

// ListTemplates returns all available template names.
func ListTemplates() []string {
	names := make([]string, 0, len(DefaultTemplates))
	for name := range DefaultTemplates {
		names = append(names, name)
	}

	return names
}
