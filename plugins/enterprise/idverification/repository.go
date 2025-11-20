package idverification

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Repository defines the interface for identity verification data operations
// Updated for V2 architecture with App → Environment → Organization hierarchy
type Repository interface {
	// Identity Verification CRUD
	CreateVerification(ctx context.Context, verification *schema.IdentityVerification) error
	GetVerificationByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerification, error)
	GetVerificationsByUserID(ctx context.Context, appID xid.ID, userID xid.ID, limit, offset int) ([]*schema.IdentityVerification, error)
	GetVerificationsByOrgID(ctx context.Context, appID xid.ID, orgID xid.ID, limit, offset int) ([]*schema.IdentityVerification, error)
	UpdateVerification(ctx context.Context, verification *schema.IdentityVerification) error
	DeleteVerification(ctx context.Context, appID xid.ID, id string) error

	// Query methods
	GetLatestVerificationByUser(ctx context.Context, appID xid.ID, userID xid.ID) (*schema.IdentityVerification, error)
	GetVerificationByProviderCheckID(ctx context.Context, appID xid.ID, providerCheckID string) (*schema.IdentityVerification, error)
	GetVerificationsByStatus(ctx context.Context, appID xid.ID, status string, limit, offset int) ([]*schema.IdentityVerification, error)
	GetVerificationsByType(ctx context.Context, appID xid.ID, verificationType string, limit, offset int) ([]*schema.IdentityVerification, error)
	CountVerificationsByUser(ctx context.Context, appID xid.ID, userID xid.ID, since time.Time) (int, error)
	GetExpiredVerifications(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerification, error)

	// Document operations
	CreateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error
	GetDocumentByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerificationDocument, error)
	GetDocumentsByVerificationID(ctx context.Context, appID xid.ID, verificationID string) ([]*schema.IdentityVerificationDocument, error)
	UpdateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error
	DeleteDocument(ctx context.Context, appID xid.ID, id string) error
	GetDocumentsForDeletion(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerificationDocument, error)

	// Session operations
	CreateSession(ctx context.Context, session *schema.IdentityVerificationSession) error
	GetSessionByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerificationSession, error)
	GetSessionsByUserID(ctx context.Context, appID xid.ID, userID xid.ID, limit, offset int) ([]*schema.IdentityVerificationSession, error)
	UpdateSession(ctx context.Context, session *schema.IdentityVerificationSession) error
	DeleteSession(ctx context.Context, appID xid.ID, id string) error
	GetExpiredSessions(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerificationSession, error)

	// User verification status
	CreateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error
	GetUserVerificationStatus(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) (*schema.UserVerificationStatus, error)
	UpdateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error
	DeleteUserVerificationStatus(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) error
	GetUsersRequiringReverification(ctx context.Context, appID xid.ID, limit int) ([]*schema.UserVerificationStatus, error)
	GetUsersByVerificationLevel(ctx context.Context, appID xid.ID, level string, limit, offset int) ([]*schema.UserVerificationStatus, error)
	GetBlockedUsers(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.UserVerificationStatus, error)

	// Analytics and reporting - Returns map[string]interface{} for flexibility
	GetVerificationStats(ctx context.Context, appID xid.ID, orgID xid.ID, from, to time.Time) (map[string]interface{}, error)
	GetProviderStats(ctx context.Context, appID xid.ID, provider string, from, to time.Time) (map[string]interface{}, error)
}
