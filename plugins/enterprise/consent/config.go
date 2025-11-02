package consent

import "time"

// Config holds the consent plugin configuration
type Config struct {
	// Enable consent plugin
	Enabled bool `json:"enabled" yaml:"enabled"`

	// GDPR compliance mode
	GDPREnabled bool `json:"gdprEnabled" yaml:"gdprEnabled"`

	// CCPA compliance mode
	CCPAEnabled bool `json:"ccpaEnabled" yaml:"ccpaEnabled"`

	// Cookie consent configuration
	CookieConsent CookieConsentConfig `json:"cookieConsent" yaml:"cookieConsent"`

	// Data export configuration
	DataExport DataExportConfig `json:"dataExport" yaml:"dataExport"`

	// Data deletion configuration
	DataDeletion DataDeletionConfig `json:"dataDeletion" yaml:"dataDeletion"`

	// Consent audit configuration
	Audit ConsentAuditConfig `json:"audit" yaml:"audit"`

	// Consent expiry configuration
	Expiry ConsentExpiryConfig `json:"expiry" yaml:"expiry"`

	// Dashboard configuration
	Dashboard ConsentDashboardConfig `json:"dashboard" yaml:"dashboard"`

	// Notifications
	Notifications ConsentNotificationsConfig `json:"notifications" yaml:"notifications"`
}

// CookieConsentConfig configures cookie consent management
type CookieConsentConfig struct {
	Enabled             bool          `json:"enabled" yaml:"enabled"`
	DefaultStyle        string        `json:"defaultStyle" yaml:"defaultStyle"` // banner, modal, popup
	RequireExplicit     bool          `json:"requireExplicit" yaml:"requireExplicit"` // No implied consent
	ValidityPeriod      time.Duration `json:"validityPeriod" yaml:"validityPeriod"` // How long consent is valid
	AllowAnonymous      bool          `json:"allowAnonymous" yaml:"allowAnonymous"` // Track consent for non-authenticated users
	BannerVersion       string        `json:"bannerVersion" yaml:"bannerVersion"` // Current banner version
	Categories          []string      `json:"categories" yaml:"categories"` // essential, functional, analytics, marketing, personalization, third_party
}

// DataExportConfig configures data portability features
type DataExportConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	AllowedFormats    []string      `json:"allowedFormats" yaml:"allowedFormats"` // json, csv, xml, pdf
	DefaultFormat     string        `json:"defaultFormat" yaml:"defaultFormat"`
	MaxRequests       int           `json:"maxRequests" yaml:"maxRequests"` // Max requests per user per period
	RequestPeriod     time.Duration `json:"requestPeriod" yaml:"requestPeriod"` // Period for max requests (e.g., 30 days)
	ExpiryHours       int           `json:"expiryHours" yaml:"expiryHours"` // How long export URL is valid
	StoragePath       string        `json:"storagePath" yaml:"storagePath"` // Where to store exports
	IncludeSections   []string      `json:"includeSections" yaml:"includeSections"` // Default sections: profile, sessions, consents, audit
	AutoCleanup       bool          `json:"autoCleanup" yaml:"autoCleanup"` // Auto-delete expired exports
	CleanupInterval   time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
	MaxExportSize     int64         `json:"maxExportSize" yaml:"maxExportSize"` // Max export size in bytes
}

// DataDeletionConfig configures right to be forgotten
type DataDeletionConfig struct {
	Enabled                 bool          `json:"enabled" yaml:"enabled"`
	RequireAdminApproval    bool          `json:"requireAdminApproval" yaml:"requireAdminApproval"`
	GracePeriodDays         int           `json:"gracePeriodDays" yaml:"gracePeriodDays"` // Days before actual deletion
	ArchiveBeforeDeletion   bool          `json:"archiveBeforeDeletion" yaml:"archiveBeforeDeletion"`
	ArchivePath             string        `json:"archivePath" yaml:"archivePath"`
	RetentionExemptions     []string      `json:"retentionExemptions" yaml:"retentionExemptions"` // Reasons to exempt from deletion
	NotifyBeforeDeletion    bool          `json:"notifyBeforeDeletion" yaml:"notifyBeforeDeletion"`
	AllowPartialDeletion    bool          `json:"allowPartialDeletion" yaml:"allowPartialDeletion"` // Allow deleting specific sections
	PreserveLegalData       bool          `json:"preserveLegalData" yaml:"preserveLegalData"` // Keep data required by law
	AutoProcessAfterGrace   bool          `json:"autoProcessAfterGrace" yaml:"autoProcessAfterGrace"` // Auto-process after grace period
}

// ConsentAuditConfig configures consent audit trail
type ConsentAuditConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	RetentionDays    int           `json:"retentionDays" yaml:"retentionDays"` // How long to keep audit logs
	Immutable        bool          `json:"immutable" yaml:"immutable"` // Prevent audit log modification
	LogAllChanges    bool          `json:"logAllChanges" yaml:"logAllChanges"` // Log all consent changes
	LogIPAddress     bool          `json:"logIpAddress" yaml:"logIpAddress"`
	LogUserAgent     bool          `json:"logUserAgent" yaml:"logUserAgent"`
	SignLogs         bool          `json:"signLogs" yaml:"signLogs"` // Cryptographic signing
	ExportFormat     string        `json:"exportFormat" yaml:"exportFormat"` // json, csv
	ArchiveOldLogs   bool          `json:"archiveOldLogs" yaml:"archiveOldLogs"`
	ArchiveInterval  time.Duration `json:"archiveInterval" yaml:"archiveInterval"`
}

// ConsentExpiryConfig configures consent expiry management
type ConsentExpiryConfig struct {
	Enabled               bool          `json:"enabled" yaml:"enabled"`
	DefaultValidityDays   int           `json:"defaultValidityDays" yaml:"defaultValidityDays"` // Default consent validity
	RenewalReminderDays   int           `json:"renewalReminderDays" yaml:"renewalReminderDays"` // Days before expiry to remind
	AutoExpireCheck       bool          `json:"autoExpireCheck" yaml:"autoExpireCheck"` // Automatically check and expire
	ExpireCheckInterval   time.Duration `json:"expireCheckInterval" yaml:"expireCheckInterval"`
	AllowRenewal          bool          `json:"allowRenewal" yaml:"allowRenewal"`
	RequireReConsent      bool          `json:"requireReConsent" yaml:"requireReConsent"` // Require explicit re-consent
}

// ConsentDashboardConfig configures the consent dashboard
type ConsentDashboardConfig struct {
	Enabled              bool   `json:"enabled" yaml:"enabled"`
	Path                 string `json:"path" yaml:"path"` // e.g., /auth/consent
	ShowConsentHistory   bool   `json:"showConsentHistory" yaml:"showConsentHistory"`
	ShowCookiePreferences bool  `json:"showCookiePreferences" yaml:"showCookiePreferences"`
	ShowDataExport       bool   `json:"showDataExport" yaml:"showDataExport"`
	ShowDataDeletion     bool   `json:"showDataDeletion" yaml:"showDataDeletion"`
	ShowAuditLog         bool   `json:"showAuditLog" yaml:"showAuditLog"`
	ShowPolicies         bool   `json:"showPolicies" yaml:"showPolicies"`
}

// ConsentNotificationsConfig configures consent notifications
type ConsentNotificationsConfig struct {
	Enabled                bool     `json:"enabled" yaml:"enabled"`
	NotifyOnGrant          bool     `json:"notifyOnGrant" yaml:"notifyOnGrant"`
	NotifyOnRevoke         bool     `json:"notifyOnRevoke" yaml:"notifyOnRevoke"`
	NotifyOnExpiry         bool     `json:"notifyOnExpiry" yaml:"notifyOnExpiry"`
	NotifyExportReady      bool     `json:"notifyExportReady" yaml:"notifyExportReady"`
	NotifyDeletionApproved bool     `json:"notifyDeletionApproved" yaml:"notifyDeletionApproved"`
	NotifyDeletionComplete bool     `json:"notifyDeletionComplete" yaml:"notifyDeletionComplete"`
	NotifyDPOEmail         string   `json:"notifyDpoEmail" yaml:"notifyDpoEmail"` // Data Protection Officer email
	Channels               []string `json:"channels" yaml:"channels"` // email, sms, webhook
}

// DefaultConfig returns the default consent configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		GDPREnabled: true,
		CCPAEnabled: false,
		CookieConsent: CookieConsentConfig{
			Enabled:         true,
			DefaultStyle:    "banner",
			RequireExplicit: true,
			ValidityPeriod:  365 * 24 * time.Hour, // 1 year
			AllowAnonymous:  true,
			BannerVersion:   "1.0",
			Categories:      []string{"essential", "functional", "analytics", "marketing", "personalization", "third_party"},
		},
		DataExport: DataExportConfig{
			Enabled:         true,
			AllowedFormats:  []string{"json", "csv"},
			DefaultFormat:   "json",
			MaxRequests:     5,
			RequestPeriod:   30 * 24 * time.Hour, // 30 days
			ExpiryHours:     72, // 3 days
			StoragePath:     "/var/lib/authsome/consent/exports",
			IncludeSections: []string{"profile", "sessions", "consents", "audit"},
			AutoCleanup:     true,
			CleanupInterval: 24 * time.Hour,
			MaxExportSize:   100 * 1024 * 1024, // 100MB
		},
		DataDeletion: DataDeletionConfig{
			Enabled:               true,
			RequireAdminApproval:  true,
			GracePeriodDays:       30, // GDPR allows up to 30 days
			ArchiveBeforeDeletion: true,
			ArchivePath:           "/var/lib/authsome/consent/archives",
			RetentionExemptions:   []string{"legal_hold", "active_investigation", "contractual_obligation"},
			NotifyBeforeDeletion:  true,
			AllowPartialDeletion:  true,
			PreserveLegalData:     true,
			AutoProcessAfterGrace: false, // Require manual processing
		},
		Audit: ConsentAuditConfig{
			Enabled:         true,
			RetentionDays:   2555, // 7 years (common legal requirement)
			Immutable:       true,
			LogAllChanges:   true,
			LogIPAddress:    true,
			LogUserAgent:    true,
			SignLogs:        true,
			ExportFormat:    "json",
			ArchiveOldLogs:  true,
			ArchiveInterval: 90 * 24 * time.Hour, // 90 days
		},
		Expiry: ConsentExpiryConfig{
			Enabled:             true,
			DefaultValidityDays: 365, // 1 year
			RenewalReminderDays: 30,
			AutoExpireCheck:     true,
			ExpireCheckInterval: 24 * time.Hour,
			AllowRenewal:        true,
			RequireReConsent:    false, // Auto-renew if re-consent not required
		},
		Dashboard: ConsentDashboardConfig{
			Enabled:               true,
			Path:                  "/auth/consent",
			ShowConsentHistory:    true,
			ShowCookiePreferences: true,
			ShowDataExport:        true,
			ShowDataDeletion:      true,
			ShowAuditLog:          true,
			ShowPolicies:          true,
		},
		Notifications: ConsentNotificationsConfig{
			Enabled:                true,
			NotifyOnGrant:          false,
			NotifyOnRevoke:         true,
			NotifyOnExpiry:         true,
			NotifyExportReady:      true,
			NotifyDeletionApproved: true,
			NotifyDeletionComplete: true,
			NotifyDPOEmail:         "",
			Channels:               []string{"email"},
		},
	}
}

// Validate ensures the configuration has sensible defaults
func (c *Config) Validate() {
	// Cookie consent validation
	if c.CookieConsent.ValidityPeriod == 0 {
		c.CookieConsent.ValidityPeriod = 365 * 24 * time.Hour
	}
	if c.CookieConsent.DefaultStyle == "" {
		c.CookieConsent.DefaultStyle = "banner"
	}

	// Data export validation
	if len(c.DataExport.AllowedFormats) == 0 {
		c.DataExport.AllowedFormats = []string{"json", "csv"}
	}
	if c.DataExport.DefaultFormat == "" {
		c.DataExport.DefaultFormat = "json"
	}
	if c.DataExport.ExpiryHours == 0 {
		c.DataExport.ExpiryHours = 72
	}
	if c.DataExport.MaxExportSize == 0 {
		c.DataExport.MaxExportSize = 100 * 1024 * 1024 // 100MB
	}

	// Data deletion validation
	if c.DataDeletion.GracePeriodDays == 0 {
		c.DataDeletion.GracePeriodDays = 30
	}

	// Audit validation
	if c.Audit.RetentionDays == 0 {
		c.Audit.RetentionDays = 2555 // 7 years
	}

	// Expiry validation
	if c.Expiry.DefaultValidityDays == 0 {
		c.Expiry.DefaultValidityDays = 365
	}
	if c.Expiry.RenewalReminderDays == 0 {
		c.Expiry.RenewalReminderDays = 30
	}

	// Dashboard validation
	if c.Dashboard.Path == "" {
		c.Dashboard.Path = "/auth/consent"
	}
}

