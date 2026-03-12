// Package bridge defines local interfaces for optional forgery extension
// integration. AuthSome defines its own slim interfaces (following the
// keysmith audit_hook pattern) to avoid hard coupling to external packages.
// Bridge adapters in separate packages translate between these local types
// and the actual extension APIs.
package bridge

import "context"

// Chronicle is a local audit interface. Implementations record auth events
// to an audit trail backend (e.g., chronicle).
type Chronicle interface {
	Record(ctx context.Context, event *AuditEvent) error
}

// AuditEvent is a local representation of an audit event.
type AuditEvent struct {
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resource_id,omitempty"`
	ActorID    string            `json:"actor_id,omitempty"`
	Tenant     string            `json:"tenant,omitempty"`
	Outcome    string            `json:"outcome"`
	Severity   string            `json:"severity"`
	Category   string            `json:"category,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

// Severity constants.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Outcome constants.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

// ChronicleFunc is an adapter to use a plain function as a Chronicle.
type ChronicleFunc func(ctx context.Context, event *AuditEvent) error

// Record implements Chronicle.
func (f ChronicleFunc) Record(ctx context.Context, event *AuditEvent) error {
	return f(ctx, event)
}
