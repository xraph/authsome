package emailotp

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Service implements email OTP flow.
type Service struct {
	repo         *repo.EmailOTPRepository
	users        user.ServiceInterface
	sessions     session.ServiceInterface
	audit        *audit.Service
	notifAdapter *notificationPlugin.Adapter
	config       Config
	logger       forge.Logger
}

func NewService(
	r *repo.EmailOTPRepository,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	auditSvc *audit.Service,
	notifAdapter *notificationPlugin.Adapter,
	cfg Config,
	logger forge.Logger,
) *Service {
	// defaults
	if cfg.OTPLength == 0 {
		cfg.OTPLength = 6
	}

	if cfg.ExpiryMinutes == 0 {
		cfg.ExpiryMinutes = 10
	}

	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 5
	}

	return &Service{
		repo:         r,
		users:        userSvc,
		sessions:     sessionSvc,
		audit:        auditSvc,
		notifAdapter: notifAdapter,
		config:       cfg,
		logger:       logger,
	}
}

func (s *Service) SendOTP(ctx context.Context, appID xid.ID, email, ip, ua string) (string, error) {
	// Validate app context
	if appID.IsNil() {
		return "", errs.New("APP_CONTEXT_REQUIRED", "App context is required", 400)
	}

	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return "", errs.New("EMAIL_REQUIRED", "Email is required", 400)
	}

	// Generate numeric OTP
	rand.Seed(time.Now().UnixNano())

	max := int64(1)
	for range s.config.OTPLength {
		max *= 10
	}

	code := int64(rand.Intn(int(max)))
	otp := fmt.Sprintf("%0*d", s.config.OTPLength, code)

	// Calculate expiry
	expiryDuration := time.Duration(s.config.ExpiryMinutes) * time.Minute

	// Persist OTP
	if err := s.repo.Create(ctx, e, otp, time.Now().Add(expiryDuration)); err != nil {
		return "", errs.Wrap(err, "OTP_CREATE_FAILED", "Failed to create OTP", 500)
	}

	// Send via notification plugin if available
	if s.notifAdapter != nil {
		err := s.notifAdapter.SendEmailOTP(ctx, appID, e, otp, s.config.ExpiryMinutes)
		if err != nil {
			// Log error but don't fail the operation
			s.logger.Error("failed to send email OTP via notification plugin", forge.F("error", err.Error()))
		}
	}

	// Audit: email OTP sent
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, string(audit.ActionEmailOTPSent), "email:"+e, ip, ua, "")
	}

	// Return OTP only if dev mode
	if s.config.DevExposeOTP {
		return otp, nil
	}

	return "", nil
}

func (s *Service) VerifyOTP(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, email, otp string, remember bool, ip, ua string) (*responses.AuthResponse, error) {
	// Validate app context
	if appID.IsNil() {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context is required", 400)
	}

	e := strings.ToLower(strings.TrimSpace(email))

	o := strings.TrimSpace(otp)
	if e == "" || o == "" {
		return nil, errs.New("MISSING_FIELDS", "Email and OTP are required", 400)
	}

	rec, err := s.repo.FindByEmail(ctx, e, time.Now())
	if err != nil {
		return nil, errs.Wrap(err, "OTP_LOOKUP_FAILED", "Failed to lookup OTP", 500)
	}

	if rec == nil {
		return nil, errs.New("OTP_NOT_FOUND", "OTP not found or expired", 404)
	}

	if rec.Attempts >= s.config.MaxAttempts {
		return nil, errs.New("TOO_MANY_ATTEMPTS", "Too many verification attempts", 429)
	}

	if rec.OTP != o {
		_ = s.repo.IncrementAttempts(ctx, rec)
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, string(audit.ActionEmailOTPVerifyFailed), "email:"+e, ip, ua, "")
		}

		return nil, errs.New("INVALID_OTP", "Invalid OTP code", 401)
	}

	// Success: consume OTP and create session for the user
	_ = s.repo.Consume(ctx, rec, time.Now())

	u, err := s.users.FindByEmail(ctx, e)
	if err != nil || u == nil {
		if !s.config.AllowImplicitSignup {
			return nil, errs.UserNotFound()
		}
		// Implicit sign-up: create user if missing
		// Generate a secure random password to satisfy validation
		pwd, genErr := crypto.GenerateToken(16)
		if genErr != nil {
			return nil, errs.Wrap(genErr, "PASSWORD_GENERATION_FAILED", "Failed to generate password", 500)
		}

		name := "Email OTP User"
		if at := strings.Index(e, "@"); at > 0 {
			name = e[:at]
		}

		u, err = s.users.Create(ctx, &user.CreateUserRequest{
			AppID:    appID,
			Email:    e,
			Password: pwd,
			Name:     name,
		})
		if err != nil {
			return nil, errs.Wrap(err, "USER_CREATION_FAILED", "Failed to create user", 500)
		}
	}

	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, string(audit.ActionEmailOTPVerifySuccess), "email:"+e, ip, ua, "")
	}

	// Create session with app/environment context
	sess, err := s.sessions.Create(ctx, &session.CreateSessionRequest{
		AppID:          appID,
		EnvironmentID:  &envID,
		OrganizationID: orgID,
		UserID:         u.ID,
		Remember:       remember,
		IPAddress:      ip,
		UserAgent:      ua,
	})
	if err != nil {
		return nil, errs.Wrap(err, "SESSION_CREATION_FAILED", "Failed to create session", 500)
	}

	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, string(audit.ActionEmailOTPLogin), "user:"+uid.String(), ip, ua, "")
	}

	return &responses.AuthResponse{
		User:    u,
		Session: sess,
		Token:   sess.Token,
	}, nil
}
