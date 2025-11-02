package idverification

import (
	"context"
	"time"

	"github.com/xraph/authsome/schema"
)

// Repository defines the interface for identity verification data operations
type Repository interface {
	// Identity Verification CRUD
	CreateVerification(ctx context.Context, verification *schema.IdentityVerification) error
	GetVerificationByID(ctx context.Context, id string) (*schema.IdentityVerification, error)
	GetVerificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerification, error)
	GetVerificationsByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*schema.IdentityVerification, error)
	UpdateVerification(ctx context.Context, verification *schema.IdentityVerification) error
	DeleteVerification(ctx context.Context, id string) error
	
	// Query methods
	GetLatestVerificationByUser(ctx context.Context, userID string) (*schema.IdentityVerification, error)
	GetVerificationByProviderCheckID(ctx context.Context, providerCheckID string) (*schema.IdentityVerification, error)
	GetVerificationsByStatus(ctx context.Context, status string, limit, offset int) ([]*schema.IdentityVerification, error)
	GetVerificationsByType(ctx context.Context, verificationType string, limit, offset int) ([]*schema.IdentityVerification, error)
	CountVerificationsByUser(ctx context.Context, userID string, since time.Time) (int, error)
	GetExpiredVerifications(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerification, error)
	
	// Document operations
	CreateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error
	GetDocumentByID(ctx context.Context, id string) (*schema.IdentityVerificationDocument, error)
	GetDocumentsByVerificationID(ctx context.Context, verificationID string) ([]*schema.IdentityVerificationDocument, error)
	UpdateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error
	DeleteDocument(ctx context.Context, id string) error
	GetDocumentsForDeletion(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationDocument, error)
	
	// Session operations
	CreateSession(ctx context.Context, session *schema.IdentityVerificationSession) error
	GetSessionByID(ctx context.Context, id string) (*schema.IdentityVerificationSession, error)
	GetSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerificationSession, error)
	UpdateSession(ctx context.Context, session *schema.IdentityVerificationSession) error
	DeleteSession(ctx context.Context, id string) error
	GetExpiredSessions(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationSession, error)
	
	// User verification status
	CreateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error
	GetUserVerificationStatus(ctx context.Context, userID string) (*schema.UserVerificationStatus, error)
	UpdateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error
	DeleteUserVerificationStatus(ctx context.Context, userID string) error
	GetUsersRequiringReverification(ctx context.Context, limit int) ([]*schema.UserVerificationStatus, error)
	GetUsersByVerificationLevel(ctx context.Context, level string, limit, offset int) ([]*schema.UserVerificationStatus, error)
	GetBlockedUsers(ctx context.Context, limit, offset int) ([]*schema.UserVerificationStatus, error)
	
	// Analytics and reporting - Returns map[string]interface{} for flexibility
	GetVerificationStats(ctx context.Context, orgID string, from, to time.Time) (map[string]interface{}, error)
	GetProviderStats(ctx context.Context, provider string, from, to time.Time) (map[string]interface{}, error)
}

