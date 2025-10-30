package twofa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// BackupCodes generates cryptographically secure recovery codes for 2FA
func (s *Service) BackupCodes(ctx context.Context, userID string, count int) ([]string, error) {
	if count <= 0 || count > 20 {
		count = 10 // Default to 10 codes
	}

	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Invalidate any existing backup codes
	_, err = s.repo.DB().NewDelete().
		Model((*schema.BackupCode)(nil)).
		Where("user_id = ? AND used_at IS NULL", uid).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to clear old backup codes: %w", err)
	}

	// Generate new backup codes
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate 8-character alphanumeric code
		code, err := generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code

		// Hash and store the code
		codeHash := hashBackupCode(code)
		backupCode := &schema.BackupCode{
			ID:       xid.New(),
			UserID:   uid,
			CodeHash: codeHash,
			UsedAt:   nil,
		}
		backupCode.AuditableModel.CreatedBy = uid
		backupCode.AuditableModel.UpdatedBy = uid

		_, err = s.repo.DB().NewInsert().Model(backupCode).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to store backup code: %w", err)
		}
	}

	return codes, nil
}

// VerifyBackupCode validates a backup code and marks it as used
func (s *Service) VerifyBackupCode(ctx context.Context, userID, code string) (bool, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	// Normalize code (uppercase, remove spaces/dashes)
	code = normalizeBackupCode(code)
	codeHash := hashBackupCode(code)

	// Find unused backup code
	var backupCode schema.BackupCode
	err = s.repo.DB().NewSelect().
		Model(&backupCode).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", uid, codeHash).
		Scan(ctx)

	if err != nil {
		return false, nil // Code not found or already used
	}

	// Mark code as used
	now := time.Now()
	backupCode.UsedAt = &now
	_, err = s.repo.DB().NewUpdate().
		Model(&backupCode).
		Column("used_at").
		Where("id = ?", backupCode.ID).
		Exec(ctx)

	if err != nil {
		return false, fmt.Errorf("failed to mark backup code as used: %w", err)
	}

	return true, nil
}

// generateBackupCode creates a cryptographically secure 8-character code
func generateBackupCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Excludes ambiguous chars (0, O, I, 1)
	const codeLength = 8

	bytes := make([]byte, codeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	code := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		code[i] = charset[int(bytes[i])%len(charset)]
	}

	// Format as XXXX-XXXX for readability
	return string(code[:4]) + "-" + string(code[4:]), nil
}

// hashBackupCode creates a SHA-256 hash of the backup code
func hashBackupCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

// normalizeBackupCode removes spaces, dashes, and converts to uppercase
func normalizeBackupCode(code string) string {
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")
	return code
}
