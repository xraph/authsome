package passkey

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/schema"
)

type Config struct {
	RPName string
	RPID   string
}

type Service struct {
	db      *bun.DB
	userSvc *user.Service
	authSvc *auth.Service
	audit   *audit.Service
	config  Config
}

func NewService(db *bun.DB, userSvc *user.Service, authSvc *auth.Service, auditSvc *audit.Service, cfg Config) *Service {
	return &Service{db: db, userSvc: userSvc, authSvc: authSvc, audit: auditSvc, config: cfg}
}

// BeginRegistration returns a simple challenge payload (stub for WebAuthn options)
func (s *Service) BeginRegistration(_ context.Context, userID xid.ID) (map[string]any, error) {
	// In a full implementation, generate WebAuthn options and store session
	return map[string]any{"challenge": time.Now().UnixNano(), "rpId": s.config.RPID, "userId": userID.String()}, nil
}

// FinishRegistration persists a passkey record with app and org scoping
func (s *Service) FinishRegistration(ctx context.Context, userID xid.ID, credentialID string, ip, ua string) error {
	// Get app and org from context
	appID := contexts.GetAppID(ctx)
	orgID := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	pk := &schema.Passkey{
		ID:                 xid.New(),
		UserID:             userID,
		CredentialID:       credentialID,
		AppID:              appID,
		UserOrganizationID: userOrgID,
	}
	pk.AuditableModel.CreatedBy = pk.ID
	pk.AuditableModel.UpdatedBy = pk.ID
	_, err := s.db.NewInsert().Model(pk).Exec(ctx)
	if err != nil {
		return err
	}
	// Audit: passkey registered
	if s.audit != nil {
		_ = s.audit.Log(ctx, &userID, "passkey_registered", "passkey:"+pk.ID.String(), ip, ua, "")
	}
	return nil
}

// BeginLogin returns a simple challenge payload (stub)
func (s *Service) BeginLogin(_ context.Context, userID xid.ID) (map[string]any, error) {
	return map[string]any{"challenge": time.Now().UnixNano(), "rpId": s.config.RPID, "userId": userID.String()}, nil
}

// FinishLogin returns an auth session for the user (stub)
func (s *Service) FinishLogin(ctx context.Context, userID xid.ID, remember bool, ip, ua string) (*auth.AuthResponse, error) {
	// Lookup user to ensure exists
	u, err := s.userSvc.FindByID(ctx, userID)
	if err != nil || u == nil {
		return nil, err
	}
	resp, err := s.authSvc.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err == nil && s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "passkey_login", "user:"+uid.String(), ip, ua, "")
	}
	return resp, err
}

// List user passkeys, scoped to app and optional org
func (s *Service) List(ctx context.Context, userID xid.ID) ([]schema.Passkey, error) {
	// Get app and org from context
	appID := contexts.GetAppID(ctx)
	orgID := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	var out []schema.Passkey
	q := s.db.NewSelect().Model(&out).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID)

	// Scope to org if provided
	if userOrgID != nil {
		q = q.Where("user_organization_id = ?", *userOrgID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	return out, err
}

// Delete passkey by ID, scoped to app and optional org
func (s *Service) Delete(ctx context.Context, id xid.ID, ip, ua string) error {
	// Get app and org from context
	appID := contexts.GetAppID(ctx)
	orgID := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	// Fetch record to verify ownership and get userID for audit
	var existing schema.Passkey
	q := s.db.NewSelect().Model(&existing).
		Where("id = ?", id).
		Where("app_id = ?", appID)

	if userOrgID != nil {
		q = q.Where("user_organization_id = ?", *userOrgID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	if err != nil {
		return err
	}

	// Delete the passkey
	_, err = s.db.NewDelete().Model((*schema.Passkey)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}

	// Audit
	if s.audit != nil {
		_ = s.audit.Log(ctx, &existing.UserID, "passkey_deleted", "passkey:"+id.String(), ip, ua, "")
	}
	return nil
}
