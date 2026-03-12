package dashui

import "time"

// SCIMConfigView is a display-only view of a SCIM configuration.
type SCIMConfigView struct {
	ID           string
	Name         string
	Enabled      bool
	AutoCreate   bool
	AutoSuspend  bool
	GroupSync    bool
	DefaultRole  string
	OrgID        string
	OrgName      string // resolved org name
	AppID        string
	TokenCount   int
	LastActivity string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SCIMTokenView is a display-only view of a SCIM bearer token.
type SCIMTokenView struct {
	ID         string
	Name       string
	LastUsedAt string
	ExpiresAt  string
	CreatedAt  time.Time
}

// SCIMLogView is a display-only view of a SCIM provision log entry.
type SCIMLogView struct {
	ID           string
	ConfigID     string
	ConfigName   string
	Action       string
	ResourceType string
	ExternalID   string
	InternalID   string
	Status       string
	Detail       string
	CreatedAt    time.Time
}

// ──────────────────────────────────────────────────
// Page data types
// ──────────────────────────────────────────────────

// SCIMListPageData holds data for the SCIM configurations list page.
type SCIMListPageData struct {
	Configs          []SCIMConfigView
	TotalConfigs     int
	ActiveTokens     int
	ProvisionedUsers int
	RecentActivity   int
	Error            string
	Success          string
	FormNonce        string
}

// SCIMDetailPageData holds data for the SCIM config detail page.
type SCIMDetailPageData struct {
	Config           SCIMConfigView
	Tokens           []SCIMTokenView
	RecentLogs       []SCIMLogView
	BaseURL          string
	UsersEndpoint    string
	GroupsEndpoint   string
	ProvisionedUsers int
	SuccessCount     int
	ErrorCount       int
	SkippedCount     int
	NewToken         string // shown once after creation
	ActiveTab        string
	Error            string
	Success          string
	FormNonce        string
}

// SCIMLogsPageData holds data for the SCIM provisioning logs page.
type SCIMLogsPageData struct {
	Logs         []SCIMLogView
	Configs      []SCIMConfigView // for filter dropdown
	SuccessCount int
	ErrorCount   int
	SkippedCount int
	TotalCount   int
	Error        string
}

// OverviewWidgetData holds data for the SCIM dashboard overview widget.
type OverviewWidgetData struct {
	ConfigCount int
	TokenCount  int
	RecentCount int
}

// SettingsPanelData holds data for the SCIM settings panel.
type SettingsPanelData struct {
	Enabled         bool
	AutoCreate      bool
	AutoSuspend     bool
	GroupSync       bool
	DefaultRole     string
	TokenExpiryDays int
}

// OrgTabData holds data for the org detail SCIM tab.
type OrgTabData struct {
	Configs          []SCIMConfigView
	ProvisionedUsers int
	RecentCount      int
	RecentLogs       []SCIMLogView
	OrgID            string
	FormNonce        string
}

// OrgSectionData holds compact SCIM data for the org detail overview section.
type OrgSectionData struct {
	HasConfig   bool
	ConfigCount int
	Enabled     bool
	LastSync    string
}
