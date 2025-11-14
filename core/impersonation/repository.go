package impersonation

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// IMPERSONATION REPOSITORY INTERFACE (ISP Compliant)
// =============================================================================

// Repository defines persistence operations for impersonation sessions
// This follows the Interface Segregation Principle from core/app and core/jwt architecture
type Repository interface {
	// Session operations
	Create(ctx context.Context, session *schema.ImpersonationSession) error
	Get(ctx context.Context, id xid.ID, appID xid.ID) (*schema.ImpersonationSession, error)
	GetBySessionID(ctx context.Context, sessionID xid.ID) (*schema.ImpersonationSession, error)
	Update(ctx context.Context, session *schema.ImpersonationSession) error

	// ListSessions lists impersonation sessions with pagination and filtering
	ListSessions(ctx context.Context, filter *ListSessionsFilter) (*pagination.PageResponse[*schema.ImpersonationSession], error)

	GetActive(ctx context.Context, impersonatorID xid.ID, appID xid.ID) (*schema.ImpersonationSession, error)
	ExpireOldSessions(ctx context.Context) (int, error) // Returns number of expired sessions

	// Audit operations
	CreateAuditEvent(ctx context.Context, event *schema.ImpersonationAuditEvent) error

	// ListAuditEvents lists audit events with pagination and filtering
	ListAuditEvents(ctx context.Context, filter *ListAuditEventsFilter) (*pagination.PageResponse[*schema.ImpersonationAuditEvent], error)
}
