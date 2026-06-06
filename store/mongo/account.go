package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Verification
// ──────────────────────────────────────────────────

// CreateVerification persists a new verification token.
func (s *Store) CreateVerification(ctx context.Context, v *account.Verification) error {
	m := toVerificationModel(v)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create verification: %w", err)
	}

	return nil
}

// GetVerification returns a verification by token.
func (s *Store) GetVerification(ctx context.Context, token string) (*account.Verification, error) {
	var m verificationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"token": token}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get verification: %w", err)
	}

	return fromVerificationModel(&m)
}

// ConsumeVerification marks a verification token as consumed.
func (s *Store) ConsumeVerification(ctx context.Context, token string) error {
	res, err := s.mdb.NewUpdate((*verificationModel)(nil)).
		Filter(bson.M{"token": token, "consumed": false}).
		Set("consumed", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: consume verification: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// GetActiveEmailVerification returns the most recent unconsumed, unexpired
// email verification for a user (OTP lookup by user, not token).
func (s *Store) GetActiveEmailVerification(ctx context.Context, userID id.UserID) (*account.Verification, error) {
	var m verificationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"user_id":    userID.String(),
			"type":       string(account.VerificationEmail),
			"consumed":   false,
			"expires_at": bson.M{"$gt": time.Now()},
		}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get active email verification: %w", err)
	}

	return fromVerificationModel(&m)
}

// UpdateVerification persists mutable fields (attempts, consumed).
func (s *Store) UpdateVerification(ctx context.Context, v *account.Verification) error {
	_, err := s.mdb.NewUpdate((*verificationModel)(nil)).
		Filter(bson.M{"_id": v.ID.String()}).
		Set("attempts", v.Attempts).
		Set("consumed", v.Consumed).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update verification: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────
// Password Reset
// ──────────────────────────────────────────────────

// CreatePasswordReset persists a new password reset token.
func (s *Store) CreatePasswordReset(ctx context.Context, pr *account.PasswordReset) error {
	m := toPasswordResetModel(pr)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create password reset: %w", err)
	}

	return nil
}

// GetPasswordReset returns a password reset by token.
func (s *Store) GetPasswordReset(ctx context.Context, token string) (*account.PasswordReset, error) {
	var m passwordResetModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"token": token}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get password reset: %w", err)
	}

	return fromPasswordResetModel(&m)
}

// ConsumePasswordReset marks a password reset token as consumed.
func (s *Store) ConsumePasswordReset(ctx context.Context, token string) error {
	res, err := s.mdb.NewUpdate((*passwordResetModel)(nil)).
		Filter(bson.M{"token": token, "consumed": false}).
		Set("consumed", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: consume password reset: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}
