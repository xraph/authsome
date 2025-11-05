package impersonation

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Repository defines persistence operations for impersonation sessions
type Repository interface {
	// Session operations
	Create(ctx context.Context, session *schema.ImpersonationSession) error
	Get(ctx context.Context, id xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error)
	GetBySessionID(ctx context.Context, sessionID xid.ID) (*schema.ImpersonationSession, error)
	Update(ctx context.Context, session *schema.ImpersonationSession) error
	List(ctx context.Context, req *ListRequest) ([]*schema.ImpersonationSession, error)
	Count(ctx context.Context, req *ListRequest) (int, error)
	GetActive(ctx context.Context, impersonatorID xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error)
	ExpireOldSessions(ctx context.Context) (int, error) // Returns number of expired sessions

	// Audit operations
	CreateAuditEvent(ctx context.Context, event *schema.ImpersonationAuditEvent) error
	ListAuditEvents(ctx context.Context, req *AuditListRequest) ([]*schema.ImpersonationAuditEvent, error)
	CountAuditEvents(ctx context.Context, req *AuditListRequest) (int, error)
}

// AuditRepository defines operations for impersonation audit logging
type AuditRepository interface {
	Create(ctx context.Context, event *schema.ImpersonationAuditEvent) error
	List(ctx context.Context, params AuditListRequest) ([]*schema.ImpersonationAuditEvent, error)
	Count(ctx context.Context, params AuditListRequest) (int, error)
}
