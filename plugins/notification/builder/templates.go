package builder

import (
	"fmt"
	"strings"
)

// SampleTemplates provides pre-built email templates
var SampleTemplates = map[string]*Document{}

// TemplateInfo holds metadata about sample templates
type TemplateInfo struct {
	Name        string
	DisplayName string
	Description string
	Category    string
}

// GetAllTemplateInfo returns info about all available templates
func GetAllTemplateInfo() []TemplateInfo {
	return []TemplateInfo{
		{"welcome", "Welcome", "Modern onboarding email with feature highlights", "Onboarding"},
		{"otp", "Verification Code", "Clean OTP/2FA verification email", "Authentication"},
		{"reset_password", "Password Reset", "Secure password reset with instructions", "Authentication"},
		{"invitation", "Team Invitation", "Organization/team invitation email", "Collaboration"},
		{"magic_link", "Magic Link", "Passwordless sign-in email", "Authentication"},
		{"order_confirmation", "Order Confirmation", "E-commerce order receipt", "Transactional"},
		{"newsletter", "Newsletter", "Multi-column newsletter layout", "Marketing"},
		{"account_alert", "Account Alert", "Security alert notification", "Security"},
	}
}

func init() {
	SampleTemplates["welcome"] = createWelcomeTemplate()
	SampleTemplates["otp"] = createOTPTemplate()
	SampleTemplates["reset_password"] = createResetPasswordTemplate()
	SampleTemplates["invitation"] = createInvitationTemplate()
	SampleTemplates["magic_link"] = createMagicLinkTemplate()
	SampleTemplates["order_confirmation"] = createOrderConfirmationTemplate()
	SampleTemplates["newsletter"] = createNewsletterTemplate()
	SampleTemplates["account_alert"] = createAccountAlertTemplate()
}

// createWelcomeTemplate creates a beautiful welcome email template
func createWelcomeTemplate() *Document {
	doc := NewDocument()

	// Update root with modern styling
	setRootStyle(doc, "#F3F4F6", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Logo/Brand area
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#4F46E5",
			"padding": map[string]interface{}{
				"top": 40, "right": 24, "bottom": 40, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#4F46E5",
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Get the container ID
	brandContainerID := getLastBlockID(doc)

	// Brand name
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":  "center",
			"color":      "#FFFFFF",
			"fontWeight": "700",
		},
		"props": map[string]interface{}{
			"text":  "{{.AppName}}",
			"level": "h1",
		},
	}, brandContainerID)

	// Welcome message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#E0E7FF",
			"fontSize":  16,
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": "Welcome aboard! We're thrilled to have you.",
		},
	}, brandContainerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Greeting
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"color": "#1F2937",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 0, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":  "Hey {{.UserName}} 👋",
			"level": "h2",
		},
	}, doc.Root)

	// Intro text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#4B5563",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Your account has been successfully created and you're ready to start exploring. Here's what you can do next:",
		},
	}, doc.Root)

	// Feature boxes using columns
	columnsID, _ := doc.AddBlock(BlockTypeColumns, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 24, "bottom": 0, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"columnsCount": 3,
			"columnsGap":   16,
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Create three columns
	col1ID := addColumnToColumns(doc, columnsID)
	col2ID := addColumnToColumns(doc, columnsID)
	col3ID := addColumnToColumns(doc, columnsID)

	// Feature 1
	addFeatureBox(doc, col1ID, "#EEF2FF", "#4F46E5", "🚀", "Quick Setup", "Get started in minutes with our guided setup")
	// Feature 2
	addFeatureBox(doc, col2ID, "#ECFDF5", "#059669", "🔒", "Secure", "Enterprise-grade security for your data")
	// Feature 3
	addFeatureBox(doc, col3ID, "#FEF3C7", "#D97706", "⚡", "Fast", "Lightning-fast performance you'll love")

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 24},
	}, doc.Root)

	// CTA Button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "Start Exploring →",
			"url":          "{{.DashboardURL}}",
			"buttonColor":  "#4F46E5",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Divider
	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 0, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E7EB",
			"lineHeight": 1,
		},
	}, doc.Root)

	// Footer text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#9CA3AF",
			"padding": map[string]interface{}{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Questions? Reply to this email or visit our <a href=\"{{.HelpURL}}\" style=\"color: #4F46E5;\">Help Center</a>.",
		},
	}, doc.Root)

	return doc
}

// createOTPTemplate creates a clean OTP verification template
func createOTPTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#F8FAFC", "#FFFFFF", "#1E293B", "MODERN_SANS")

	// Top accent bar
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#0EA5E9",
			"padding": map[string]interface{}{
				"top": 4, "right": 0, "bottom": 4, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#0EA5E9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 40},
	}, doc.Root)

	// Lock icon
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  48,
		},
		"props": map[string]interface{}{
			"text": "🔐",
		},
	}, doc.Root)

	// Heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#0F172A",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 8, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Your Verification Code",
			"level": "h2",
		},
	}, doc.Root)

	// Subtitle
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#64748B",
			"fontSize":  15,
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Enter this code to verify your identity",
		},
	}, doc.Root)

	// OTP Container
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F1F5F9",
			"padding": map[string]interface{}{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
			"borderRadius": 12,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F1F5F9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	otpContainerID := getLastBlockID(doc)

	// OTP Code
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":     "center",
			"fontSize":      42,
			"fontWeight":    "bold",
			"color":         "#0EA5E9",
			"letterSpacing": "0.3em",
			"fontFamily":    "monospace",
		},
		"props": map[string]interface{}{
			"text": "{{.OTPCode}}",
		},
	}, otpContainerID)

	// Expiry notice
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#94A3B8",
			"padding": map[string]interface{}{
				"top": 12, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": "Expires in {{.ExpiresIn}} minutes",
		},
	}, otpContainerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Warning box
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]interface{}{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	warningContainerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize": 13,
			"color":    "#92400E",
		},
		"props": map[string]interface{}{
			"text": "⚠️ Never share this code with anyone. Our team will never ask for your verification code.",
		},
	}, warningContainerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	return doc
}

// createResetPasswordTemplate creates a secure password reset template
func createResetPasswordTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red accent header
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#DC2626",
			"padding": map[string]interface{}{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#DC2626",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]interface{}{
			"text": "🔑",
		},
	}, headerID)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]interface{}{
				"top": 12, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "Password Reset Request",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Greeting
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Hi <strong>{{.UserName}}</strong>,",
		},
	}, doc.Root)

	// Message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#4B5563",
			"fontSize":   15,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "We received a request to reset the password for your account. Click the button below to create a new password.",
		},
	}, doc.Root)

	// CTA Button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "Reset My Password",
			"url":          "{{.ResetURL}}",
			"buttonColor":  "#DC2626",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Link fallback
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize": 13,
			"color":    "#6B7280",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Or copy and paste this link: <br/><span style=\"color: #DC2626; word-break: break-all;\">{{.ResetURL}}</span>",
		},
	}, doc.Root)

	// Divider
	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 8, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E7EB",
			"lineHeight": 1,
		},
	}, doc.Root)

	// Security notice
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F3F4F6",
			"padding": map[string]interface{}{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F3F4F6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	securityBoxID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   13,
			"color":      "#6B7280",
			"lineHeight": "1.6",
		},
		"props": map[string]interface{}{
			"text": "🔒 <strong>Security Notice:</strong> This link will expire in {{.ExpiresIn}} hours. If you didn't request this reset, please ignore this email or contact support if you have concerns.",
		},
	}, securityBoxID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 24},
	}, doc.Root)

	return doc
}

// createInvitationTemplate creates a team invitation template
func createInvitationTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#F0FDF4", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#059669",
			"padding": map[string]interface{}{
				"top": 40, "right": 24, "bottom": 40, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#059669",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  48,
		},
		"props": map[string]interface{}{
			"text": "🎉",
		},
	}, headerID)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]interface{}{
				"top": 12, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "You've Been Invited!",
			"level": "h1",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Invitation message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":  "center",
			"color":      "#374151",
			"fontSize":   17,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "<strong>{{.InviterName}}</strong> has invited you to join",
		},
	}, doc.Root)

	// Organization card
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F0FDF4",
			"padding": map[string]interface{}{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
			"borderRadius": 12,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F0FDF4",
		},
		"childrenIds": []string{},
	}, doc.Root)

	orgCardID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#059669",
		},
		"props": map[string]interface{}{
			"text":  "{{.OrganizationName}}",
			"level": "h2",
		},
	}, orgCardID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": "You'll be joining as: <strong style=\"color: #059669;\">{{.Role}}</strong>",
		},
	}, orgCardID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 24},
	}, doc.Root)

	// Accept button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "Accept Invitation",
			"url":          "{{.InvitationURL}}",
			"buttonColor":  "#059669",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Decline option
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#9CA3AF",
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Not interested? You can safely ignore this email.",
		},
	}, doc.Root)

	return doc
}

// createMagicLinkTemplate creates a passwordless sign-in template
func createMagicLinkTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#EEF2FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Purple gradient header
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#7C3AED",
			"padding": map[string]interface{}{
				"top": 40, "right": 24, "bottom": 40, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#7C3AED",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  48,
		},
		"props": map[string]interface{}{
			"text": "✨",
		},
	}, headerID)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]interface{}{
				"top": 12, "right": 0, "bottom": 4, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "Sign In with Magic Link",
			"level": "h2",
		},
	}, headerID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#DDD6FE",
			"fontSize":  14,
		},
		"props": map[string]interface{}{
			"text": "No password needed",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":  "center",
			"color":      "#4B5563",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Click the button below to securely sign in to your account. This link is valid for {{.ExpiresIn}} minutes.",
		},
	}, doc.Root)

	// Magic link button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "✨ Sign In Now",
			"url":          "{{.MagicLinkURL}}",
			"buttonColor":  "#7C3AED",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Device info box
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F5F3FF",
			"padding": map[string]interface{}{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F5F3FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	deviceBoxID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   13,
			"color":      "#5B21B6",
			"lineHeight": "1.6",
		},
		"props": map[string]interface{}{
			"text": "📱 <strong>Request details:</strong><br/>Device: {{.DeviceName}}<br/>Location: {{.Location}}<br/>Time: {{.RequestTime}}",
		},
	}, deviceBoxID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 16},
	}, doc.Root)

	// Security note
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  12,
			"color":     "#9CA3AF",
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "If you didn't request this link, you can safely ignore this email.",
		},
	}, doc.Root)

	return doc
}

// createOrderConfirmationTemplate creates an e-commerce order confirmation
func createOrderConfirmationTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#F9FAFB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Success header
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#10B981",
			"padding": map[string]interface{}{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#10B981",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  48,
		},
		"props": map[string]interface{}{
			"text": "✓",
		},
	}, headerID)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "Order Confirmed!",
			"level": "h2",
		},
	}, headerID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#D1FAE5",
			"fontSize":  14,
			"padding": map[string]interface{}{
				"top": 4, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": "Order #{{.OrderNumber}}",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Thank you message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Thank you for your order, <strong>{{.CustomerName}}</strong>! We're getting your items ready to ship.",
		},
	}, doc.Root)

	// Order details card
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F9FAFB",
			"padding": map[string]interface{}{
				"top": 24, "right": 24, "bottom": 24, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F9FAFB",
		},
		"childrenIds": []string{},
	}, doc.Root)

	orderCardID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"color":    "#1F2937",
			"fontSize": 16,
			"padding": map[string]interface{}{
				"top": 0, "right": 0, "bottom": 16, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "Order Summary",
			"level": "h3",
		},
	}, orderCardID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   14,
			"color":      "#4B5563",
			"lineHeight": "2",
		},
		"props": map[string]interface{}{
			"text": "{{.OrderItems}}",
		},
	}, orderCardID)

	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 0, "bottom": 16, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E7EB",
			"lineHeight": 1,
		},
	}, orderCardID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   16,
			"fontWeight": "bold",
			"color":      "#1F2937",
		},
		"props": map[string]interface{}{
			"text": "Total: <span style=\"color: #10B981;\">{{.OrderTotal}}</span>",
		},
	}, orderCardID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 24},
	}, doc.Root)

	// Shipping info
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   14,
			"color":      "#6B7280",
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "📦 <strong>Shipping to:</strong><br/>{{.ShippingAddress}}",
		},
	}, doc.Root)

	// Track order button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "Track Your Order",
			"url":          "{{.TrackingURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	return doc
}

// createNewsletterTemplate creates a multi-column newsletter
func createNewsletterTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#1E293B", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Header with brand
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#1E293B",
			"padding": map[string]interface{}{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#1E293B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
		},
		"props": map[string]interface{}{
			"text":  "{{.NewsletterName}}",
			"level": "h1",
		},
	}, headerID)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#94A3B8",
			"fontSize":  14,
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": "{{.EditionDate}} • Issue #{{.IssueNumber}}",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Featured article
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"color": "#1F2937",
			"padding": map[string]interface{}{
				"top": 0, "right": 24, "bottom": 8, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "{{.FeaturedTitle}}",
			"level": "h2",
		},
	}, doc.Root)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#4B5563",
			"fontSize":   15,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text": "{{.FeaturedExcerpt}}",
		},
	}, doc.Root)

	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 24, "bottom": 24, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "Read More →",
			"url":          "{{.FeaturedURL}}",
			"buttonColor":  "#1E293B",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
			"fullWidth":    false,
		},
	}, doc.Root)

	// Divider
	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 8, "right": 24, "bottom": 24, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E7EB",
			"lineHeight": 1,
		},
	}, doc.Root)

	// More articles section title
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"color":    "#1F2937",
			"fontSize": 18,
			"padding": map[string]interface{}{
				"top": 0, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "More Stories",
			"level": "h3",
		},
	}, doc.Root)

	// Two-column articles
	columnsID, _ := doc.AddBlock(BlockTypeColumns, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 16, "bottom": 24, "left": 16,
			},
		},
		"props": map[string]interface{}{
			"columnsCount": 2,
			"columnsGap":   16,
		},
		"childrenIds": []string{},
	}, doc.Root)

	col1ID := addColumnToColumns(doc, columnsID)
	col2ID := addColumnToColumns(doc, columnsID)

	// Article 1
	addArticleCard(doc, col1ID, "{{.Article1Title}}", "{{.Article1Excerpt}}", "{{.Article1URL}}")
	// Article 2
	addArticleCard(doc, col2ID, "{{.Article2Title}}", "{{.Article2Excerpt}}", "{{.Article2URL}}")

	// Footer
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F1F5F9",
			"padding": map[string]interface{}{
				"top": 24, "right": 24, "bottom": 24, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F1F5F9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	footerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#64748B",
		},
		"props": map[string]interface{}{
			"text": "You're receiving this because you subscribed to {{.NewsletterName}}.<br/><a href=\"{{.UnsubscribeURL}}\" style=\"color: #64748B;\">Unsubscribe</a> • <a href=\"{{.PreferencesURL}}\" style=\"color: #64748B;\">Update preferences</a>",
		},
	}, footerID)

	return doc
}

// createAccountAlertTemplate creates a security alert template
func createAccountAlertTemplate() *Document {
	doc := NewDocument()

	// Update root styling
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Alert header
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#EF4444",
			"padding": map[string]interface{}{
				"top": 24, "right": 24, "bottom": 24, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#EF4444",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  36,
		},
		"props": map[string]interface{}{
			"text": "⚠️",
		},
	}, headerID)

	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text":  "Security Alert",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 32},
	}, doc.Root)

	// Alert message
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Hi <strong>{{.UserName}}</strong>,<br/><br/>We detected {{.AlertType}} on your account:",
		},
	}, doc.Root)

	// Details card
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#FEF2F2",
			"padding": map[string]interface{}{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#FEF2F2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.8",
		},
		"props": map[string]interface{}{
			"text": "🕐 <strong>Time:</strong> {{.AlertTime}}<br/>📍 <strong>Location:</strong> {{.AlertLocation}}<br/>💻 <strong>Device:</strong> {{.AlertDevice}}<br/>🌐 <strong>IP Address:</strong> {{.AlertIP}}",
		},
	}, detailsID)

	// Spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{"height": 24},
	}, doc.Root)

	// Action buttons
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 12, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "This Was Me",
			"url":          "{{.ConfirmURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text":         "Secure My Account",
			"url":          "{{.SecureAccountURL}}",
			"buttonColor":  "#EF4444",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Help text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#6B7280",
			"padding": map[string]interface{}{
				"top": 0, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "If you don't recognize this activity, please secure your account immediately and contact our support team.",
		},
	}, doc.Root)

	return doc
}

// Helper functions

// setRootStyle updates the root block styling
func setRootStyle(doc *Document, backdropColor, canvasColor, textColor, fontFamily string) {
	if rootBlock, exists := doc.Blocks["root"]; exists {
		// Get existing childrenIds
		childrenIds := []string{}
		if rootBlock.Data != nil {
			if ids, ok := rootBlock.Data["childrenIds"].([]string); ok {
				childrenIds = ids
			}
		}
		doc.Blocks["root"] = Block{
			Type: rootBlock.Type,
			Data: map[string]interface{}{
				"backdropColor": backdropColor,
				"canvasColor":   canvasColor,
				"textColor":     textColor,
				"fontFamily":    fontFamily,
				"childrenIds":   childrenIds,
			},
		}
	}
}

func getLastBlockID(doc *Document) string {
	// Find the highest block number
	maxNum := 0
	for id := range doc.Blocks {
		if strings.HasPrefix(id, "block-") {
			var num int
			fmt.Sscanf(id, "block-%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("block-%d", maxNum)
}

func addColumnToColumns(doc *Document, columnsID string) string {
	colID := fmt.Sprintf("col-%d-%d", len(doc.Blocks), 0)
	for i := 0; ; i++ {
		colID = fmt.Sprintf("col-%d-%d", len(doc.Blocks), i)
		if _, exists := doc.Blocks[colID]; !exists {
			break
		}
	}

	doc.Blocks[colID] = Block{
		Type: "Column",
		Data: map[string]interface{}{
			"style":       map[string]interface{}{},
			"props":       map[string]interface{}{},
			"childrenIds": []string{},
		},
	}

	// Add to columns' childrenIds
	if columnsBlock, exists := doc.Blocks[columnsID]; exists && columnsBlock.Data != nil {
		if childrenIds, ok := columnsBlock.Data["childrenIds"].([]string); ok {
			columnsBlock.Data["childrenIds"] = append(childrenIds, colID)
		} else {
			columnsBlock.Data["childrenIds"] = []string{colID}
		}
		doc.Blocks[columnsID] = columnsBlock
	}

	return colID
}

func addFeatureBox(doc *Document, parentID, bgColor, iconColor, icon, title, description string) {
	// Container
	containerID, _ := doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": bgColor,
			"padding": map[string]interface{}{
				"top": 20, "right": 16, "bottom": 20, "left": 16,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": bgColor,
		},
		"childrenIds": []string{},
	}, parentID)

	// Icon
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  28,
		},
		"props": map[string]interface{}{
			"text": icon,
		},
	}, containerID)

	// Title
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":  "center",
			"fontWeight": "bold",
			"fontSize":   14,
			"color":      iconColor,
			"padding": map[string]interface{}{
				"top": 8, "right": 0, "bottom": 4, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": title,
		},
	}, containerID)

	// Description
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  12,
			"color":     "#6B7280",
		},
		"props": map[string]interface{}{
			"text": description,
		},
	}, containerID)
}

func addArticleCard(doc *Document, parentID, title, excerpt, url string) {
	// Container
	containerID, _ := doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F8FAFC",
			"padding": map[string]interface{}{
				"top": 16, "right": 16, "bottom": 16, "left": 16,
			},
			"borderRadius": 8,
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F8FAFC",
		},
		"childrenIds": []string{},
	}, parentID)

	// Title
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontWeight": "bold",
			"fontSize":   14,
			"color":      "#1E293B",
			"padding": map[string]interface{}{
				"top": 0, "right": 0, "bottom": 8, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": title,
		},
	}, containerID)

	// Excerpt
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   13,
			"color":      "#64748B",
			"lineHeight": "1.5",
			"padding": map[string]interface{}{
				"top": 0, "right": 0, "bottom": 12, "left": 0,
			},
		},
		"props": map[string]interface{}{
			"text": excerpt,
		},
	}, containerID)

	// Link
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize": 13,
		},
		"props": map[string]interface{}{
			"text": fmt.Sprintf("<a href=\"%s\" style=\"color: #1E293B; font-weight: 600;\">Read more →</a>", url),
		},
	}, containerID)
}

// GetSampleTemplate returns a sample template by name
func GetSampleTemplate(name string) (*Document, error) {
	if template, exists := SampleTemplates[name]; exists {
		// Return a copy
		jsonStr, err := template.ToJSON()
		if err != nil {
			return nil, err
		}
		return FromJSON(jsonStr)
	}
	return nil, fmt.Errorf("template %s not found", name)
}

// ListSampleTemplates returns list of available sample templates
func ListSampleTemplates() []string {
	names := make([]string, 0, len(SampleTemplates))
	for name := range SampleTemplates {
		names = append(names, name)
	}
	return names
}

// RenderTemplate is a helper to render a template to HTML
func RenderTemplate(doc *Document, variables map[string]interface{}) (string, error) {
	// First render to HTML
	renderer := NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	// If variables provided, apply template engine
	if len(variables) > 0 {
		// Simple variable replacement - you can enhance this with the notification template engine
		for key, value := range variables {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			valueStr := fmt.Sprintf("%v", value)
			html = strings.Replace(html, placeholder, valueStr, -1)
		}
	}

	return html, nil
}
