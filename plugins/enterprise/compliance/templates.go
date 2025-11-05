package compliance

// ComplianceTemplates provides predefined compliance templates
var ComplianceTemplates = map[ComplianceStandard]ComplianceTemplate{
	StandardSOC2: {
		Standard:    StandardSOC2,
		Name:        "SOC 2 Type II",
		Description: "Service Organization Control 2 - Trust Services Criteria",

		// Security requirements
		MFARequired:       true,
		PasswordMinLength: 12,
		SessionMaxAge:     86400, // 24 hours

		// Audit requirements
		RetentionDays:      90, // Minimum 90 days
		DataResidency:      "", // Not specified
		AuditFrequencyDays: 90, // Quarterly reviews

		RequiredPolicies: []string{
			"access_control",
			"password_policy",
			"data_classification",
			"incident_response",
			"change_management",
			"vendor_management",
			"backup_recovery",
		},

		RequiredTraining: []string{
			"security_awareness",
			"data_handling",
			"incident_reporting",
		},
	},

	StandardHIPAA: {
		Standard:    StandardHIPAA,
		Name:        "HIPAA (Health Insurance Portability and Accountability Act)",
		Description: "Healthcare data protection and privacy requirements",

		// Security requirements (stricter)
		MFARequired:       true,
		PasswordMinLength: 14,
		SessionMaxAge:     3600, // 1 hour max

		// Audit requirements (7 years retention)
		RetentionDays:      2555, // 7 years
		DataResidency:      "US", // Must be US
		AuditFrequencyDays: 30,   // Monthly reviews

		RequiredPolicies: []string{
			"access_control",
			"password_policy",
			"data_encryption",
			"audit_controls",
			"breach_notification",
			"business_associate_agreement",
			"minimum_necessary",
			"emergency_access",
			"data_integrity",
			"transmission_security",
		},

		RequiredTraining: []string{
			"hipaa_basics",
			"phi_handling",
			"privacy_practices",
			"security_awareness",
			"breach_prevention",
		},
	},

	StandardPCIDSS: {
		Standard:    StandardPCIDSS,
		Name:        "PCI-DSS (Payment Card Industry Data Security Standard)",
		Description: "Payment card data security requirements",

		// Security requirements (very strict)
		MFARequired:       true,
		PasswordMinLength: 15,
		SessionMaxAge:     900, // 15 minutes

		// Audit requirements
		RetentionDays:      365, // 1 year minimum
		DataResidency:      "",  // Not specified
		AuditFrequencyDays: 90,  // Quarterly

		RequiredPolicies: []string{
			"firewall_configuration",
			"password_policy",
			"cardholder_data_protection",
			"encryption_transmission",
			"antivirus",
			"secure_systems",
			"access_control",
			"unique_ids",
			"physical_access",
			"network_monitoring",
			"security_testing",
			"information_security_policy",
		},

		RequiredTraining: []string{
			"pci_awareness",
			"cardholder_data_handling",
			"security_best_practices",
			"incident_response",
		},
	},

	StandardGDPR: {
		Standard:    StandardGDPR,
		Name:        "GDPR (General Data Protection Regulation)",
		Description: "EU data protection and privacy regulation",

		// Security requirements
		MFARequired:       true,
		PasswordMinLength: 12,
		SessionMaxAge:     86400, // 24 hours

		// Audit requirements
		RetentionDays:      90,   // Vary by data type
		DataResidency:      "EU", // EU for EU citizens
		AuditFrequencyDays: 90,

		RequiredPolicies: []string{
			"privacy_policy",
			"data_processing_agreement",
			"consent_management",
			"data_breach_notification",
			"right_to_access",
			"right_to_erasure",
			"right_to_portability",
			"data_protection_impact_assessment",
			"data_retention",
			"vendor_management",
		},

		RequiredTraining: []string{
			"gdpr_fundamentals",
			"data_subject_rights",
			"privacy_by_design",
			"breach_notification",
			"lawful_basis",
		},
	},

	StandardISO27001: {
		Standard:    StandardISO27001,
		Name:        "ISO/IEC 27001",
		Description: "Information Security Management System standard",

		// Security requirements
		MFARequired:       true,
		PasswordMinLength: 12,
		SessionMaxAge:     86400, // 24 hours

		// Audit requirements
		RetentionDays:      180, // 6 months minimum
		DataResidency:      "",  // Not specified
		AuditFrequencyDays: 180, // Biannual

		RequiredPolicies: []string{
			"information_security_policy",
			"access_control",
			"asset_management",
			"cryptography",
			"physical_security",
			"operations_security",
			"communications_security",
			"acquisition_development",
			"supplier_relationships",
			"incident_management",
			"business_continuity",
			"compliance",
		},

		RequiredTraining: []string{
			"security_awareness",
			"isms_overview",
			"risk_management",
			"incident_handling",
		},
	},

	StandardCCPA: {
		Standard:    StandardCCPA,
		Name:        "CCPA (California Consumer Privacy Act)",
		Description: "California privacy rights and consumer protection",

		// Security requirements
		MFARequired:       false, // Recommended but not required
		PasswordMinLength: 12,
		SessionMaxAge:     86400, // 24 hours

		// Audit requirements
		RetentionDays:      365,  // 12 months
		DataResidency:      "US", // US preferred
		AuditFrequencyDays: 90,

		RequiredPolicies: []string{
			"privacy_notice",
			"consumer_rights",
			"data_collection_notice",
			"opt_out_rights",
			"data_deletion",
			"data_disclosure",
			"non_discrimination",
			"authorized_agent",
		},

		RequiredTraining: []string{
			"ccpa_overview",
			"consumer_requests",
			"data_inventory",
			"privacy_rights",
		},
	},
}

// GetTemplate returns a compliance template for a standard
func GetTemplate(standard ComplianceStandard) (ComplianceTemplate, bool) {
	template, ok := ComplianceTemplates[standard]
	return template, ok
}

// GetTemplateNames returns all available template names
func GetTemplateNames() []string {
	names := make([]string, 0, len(ComplianceTemplates))
	for standard := range ComplianceTemplates {
		names = append(names, string(standard))
	}
	return names
}

// CreateProfileFromTemplate creates a compliance profile from a template
func CreateProfileFromTemplate(orgID string, standard ComplianceStandard) (*ComplianceProfile, error) {
	template, ok := GetTemplate(standard)
	if !ok {
		return nil, ErrTemplateNotFound
	}

	profile := &ComplianceProfile{
		OrganizationID: orgID,
		Name:           template.Name,
		Standards:      []ComplianceStandard{standard},
		Status:         "active",

		// Security Requirements from template
		MFARequired:           template.MFARequired,
		PasswordMinLength:     template.PasswordMinLength,
		PasswordRequireUpper:  true,
		PasswordRequireLower:  true,
		PasswordRequireNumber: true,
		PasswordRequireSymbol: true,
		PasswordExpiryDays:    90, // Default 90 days

		// Session Requirements from template
		SessionMaxAge:      template.SessionMaxAge,
		SessionIdleTimeout: template.SessionMaxAge / 2,
		SessionIPBinding:   standard == StandardPCIDSS, // Only for PCI-DSS by default

		// Audit Requirements from template
		RetentionDays:      template.RetentionDays,
		AuditLogExport:     true,
		DetailedAuditTrail: true,

		// Data Requirements from template
		DataResidency:       template.DataResidency,
		EncryptionAtRest:    true,
		EncryptionInTransit: true,

		// Access Control
		RBACRequired:        true,
		LeastPrivilege:      true,
		RegularAccessReview: true,

		Metadata: map[string]interface{}{
			"created_from_template": string(standard),
			"template_version":      "1.0",
			"required_policies":     template.RequiredPolicies,
			"required_training":     template.RequiredTraining,
		},
	}

	return profile, nil
}
