package sms

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Template represents an SMS template.
type Template struct {
	Name string
	Body string
}

// TemplateData represents data passed to SMS templates.
type TemplateData struct {
	UserName         string
	UserEmail        string
	OrganizationName string
	VerificationCode string
	ResetCode        string
	LoginCode        string
	IPAddress        string
	DeviceName       string
	Location         string
	Timestamp        string
	ExpiryTime       string
	AppName          string
	SupportPhone     string
}

// DefaultTemplates contains the default SMS templates.
var DefaultTemplates = map[string]*Template{
	"verification": {
		Name: "verification",
		Body: `{{.AppName}}: Your verification code is {{.VerificationCode}}. This code expires in {{.ExpiryTime}}. If you didn't request this, please ignore this message.`,
	},

	"two_factor": {
		Name: "two_factor",
		Body: `{{.AppName}}: Your 2FA code is {{.VerificationCode}}. This code expires in {{.ExpiryTime}}. Do not share this code with anyone.`,
	},

	"password_reset": {
		Name: "password_reset",
		Body: `{{.AppName}}: Your password reset code is {{.ResetCode}}. This code expires in {{.ExpiryTime}}. If you didn't request this, please contact support.`,
	},

	"login_code": {
		Name: "login_code",
		Body: `{{.AppName}}: Your login code is {{.LoginCode}}. This code expires in {{.ExpiryTime}}. Use this code to complete your sign-in.`,
	},

	"login_notification": {
		Name: "login_notification",
		Body: `{{.AppName}}: New login detected from {{.Location}} at {{.Timestamp}}. If this wasn't you, secure your account immediately.`,
	},

	"account_locked": {
		Name: "account_locked",
		Body: `{{.AppName}}: Your account has been temporarily locked due to suspicious activity. Contact support at {{.SupportPhone}} if you need assistance.`,
	},

	"device_verification": {
		Name: "device_verification",
		Body: `{{.AppName}}: New device login detected: {{.DeviceName}} from {{.Location}}. Your verification code is {{.VerificationCode}}. Expires in {{.ExpiryTime}}.`,
	},

	"password_changed": {
		Name: "password_changed",
		Body: `{{.AppName}}: Your password was successfully changed at {{.Timestamp}}. If you didn't make this change, contact support immediately.`,
	},

	"phone_verification": {
		Name: "phone_verification",
		Body: `{{.AppName}}: Verify your phone number with code {{.VerificationCode}}. This code expires in {{.ExpiryTime}}.`,
	},

	"backup_codes": {
		Name: "backup_codes",
		Body: `{{.AppName}}: Your backup codes have been regenerated. Make sure to save them in a secure location. Generated at {{.Timestamp}}.`,
	},
}

// RenderTemplate renders an SMS template with the provided data.
func RenderTemplate(templateName string, data *TemplateData) (*RenderedTemplate, error) {
	tmpl, exists := DefaultTemplates[templateName]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	// Parse and render the template
	bodyTmpl, err := template.New("sms").Parse(tmpl.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SMS template: %w", err)
	}

	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render SMS body: %w", err)
	}

	body := strings.TrimSpace(bodyBuf.String())

	// Validate SMS length (most carriers support up to 160 characters for single SMS)
	if len(body) > 160 {
		return nil, fmt.Errorf("SMS body too long: %d characters (max 160)", len(body))
	}

	return &RenderedTemplate{
		Body: body,
	}, nil
}

// RenderedTemplate represents a rendered SMS template.
type RenderedTemplate struct {
	Body string
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

// ValidateTemplate validates that a template renders correctly with sample data.
func ValidateTemplate(templateName string) error {
	sampleData := &TemplateData{
		UserName:         "John Doe",
		UserEmail:        "john@example.com",
		OrganizationName: "Example Org",
		VerificationCode: "123456",
		ResetCode:        "789012",
		LoginCode:        "345678",
		IPAddress:        "192.168.1.1",
		DeviceName:       "iPhone 15",
		Location:         "New York, NY",
		Timestamp:        "2024-01-15 10:30 AM",
		ExpiryTime:       "10 minutes",
		AppName:          "MyApp",
		SupportPhone:     "+1-555-0123",
	}

	_, err := RenderTemplate(templateName, sampleData)

	return err
}

// GetCharacterCount returns the character count for a rendered template.
func GetCharacterCount(templateName string, data *TemplateData) (int, error) {
	rendered, err := RenderTemplate(templateName, data)
	if err != nil {
		return 0, err
	}

	return len(rendered.Body), nil
}
