package username

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
)

// Service provides username-based auth operations backed by core services.
type Service struct {
	users        *user.Service
	auth         *auth.Service
	audit        *audit.Service
	usernameRepo *repo.UsernameRepository
	config       Config
}

func NewService(users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, usernameRepo *repo.UsernameRepository, config Config) *Service {
	return &Service{
		users:        users,
		auth:         authSvc,
		audit:        auditSvc,
		usernameRepo: usernameRepo,
		config:       config,
	}
}

// =============================================================================
// PASSWORD VALIDATION
// =============================================================================

// ValidatePassword validates password against configured requirements.
func (s *Service) ValidatePassword(password string) error {
	if len(password) < s.config.MinPasswordLength {
		return errs.WeakPassword(fmt.Sprintf("password must be at least %d characters", s.config.MinPasswordLength))
	}

	if len(password) > s.config.MaxPasswordLength {
		return errs.WeakPassword(fmt.Sprintf("password must be at most %d characters", s.config.MaxPasswordLength))
	}

	if s.config.RequireUppercase {
		hasUpper := false

		for _, c := range password {
			if c >= 'A' && c <= 'Z' {
				hasUpper = true

				break
			}
		}

		if !hasUpper {
			return errs.WeakPassword("password must contain at least one uppercase letter")
		}
	}

	if s.config.RequireLowercase {
		hasLower := false

		for _, c := range password {
			if c >= 'a' && c <= 'z' {
				hasLower = true

				break
			}
		}

		if !hasLower {
			return errs.WeakPassword("password must contain at least one lowercase letter")
		}
	}

	if s.config.RequireNumber {
		hasNumber := false

		for _, c := range password {
			if c >= '0' && c <= '9' {
				hasNumber = true

				break
			}
		}

		if !hasNumber {
			return errs.WeakPassword("password must contain at least one number")
		}
	}

	if s.config.RequireSpecialChar {
		hasSpecial := false

		specialChars := "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"
		for _, c := range password {
			if strings.ContainsRune(specialChars, c) {
				hasSpecial = true

				break
			}
		}

		if !hasSpecial {
			return errs.WeakPassword("password must contain at least one special character")
		}
	}

	return nil
}

// =============================================================================
// ACCOUNT LOCKOUT METHODS
// =============================================================================

// checkAccountLockout verifies if a user account is locked.
func (s *Service) checkAccountLockout(ctx context.Context, userID xid.ID) error {
	if !s.config.LockoutEnabled {
		return nil
	}

	locked, lockedUntil, err := s.usernameRepo.IsAccountLocked(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check account lockout: %w", err)
	}

	if locked && lockedUntil != nil {
		minutesLeft := max(int(time.Until(*lockedUntil).Minutes()), 0)

		return &AccountLockoutError{
			LockedUntil:   *lockedUntil,
			LockedMinutes: minutesLeft,
		}
	}

	return nil
}

// recordFailedLoginAttempt logs a failed login attempt.
func (s *Service) recordFailedLoginAttempt(ctx context.Context, username string, appID xid.ID, ip, ua string) error {
	if !s.config.LockoutEnabled {
		return nil
	}

	return s.usernameRepo.RecordFailedAttempt(ctx, username, appID, ip, ua)
}

// handleAccountLockout checks failed attempts and locks account if threshold exceeded.
func (s *Service) handleAccountLockout(ctx context.Context, userID xid.ID, username string, appID xid.ID) error {
	if !s.config.LockoutEnabled {
		return nil
	}

	// Count failed attempts within the window
	since := time.Now().Add(-s.config.FailedAttemptWindow)

	count, err := s.usernameRepo.GetFailedAttempts(ctx, username, appID, since)
	if err != nil {
		return fmt.Errorf("failed to get failed attempts: %w", err)
	}

	if count >= s.config.MaxFailedAttempts {
		// Lock the account
		reason := fmt.Sprintf("Too many failed login attempts (%d)", count)
		if err := s.usernameRepo.LockAccount(ctx, userID, s.config.LockoutDuration, reason); err != nil {
			return fmt.Errorf("failed to lock account: %w", err)
		}

		// Audit the lockout
		if s.audit != nil {
			_ = s.audit.Log(ctx, &userID, string(audit.ActionUsernameAccountLockedAuto),
				fmt.Sprintf("username:%s attempts:%d", username, count),
				"", "",
				fmt.Sprintf(`{"username":"%s","attempts":%d,"lockout_minutes":%d}`,
					username, count, int(s.config.LockoutDuration.Minutes())))
		}

		return errs.BadRequest("account locked due to too many failed attempts")
	}

	return nil
}

// clearFailedAttempts removes all failed attempts for a username.
func (s *Service) clearFailedAttempts(ctx context.Context, username string, appID xid.ID) error {
	if !s.config.LockoutEnabled {
		return nil
	}

	return s.usernameRepo.ClearFailedAttempts(ctx, username, appID)
}

// =============================================================================
// PASSWORD HISTORY METHODS
// =============================================================================

// validatePasswordHistory checks if password is in user's history.
func (s *Service) validatePasswordHistory(ctx context.Context, userID xid.ID, password string) error {
	if !s.config.PreventPasswordReuse || s.config.PasswordHistorySize == 0 {
		return nil
	}

	inHistory, err := s.usernameRepo.CheckPasswordInHistory(ctx, userID, password, s.config.PasswordHistorySize)
	if err != nil {
		return fmt.Errorf("failed to check password history: %w", err)
	}

	if inHistory {
		return errs.WeakPassword(fmt.Sprintf("password was recently used, please choose a different password (last %d passwords are tracked)", s.config.PasswordHistorySize))
	}

	return nil
}

// savePasswordHistory saves a password hash to history.
func (s *Service) savePasswordHistory(ctx context.Context, userID xid.ID, passwordHash string) error {
	if !s.config.PreventPasswordReuse || s.config.PasswordHistorySize == 0 {
		return nil
	}

	if err := s.usernameRepo.SavePasswordHistory(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("failed to save password history: %w", err)
	}

	// Cleanup old history entries
	if err := s.usernameRepo.CleanupOldPasswordHistory(ctx, userID, s.config.PasswordHistorySize); err != nil {
		// Log error but don't fail
	}

	return nil
}

// =============================================================================
// PASSWORD EXPIRY METHODS
// =============================================================================

// checkPasswordExpiry checks if user's password has expired
// Note: Uses account creation date as password change date is not tracked in current schema.
func (s *Service) checkPasswordExpiry(ctx context.Context, u *user.User) error {
	if !s.config.PasswordExpiryEnabled {
		return nil
	}

	// Use creation date as reference (password change tracking not in current schema)
	daysSinceCreation := int(time.Since(u.CreatedAt).Hours() / 24)
	if daysSinceCreation > s.config.PasswordExpiryDays {
		return errs.PasswordExpired()
	}

	return nil
}

// isPasswordExpired checks if password is expired (no error, just bool)
// Note: Uses account creation date as password change date is not tracked in current schema.
func (s *Service) isPasswordExpired(u *user.User) bool {
	if !s.config.PasswordExpiryEnabled {
		return false
	}

	daysSinceCreation := int(time.Since(u.CreatedAt).Hours() / 24)

	return daysSinceCreation > s.config.PasswordExpiryDays
}

// daysUntilPasswordExpiry returns days until password expires
// Note: Uses account creation date as password change date is not tracked in current schema.
func (s *Service) daysUntilPasswordExpiry(u *user.User) int {
	if !s.config.PasswordExpiryEnabled {
		return -1 // Never expires
	}

	daysSinceCreation := int(time.Since(u.CreatedAt).Hours() / 24)

	remaining := s.config.PasswordExpiryDays - daysSinceCreation
	if remaining < 0 {
		return 0
	}

	return remaining
}

// =============================================================================
// SIGNUP METHOD
// =============================================================================

// SignUpWithUsername creates a new user with username and password.
func (s *Service) SignUpWithUsername(ctx context.Context, username, password, ip, ua string) error {
	// Get app and org from context
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	// Normalize and validate inputs
	disp := strings.TrimSpace(username)
	canonical := strings.ToLower(disp)

	if disp == "" || strings.TrimSpace(password) == "" {
		// Audit attempt
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameSignupFailed),
				"reason:missing_fields", ip, ua,
				fmt.Sprintf(`{"username":"%s","reason":"missing_fields"}`, disp))
		}

		return errs.RequiredField("username and password")
	}

	// Audit signup attempt
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameSignupAttempt),
			"username:"+canonical, ip, ua,
			fmt.Sprintf(`{"username":"%s","app_id":"%s"}`, canonical, appID.String()))
	}

	// Check if username already exists
	existing, _ := s.users.FindByUsername(ctx, canonical)
	if existing != nil {
		// Audit username collision
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameAlreadyExists),
				"username:"+canonical, ip, ua,
				fmt.Sprintf(`{"username":"%s"}`, canonical))
		}

		return errs.UsernameAlreadyExists(canonical)
	}

	// Validate password against configured requirements
	if err := s.ValidatePassword(password); err != nil {
		// Audit weak password
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameWeakPassword),
				"username:"+canonical, ip, ua,
				fmt.Sprintf(`{"username":"%s","reason":"%s"}`, canonical, err.Error()))
		}

		return err
	}

	// Generate a temporary, unique email to satisfy non-null/unique constraints
	// Scoped to app to prevent collisions across apps
	id := xid.New()
	tempEmail := fmt.Sprintf("u-%s@%s.temp.local", id.String(), appID.String())

	// Create user via core user service to reuse password hashing and validations
	u, err := s.users.Create(ctx, &user.CreateUserRequest{
		Email:    tempEmail,
		Password: password,
		Name:     disp,
	})
	if err != nil {
		// Audit creation failure
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameSignupFailed),
				fmt.Sprintf("username:%s error:%s", canonical, err.Error()), ip, ua,
				fmt.Sprintf(`{"username":"%s","error":"%s"}`, canonical, err.Error()))
		}

		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update canonical and display username
	req := &user.UpdateUserRequest{Username: &canonical, DisplayUsername: &disp}
	if _, err := s.users.Update(ctx, u, req); err != nil {
		// Audit update failure
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameSignupFailed),
				fmt.Sprintf("username:%s user_id:%s error:%s", canonical, uid.String(), err.Error()),
				ip, ua,
				fmt.Sprintf(`{"username":"%s","user_id":"%s","error":"%s"}`, canonical, uid.String(), err.Error()))
		}

		return fmt.Errorf("failed to update username: %w", err)
	}

	// Save password to history
	if err := s.savePasswordHistory(ctx, u.ID, u.PasswordHash); err != nil {
		// Log error but don't fail signup
	}

	// Audit successful signup
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameSignupSuccess),
			fmt.Sprintf("username:%s user_id:%s", canonical, uid.String()),
			ip, ua,
			fmt.Sprintf(`{"username":"%s","user_id":"%s","app_id":"%s","org_id":"%s"}`,
				canonical, uid.String(), appID.String(), func() string {
					if orgID != xid.NilID() {
						return orgID.String()
					}

					return ""
				}()))
	}

	return nil
}

// =============================================================================
// SIGNIN METHOD
// =============================================================================

// SignInWithUsername authenticates by username and password.
func (s *Service) SignInWithUsername(ctx context.Context, username, password string, remember bool, ip, ua string) (*responses.AuthResponse, error) {
	// Get app and org from context
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	un := strings.ToLower(strings.TrimSpace(username))
	if un == "" || password == "" {
		return nil, errs.RequiredField("username and password")
	}

	// Audit signin attempt
	if s.audit != nil {
		_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameSigninAttempt),
			"username:"+un, ip, ua,
			fmt.Sprintf(`{"username":"%s","app_id":"%s"}`, un, appID.String()))
	}

	// Find user by username
	u, err := s.users.FindByUsername(ctx, un)
	if err != nil || u == nil {
		// Record failed attempt and calculate remaining
		if s.config.LockoutEnabled {
			_ = s.recordFailedLoginAttempt(ctx, un, appID, ip, ua)

			// Get current attempt count and calculate remaining
			since := time.Now().Add(-s.config.FailedAttemptWindow)
			count, _ := s.usernameRepo.GetFailedAttempts(ctx, un, appID, since)
			attemptsRemaining := s.config.MaxFailedAttempts - count

			// Audit invalid credentials
			if s.audit != nil {
				_ = s.audit.Log(ctx, nil, string(audit.ActionUsernameInvalidCredentials),
					fmt.Sprintf("username:%s reason:user_not_found attempts:%d remaining:%d", un, count, attemptsRemaining), ip, ua,
					fmt.Sprintf(`{"username":"%s","reason":"user_not_found","attempts":%d,"remaining":%d}`, un, count, attemptsRemaining))
			}

			return nil, errs.InvalidCredentialsWithAttempts(attemptsRemaining)
		}

		// Lockout not enabled - return basic error
		// Audit invalid credentials
		if s.audit != nil {
			_ = s.audit.Log(ctx, nil, "username_invalid_credentials",
				fmt.Sprintf("username:%s reason:user_not_found", un), ip, ua,
				fmt.Sprintf(`{"username":"%s","reason":"user_not_found"}`, un))
		}

		return nil, errs.InvalidCredentials()
	}

	// Check account lockout
	if err := s.checkAccountLockout(ctx, u.ID); err != nil {
		// Check if it's an AccountLockoutError
		lockoutErr := &AccountLockoutError{}
		if errors.As(err, &lockoutErr) {
			// Audit lockout attempt
			if s.audit != nil {
				uid := u.ID
				_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameAccountLocked),
					fmt.Sprintf("username:%s user_id:%s", un, uid.String()), ip, ua,
					fmt.Sprintf(`{"username":"%s","user_id":"%s","locked_until":"%s"}`,
						un, uid.String(), lockoutErr.LockedUntil.Format(time.RFC3339)))
			}

			return nil, errs.AccountLockedWithTime(
				fmt.Sprintf("locked for %d minutes", lockoutErr.LockedMinutes),
				lockoutErr.LockedUntil,
			)
		}

		return nil, fmt.Errorf("lockout check failed: %w", err)
	}

	// Check password expiry
	if err := s.checkPasswordExpiry(ctx, u); err != nil {
		// Audit password expired
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernamePasswordExpired),
				fmt.Sprintf("username:%s user_id:%s", un, uid.String()), ip, ua,
				fmt.Sprintf(`{"username":"%s","user_id":"%s"}`, un, uid.String()))
		}

		return nil, err
	}

	// Verify password
	if ok := crypto.CheckPassword(password, u.PasswordHash); !ok {
		// Record failed attempt
		if s.config.LockoutEnabled {
			_ = s.recordFailedLoginAttempt(ctx, un, appID, ip, ua)

			// Check if we should lock the account
			if err := s.handleAccountLockout(ctx, u.ID, un, appID); err != nil {
				// Account was locked
				if s.audit != nil {
					uid := u.ID
					_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameSigninFailed),
						fmt.Sprintf("username:%s user_id:%s reason:invalid_password_locked", un, uid.String()),
						ip, ua,
						fmt.Sprintf(`{"username":"%s","user_id":"%s","reason":"invalid_password","locked":true}`,
							un, uid.String()))
				}

				return nil, errs.InvalidCredentials()
			}

			// Get current attempt count and calculate remaining
			since := time.Now().Add(-s.config.FailedAttemptWindow)
			count, _ := s.usernameRepo.GetFailedAttempts(ctx, un, appID, since)
			attemptsRemaining := s.config.MaxFailedAttempts - count

			// Audit failed attempt recorded
			if s.audit != nil {
				uid := u.ID
				_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameFailedAttemptRecorded),
					fmt.Sprintf("username:%s user_id:%s attempts:%d remaining:%d", un, uid.String(), count, attemptsRemaining), ip, ua,
					fmt.Sprintf(`{"username":"%s","user_id":"%s","attempts":%d,"remaining":%d}`, un, uid.String(), count, attemptsRemaining))
			}

			// Audit invalid credentials
			if s.audit != nil {
				uid := u.ID
				_ = s.audit.Log(ctx, &uid, "username_invalid_credentials",
					fmt.Sprintf("username:%s user_id:%s reason:invalid_password", un, uid.String()),
					ip, ua,
					fmt.Sprintf(`{"username":"%s","user_id":"%s","reason":"invalid_password"}`, un, uid.String()))
			}

			// Return error with attempts remaining warning
			return nil, errs.InvalidCredentialsWithAttempts(attemptsRemaining)
		}

		// Lockout not enabled - return basic error
		// Audit invalid credentials
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, "username_invalid_credentials",
				fmt.Sprintf("username:%s user_id:%s reason:invalid_password", un, uid.String()),
				ip, ua,
				fmt.Sprintf(`{"username":"%s","user_id":"%s","reason":"invalid_password"}`, un, uid.String()))
		}

		return nil, errs.InvalidCredentials()
	}

	// Password is correct - clear failed attempts
	if s.config.LockoutEnabled {
		if err := s.clearFailedAttempts(ctx, un, appID); err != nil {
			// Log error but don't fail signin
		} else {
			// Audit cleared attempts
			if s.audit != nil {
				uid := u.ID
				_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameFailedAttemptsCleared),
					fmt.Sprintf("username:%s user_id:%s", un, uid.String()), ip, ua,
					fmt.Sprintf(`{"username":"%s","user_id":"%s"}`, un, uid.String()))
			}
		}
	}

	// Create session
	res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
	if err != nil {
		// Audit session creation failure
		if s.audit != nil {
			uid := u.ID
			_ = s.audit.Log(ctx, &uid, "username_signin_failed",
				fmt.Sprintf("username:%s user_id:%s error:%s", un, uid.String(), err.Error()),
				ip, ua,
				fmt.Sprintf(`{"username":"%s","user_id":"%s","error":"%s"}`, un, uid.String(), err.Error()))
		}

		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Audit successful signin
	if s.audit != nil {
		uid := u.ID
		_ = s.audit.Log(ctx, &uid, string(audit.ActionUsernameSigninSuccess),
			fmt.Sprintf("username:%s user_id:%s session_id:%s", un, uid.String(), res.Session.ID.String()),
			ip, ua,
			fmt.Sprintf(`{"username":"%s","user_id":"%s","session_id":"%s","remember":%t,"app_id":"%s","org_id":"%s"}`,
				un, uid.String(), res.Session.ID.String(), remember, appID.String(), func() string {
					if orgID != xid.NilID() {
						return orgID.String()
					}

					return ""
				}()))
	}

	return res, nil
}

// =============================================================================
// ERROR TYPES
// =============================================================================

// AccountLockoutError represents an account lockout error.
type AccountLockoutError struct {
	LockedUntil   time.Time
	LockedMinutes int
}

func (e *AccountLockoutError) Error() string {
	return fmt.Sprintf("account locked until %s (%d minutes)", e.LockedUntil.Format(time.RFC3339), e.LockedMinutes)
}
