package passkey

import (
	"context"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/schema"
	"time"
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
func (s *Service) BeginRegistration(_ context.Context, userID string) (map[string]any, error) {
	// In a full implementation, generate WebAuthn options and store session
	return map[string]any{"challenge": time.Now().UnixNano(), "rpId": s.config.RPID, "userId": userID}, nil
}

// FinishRegistration persists a passkey record (stub)
func (s *Service) FinishRegistration(ctx context.Context, userID string, credentialID string, ip, ua string) error {
	pk := &schema.Passkey{ID: xid.New(), UserID: userID, CredentialID: credentialID}
	pk.AuditableModel.CreatedBy = pk.ID
	pk.AuditableModel.UpdatedBy = pk.ID
	_, err := s.db.NewInsert().Model(pk).Exec(ctx)
	if err != nil {
		return err
	}
	// Audit: passkey registered
	if s.audit != nil {
		uid, _ := xid.FromString(userID)
		_ = s.audit.Log(ctx, &uid, "passkey_registered", "passkey:"+pk.ID.String(), ip, ua, "")
	}
	return nil
}

// BeginLogin returns a simple challenge payload (stub)
func (s *Service) BeginLogin(_ context.Context, userID string) (map[string]any, error) {
	return map[string]any{"challenge": time.Now().UnixNano(), "rpId": s.config.RPID, "userId": userID}, nil
}

// FinishLogin returns an auth session for the user (stub)
func (s *Service) FinishLogin(ctx context.Context, userID string, remember bool, ip, ua string) (*auth.AuthResponse, error) {
	// Lookup user to ensure exists
	id, err := xid.FromString(userID)
	if err != nil {
		return nil, err
	}
	u, err := s.userSvc.FindByID(ctx, id)
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

// List user passkeys
func (s *Service) List(ctx context.Context, userID string) ([]schema.Passkey, error) {
	var out []schema.Passkey
	err := s.db.NewSelect().Model(&out).Where("user_id = ?", userID).Scan(ctx)
	return out, err
}

// Delete passkey by ID
func (s *Service) Delete(ctx context.Context, id string, ip, ua string) error {
	xidVal, err := xid.FromString(id)
	if err != nil {
		return err
	}
	// Fetch record to get userID for audit context
	var existing schema.Passkey
	_ = s.db.NewSelect().Model(&existing).Where("id = ?", xidVal).Scan(ctx)
	_, err = s.db.NewDelete().Model((*schema.Passkey)(nil)).Where("id = ?", xidVal).Exec(ctx)
	if err != nil {
		return err
	}
	if s.audit != nil {
		var uidPtr *xid.ID
		if existing.UserID != "" {
			if uid, e := xid.FromString(existing.UserID); e == nil {
				uidPtr = &uid
			}
		}
		_ = s.audit.Log(ctx, uidPtr, "passkey_deleted", "passkey:"+id, ip, ua, "")
	}
	return nil
}
