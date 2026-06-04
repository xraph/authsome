package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// User email model (multiple emails per account)
// ──────────────────────────────────────────────────
//
// MongoDB lacks the multi-statement transactions the SQL backends use for the
// user+primary-email and set-primary flows (outside a replica set), so those
// methods perform best-effort ordered writes — consistent with this store's
// WithTx no-op stance. The unique index on (app_id, env_id, email) still
// guarantees an address belongs to at most one account.

type userEmailModel struct {
	grove.BaseModel `grove:"table:authsome_user_emails"`

	ID        string     `grove:"id,pk"        bson:"_id"`
	UserID    string     `grove:"user_id"      bson:"user_id"`
	AppID     string     `grove:"app_id"       bson:"app_id"`
	EnvID     string     `grove:"env_id"       bson:"env_id"`
	Email     string     `grove:"email"        bson:"email"`
	Verified  bool       `grove:"verified"     bson:"verified"`
	IsPrimary bool       `grove:"is_primary"   bson:"is_primary"`
	Source    string     `grove:"source"       bson:"source"`
	CreatedAt time.Time  `grove:"created_at"   bson:"created_at"`
	UpdatedAt time.Time  `grove:"updated_at"   bson:"updated_at"`
	DeletedAt *time.Time `grove:"deleted_at"   bson:"deleted_at,omitempty"`
}

func toUserEmailModel(e *user.UserEmail) *userEmailModel {
	now := now()
	m := &userEmailModel{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		AppID:     e.AppID.String(),
		EnvID:     e.EnvID.String(),
		Email:     user.NormalizeEmail(e.Email),
		Verified:  e.Verified,
		IsPrimary: e.IsPrimary,
		Source:    e.Source,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
	}
	if m.ID == "" {
		m.ID = id.NewUserEmailID().String()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	return m
}

func fromUserEmailModel(m *userEmailModel) (*user.UserEmail, error) {
	ueID, err := id.ParseUserEmailID(m.ID)
	if err != nil {
		return nil, err
	}
	uid, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
	return &user.UserEmail{
		ID:        ueID,
		UserID:    uid,
		AppID:     appID,
		EnvID:     envID,
		Email:     m.Email,
		Verified:  m.Verified,
		IsPrimary: m.IsPrimary,
		Source:    m.Source,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// User email store methods
// ──────────────────────────────────────────────────

func (s *Store) CreateUserWithPrimaryEmail(ctx context.Context, u *user.User, primary *user.UserEmail) error {
	if err := s.CreateUser(ctx, u); err != nil {
		return err
	}
	pm := toUserEmailModel(primary)
	pm.IsPrimary = true
	if _, err := s.mdb.NewInsert(pm).Exec(ctx); err != nil {
		if mapped := mapWriteErr(err); mapped != err { //nolint:errorlint // identity check
			return mapped
		}
		return fmt.Errorf("authsome/mongo: create primary email: %w", err)
	}
	return nil
}

func (s *Store) AddUserEmail(ctx context.Context, e *user.UserEmail) error {
	if _, err := s.mdb.NewInsert(toUserEmailModel(e)).Exec(ctx); err != nil {
		if mapped := mapWriteErr(err); mapped != err { //nolint:errorlint // identity check
			return mapped
		}
		return fmt.Errorf("authsome/mongo: add user email: %w", err)
	}
	return nil
}

func (s *Store) GetUserByAnyEmail(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.User, error) {
	var m userEmailModel
	err := s.mdb.NewFind(&m).
		Filter(userEmailFilter(appID, envID, email)).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get user by any email: %w", err)
	}
	uid, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, uid)
}

func (s *Store) GetUserEmailRecord(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.UserEmail, error) {
	var m userEmailModel
	err := s.mdb.NewFind(&m).
		Filter(userEmailFilter(appID, envID, email)).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get user email record: %w", err)
	}
	return fromUserEmailModel(&m)
}

// userEmailFilter builds the lookup filter. A nil envID matches within the app
// across all environments (used by lookup flows with no environment in scope).
func userEmailFilter(appID id.AppID, envID id.EnvironmentID, email string) bson.M {
	f := bson.M{
		"app_id":     appID.String(),
		"email":      user.NormalizeEmail(email),
		"deleted_at": nil,
	}
	if !envID.IsNil() {
		f["env_id"] = envID.String()
	}
	return f
}

func (s *Store) GetUserEmails(ctx context.Context, userID id.UserID) ([]*user.UserEmail, error) {
	var models []userEmailModel
	err := s.mdb.NewFind(&models).
		Filter(bson.M{"user_id": userID.String(), "deleted_at": nil}).
		Sort(bson.D{{Key: "is_primary", Value: -1}, {Key: "created_at", Value: 1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: get user emails: %w", err)
	}
	out := make([]*user.UserEmail, 0, len(models))
	for i := range models {
		e, err := fromUserEmailModel(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) MarkUserEmailVerified(ctx context.Context, userID id.UserID, email string) error {
	var m userEmailModel
	err := s.mdb.NewFind(&m).
		Filter(bson.M{"user_id": userID.String(), "email": user.NormalizeEmail(email), "deleted_at": nil}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return store.ErrNotFound
		}
		return fmt.Errorf("authsome/mongo: find email to verify: %w", err)
	}

	t := now()
	if _, err := s.mdb.NewUpdate((*userEmailModel)(nil)).
		Filter(bson.M{"_id": m.ID}).
		Set("verified", true).
		Set("updated_at", t).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: mark email verified: %w", err)
	}
	if m.IsPrimary {
		if _, err := s.mdb.NewUpdate((*userModel)(nil)).
			Filter(bson.M{"_id": userID.String()}).
			Set("email_verified", true).
			Set("updated_at", t).
			Exec(ctx); err != nil {
			return fmt.Errorf("authsome/mongo: mirror email_verified: %w", err)
		}
	}
	return nil
}

func (s *Store) SetPrimaryEmail(ctx context.Context, userID id.UserID, email string) error {
	norm := user.NormalizeEmail(email)
	var target userEmailModel
	err := s.mdb.NewFind(&target).
		Filter(bson.M{"user_id": userID.String(), "email": norm, "deleted_at": nil}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return store.ErrNotFound
		}
		return fmt.Errorf("authsome/mongo: find email to set primary: %w", err)
	}
	if !target.Verified {
		return account.ErrEmailNotVerified
	}

	t := now()
	// Clear the existing primary BEFORE setting the new one so the partial
	// unique index on (user_id) WHERE is_primary never sees two primaries.
	if _, err := s.mdb.NewUpdate((*userEmailModel)(nil)).
		Filter(bson.M{"user_id": userID.String(), "is_primary": true, "deleted_at": nil}).
		Set("is_primary", false).
		Set("updated_at", t).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: clear primary email: %w", err)
	}
	if _, err := s.mdb.NewUpdate((*userEmailModel)(nil)).
		Filter(bson.M{"_id": target.ID}).
		Set("is_primary", true).
		Set("updated_at", t).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: set primary email: %w", err)
	}
	if _, err := s.mdb.NewUpdate((*userModel)(nil)).
		Filter(bson.M{"_id": userID.String()}).
		Set("email", target.Email).
		Set("email_verified", target.Verified).
		Set("updated_at", t).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: mirror primary email: %w", err)
	}
	return nil
}

func (s *Store) DeleteUserEmail(ctx context.Context, userID id.UserID, email string) error {
	var m userEmailModel
	err := s.mdb.NewFind(&m).
		Filter(bson.M{"user_id": userID.String(), "email": user.NormalizeEmail(email), "deleted_at": nil}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return store.ErrNotFound
		}
		return fmt.Errorf("authsome/mongo: find email to delete: %w", err)
	}
	if m.IsPrimary {
		return store.ErrConflict
	}
	// Hard-delete: the email-uniqueness index is non-partial on mongo (partial
	// filters can't express $exists:false), so a soft-deleted row would keep
	// the address reserved. Removing the doc frees it, matching the SQL
	// backends' observable behavior.
	if _, err := s.mdb.NewDelete((*userEmailModel)(nil)).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: delete user email: %w", err)
	}
	return nil
}
