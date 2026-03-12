package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Sentinel errors for account operations.
var (
	ErrInvalidCredentials = errors.New("account: invalid credentials")
	ErrEmailTaken         = errors.New("account: email already taken")
	ErrUsernameTaken      = errors.New("account: username already taken")
	ErrUserBanned         = errors.New("account: user is banned")
	ErrAccountLocked      = errors.New("account: account temporarily locked due to too many failed attempts")
	ErrSessionExpired     = errors.New("account: session expired")
	ErrWeakPassword       = errors.New("account: password does not meet requirements")
	ErrPasswordExpired    = errors.New("account: password has expired and must be changed")
	ErrPasswordReused     = errors.New("account: password was recently used and cannot be reused")
)

// PasswordPolicy configures password validation rules.
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit     bool
	RequireSpecial   bool
	BcryptCost       int
	Algorithm        string       // "bcrypt" (default) or "argon2id"
	Argon2Params     Argon2Params // Used when Algorithm = "argon2id"
	CheckBreached    bool         // Enable HIBP breach checking
}

// ValidatePassword checks a password against the policy.
func (p PasswordPolicy) ValidatePassword(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("%w: minimum length %d", ErrWeakPassword, p.MinLength)
	}
	if p.RequireUppercase && !containsRune(password, unicode.IsUpper) {
		return fmt.Errorf("%w: must contain uppercase letter", ErrWeakPassword)
	}
	if p.RequireLowercase && !containsRune(password, unicode.IsLower) {
		return fmt.Errorf("%w: must contain lowercase letter", ErrWeakPassword)
	}
	if p.RequireDigit && !containsRune(password, unicode.IsDigit) {
		return fmt.Errorf("%w: must contain digit", ErrWeakPassword)
	}
	if p.RequireSpecial && !containsRune(password, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		return fmt.Errorf("%w: must contain special character", ErrWeakPassword)
	}
	return nil
}

func containsRune(s string, f func(rune) bool) bool {
	for _, r := range s {
		if f(r) {
			return true
		}
	}
	return false
}

// HashPassword hashes a password using bcrypt (legacy convenience function).
func HashPassword(password string, cost int) (string, error) {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("account: hash password: %w", err)
	}
	return string(hash), nil
}

// HashPasswordWithPolicy hashes a password using the algorithm specified in the policy.
func HashPasswordWithPolicy(password string, policy PasswordPolicy) (string, error) {
	switch policy.Algorithm {
	case "argon2id":
		params := policy.Argon2Params
		if params.Memory == 0 {
			params = DefaultArgon2Params()
		}
		return HashPasswordArgon2(password, params)
	default: // "bcrypt" or empty
		return HashPassword(password, policy.BcryptCost)
	}
}

// CheckPassword verifies a password against a hash, auto-detecting the algorithm
// from the hash prefix. Supports bcrypt ($2a$/$2b$) and Argon2id ($argon2id$).
func CheckPassword(hash, password string) error {
	if IsArgon2Hash(hash) {
		return CheckPasswordArgon2(hash, password)
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// NeedsRehash returns true if the hash was produced by a different algorithm
// than the one currently configured in the policy. This enables transparent
// migration from bcrypt to argon2id (or vice versa) on successful login.
func NeedsRehash(hash string, policy PasswordPolicy) bool {
	switch policy.Algorithm {
	case "argon2id":
		return !IsArgon2Hash(hash)
	default: // "bcrypt"
		return IsArgon2Hash(hash)
	}
}

// SessionConfig holds session token configuration.
type SessionConfig struct {
	TokenTTL           time.Duration
	RefreshTokenTTL    time.Duration
	MaxActiveSessions  int
	RotateRefreshToken bool
}

// NewSession creates a new session for a user.
func NewSession(appID id.AppID, userID id.UserID, cfg SessionConfig) (*session.Session, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}
	refreshToken, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 appID,
		UserID:                userID,
		Token:                 token,
		RefreshToken:          refreshToken,
		ExpiresAt:             now.Add(cfg.TokenTTL),
		RefreshTokenExpiresAt: now.Add(cfg.RefreshTokenTTL),
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

// NewUser creates a new user from a signup request.
func NewUser(req *SignUpRequest, passwordHash string) *user.User {
	now := time.Now()
	u := &user.User{
		ID:           id.NewUserID(),
		AppID:        req.AppID,
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Username:     req.Username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if len(req.Metadata) > 0 {
		u.Metadata = user.Metadata(req.Metadata)
	}
	return u
}

// RefreshSession generates new tokens for an existing session.
// When cfg.RotateRefreshToken is true, both the access token and refresh
// token are regenerated (the old refresh token becomes invalid). When false,
// only the access token is rotated and the refresh token is reused.
func RefreshSession(sess *session.Session, cfg SessionConfig) error {
	token, err := generateSecureToken(32)
	if err != nil {
		return err
	}

	now := time.Now()
	sess.Token = token
	sess.ExpiresAt = now.Add(cfg.TokenTTL)
	sess.UpdatedAt = now

	if cfg.RotateRefreshToken {
		refreshToken, err := generateSecureToken(32)
		if err != nil {
			return err
		}
		sess.RefreshToken = refreshToken
		sess.RefreshTokenExpiresAt = now.Add(cfg.RefreshTokenTTL)
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random hex token.
func generateSecureToken(bytes int) (string, error) { //nolint:unparam // keep bytes configurable for future use
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("account: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateVerificationToken generates a new verification token string.
func GenerateVerificationToken() (string, error) {
	return generateSecureToken(32)
}

// NewVerification creates a new email/phone verification record.
func NewVerification(_ context.Context, appID id.AppID, userID id.UserID, vType VerificationType, ttl time.Duration) (*Verification, error) {
	token, err := GenerateVerificationToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &Verification{
		ID:        id.NewVerificationID(),
		AppID:     appID,
		UserID:    userID,
		Token:     token,
		Type:      vType,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

// NewPasswordReset creates a new password reset record.
func NewPasswordReset(_ context.Context, appID id.AppID, userID id.UserID, ttl time.Duration) (*PasswordReset, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &PasswordReset{
		ID:        id.NewPasswordResetID(),
		AppID:     appID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}
