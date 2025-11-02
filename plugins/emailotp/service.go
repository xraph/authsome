package emailotp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
)

// Config for Email OTP service
type Config struct {
	OTPLength           int
	ExpiryMinutes       int
	MaxAttempts         int
	DevExposeOTP        bool // if true, return OTP in response for dev testing
	AllowImplicitSignup bool // if true, create user when verifying if missing
}

// Service implements email OTP flow
type Service struct {
	repo         *repo.EmailOTPRepository
	users        *user.Service
	auth         *auth.Service
	audit        *audit.Service
	notifAdapter *notificationPlugin.Adapter
	config       Config
}

func NewService(r *repo.EmailOTPRepository, users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, notifAdapter *notificationPlugin.Adapter, cfg Config) *Service {
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
		users:        users,
		auth:         authSvc,
		audit:        auditSvc,
		notifAdapter: notifAdapter,
		config:       cfg,
	}
}

func (s *Service) SendOTP(ctx context.Context, email, ip, ua string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return "", fmt.Errorf("missing email")
	}
	
	// Generate numeric OTP
	rand.Seed(time.Now().UnixNano())
	max := int64(1)
	for i := 0; i < s.config.OTPLength; i++ {
		max *= 10
	}
	code := int64(rand.Intn(int(max)))
	otp := fmt.Sprintf("%0*d", s.config.OTPLength, code)
	
	// Calculate expiry
	expiryDuration := time.Duration(s.config.ExpiryMinutes) * time.Minute
	
	// Persist OTP
	if err := s.repo.Create(ctx, e, otp, time.Now().Add(expiryDuration)); err != nil {
		return "", err
	}
	
	// Send via notification plugin if available
	if s.notifAdapter != nil {
		err := s.notifAdapter.SendEmailOTP(ctx, "default", e, otp, s.config.ExpiryMinutes)
		if err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send email OTP via notification plugin: %v\n", err)
		}
	}
	
	// Audit: email OTP sent
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "emailotp_sent", "email:"+e, ip, ua, "")
	}
	
	// Return OTP only if dev mode
	if s.config.DevExposeOTP {
		return otp, nil
	}
	return "", nil
}

func (s *Service) VerifyOTP(ctx context.Context, email, otp string, remember bool, ip, ua string) (*auth.AuthResponse, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	o := strings.TrimSpace(otp)
	if e == "" || o == "" {
		return nil, fmt.Errorf("missing fields")
	}
	rec, err := s.repo.FindByEmail(ctx, e, time.Now())
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.New("otp not found or expired")
	}
	if rec.Attempts >= s.config.MaxAttempts {
		return nil, errors.New("too many attempts")
	}
	if rec.OTP != o {
		_ = s.repo.IncrementAttempts(ctx, rec)
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "emailotp_verify_failed", "email:"+e, ip, ua, "")
		}
		return nil, nil
	}
	// Success: consume OTP and create session for the user
	_ = s.repo.Consume(ctx, rec, time.Now())
	u, err := s.users.FindByEmail(ctx, e)
	if err != nil || u == nil {
		if !s.config.AllowImplicitSignup {
			return nil, fmt.Errorf("user not found")
		}
		// Implicit sign-up: create user if missing
		// Generate a secure random password to satisfy validation
		pwd, genErr := crypto.GenerateToken(16)
		if genErr != nil {
			return nil, genErr
		}
		name := "Email OTP User"
		if at := strings.Index(e, "@"); at > 0 {
			name = e[:at]
		}
		u, err = s.users.Create(ctx, &user.CreateUserRequest{Email: e, Password: pwd, Name: name})
		if err != nil {
			return nil, err
		}
	}
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "emailotp_verify_success", "email:"+e, ip, ua, "")
	}
	res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err == nil && s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "emailotp_login", "user:"+uid.String(), ip, ua, "")
	}
	return res, err
}
