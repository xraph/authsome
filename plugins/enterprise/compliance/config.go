package compliance

import "time"

// Config holds the compliance plugin configuration
type Config struct {
	// Enable compliance plugin
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Default compliance standard for new organizations
	DefaultStandard ComplianceStandard `json:"defaultStandard" yaml:"defaultStandard"`
	
	// Automated checks configuration
	AutomatedChecks AutomatedChecksConfig `json:"automatedChecks" yaml:"automatedChecks"`
	
	// Audit configuration
	Audit AuditConfig `json:"audit" yaml:"audit"`
	
	// Report configuration
	Reports ReportsConfig `json:"reports" yaml:"reports"`
	
	// Retention configuration
	Retention RetentionConfig `json:"retention" yaml:"retention"`
	
	// Notifications
	Notifications NotificationsConfig `json:"notifications" yaml:"notifications"`
	
	// Dashboard configuration
	Dashboard DashboardConfig `json:"dashboard" yaml:"dashboard"`
}

// AutomatedChecksConfig configures automated compliance checks
type AutomatedChecksConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	CheckInterval time.Duration `json:"checkInterval" yaml:"checkInterval"` // e.g., 24h
	
	// Specific checks
	MFACoverage          bool `json:"mfaCoverage" yaml:"mfaCoverage"`
	PasswordPolicy       bool `json:"passwordPolicy" yaml:"passwordPolicy"`
	SessionPolicy        bool `json:"sessionPolicy" yaml:"sessionPolicy"`
	AccessReview         bool `json:"accessReview" yaml:"accessReview"`
	InactiveUsers        bool `json:"inactiveUsers" yaml:"inactiveUsers"`
	SuspiciousActivity   bool `json:"suspiciousActivity" yaml:"suspiciousActivity"`
	DataRetention        bool `json:"dataRetention" yaml:"dataRetention"`
}

// AuditConfig configures audit trail settings
type AuditConfig struct {
	// Minimum retention days (enforced for all orgs)
	MinRetentionDays int `json:"minRetentionDays" yaml:"minRetentionDays"`
	
	// Maximum retention days
	MaxRetentionDays int `json:"maxRetentionDays" yaml:"maxRetentionDays"`
	
	// Detailed audit trail (log all field changes)
	DetailedTrail bool `json:"detailedTrail" yaml:"detailedTrail"`
	
	// Immutable audit logs (cannot be deleted/modified)
	Immutable bool `json:"immutable" yaml:"immutable"`
	
	// Audit log export format
	ExportFormat string `json:"exportFormat" yaml:"exportFormat"` // json, csv, pdf
	
	// Enable audit log signing (for tamper detection)
	SignLogs bool `json:"signLogs" yaml:"signLogs"`
}

// ReportsConfig configures compliance reporting
type ReportsConfig struct {
	// Enable automated report generation
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Report generation schedule
	Schedule string `json:"schedule" yaml:"schedule"` // cron format
	
	// Report formats
	Formats []string `json:"formats" yaml:"formats"` // pdf, json, csv
	
	// Report storage location
	StoragePath string `json:"storagePath" yaml:"storagePath"`
	
	// Report retention days
	RetentionDays int `json:"retentionDays" yaml:"retentionDays"`
	
	// Include evidence in reports
	IncludeEvidence bool `json:"includeEvidence" yaml:"includeEvidence"`
}

// RetentionConfig configures data retention policies
type RetentionConfig struct {
	// Enable automated data retention
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Purge schedule (cron format)
	PurgeSchedule string `json:"purgeSchedule" yaml:"purgeSchedule"`
	
	// Grace period before purging (days)
	GracePeriodDays int `json:"gracePeriodDays" yaml:"gracePeriodDays"`
	
	// Archive before purging
	ArchiveBeforePurge bool `json:"archiveBeforePurge" yaml:"archiveBeforePurge"`
	
	// Archive location
	ArchivePath string `json:"archivePath" yaml:"archivePath"`
}

// NotificationsConfig configures compliance notifications
type NotificationsConfig struct {
	// Enable notifications
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Notify on violations
	Violations bool `json:"violations" yaml:"violations"`
	
	// Notify on failed checks
	FailedChecks bool `json:"failedChecks" yaml:"failedChecks"`
	
	// Notify before audit
	AuditReminders bool `json:"auditReminders" yaml:"auditReminders"`
	
	// Notify compliance contact
	NotifyComplianceContact bool `json:"notifyComplianceContact" yaml:"notifyComplianceContact"`
	
	// Notify organization owners
	NotifyOwners bool `json:"notifyOwners" yaml:"notifyOwners"`
	
	// Notification channels
	Channels NotificationChannels `json:"channels" yaml:"channels"`
}

// NotificationChannels defines notification delivery channels
type NotificationChannels struct {
	Email   bool `json:"email" yaml:"email"`
	Slack   bool `json:"slack" yaml:"slack"`
	Webhook bool `json:"webhook" yaml:"webhook"`
}

// DashboardConfig configures the compliance dashboard
type DashboardConfig struct {
	// Enable compliance dashboard
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Dashboard path
	Path string `json:"path" yaml:"path"` // e.g., /auth/compliance
	
	// Show overall compliance score
	ShowScore bool `json:"showScore" yaml:"showScore"`
	
	// Show violations
	ShowViolations bool `json:"showViolations" yaml:"showViolations"`
	
	// Show recent checks
	ShowRecentChecks bool `json:"showRecentChecks" yaml:"showRecentChecks"`
	
	// Show reports
	ShowReports bool `json:"showReports" yaml:"showReports"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		DefaultStandard: StandardSOC2,
		AutomatedChecks: AutomatedChecksConfig{
			Enabled:       true,
			CheckInterval: 24 * time.Hour,
			MFACoverage:          true,
			PasswordPolicy:       true,
			SessionPolicy:        true,
			AccessReview:         true,
			InactiveUsers:        true,
			SuspiciousActivity:   true,
			DataRetention:        true,
		},
		Audit: AuditConfig{
			MinRetentionDays: 90,  // SOC 2 minimum
			MaxRetentionDays: 2555, // HIPAA 7 years
			DetailedTrail:    true,
			Immutable:        true,
			ExportFormat:     "json",
			SignLogs:         true,
		},
		Reports: ReportsConfig{
			Enabled:         true,
			Schedule:        "0 0 1 * *", // Monthly on 1st
			Formats:         []string{"pdf", "json"},
			StoragePath:     "/var/lib/authsome/compliance/reports",
			RetentionDays:   365,
			IncludeEvidence: true,
		},
		Retention: RetentionConfig{
			Enabled:            true,
			PurgeSchedule:      "0 2 * * 0", // Weekly on Sunday at 2am
			GracePeriodDays:    30,
			ArchiveBeforePurge: true,
			ArchivePath:        "/var/lib/authsome/compliance/archive",
		},
		Notifications: NotificationsConfig{
			Enabled:                 true,
			Violations:              true,
			FailedChecks:            true,
			AuditReminders:          true,
			NotifyComplianceContact: true,
			NotifyOwners:            false,
			Channels: NotificationChannels{
				Email:   true,
				Slack:   false,
				Webhook: false,
			},
		},
		Dashboard: DashboardConfig{
			Enabled:          true,
			Path:             "/auth/compliance",
			ShowScore:        true,
			ShowViolations:   true,
			ShowRecentChecks: true,
			ShowReports:      true,
		},
	}
}

// Validate ensures the configuration has sensible defaults
func (c *Config) Validate() {
	// Set defaults if not configured
	if c.AutomatedChecks.CheckInterval == 0 {
		c.AutomatedChecks.CheckInterval = 24 * time.Hour
	}
	if c.Audit.MinRetentionDays == 0 {
		c.Audit.MinRetentionDays = 90
	}
	if c.Audit.MaxRetentionDays == 0 {
		c.Audit.MaxRetentionDays = 2555
	}
	if c.Dashboard.Path == "" {
		c.Dashboard.Path = "/auth/compliance"
	}
	if c.Reports.StoragePath == "" {
		c.Reports.StoragePath = "/var/lib/authsome/compliance/reports"
	}
	if c.Retention.ArchivePath == "" {
		c.Retention.ArchivePath = "/var/lib/authsome/compliance/archive"
	}
}
