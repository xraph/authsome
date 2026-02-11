package builder

// auth_templates.go contains all authentication, organization, account, session, reminder, and admin templates
// Generated using the builder's Document/Block API for consistent, UI-editable email templates

// mustAddBlock is a helper that panics if AddBlock returns an error.
// This is appropriate for template generation code where errors are unrecoverable.
func mustAddBlock(doc *Document, blockType BlockType, data map[string]any, parentID string) string {
	id, err := doc.AddBlock(blockType, data, parentID)
	if err != nil {
		panic("failed to add block to template: " + err.Error())
	}

	return id
}

// =============================================================================
// ORGANIZATION TEMPLATES (Green Theme: #059669)
// =============================================================================

// createOrgInviteTemplate creates organization invitation template.
func createOrgInviteTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0FDF4", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#059669",
			"padding": map[string]any{
				"top": 40, "right": 24, "bottom": 40, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#059669",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  48,
		},
		"props": map[string]any{
			"text": "🎉",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 12, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "You've Been Invited!",
			"level": "h1",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign":  "center",
			"color":      "#374151",
			"fontSize":   17,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "<strong>{{.inviterName}}</strong> has invited you to join",
		},
	}, doc.Root)

	// Organization card
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F0FDF4",
			"padding": map[string]any{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
			"borderRadius": 12,
		},
		"props": map[string]any{
			"backgroundColor": "#F0FDF4",
		},
		"childrenIds": []string{},
	}, doc.Root)

	orgCardID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#059669",
		},
		"props": map[string]any{
			"text":  "{{.orgName}}",
			"level": "h2",
		},
	}, orgCardID)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text": "You'll be joining as: <strong style=\"color: #059669;\">{{.role}}</strong>",
		},
	}, orgCardID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	// Accept button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Accept Invitation",
			"url":          "{{.inviteURL}}",
			"buttonColor":  "#059669",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Expiry notice
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#9CA3AF",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "This invitation expires in {{.expiresIn}}. Not interested? You can safely ignore this email.",
		},
	}, doc.Root)

	return doc
}

// createOrgMemberAddedTemplate creates template for when member is added to org.
func createOrgMemberAddedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0FDF4", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#10B981",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#10B981",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "✅",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "New Team Member Added",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/><strong>{{.memberName}}</strong> has been added to <strong>{{.orgName}}</strong> as a <strong>{{.role}}</strong>.",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#ECFDF5",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#ECFDF5",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#065F46",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "👤 <strong>Member:</strong> {{.memberName}}<br/>🏢 <strong>Organization:</strong> {{.orgName}}<br/>🎭 <strong>Role:</strong> {{.role}}",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createOrgMemberRemovedTemplate creates template for when member is removed from org.
func createOrgMemberRemovedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#EF4444",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#EF4444",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🚫",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Team Member Removed",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/><strong>{{.memberName}}</strong> has been removed from <strong>{{.orgName}}</strong>.",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "👤 <strong>Member:</strong> {{.memberName}}<br/>🏢 <strong>Organization:</strong> {{.orgName}}<br/>🕐 <strong>Removed:</strong> {{.timestamp}}",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createOrgRoleChangedTemplate creates template for role changes.
func createOrgRoleChangedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0FDF4", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#059669",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#059669",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🎭",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Role Updated",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your role in <strong>{{.orgName}}</strong> has been updated.",
		},
	}, doc.Root)

	// Role change box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F0FDF4",
			"padding": map[string]any{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F0FDF4",
		},
		"childrenIds": []string{},
	}, doc.Root)

	roleBoxID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign":  "center",
			"fontSize":   16,
			"color":      "#059669",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "<strong>{{.oldRole}}</strong> → <strong>{{.newRole}}</strong>",
		},
	}, roleBoxID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createOrgTransferTemplate creates template for organization ownership transfer.
func createOrgTransferTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0FDF4", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#059669",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#059669",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "👑",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Organization Transferred",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Ownership of <strong>{{.orgName}}</strong> has been transferred to <strong>{{.transferredTo}}</strong>.",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F0FDF4",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F0FDF4",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#065F46",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "🏢 <strong>Organization:</strong> {{.orgName}}<br/>👤 <strong>New Owner:</strong> {{.transferredTo}}<br/>🕐 <strong>Transferred:</strong> {{.timestamp}}",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createOrgDeletedTemplate creates template for organization deletion.
func createOrgDeletedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DC2626",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#DC2626",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "⚠️",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Organization Deleted",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>The organization <strong>{{.orgName}}</strong> has been permanently deleted.",
		},
	}, doc.Root)

	// Warning box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	warningID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "⚠️ This action is permanent. All data associated with this organization has been removed.",
		},
	}, warningID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createOrgMemberLeftTemplate creates template when member leaves organization.
func createOrgMemberLeftTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F9FAFB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Gray header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#6B7280",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#6B7280",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "👋",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Member Left Organization",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/><strong>{{.memberName}}</strong> has left <strong>{{.orgName}}</strong>.",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F3F4F6",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F3F4F6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#374151",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "👤 <strong>Member:</strong> {{.memberName}}<br/>🏢 <strong>Organization:</strong> {{.orgName}}<br/>🕐 <strong>Left:</strong> {{.timestamp}}",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// =============================================================================
// ACCOUNT MANAGEMENT TEMPLATES (Blue Theme: #0EA5E9)
// =============================================================================

// createEmailChangeRequestTemplate creates email change confirmation template.
func createEmailChangeRequestTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#0EA5E9",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#0EA5E9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "📧",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Confirm Email Change",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We received a request to change your email address from <strong>{{.oldEmail}}</strong> to <strong>{{.newEmail}}</strong>.",
		},
	}, doc.Root)

	// Confirm button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Confirm Email Change",
			"url":          "{{.confirmURL}}",
			"buttonColor":  "#0EA5E9",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Security notice
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DBEAFE",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#DBEAFE",
		},
		"childrenIds": []string{},
	}, doc.Root)

	securityID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#1E40AF",
		},
		"props": map[string]any{
			"text": "🔒 If you didn't request this change, please ignore this email and your email address will remain unchanged.",
		},
	}, securityID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createEmailChangedTemplate creates email changed confirmation template.
func createEmailChangedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#3B82F6",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#3B82F6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "✅",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Email Address Changed",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your email address has been successfully changed.",
		},
	}, doc.Root)

	// Change details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#EFF6FF",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#EFF6FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#1E40AF",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📧 <strong>Old Email:</strong> {{.oldEmail}}<br/>📧 <strong>New Email:</strong> {{.newEmail}}<br/>🕐 <strong>Changed:</strong> {{.changeTime}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createPasswordChangedTemplate creates password changed notification template.
func createPasswordChangedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#0EA5E9",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#0EA5E9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔑",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Password Changed",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your password was successfully changed on {{.changeTime}}.",
		},
	}, doc.Root)

	// Security warning
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	warningID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#92400E",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "⚠️ <strong>Didn't change your password?</strong><br/>If this wasn't you, please secure your account immediately by contacting support.",
		},
	}, warningID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createUsernameChangedTemplate creates username changed notification template.
func createUsernameChangedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#3B82F6",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#3B82F6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "👤",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Username Updated",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your username has been successfully updated.",
		},
	}, doc.Root)

	// Username box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#EFF6FF",
			"padding": map[string]any{
				"top": 24, "right": 32, "bottom": 24, "left": 32,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#EFF6FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	usernameBoxID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign":  "center",
			"fontSize":   18,
			"color":      "#1E40AF",
			"fontWeight": "bold",
		},
		"props": map[string]any{
			"text": "{{.newUsername}}",
		},
	}, usernameBoxID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createAccountDeletedTemplate creates account deletion confirmation template.
func createAccountDeletedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DC2626",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#DC2626",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "👋",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Account Deleted",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your account has been permanently deleted. We're sorry to see you go!",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "⚠️ All your data has been permanently removed. This action cannot be undone.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createAccountSuspendedTemplate creates account suspension notification template.
func createAccountSuspendedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF3C7", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Orange/warning header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "⏸️",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Account Suspended",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your account has been temporarily suspended.",
		},
	}, doc.Root)

	// Suspension details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#92400E",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📋 <strong>Reason:</strong> {{.reason}}<br/>⏰ <strong>Suspended Until:</strong> {{.suspendedUntil}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 16},
	}, doc.Root)

	// Contact support text
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "If you believe this is a mistake, please contact our support team.",
		},
	}, doc.Root)

	return doc
}

// createAccountReactivatedTemplate creates account reactivation notification template.
func createAccountReactivatedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#ECFDF5", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#10B981",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#10B981",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "✅",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Welcome Back!",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Great news! Your account has been reactivated. You can now sign in and access all features.",
		},
	}, doc.Root)

	// Login button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Sign In Now",
			"url":          "{{.loginURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	return doc
}

// createDataExportReadyTemplate creates data export ready notification template.
func createDataExportReadyTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#0EA5E9",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#0EA5E9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "📦",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Your Data Export is Ready",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your requested data export has been processed and is now ready for download.",
		},
	}, doc.Root)

	// Download button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Download Your Data",
			"url":          "{{.downloadURL}}",
			"buttonColor":  "#0EA5E9",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Expiry notice
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DBEAFE",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#DBEAFE",
		},
		"childrenIds": []string{},
	}, doc.Root)

	expiryID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#1E40AF",
		},
		"props": map[string]any{
			"text": "⏰ This download link will expire in 7 days for security reasons.",
		},
	}, expiryID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// =============================================================================
// SESSION/DEVICE TEMPLATES (Purple Theme: #7C3AED)
// =============================================================================

// createNewDeviceLoginTemplate creates new device login notification template.
func createNewDeviceLoginTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F5F3FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Purple header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#7C3AED",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#7C3AED",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "📱",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "New Device Sign-In",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We noticed a sign-in from a new device.",
		},
	}, doc.Root)

	// Device details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F5F3FF",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F5F3FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#5B21B6",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📱 <strong>Device:</strong> {{.deviceName}}<br/>🌐 <strong>Browser:</strong> {{.browserName}}<br/>💻 <strong>OS:</strong> {{.osName}}<br/>📍 <strong>Location:</strong> {{.location}}<br/>🕐 <strong>Time:</strong> {{.timestamp}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	// Action buttons
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 12, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "This Was Me",
			"url":          "{{.confirmURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Secure My Account",
			"url":          "{{.secureAccountURL}}",
			"buttonColor":  "#7C3AED",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	return doc
}

// createNewLocationLoginTemplate creates new location login notification template.
func createNewLocationLoginTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F5F3FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Purple header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#8B5CF6",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#8B5CF6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🌍",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "New Location Sign-In",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We noticed a sign-in from a new location.",
		},
	}, doc.Root)

	// Location details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F5F3FF",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F5F3FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#5B21B6",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📍 <strong>Location:</strong> {{.location}}<br/>🌐 <strong>IP Address:</strong> {{.ipAddress}}<br/>🕐 <strong>Time:</strong> {{.timestamp}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 16},
	}, doc.Root)

	// Help text
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "If this wasn't you, please <a href=\"{{.secureAccountURL}}\" style=\"color: #7C3AED;\">secure your account</a> immediately.",
		},
	}, doc.Root)

	return doc
}

// createSuspiciousLoginTemplate creates suspicious login alert template.
func createSuspiciousLoginTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red/warning header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#EF4444",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#EF4444",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🚨",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Suspicious Login Detected",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We detected a suspicious login attempt on your account.",
		},
	}, doc.Root)

	// Suspicious activity details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📍 <strong>Location:</strong> {{.location}}<br/>🌐 <strong>IP Address:</strong> {{.ipAddress}}<br/>💻 <strong>Device:</strong> {{.deviceName}}<br/>🕐 <strong>Time:</strong> {{.timestamp}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	// Action button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Secure My Account Now",
			"url":          "{{.secureAccountURL}}",
			"buttonColor":  "#EF4444",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Urgent notice
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign":  "center",
			"fontSize":   14,
			"color":      "#DC2626",
			"fontWeight": "bold",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "⚠️ Please change your password immediately if this wasn't you!",
		},
	}, doc.Root)

	return doc
}

// createDeviceRemovedTemplate creates device removed notification template.
func createDeviceRemovedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F5F3FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Purple header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#7C3AED",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#7C3AED",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔌",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Device Removed",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>A device has been removed from your account.",
		},
	}, doc.Root)

	// Device details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F5F3FF",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#F5F3FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#5B21B6",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📱 <strong>Device:</strong> {{.deviceName}}<br/>💻 <strong>Type:</strong> {{.deviceType}}<br/>🕐 <strong>Removed:</strong> {{.timestamp}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 16},
	}, doc.Root)

	// Security text
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "If you didn't remove this device, please <a href=\"{{.secureAccountURL}}\" style=\"color: #7C3AED;\">secure your account</a>.",
		},
	}, doc.Root)

	return doc
}

// createAllSessionsRevokedTemplate creates all sessions revoked notification template.
func createAllSessionsRevokedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF3C7", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Orange/warning header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔐",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "All Sessions Signed Out",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>All active sessions on your account have been signed out for security.",
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#92400E",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "🔒 You'll need to sign in again on all your devices. This helps keep your account secure.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	// Sign in button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Sign In Again",
			"url":          "{{.loginURL}}",
			"buttonColor":  "#F59E0B",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	return doc
}

// =============================================================================
// REMINDER TEMPLATES (Amber Theme: #F59E0B)
// =============================================================================

// createVerificationReminderTemplate creates verification reminder template.
func createVerificationReminderTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "📧",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Verify Your Email",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>You haven't verified your email address yet. Please verify to access all features.",
		},
	}, doc.Root)

	// Verify button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Verify Email Now",
			"url":          "{{.verifyURL}}",
			"buttonColor":  "#F59E0B",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Reminder box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	reminderID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#92400E",
		},
		"props": map[string]any{
			"text": "⏰ Some features may be limited until you verify your email address.",
		},
	}, reminderID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createInactiveAccountTemplate creates inactive account reminder template.
func createInactiveAccountTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#D97706",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#D97706",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "💤",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "We Miss You!",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We noticed you haven't been active lately. We'd love to have you back!",
		},
	}, doc.Root)

	// Return button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Return to Dashboard",
			"url":          "{{.loginURL}}",
			"buttonColor":  "#D97706",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Info text
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  13,
			"color":     "#9CA3AF",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Don't want these reminders? You can update your preferences anytime.",
		},
	}, doc.Root)

	return doc
}

// createTrialExpiringTemplate creates trial expiring reminder template.
func createTrialExpiringTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "⏳",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Your Trial is Ending Soon",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your trial of <strong>{{.planName}}</strong> will expire in <strong>{{.daysRemaining}} days</strong> on {{.expiryDate}}.",
		},
	}, doc.Root)

	// Upgrade button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Upgrade Now",
			"url":          "{{.renewURL}}",
			"buttonColor":  "#F59E0B",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Trial info
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#92400E",
			"lineHeight": "1.6",
		},
		"props": map[string]any{
			"text": "⏰ Upgrade now to keep all your data and continue using premium features.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createSubscriptionExpiringTemplate creates subscription expiring reminder template.
func createSubscriptionExpiringTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#D97706",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#D97706",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "💳",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Subscription Expiring",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your <strong>{{.planName}}</strong> subscription will expire on <strong>{{.expiryDate}}</strong> ({{.daysRemaining}} days remaining).",
		},
	}, doc.Root)

	// Renew button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Renew Subscription",
			"url":          "{{.renewURL}}",
			"buttonColor":  "#D97706",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#92400E",
		},
		"props": map[string]any{
			"text": "💡 Renew now to avoid any interruption to your service.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createPasswordExpiringTemplate creates password expiring reminder template.
func createPasswordExpiringTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔑",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Time to Update Your Password",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your password will expire in <strong>{{.daysRemaining}} days</strong>. For your security, please update it soon.",
		},
	}, doc.Root)

	// Change password button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Change Password",
			"url":          "{{.changePasswordURL}}",
			"buttonColor":  "#F59E0B",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Security tip
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	tipID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#92400E",
		},
		"props": map[string]any{
			"text": "🔒 <strong>Security Tip:</strong> Use a strong, unique password and consider using a password manager.",
		},
	}, tipID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// =============================================================================
// ADMIN/MODERATION TEMPLATES (Red Theme: #EF4444)
// =============================================================================

// createAccountLockedTemplate creates account locked notification template.
func createAccountLockedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DC2626",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#DC2626",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔒",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Account Locked",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Your account has been locked by an administrator.",
		},
	}, doc.Root)

	// Lock details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "📋 <strong>Reason:</strong> {{.lockReason}}<br/>⏰ <strong>Locked Until:</strong> {{.unlockTime}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 16},
	}, doc.Root)

	// Contact support
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "If you believe this is a mistake, please contact our support team.",
		},
	}, doc.Root)

	return doc
}

// createAccountUnlockedTemplate creates account unlocked notification template.
func createAccountUnlockedTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#ECFDF5", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Green header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#10B981",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#10B981",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔓",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Account Unlocked",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>Good news! Your account has been unlocked and you can now sign in again.",
		},
	}, doc.Root)

	// Sign in button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 32, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Sign In Now",
			"url":          "{{.loginURL}}",
			"buttonColor":  "#10B981",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	return doc
}

// createTermsUpdateTemplate creates terms of service update notification template.
func createTermsUpdateTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#3B82F6",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#3B82F6",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "📄",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Terms of Service Updated",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We've updated our Terms of Service. Please review the changes at your convenience.",
		},
	}, doc.Root)

	// Review button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Review Terms",
			"url":          "{{.termsURL}}",
			"buttonColor":  "#3B82F6",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#EFF6FF",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#EFF6FF",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#1E40AF",
		},
		"props": map[string]any{
			"text": "📅 The updated terms will take effect on {{.effectiveDate}}. Continued use of our service means you accept these changes.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createPrivacyUpdateTemplate creates privacy policy update notification template.
func createPrivacyUpdateTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#F0F9FF", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Blue header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#0EA5E9",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#0EA5E9",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🔒",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Privacy Policy Updated",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We've updated our Privacy Policy to better explain how we protect your data.",
		},
	}, doc.Root)

	// Review button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Review Privacy Policy",
			"url":          "{{.privacyURL}}",
			"buttonColor":  "#0EA5E9",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Info box
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#DBEAFE",
			"padding": map[string]any{
				"top": 16, "right": 20, "bottom": 16, "left": 20,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#DBEAFE",
		},
		"childrenIds": []string{},
	}, doc.Root)

	infoID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize": 13,
			"color":    "#1E40AF",
		},
		"props": map[string]any{
			"text": "🔐 Your privacy is important to us. We've made these changes to be more transparent about how we handle your information.",
		},
	}, infoID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	return doc
}

// createMaintenanceScheduledTemplate creates scheduled maintenance notification template.
func createMaintenanceScheduledTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FFFBEB", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Amber header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#F59E0B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#F59E0B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🛠️",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "Scheduled Maintenance",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We'll be performing scheduled maintenance to improve our services.",
		},
	}, doc.Root)

	// Maintenance details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEF3C7",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEF3C7",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#92400E",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "🕐 <strong>Start:</strong> {{.maintenanceStart}}<br/>🕐 <strong>End:</strong> {{.maintenanceEnd}}<br/>⚠️ <strong>Impact:</strong> {{.actionRequired}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 16},
	}, doc.Root)

	// Info text
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  14,
			"color":     "#6B7280",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "We apologize for any inconvenience. Thank you for your patience!",
		},
	}, doc.Root)

	return doc
}

// createSecurityBreachTemplate creates security breach notification template.
func createSecurityBreachTemplate() *Document {
	doc := NewDocument()
	setRootStyle(doc, "#FEF2F2", "#FFFFFF", "#1F2937", "MODERN_SANS")

	// Red/critical header
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#991B1B",
			"padding": map[string]any{
				"top": 32, "right": 24, "bottom": 32, "left": 24,
			},
		},
		"props": map[string]any{
			"backgroundColor": "#991B1B",
		},
		"childrenIds": []string{},
	}, doc.Root)

	headerID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"fontSize":  40,
		},
		"props": map[string]any{
			"text": "🚨",
		},
	}, headerID)

	mustAddBlock(doc, BlockTypeHeading, map[string]any{
		"style": map[string]any{
			"textAlign": "center",
			"color":     "#FFFFFF",
			"padding": map[string]any{
				"top": 8, "right": 0, "bottom": 0, "left": 0,
			},
		},
		"props": map[string]any{
			"text":  "URGENT: Security Notice",
			"level": "h2",
		},
	}, headerID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 32},
	}, doc.Root)

	// Message
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"color":      "#374151",
			"fontSize":   16,
			"lineHeight": "1.7",
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "Hi <strong>{{.userName}}</strong>,<br/><br/>We're writing to inform you about a security incident that may have affected your account.",
		},
	}, doc.Root)

	// Breach details
	mustAddBlock(doc, BlockTypeContainer, map[string]any{
		"style": map[string]any{
			"backgroundColor": "#FEE2E2",
			"padding": map[string]any{
				"top": 20, "right": 24, "bottom": 20, "left": 24,
			},
			"borderRadius": 8,
		},
		"props": map[string]any{
			"backgroundColor": "#FEE2E2",
		},
		"childrenIds": []string{},
	}, doc.Root)

	detailsID := getLastBlockID(doc)

	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"fontSize":   14,
			"color":      "#991B1B",
			"lineHeight": "1.8",
		},
		"props": map[string]any{
			"text": "⚠️ <strong>What Happened:</strong><br/>{{.breachDetails}}<br/><br/>🔐 <strong>Action Required:</strong><br/>{{.actionRequired}}",
		},
	}, detailsID)

	// Spacer
	mustAddBlock(doc, BlockTypeSpacer, map[string]any{
		"style": map[string]any{},
		"props": map[string]any{"height": 24},
	}, doc.Root)

	// Secure account button
	mustAddBlock(doc, BlockTypeButton, map[string]any{
		"style": map[string]any{
			"padding": map[string]any{
				"top": 0, "right": 32, "bottom": 16, "left": 32,
			},
		},
		"props": map[string]any{
			"text":         "Secure My Account",
			"url":          "{{.secureAccountURL}}",
			"buttonColor":  "#DC2626",
			"textColor":    "#FFFFFF",
			"borderRadius": 8,
			"fullWidth":    true,
		},
	}, doc.Root)

	// Urgent notice
	mustAddBlock(doc, BlockTypeText, map[string]any{
		"style": map[string]any{
			"textAlign":  "center",
			"fontSize":   14,
			"color":      "#DC2626",
			"fontWeight": "bold",
			"padding": map[string]any{
				"top": 8, "right": 32, "bottom": 24, "left": 32,
			},
		},
		"props": map[string]any{
			"text": "⚠️ Please take action immediately to protect your account. We sincerely apologize for this incident.",
		},
	}, doc.Root)

	return doc
}
