package mfa

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
)

// DefaultRecoveryCodeCount is the number of recovery codes generated per enrollment.
const DefaultRecoveryCodeCount = 8

// recoveryCodeAlphabet is the character set for recovery codes (no ambiguous chars).
const recoveryCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// recoveryCodeLength is the length of each plaintext recovery code.
const recoveryCodeLength = 8

// RecoveryCode represents a single MFA recovery (backup) code.
type RecoveryCode struct {
	ID        id.RecoveryCodeID `json:"id"`
	UserID    id.UserID         `json:"user_id"`
	CodeHash  string            `json:"-"`    // bcrypt hash of the plaintext code
	Used      bool              `json:"used"` // true after one-time use
	UsedAt    *time.Time        `json:"used_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// GenerateRecoveryCodes creates a set of plaintext recovery codes and their
// bcrypt-hashed RecoveryCode records. Returns both the records (for storage)
// and the plaintext codes (to show the user once).
func GenerateRecoveryCodes(userID id.UserID, count int) ([]*RecoveryCode, []string, error) {
	if count <= 0 {
		count = DefaultRecoveryCodeCount
	}

	now := time.Now()
	codes := make([]*RecoveryCode, 0, count)
	plaintexts := make([]string, 0, count)

	for i := 0; i < count; i++ {
		plain, err := generateRandomCode(recoveryCodeLength)
		if err != nil {
			return nil, nil, fmt.Errorf("mfa: generate recovery code: %w", err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("mfa: hash recovery code: %w", err)
		}

		codes = append(codes, &RecoveryCode{
			ID:        id.NewRecoveryCodeID(),
			UserID:    userID,
			CodeHash:  string(hash),
			Used:      false,
			CreatedAt: now,
		})
		plaintexts = append(plaintexts, plain)
	}

	return codes, plaintexts, nil
}

// VerifyRecoveryCode checks if the given plaintext code matches the hashed code.
func VerifyRecoveryCode(plaintext string, code *RecoveryCode) bool {
	if code.Used {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(code.CodeHash), []byte(plaintext)) == nil
}

// generateRandomCode creates a cryptographically random code of the given length.
func generateRandomCode(length int) (string, error) {
	alphabetLen := big.NewInt(int64(len(recoveryCodeAlphabet)))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		result[i] = recoveryCodeAlphabet[idx.Int64()]
	}

	return string(result), nil
}
