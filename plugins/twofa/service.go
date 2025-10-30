package twofa

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/rs/xid"
	repo "github.com/xraph/authsome/repository"
)

// Service provides Two-Factor Authentication operations
type Service struct{ repo *repo.TwoFARepository }

func NewService(r *repo.TwoFARepository) *Service { return &Service{repo: r} }

type EnableRequest struct {
	Method string // "totp" or "otp"
}

type VerifyRequest struct {
	Code string
}

// Status provides the current 2FA status and device trust state
type Status struct {
	Enabled bool
	Method  string
	Trusted bool
}

// Enable sets up 2FA for a user using the specified method
func (s *Service) Enable(ctx context.Context, userID string, req *EnableRequest) (*TOTPSecret, error) {
	_ = ctx
	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, err
	}
	switch req.Method {
	case "totp":
		// Generate and store real secret
		bundle, genErr := s.GenerateTOTPSecret(ctx, userID)
		if genErr != nil {
			return nil, genErr
		}
		if err := s.repo.UpsertSecret(ctx, uid, "totp", bundle.Secret, true); err != nil {
			return nil, err
		}
		return bundle, nil
	case "otp":
		if err := s.repo.UpsertSecret(ctx, uid, "otp", "", true); err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, nil
	}
}

// Verify validates a provided 2FA code
func (s *Service) Verify(ctx context.Context, userID string, req *VerifyRequest) (bool, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return false, err
	}
	// Check configured method first
	sec, err := s.repo.GetSecret(ctx, uid)
	if err != nil {
		return false, err
	}
	if sec != nil && sec.Enabled {
		switch sec.Method {
		case "totp":
			ok, verr := s.VerifyTOTP(userID, req.Code)
			if verr != nil {
				return false, verr
			}
			if ok {
				return true, nil
			}
		case "otp":
			ok, verr := s.VerifyOTP(ctx, userID, req.Code)
			if verr != nil {
				return false, verr
			}
			if ok {
				return true, nil
			}
		}
	}
	// Fallback to backup code verification
	ok, berr := s.repo.VerifyAndUseBackupCode(ctx, uid, req.Code)
	if berr != nil {
		return false, berr
	}
	return ok, nil
}

// GetStatus returns 2FA enabled/method and whether device is trusted
func (s *Service) GetStatus(ctx context.Context, userID, deviceID string) (*Status, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, err
	}
	sec, err := s.repo.GetSecret(ctx, uid)
	if err != nil {
		return nil, err
	}
	st := &Status{Enabled: false, Method: "", Trusted: false}
	if sec != nil && sec.Enabled {
		st.Enabled = true
		st.Method = sec.Method
		if deviceID != "" {
			trusted, _ := s.repo.IsTrustedDevice(ctx, uid, deviceID, time.Now())
			st.Trusted = trusted
		}
	}
	return st, nil
}

// Disable removes 2FA for a user
func (s *Service) Disable(ctx context.Context, userID string) error {
	uid, err := xid.FromString(userID)
	if err != nil {
		return err
	}
	return s.repo.DisableSecret(ctx, uid)
}

// GenerateBackupCodes returns a set of backup recovery codes
func (s *Service) GenerateBackupCodes(ctx context.Context, userID string, count int) ([]string, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 10
	}
	// Generate codes and store hashed versions
	hashes := make([]string, 0, count)
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		c := "backup-" + xid.New().String()
		codes = append(codes, c)
		hashes = append(hashes, hashCode(c))
	}
	if err := s.repo.CreateBackupCodes(ctx, uid, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// hashCode provides a basic hashing for backup codes
func hashCode(s string) string {
	// Lightweight SHA-256 hex encoding
	// In later phases, add salt and stretching
	return sha256Hex(s)
}

// sha256Hex returns hex-encoded SHA-256 of input
func sha256Hex(in string) string {
	// Local inline to avoid extra utility imports
	h := sha256.Sum256([]byte(in))
	return fmt.Sprintf("%x", h[:])
}

// Trusted devices helpers (stubs)
func (s *Service) MarkTrusted(ctx context.Context, userID, deviceID string, days int) error {
	uid, err := xid.FromString(userID)
	if err != nil {
		return err
	}
	return s.repo.MarkTrustedDevice(ctx, uid, deviceID, time.Now().Add(time.Duration(days)*24*time.Hour))
}

func (s *Service) IsTrusted(ctx context.Context, userID, deviceID string) (bool, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return false, err
	}
	return s.repo.IsTrustedDevice(ctx, uid, deviceID, time.Now())
}
