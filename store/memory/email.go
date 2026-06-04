package memory

import (
	"context"
	"sort"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// User Email Store (multiple emails per account)
// ──────────────────────────────────────────────────

// liveEmail reports whether an email row is active (not soft-deleted).
func liveEmail(e *user.UserEmail) bool { return e != nil && e.DeletedAt == nil }

// findEmailLocked returns the live email row matching (appID, envID, email).
// A nil envID matches within the app across all environments (used by lookup
// flows with no environment in scope). Callers must hold s.mu.
func (s *Store) findEmailLocked(appID id.AppID, envID id.EnvironmentID, email string) *user.UserEmail {
	norm := user.NormalizeEmail(email)
	for _, e := range s.userEmails {
		if !liveEmail(e) {
			continue
		}
		if e.AppID.String() != appID.String() || e.Email != norm {
			continue
		}
		if !envID.IsNil() && e.EnvID.String() != envID.String() {
			continue
		}
		return e
	}
	return nil
}

// CreateUserWithPrimaryEmail creates a user together with its primary email
// row. The email address must be free within (app_id, env_id).
func (s *Store) CreateUserWithPrimaryEmail(_ context.Context, u *user.User, primary *user.UserEmail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing := s.findEmailLocked(primary.AppID, primary.EnvID, primary.Email); existing != nil {
		return account.ErrEmailTaken
	}

	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = u.CreatedAt
	s.users[u.ID.String()] = u

	cp := *primary
	cp.Email = user.NormalizeEmail(primary.Email)
	cp.IsPrimary = true
	if cp.ID.IsNil() {
		cp.ID = id.NewUserEmailID()
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now
	s.userEmails[cp.ID.String()] = &cp
	return nil
}

// AddUserEmail attaches an additional email to a user. The address must be
// free within (app_id, env_id), else account.ErrEmailTaken.
func (s *Store) AddUserEmail(_ context.Context, e *user.UserEmail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing := s.findEmailLocked(e.AppID, e.EnvID, e.Email); existing != nil {
		return account.ErrEmailTaken
	}

	cp := *e
	cp.Email = user.NormalizeEmail(e.Email)
	if cp.ID.IsNil() {
		cp.ID = id.NewUserEmailID()
	}
	now := time.Now()
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now
	s.userEmails[cp.ID.String()] = &cp
	return nil
}

// GetUserByAnyEmail resolves the user owning any (primary or secondary) email
// matching (appID, envID, email).
func (s *Store) GetUserByAnyEmail(_ context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e := s.findEmailLocked(appID, envID, email)
	if e == nil {
		return nil, store.ErrNotFound
	}
	u, ok := s.users[e.UserID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return u, nil
}

// GetUserEmailRecord returns the email row matching (appID, envID, email).
func (s *Store) GetUserEmailRecord(_ context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.UserEmail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e := s.findEmailLocked(appID, envID, email)
	if e == nil {
		return nil, store.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

// GetUserEmails returns all live emails for a user, primary first then oldest.
func (s *Store) GetUserEmails(_ context.Context, userID id.UserID) ([]*user.UserEmail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*user.UserEmail
	for _, e := range s.userEmails {
		if !liveEmail(e) || e.UserID.String() != userID.String() {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsPrimary != out[j].IsPrimary {
			return out[i].IsPrimary
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// MarkUserEmailVerified marks a user's email verified, mirroring onto the user
// record when the email is primary.
func (s *Store) MarkUserEmailVerified(_ context.Context, userID id.UserID, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	norm := user.NormalizeEmail(email)
	for _, e := range s.userEmails {
		if !liveEmail(e) || e.UserID.String() != userID.String() || e.Email != norm {
			continue
		}
		now := time.Now()
		e.Verified = true
		e.UpdatedAt = now
		if e.IsPrimary {
			if u, ok := s.users[userID.String()]; ok {
				u.EmailVerified = true
				u.UpdatedAt = now
			}
		}
		return nil
	}
	return store.ErrNotFound
}

// SetPrimaryEmail makes a verified, user-owned email the primary, clearing the
// previous primary and mirroring onto the user record.
func (s *Store) SetPrimaryEmail(_ context.Context, userID id.UserID, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	norm := user.NormalizeEmail(email)
	var target *user.UserEmail
	owned := make([]*user.UserEmail, 0)
	for _, e := range s.userEmails {
		if liveEmail(e) && e.UserID.String() == userID.String() {
			owned = append(owned, e)
			if e.Email == norm {
				target = e
			}
		}
	}
	if target == nil {
		return store.ErrNotFound
	}
	if !target.Verified {
		return account.ErrEmailNotVerified
	}

	now := time.Now()
	for _, e := range owned {
		if e.IsPrimary && e != target {
			e.IsPrimary = false
			e.UpdatedAt = now
		}
	}
	target.IsPrimary = true
	target.UpdatedAt = now
	if u, ok := s.users[userID.String()]; ok {
		u.Email = target.Email
		u.EmailVerified = target.Verified
		u.UpdatedAt = now
	}
	return nil
}

// DeleteUserEmail soft-deletes a non-primary email. Deleting the primary is
// refused with store.ErrConflict.
func (s *Store) DeleteUserEmail(_ context.Context, userID id.UserID, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	norm := user.NormalizeEmail(email)
	for _, e := range s.userEmails {
		if !liveEmail(e) || e.UserID.String() != userID.String() || e.Email != norm {
			continue
		}
		if e.IsPrimary {
			return store.ErrConflict
		}
		now := time.Now()
		e.DeletedAt = &now
		e.UpdatedAt = now
		return nil
	}
	return store.ErrNotFound
}
