package phone

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
)

var (
	// E.164 phone number format: +[country code][subscriber number]
	// Example: +1234567890, +442071838750
	// Minimum 7 digits (e.g., +1234567), maximum 15 digits total
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

	// Common errors
	ErrInvalidPhoneFormat = errors.New("invalid phone number format, must be E.164 format (e.g., +1234567890)")
	ErrMissingPhone       = errors.New("phone number is required")
	ErrMissingCode        = errors.New("verification code is required")
	ErrMissingEmail       = errors.New("email is required")
	ErrCodeExpired        = errors.New("verification code not found or expired")
	ErrTooManyAttempts    = errors.New("too many verification attempts, please request a new code")
	ErrInvalidCode        = errors.New("invalid verification code")
)

type Service struct {
	repo         *repo.PhoneRepository
	users        *user.Service
	auth         *auth.Service
	audit        *audit.Service
	notifAdapter *notificationPlugin.Adapter
	config       Config
}

func NewService(r *repo.PhoneRepository, users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, notifAdapter *notificationPlugin.Adapter, cfg Config) *Service {
	if cfg.CodeLength == 0 {
		cfg.CodeLength = 6
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

// validatePhone validates phone number in E.164 format
func validatePhone(phone string) error {
	p := strings.TrimSpace(phone)
	if p == "" {
		return ErrMissingPhone
	}
	if !phoneRegex.MatchString(p) {
		return ErrInvalidPhoneFormat
	}
	return nil
}

// generateSecureCode generates a cryptographically secure numeric code
func generateSecureCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("code length must be positive")
	}

	// Calculate max value (10^length)
	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}

	// Generate random number in range [0, max)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	// Format with leading zeros
	return fmt.Sprintf("%0*d", length, n), nil
}

func (s *Service) SendCode(ctx context.Context, phone, ip, ua string) (string, error) {
	p := strings.TrimSpace(phone)
	if err := validatePhone(p); err != nil {
		return "", err
	}

	// Get app and org from context
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	// Generate cryptographically secure numeric code
	otp, err := generateSecureCode(s.config.CodeLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate verification code: %w", err)
	}

	// Calculate expiry
	expiryDuration := time.Duration(s.config.ExpiryMinutes) * time.Minute

	if err := s.repo.Create(ctx, p, otp, appID, userOrgID, time.Now().Add(expiryDuration)); err != nil {
		// Audit failed attempt
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "phone_code_send_failed",
				fmt.Sprintf("phone:%s error:%s", p, err.Error()), ip, ua,
				fmt.Sprintf(`{"phone":"%s","error":"%s","app_id":"%s"}`, p, err.Error(), appID.String()))
		}
		return "", fmt.Errorf("failed to create verification code: %w", err)
	}

	// Send via notification plugin if available
	if s.notifAdapter != nil {
		err := s.notifAdapter.SendPhoneOTP(ctx, appID, p, otp)
		if err != nil {
			// Audit SMS send failure but don't fail the operation
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, "phone_sms_send_failed",
					fmt.Sprintf("phone:%s provider:%s error:%s", p, s.config.SMSProvider, err.Error()),
					ip, ua,
					fmt.Sprintf(`{"phone":"%s","provider":"%s","error":"%s","app_id":"%s"}`,
						p, s.config.SMSProvider, err.Error(), appID.String()))
			}
			// Log error but don't fail - code is still valid for verification
			fmt.Printf("Failed to send phone OTP via notification plugin: %v\n", err)
		} else {
			// Audit successful SMS send
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, "phone_sms_sent",
					fmt.Sprintf("phone:%s provider:%s", p, s.config.SMSProvider),
					ip, ua,
					fmt.Sprintf(`{"phone":"%s","provider":"%s","app_id":"%s","org_id":"%s"}`,
						p, s.config.SMSProvider, appID.String(), func() string {
							if userOrgID != nil {
								return userOrgID.String()
							}
							return ""
						}()))
			}
		}
	}

	// Audit code creation
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, "phone_code_created",
			fmt.Sprintf("phone:%s expires_in:%dm", p, s.config.ExpiryMinutes),
			ip, ua,
			fmt.Sprintf(`{"phone":"%s","expires_in_minutes":%d,"app_id":"%s"}`,
				p, s.config.ExpiryMinutes, appID.String()))
	}

	if s.config.DevExposeCode {
		return otp, nil
	}
	return "", nil
}

func (s *Service) Verify(ctx context.Context, phone, code, email string, remember bool, ip, ua string) (*responses.AuthResponse, error) {
	p := strings.TrimSpace(phone)
	c := strings.TrimSpace(code)
	e := strings.ToLower(strings.TrimSpace(email))

	// Validate inputs
	if err := validatePhone(p); err != nil {
		return nil, err
	}
	if c == "" {
		return nil, ErrMissingCode
	}
	if e == "" {
		return nil, ErrMissingEmail
	}

	// Get app and org from context
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	rec, err := s.repo.FindByPhone(ctx, p, appID, userOrgID, time.Now())
	if err != nil {
		// Audit database error
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "phone_verify_db_error",
				fmt.Sprintf("phone:%s error:%s", p, err.Error()),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","error":"%s"}`, p, e, err.Error()))
		}
		return nil, fmt.Errorf("failed to find verification code: %w", err)
	}
	if rec == nil {
		// Audit expired/not found
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "phone_verify_code_not_found",
				fmt.Sprintf("phone:%s email:%s", p, e),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","reason":"expired_or_not_found"}`, p, e))
		}
		return nil, ErrCodeExpired
	}
	if rec.Attempts >= s.config.MaxAttempts {
		// Audit too many attempts
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "phone_verify_too_many_attempts",
				fmt.Sprintf("phone:%s email:%s attempts:%d max:%d", p, e, rec.Attempts, s.config.MaxAttempts),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","attempts":%d,"max_attempts":%d}`,
					p, e, rec.Attempts, s.config.MaxAttempts))
		}
		return nil, ErrTooManyAttempts
	}
	if rec.Code != c {
		_ = s.repo.IncrementAttempts(ctx, rec)
		// Audit failed verification attempt
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "phone_verify_invalid_code",
				fmt.Sprintf("phone:%s email:%s attempt:%d", p, e, rec.Attempts+1),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","attempt":%d,"remaining_attempts":%d}`,
					p, e, rec.Attempts+1, s.config.MaxAttempts-rec.Attempts-1))
		}
		return nil, ErrInvalidCode
	}

	// Mark code as consumed
	_ = s.repo.Consume(ctx, rec, time.Now())
	u, err := s.users.FindByEmail(ctx, e)
	if err != nil || u == nil {
		if !s.config.AllowImplicitSignup {
			// Audit user not found with implicit signup disabled
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, "phone_verify_user_not_found",
					fmt.Sprintf("phone:%s email:%s implicit_signup:disabled", p, e),
					ip, ua,
					fmt.Sprintf(`{"phone":"%s","email":"%s","implicit_signup_enabled":false}`, p, e))
			}
			return nil, fmt.Errorf("user not found and implicit signup is disabled")
		}

		// Create user via implicit signup
		pwd, genErr := crypto.GenerateToken(16)
		if genErr != nil {
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, "phone_verify_password_gen_failed",
					fmt.Sprintf("phone:%s email:%s error:%s", p, e, genErr.Error()),
					ip, ua,
					fmt.Sprintf(`{"phone":"%s","email":"%s","error":"%s"}`, p, e, genErr.Error()))
			}
			return nil, fmt.Errorf("failed to generate password: %w", genErr)
		}
		name := e
		u, err = s.users.Create(ctx, &user.CreateUserRequest{Email: e, Password: pwd, Name: name})
		if err != nil {
			// Audit user creation failure
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, "phone_verify_user_creation_failed",
					fmt.Sprintf("phone:%s email:%s error:%s", p, e, err.Error()),
					ip, ua,
					fmt.Sprintf(`{"phone":"%s","email":"%s","error":"%s"}`, p, e, err.Error()))
			}
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Audit successful implicit signup
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, "phone_verify_implicit_signup",
				fmt.Sprintf("phone:%s email:%s user_id:%s", p, e, uid.String()),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","user_id":"%s","method":"implicit_signup"}`,
					p, e, uid.String()))
		}
	}

	// Audit successful verification
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "phone_verify_success",
			fmt.Sprintf("phone:%s email:%s user_id:%s", p, e, uid.String()),
			ip, ua,
			fmt.Sprintf(`{"phone":"%s","email":"%s","user_id":"%s","app_id":"%s"}`,
				p, e, uid.String(), appID.String()))
	}

	// Create session
	res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err != nil {
		// Audit session creation failure
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, "phone_verify_session_failed",
				fmt.Sprintf("phone:%s email:%s user_id:%s error:%s", p, e, uid.String(), err.Error()),
				ip, ua,
				fmt.Sprintf(`{"phone":"%s","email":"%s","user_id":"%s","error":"%s"}`,
					p, e, uid.String(), err.Error()))
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Audit successful login
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, "phone_login_success",
			fmt.Sprintf("phone:%s user_id:%s session_id:%s", p, uid.String(), res.Session.ID.String()),
			ip, ua,
			fmt.Sprintf(`{"phone":"%s","user_id":"%s","session_id":"%s","remember":%t}`,
				p, uid.String(), res.Session.ID.String(), remember))
	}

	return res, nil
}
