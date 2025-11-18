package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// NotificationTestType represents the type of test being performed
type NotificationTestType string

const (
	NotificationTestTypePreview NotificationTestType = "preview" // Template preview/render test
	NotificationTestTypeSend    NotificationTestType = "send"    // Single test send
	NotificationTestTypeBulk    NotificationTestType = "bulk"    // Bulk test with multiple recipients
)

// NotificationTestStatus represents the test execution status
type NotificationTestStatus string

const (
	NotificationTestStatusPending    NotificationTestStatus = "pending"    // Test queued
	NotificationTestStatusRunning    NotificationTestStatus = "running"    // Test in progress
	NotificationTestStatusCompleted  NotificationTestStatus = "completed"  // Test completed successfully
	NotificationTestStatusFailed     NotificationTestStatus = "failed"     // Test failed
	NotificationTestStatusPartial    NotificationTestStatus = "partial"    // Some tests succeeded, some failed
)

// NotificationTest represents a test execution for notification templates
type NotificationTest struct {
	bun.BaseModel `bun:"table:notification_tests,alias:ntest"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	TemplateID     xid.ID                 `bun:"template_id,notnull,type:varchar(20)" json:"templateId"`
	AppID          xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	OrganizationID *xid.ID                `bun:"organization_id,type:varchar(20)" json:"organizationId,omitempty"`
	TestType       string                 `bun:"test_type,notnull" json:"testType"` // preview, send, bulk
	Recipients     []string               `bun:"recipients,array" json:"recipients"` // Test recipient(s)
	Variables      map[string]interface{} `bun:"variables,type:jsonb" json:"variables,omitempty"` // Test variables
	Results        map[string]interface{} `bun:"results,type:jsonb" json:"results,omitempty"`     // Test results (success/failure for each recipient)
	Status         string                 `bun:"status,notnull,default:'pending'" json:"status"`  // pending, running, completed, failed, partial
	Error          string                 `bun:"error" json:"error,omitempty"`                    // Error message if failed
	SuccessCount   int                    `bun:"success_count,notnull,default:0" json:"successCount"`
	FailureCount   int                    `bun:"failure_count,notnull,default:0" json:"failureCount"`
	CreatedBy      *xid.ID                `bun:"created_by,type:varchar(20)" json:"createdBy,omitempty"` // User who initiated the test
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time              `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	CompletedAt    *time.Time             `bun:"completed_at" json:"completedAt,omitempty"`

	// Relations
	Template     *NotificationTemplate `bun:"rel:belongs-to,join:template_id=id" json:"template,omitempty"`
	App          *App                  `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Organization *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	User         *User                 `bun:"rel:belongs-to,join:created_by=id" json:"user,omitempty"`
}

