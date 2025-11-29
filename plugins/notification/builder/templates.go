package builder

import (
	"fmt"
	"strings"
)

// SampleTemplates provides pre-built email templates
var SampleTemplates = map[string]*Document{}

func init() {
	SampleTemplates["welcome"] = createWelcomeTemplate()
	SampleTemplates["otp"] = createOTPTemplate()
	SampleTemplates["reset_password"] = createResetPasswordTemplate()
	SampleTemplates["invitation"] = createInvitationTemplate()
	SampleTemplates["notification"] = createNotificationTemplate()
}

// createWelcomeTemplate creates a beautiful welcome email template
func createWelcomeTemplate() *Document {
	doc := NewDocument()

	// Add spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 8, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Welcome to {{.AppName}}!",
			"level": "h1",
		},
	}, doc.Root)

	// Add text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#666666",
			"fontSize":  16,
			"padding": map[string]interface{}{
				"top": 8, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text": "Hi {{.UserName}}, we're excited to have you on board!",
		},
	}, doc.Root)

	// Add divider
	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E5E5",
			"lineHeight": 1,
		},
	}, doc.Root)

	// Add main text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize":   15,
			"lineHeight": "1.6",
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Your account has been successfully created. You can now access all features and start exploring.",
		},
	}, doc.Root)

	// Add button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "Get Started",
			"url":          "{{.DashboardURL}}",
			"buttonColor":  "#0066CC",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
			"fullWidth":    false,
		},
	}, doc.Root)

	return doc
}

// createOTPTemplate creates a one-time password template
func createOTPTemplate() *Document {
	doc := NewDocument()

	// Add spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Verification Code",
			"level": "h2",
		},
	}, doc.Root)

	// Add text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#666666",
			"fontSize":  15,
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Use the following code to complete your verification:",
		},
	}, doc.Root)

	// Add OTP code container
	doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F8F9FA",
			"padding": map[string]interface{}{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F8F9FA",
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Add OTP code text
	lastBlockID := fmt.Sprintf("block-%d", len(doc.Blocks)-1)
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign":     "center",
			"fontSize":      32,
			"fontWeight":    "bold",
			"color":         "#0066CC",
			"letterSpacing": "0.25em",
		},
		"props": map[string]interface{}{
			"text": "{{.OTPCode}}",
		},
	}, lastBlockID)

	// Add expiration text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#999999",
			"padding": map[string]interface{}{
				"top": 24, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "This code expires in {{.ExpiresIn}} minutes.",
		},
	}, doc.Root)

	return doc
}

// createResetPasswordTemplate creates a password reset template
func createResetPasswordTemplate() *Document {
	doc := NewDocument()

	// Add spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "Reset Your Password",
			"level": "h2",
		},
	}, doc.Root)

	// Add text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#666666",
			"fontSize":  15,
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "Hi {{.UserName}}, we received a request to reset your password.",
		},
	}, doc.Root)

	// Add button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 24, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "Reset Password",
			"url":          "{{.ResetURL}}",
			"buttonColor":  "#DC2626",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
			"fullWidth":    false,
		},
	}, doc.Root)

	// Add divider
	doc.AddBlock(BlockTypeDivider, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"lineColor":  "#E5E5E5",
			"lineHeight": 1,
		},
	}, doc.Root)

	// Add security text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize": 13,
			"color":    "#999999",
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "If you didn't request this, you can safely ignore this email. This link expires in {{.ExpiresIn}} hours.",
		},
	}, doc.Root)

	return doc
}

// createInvitationTemplate creates an organization invitation template
func createInvitationTemplate() *Document {
	doc := NewDocument()

	// Add spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 8, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "You're Invited!",
			"level": "h2",
		},
	}, doc.Root)

	// Add text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#666666",
			"fontSize":  15,
			"padding": map[string]interface{}{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "{{.InviterName}} has invited you to join <strong>{{.OrganizationName}}</strong>.",
		},
	}, doc.Root)

	// Add container
	containerID, _ := doc.AddBlock(BlockTypeContainer, map[string]interface{}{
		"style": map[string]interface{}{
			"backgroundColor": "#F8F9FA",
			"padding": map[string]interface{}{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"backgroundColor": "#F8F9FA",
		},
		"childrenIds": []string{},
	}, doc.Root)

	// Add role info
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#666666",
		},
		"props": map[string]interface{}{
			"text": "Role: <strong>{{.Role}}</strong>",
		},
	}, containerID)

	// Add accept button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 24, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "Accept Invitation",
			"url":          "{{.InvitationURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
			"fullWidth":    false,
		},
	}, doc.Root)

	return doc
}

// createNotificationTemplate creates a general notification template
func createNotificationTemplate() *Document {
	doc := NewDocument()

	// Add spacer
	doc.AddBlock(BlockTypeSpacer, map[string]interface{}{
		"style": map[string]interface{}{},
		"props": map[string]interface{}{
			"height": 20,
		},
	}, doc.Root)

	// Add heading
	doc.AddBlock(BlockTypeHeading, map[string]interface{}{
		"style": map[string]interface{}{
			"textAlign": "center",
			"color":     "#1a1a1a",
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 16, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":  "{{.Title}}",
			"level": "h2",
		},
	}, doc.Root)

	// Add text
	doc.AddBlock(BlockTypeText, map[string]interface{}{
		"style": map[string]interface{}{
			"fontSize": 15,
			"color":    "#666666",
			"padding": map[string]interface{}{
				"top": 16, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]interface{}{
			"text": "{{.Message}}",
		},
	}, doc.Root)

	// Add optional button
	doc.AddBlock(BlockTypeButton, map[string]interface{}{
		"style": map[string]interface{}{
			"padding": map[string]interface{}{
				"top": 16, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]interface{}{
			"text":         "{{.ActionText}}",
			"url":          "{{.ActionURL}}",
			"buttonColor":  "#0066CC",
			"textColor":    "#FFFFFF",
			"borderRadius": 6,
			"fullWidth":    false,
		},
	}, doc.Root)

	return doc
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
