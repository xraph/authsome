package cms

import "time"

// Config holds the CMS plugin configuration
type Config struct {
	// Features
	Features FeaturesConfig `json:"features" yaml:"features"`

	// Limits
	Limits LimitsConfig `json:"limits" yaml:"limits"`

	// Revisions
	Revisions RevisionsConfig `json:"revisions" yaml:"revisions"`

	// Search
	Search SearchConfig `json:"search" yaml:"search"`

	// API
	API APIConfig `json:"api" yaml:"api"`

	// Dashboard
	Dashboard DashboardConfig `json:"dashboard" yaml:"dashboard"`
}

// FeaturesConfig holds feature toggles
type FeaturesConfig struct {
	// EnableRevisions enables content versioning
	// Default: true
	EnableRevisions bool `json:"enableRevisions" yaml:"enableRevisions"`

	// EnableDrafts enables draft/publish workflow
	// Default: true
	EnableDrafts bool `json:"enableDrafts" yaml:"enableDrafts"`

	// EnableScheduling enables scheduled publishing
	// Default: true
	EnableScheduling bool `json:"enableScheduling" yaml:"enableScheduling"`

	// EnableSearch enables full-text search
	// Default: false (requires PostgreSQL full-text search setup)
	EnableSearch bool `json:"enableSearch" yaml:"enableSearch"`

	// EnableRelations enables relations between content types
	// Default: true
	EnableRelations bool `json:"enableRelations" yaml:"enableRelations"`

	// EnableLocalization enables content localization
	// Default: false
	EnableLocalization bool `json:"enableLocalization" yaml:"enableLocalization"`

	// EnableSoftDelete enables soft delete for entries
	// Default: true
	EnableSoftDelete bool `json:"enableSoftDelete" yaml:"enableSoftDelete"`
}

// LimitsConfig holds resource limits
type LimitsConfig struct {
	// MaxContentTypes is the maximum number of content types per app/environment
	// 0 means unlimited
	// Default: 100
	MaxContentTypes int `json:"maxContentTypes" yaml:"maxContentTypes"`

	// MaxFieldsPerType is the maximum number of fields per content type
	// Default: 50
	MaxFieldsPerType int `json:"maxFieldsPerType" yaml:"maxFieldsPerType"`

	// MaxEntriesPerType is the maximum number of entries per content type
	// 0 means unlimited (can be overridden per content type)
	// Default: 0
	MaxEntriesPerType int `json:"maxEntriesPerType" yaml:"maxEntriesPerType"`

	// MaxEntryDataSize is the maximum size of entry data in bytes
	// Default: 1MB
	MaxEntryDataSize int64 `json:"maxEntryDataSize" yaml:"maxEntryDataSize"`

	// MaxRelationsPerEntry is the maximum number of relations per entry
	// Default: 100
	MaxRelationsPerEntry int `json:"maxRelationsPerEntry" yaml:"maxRelationsPerEntry"`
}

// RevisionsConfig holds revision settings
type RevisionsConfig struct {
	// MaxRevisionsPerEntry is the maximum number of revisions to keep per entry
	// When exceeded, oldest revisions are automatically deleted
	// Default: 50
	MaxRevisionsPerEntry int `json:"maxRevisionsPerEntry" yaml:"maxRevisionsPerEntry"`

	// RetentionDays is how long to keep old revisions in days
	// Revisions older than this are eligible for cleanup
	// Default: 90
	RetentionDays int `json:"retentionDays" yaml:"retentionDays"`

	// AutoCleanup enables automatic cleanup of old revisions
	// Default: true
	AutoCleanup bool `json:"autoCleanup" yaml:"autoCleanup"`

	// CleanupInterval is how often to run revision cleanup
	// Default: 24 hours
	CleanupInterval time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
}

// SearchConfig holds search settings
type SearchConfig struct {
	// Language is the PostgreSQL text search configuration
	// Default: "english"
	Language string `json:"language" yaml:"language"`

	// MinSearchLength is the minimum query length for search
	// Default: 2
	MinSearchLength int `json:"minSearchLength" yaml:"minSearchLength"`

	// MaxSearchResults is the maximum number of search results
	// Default: 100
	MaxSearchResults int `json:"maxSearchResults" yaml:"maxSearchResults"`

	// EnableHighlighting enables search result highlighting
	// Default: true
	EnableHighlighting bool `json:"enableHighlighting" yaml:"enableHighlighting"`
}

// APIConfig holds API settings
type APIConfig struct {
	// EnablePublicAPI allows unauthenticated read access to published content
	// Default: false
	EnablePublicAPI bool `json:"enablePublicAPI" yaml:"enablePublicAPI"`

	// DefaultPageSize is the default page size for list endpoints
	// Default: 20
	DefaultPageSize int `json:"defaultPageSize" yaml:"defaultPageSize"`

	// MaxPageSize is the maximum page size for list endpoints
	// Default: 100
	MaxPageSize int `json:"maxPageSize" yaml:"maxPageSize"`

	// RateLimitPerMinute is the rate limit for API requests per minute
	// 0 means no rate limiting
	// Default: 0
	RateLimitPerMinute int `json:"rateLimitPerMinute" yaml:"rateLimitPerMinute"`

	// EnableGraphQL enables GraphQL API endpoint
	// Default: false
	EnableGraphQL bool `json:"enableGraphql" yaml:"enableGraphql"`
}

// DashboardConfig holds dashboard-specific settings
type DashboardConfig struct {
	// EnableFieldDragDrop enables drag and drop field reordering
	// Default: true
	EnableFieldDragDrop bool `json:"enableFieldDragDrop" yaml:"enableFieldDragDrop"`

	// EnableBulkOperations enables bulk operations on entries
	// Default: true
	EnableBulkOperations bool `json:"enableBulkOperations" yaml:"enableBulkOperations"`

	// EnableImportExport enables import/export functionality
	// Default: false
	EnableImportExport bool `json:"enableImportExport" yaml:"enableImportExport"`

	// EntriesPerPage is the default number of entries per page in the dashboard
	// Default: 25
	EntriesPerPage int `json:"entriesPerPage" yaml:"entriesPerPage"`

	// ShowRevisionHistory shows revision history in entry detail
	// Default: true
	ShowRevisionHistory bool `json:"showRevisionHistory" yaml:"showRevisionHistory"`

	// ShowRelatedEntries shows related entries in entry detail
	// Default: true
	ShowRelatedEntries bool `json:"showRelatedEntries" yaml:"showRelatedEntries"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Features: FeaturesConfig{
			EnableRevisions:    true,
			EnableDrafts:       true,
			EnableScheduling:   true,
			EnableSearch:       false,
			EnableRelations:    true,
			EnableLocalization: false,
			EnableSoftDelete:   true,
		},
		Limits: LimitsConfig{
			MaxContentTypes:      100,
			MaxFieldsPerType:     50,
			MaxEntriesPerType:    0, // Unlimited
			MaxEntryDataSize:     1 * 1024 * 1024, // 1MB
			MaxRelationsPerEntry: 100,
		},
		Revisions: RevisionsConfig{
			MaxRevisionsPerEntry: 50,
			RetentionDays:        90,
			AutoCleanup:          true,
			CleanupInterval:      24 * time.Hour,
		},
		Search: SearchConfig{
			Language:           "english",
			MinSearchLength:    2,
			MaxSearchResults:   100,
			EnableHighlighting: true,
		},
		API: APIConfig{
			EnablePublicAPI:    false,
			DefaultPageSize:    20,
			MaxPageSize:        100,
			RateLimitPerMinute: 0,
			EnableGraphQL:      false,
		},
		Dashboard: DashboardConfig{
			EnableFieldDragDrop:  true,
			EnableBulkOperations: true,
			EnableImportExport:   false,
			EntriesPerPage:       25,
			ShowRevisionHistory:  true,
			ShowRelatedEntries:   true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Set minimum values
	if c.Limits.MaxFieldsPerType < 1 {
		c.Limits.MaxFieldsPerType = 1
	}

	if c.Limits.MaxEntryDataSize < 1024 {
		c.Limits.MaxEntryDataSize = 1024 // Minimum 1KB
	}

	if c.Revisions.MaxRevisionsPerEntry < 1 {
		c.Revisions.MaxRevisionsPerEntry = 1
	}

	if c.Revisions.RetentionDays < 1 {
		c.Revisions.RetentionDays = 1
	}

	if c.Revisions.CleanupInterval < time.Minute {
		c.Revisions.CleanupInterval = time.Minute
	}

	if c.Search.MinSearchLength < 1 {
		c.Search.MinSearchLength = 1
	}

	if c.API.DefaultPageSize < 1 {
		c.API.DefaultPageSize = 1
	}

	if c.API.MaxPageSize < c.API.DefaultPageSize {
		c.API.MaxPageSize = c.API.DefaultPageSize
	}

	if c.Dashboard.EntriesPerPage < 1 {
		c.Dashboard.EntriesPerPage = 1
	}

	return nil
}

// Merge merges another config into this one (non-zero values override)
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Features - booleans are tricky, merge explicitly
	// Note: In Go, we can't distinguish between "false" and "not set"
	// So we always merge features

	// Limits
	if other.Limits.MaxContentTypes > 0 {
		c.Limits.MaxContentTypes = other.Limits.MaxContentTypes
	}
	if other.Limits.MaxFieldsPerType > 0 {
		c.Limits.MaxFieldsPerType = other.Limits.MaxFieldsPerType
	}
	if other.Limits.MaxEntriesPerType > 0 {
		c.Limits.MaxEntriesPerType = other.Limits.MaxEntriesPerType
	}
	if other.Limits.MaxEntryDataSize > 0 {
		c.Limits.MaxEntryDataSize = other.Limits.MaxEntryDataSize
	}
	if other.Limits.MaxRelationsPerEntry > 0 {
		c.Limits.MaxRelationsPerEntry = other.Limits.MaxRelationsPerEntry
	}

	// Revisions
	if other.Revisions.MaxRevisionsPerEntry > 0 {
		c.Revisions.MaxRevisionsPerEntry = other.Revisions.MaxRevisionsPerEntry
	}
	if other.Revisions.RetentionDays > 0 {
		c.Revisions.RetentionDays = other.Revisions.RetentionDays
	}
	if other.Revisions.CleanupInterval > 0 {
		c.Revisions.CleanupInterval = other.Revisions.CleanupInterval
	}

	// Search
	if other.Search.Language != "" {
		c.Search.Language = other.Search.Language
	}
	if other.Search.MinSearchLength > 0 {
		c.Search.MinSearchLength = other.Search.MinSearchLength
	}
	if other.Search.MaxSearchResults > 0 {
		c.Search.MaxSearchResults = other.Search.MaxSearchResults
	}

	// API
	if other.API.DefaultPageSize > 0 {
		c.API.DefaultPageSize = other.API.DefaultPageSize
	}
	if other.API.MaxPageSize > 0 {
		c.API.MaxPageSize = other.API.MaxPageSize
	}
	if other.API.RateLimitPerMinute > 0 {
		c.API.RateLimitPerMinute = other.API.RateLimitPerMinute
	}

	// Dashboard
	if other.Dashboard.EntriesPerPage > 0 {
		c.Dashboard.EntriesPerPage = other.Dashboard.EntriesPerPage
	}
}

