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
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/interfaces"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
)

// Config for Magic Link service
type Config struct {
	ExpiryMinutes       int
	BaseURL             string
	DevExposeURL        bool
	AllowImplicitSignup bool
}

type Service struct {
	repo         *repo.MagicLinkRepository
	users        *user.Service
	auth         *auth.Service
	audit        *audit.Service
	notifAdapter *notificationPlugin.Adapter
	config       Config
}

func NewService(r *repo.MagicLinkRepository, users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, notifAdapter *notificationPlugin.Adapter, cfg Config) *Service {
	if cfg.ExpiryMinutes == 0 {
		cfg.ExpiryMinutes = 15
	}
	return &Service{
		repo:         r,
		users:        users,
		auth:         authSvc,
		audit:        auditSvc,
		notifAdapter: notifAdapter,
		config:       cfg,
	}
}

func (s *Service) Send(ctx context.Context, email, ip, ua string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return "", fmt.Errorf("missing email")
	}

	// Get app and org from context
	appID := interfaces.GetAppID(ctx)
	orgID := interfaces.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	token, err := crypto.GenerateToken(32)
	if err != nil {
		return "", err
	}

	// Calculate expiry
	expiryDuration := time.Duration(s.config.ExpiryMinutes) * time.Minute

	if err := s.repo.Create(ctx, e, token, appID, userOrgID, time.Now().Add(expiryDuration)); err != nil {
		return "", err
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
		err := s.notifAdapter.SendMagicLink(ctx, "default", e, userName, magicLink, s.config.ExpiryMinutes)
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

func (s *Service) Verify(ctx context.Context, token string, remember bool, ip, ua string) (*auth.AuthResponse, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return nil, fmt.Errorf("missing token")
	}

	// Get app and org from context
	appID := interfaces.GetAppID(ctx)
	orgID := interfaces.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	rec, err := s.repo.FindByToken(ctx, t, appID, userOrgID, time.Now())
	if err != nil || rec == nil {
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "magiclink_verify_failed", "token:"+t, ip, ua, "")
		}
		return nil, fmt.Errorf("invalid or expired token")
	}
	_ = s.repo.Consume(ctx, rec, time.Now())
	// Find or create user
	u, err := s.users.FindByEmail(ctx, rec.Email)
	if err != nil || u == nil {
		if !s.config.AllowImplicitSignup {
			return nil, fmt.Errorf("user not found")
		}
		pwd, genErr := crypto.GenerateToken(16)
		if genErr != nil {
			return nil, genErr
		}
		name := rec.Email
		if at := strings.Index(rec.Email, "@"); at > 0 {
			name = rec.Email[:at]
		}
		u, err = s.users.Create(ctx, &user.CreateUserRequest{Email: rec.Email, Password: pwd, Name: name})
		if err != nil {
			return nil, err
		}
	}
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "magiclink_verify_success", "email:"+rec.Email, ip, ua, "")
	}
	res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err == nil && s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "magiclink_login", "user:"+uid.String(), ip, ua, "")
	}
	return res, err
}
