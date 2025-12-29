package magiclink

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
)

type Service struct {
	repo         *repo.MagicLinkRepository
	users        *user.Service
	sessions     *session.Service
	auth         *auth.Service
	audit        *audit.Service
	notifAdapter *notificationPlugin.Adapter
	config       Config
}

func NewService(r *repo.MagicLinkRepository, users *user.Service, sessionSvc *session.Service, authSvc *auth.Service, auditSvc *audit.Service, notifAdapter *notificationPlugin.Adapter, cfg Config) *Service {
	if cfg.ExpiryMinutes == 0 {
		cfg.ExpiryMinutes = 15
	}
	return &Service{
		repo:         r,
		users:        users,
		sessions:     sessionSvc,
		auth:         authSvc,
		audit:        auditSvc,
		notifAdapter: notifAdapter,
		config:       cfg,
	}
}

func (s *Service) Send(ctx context.Context, appID xid.ID, email, ip, ua string) (string, error) {
	// Validate app context
	if appID.IsNil() {
		return "", errs.New("APP_CONTEXT_REQUIRED", "App context is required", 400)
	}

	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return "", errs.RequiredField("email")
	}

	// Organization context is optional for magiclink (can be nil)
	var userOrgID *xid.ID
	orgID, ok := contexts.GetOrganizationID(ctx)
	if ok && !orgID.IsNil() {
		userOrgID = &orgID
	}

	token, err := crypto.GenerateToken(32)
	if err != nil {
		return "", errs.Wrap(err, "TOKEN_GENERATION_FAILED", "Failed to generate token", 500)
	}

	// Calculate expiry
	expiryDuration := time.Duration(s.config.ExpiryMinutes) * time.Minute

	if err := s.repo.Create(ctx, e, token, appID, userOrgID, time.Now().Add(expiryDuration)); err != nil {
		return "", errs.Wrap(err, "MAGIC_LINK_CREATE_FAILED", "Failed to create magic link", 500)
	}

	esc := url.QueryEscape(token)
	magicLink := s.config.BaseURL + "/api/auth/magic-link/verify?token=" + esc

	// Get user name for personalization
	userName := e
	if u, _ := s.users.FindByEmail(ctx, e); u != nil && u.Name != "" {
		userName = u.Name
	} else if at := strings.Index(e, "@"); at > 0 {
		userName = e[:at]
	}

	// Send via notification plugin if available
	if s.notifAdapter != nil {
		err := s.notifAdapter.SendMagicLink(ctx, appID, e, userName, magicLink, s.config.ExpiryMinutes)
		if err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send magic link via notification plugin: %v\n", err)
		}
	}

	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "magiclink_sent", "email:"+e, ip, ua, "")
	}

	if s.config.DevExposeURL || s.notifAdapter == nil {
		return magicLink, nil
	}
	return "", nil
}

// VerifyResult holds the result of magic link verification
type VerifyResult struct {
	Email     string      // Email from magic link
	User      *user.User  // Nil for new users, populated for existing users
	IsNewUser bool        // True if this is a new user signup
	AppID     xid.ID      // App ID for context
	EnvID     xid.ID      // Environment ID for context
	OrgID     *xid.ID     // Optional organization ID
}

func (s *Service) Verify(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, token string, remember bool, ip, ua string) (*VerifyResult, error) {
	// Validate app context
	if appID.IsNil() {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context is required", 400)
	}

	t := strings.TrimSpace(token)
	if t == "" {
		return nil, errs.RequiredField("token")
	}

	rec, err := s.repo.FindByToken(ctx, t, appID, orgID, time.Now())
	if err != nil {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "magiclink_verify_failed", "token:"+t, ip, ua, "")
		}
		return nil, errs.Wrap(err, "MAGIC_LINK_LOOKUP_FAILED", "Failed to lookup magic link", 500)
	}
	if rec == nil {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "magiclink_verify_failed", "token:"+t, ip, ua, "")
		}
		return nil, errs.MagicLinkExpired()
	}

	_ = s.repo.Consume(ctx, rec, time.Now())

	// Check if user exists
	u, err := s.users.FindByEmail(ctx, rec.Email)
	if err != nil || u == nil {
		if !s.config.AllowImplicitSignup {
			return nil, errs.UserNotFound()
		}
		// Return info for new user - handler will create via authService.SignUp
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "magiclink_verify_success_new_user", "email:"+rec.Email, ip, ua, "")
		}
		return &VerifyResult{
			Email:     rec.Email,
			User:      nil,
			IsNewUser: true,
			AppID:     appID,
			EnvID:     envID,
			OrgID:     orgID,
		}, nil
	}

	// Return existing user info - handler will create session via authService
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "magiclink_verify_success_existing_user", "email:"+rec.Email, ip, ua, "")
	}

	return &VerifyResult{
		Email:     rec.Email,
		User:      u,
		IsNewUser: false,
		AppID:     appID,
		EnvID:     envID,
		OrgID:     orgID,
	}, nil
}
