// Package memory provides an in-memory implementation of store.Store for testing.
package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"

	"github.com/xraph/grove/migrate"
)

// Compile-time checks.
var (
	_ store.Store       = (*Store)(nil)
	_ environment.Store = (*Store)(nil)
)

// Store is an in-memory implementation of the composite store interface.
type Store struct {
	mu sync.RWMutex

	users             map[string]*user.User
	sessions          map[string]*session.Session
	verifications     map[string]*account.Verification
	passwordResets    map[string]*account.PasswordReset
	apps              map[string]*app.App
	orgs              map[string]*organization.Organization
	members           map[string]*organization.Member
	invitations       map[string]*organization.Invitation
	teams             map[string]*organization.Team
	devices           map[string]*device.Device
	webhooks          map[string]*webhook.Webhook
	notifications     map[string]*notification.Notification
	apikeys           map[string]*apikey.APIKey
	environments      map[string]*environment.Environment
	formConfigs       map[string]*formconfig.FormConfig
	brandingConfigs   map[string]*formconfig.BrandingConfig
	appSessionConfigs map[string]*appsessionconfig.Config
	appClientConfigs  map[string]*appclientconfig.Config
	settingsMap       map[string]*settings.Setting

	serviceAccounts map[string]*serviceaccount.ServiceAccount

	// userEmails maps UserEmail.ID -> record. A user may own several emails;
	// uniqueness is enforced per (app_id, env_id, email) among non-deleted rows.
	userEmails map[string]*user.UserEmail

	// revokedRefreshTokens maps SHA-256(refresh_token) -> revocation record.
	// Used to detect refresh-token replay (RFC 6819 §5.2.2.3): every
	// successful rotation records the OLD hash with reason="rotated", and
	// every refresh exchange checks this map first. A hit means the
	// presented token has already been rotated — replay detected.
	revokedRefreshTokens map[string]*session.RevokedRefreshToken

	// faults is a TEST-ONLY one-shot fault map. Keys are method names
	// (e.g. "DeleteTeam"); the named method consumes and returns the
	// stored error on its next call. Production code MUST NOT depend on
	// this field.
	faults map[string]error
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{
		users:                make(map[string]*user.User),
		sessions:             make(map[string]*session.Session),
		verifications:        make(map[string]*account.Verification),
		passwordResets:       make(map[string]*account.PasswordReset),
		apps:                 make(map[string]*app.App),
		orgs:                 make(map[string]*organization.Organization),
		members:              make(map[string]*organization.Member),
		invitations:          make(map[string]*organization.Invitation),
		teams:                make(map[string]*organization.Team),
		devices:              make(map[string]*device.Device),
		webhooks:             make(map[string]*webhook.Webhook),
		notifications:        make(map[string]*notification.Notification),
		apikeys:              make(map[string]*apikey.APIKey),
		environments:         make(map[string]*environment.Environment),
		formConfigs:          make(map[string]*formconfig.FormConfig),
		brandingConfigs:      make(map[string]*formconfig.BrandingConfig),
		appSessionConfigs:    make(map[string]*appsessionconfig.Config),
		appClientConfigs:     make(map[string]*appclientconfig.Config),
		settingsMap:          make(map[string]*settings.Setting),
		serviceAccounts:      make(map[string]*serviceaccount.ServiceAccount),
		userEmails:           make(map[string]*user.UserEmail),
		revokedRefreshTokens: make(map[string]*session.RevokedRefreshToken),
		faults:               make(map[string]error),
	}
}

// InjectOneShotFault is a TEST-ONLY helper. The next call to <method>
// returns err. Production code must not depend on this.
func (s *Store) InjectOneShotFault(method string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.faults == nil {
		s.faults = make(map[string]error)
	}
	s.faults[method] = err
}

// takeFault consumes a one-shot fault for method, if one was injected.
// Returns nil if no fault was queued.
func (s *Store) takeFault(method string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err, ok := s.faults[method]; ok {
		delete(s.faults, method)
		return err
	}
	return nil
}

// orgTxSnapshot captures only the organization-related tables (orgs,
// members, invitations, teams). It exists to support WithTx on the memory
// store and intentionally does NOT cover other tables; do not use this
// WithTx for cross-table mutations until the snapshot is broadened.
//
// Map values are shared pointers (shallow copy). fn must replace map
// entries (insert/delete) rather than mutate struct fields in place;
// in-place mutations to a *Member, *Team, etc. survive rollback because
// the restored map still points to the same struct.
type orgTxSnapshot struct {
	orgs        map[string]*organization.Organization
	members     map[string]*organization.Member
	invitations map[string]*organization.Invitation
	teams       map[string]*organization.Team
}

func (s *Store) snapshotOrgState() *orgTxSnapshot {
	cpOrg := make(map[string]*organization.Organization, len(s.orgs))
	for k, v := range s.orgs {
		cpOrg[k] = v
	}
	cpMem := make(map[string]*organization.Member, len(s.members))
	for k, v := range s.members {
		cpMem[k] = v
	}
	cpInv := make(map[string]*organization.Invitation, len(s.invitations))
	for k, v := range s.invitations {
		cpInv[k] = v
	}
	cpTeam := make(map[string]*organization.Team, len(s.teams))
	for k, v := range s.teams {
		cpTeam[k] = v
	}
	return &orgTxSnapshot{orgs: cpOrg, members: cpMem, invitations: cpInv, teams: cpTeam}
}

func (s *Store) restoreOrgState(snap *orgTxSnapshot) {
	s.orgs = snap.orgs
	s.members = snap.members
	s.invitations = snap.invitations
	s.teams = snap.teams
}

// WithTx runs fn against the memory store with snapshot/restore semantics
// for the four organization-related tables (orgs, members, invitations,
// teams). On error, those tables are restored to their pre-fn state.
//
// snapshotOrgState covers only organization tables; do not use this
// WithTx for cross-table mutations until the snapshot is broadened.
//
// Concurrency: NOT safe for concurrent calls — the lock is released
// while fn runs, so two concurrent WithTx calls each take their own
// snapshot and rollback is last-writer-wins on the four org tables.
// Real backends (postgres/sqlite/mongo) will get serializability from
// the DB once those WithTx implementations are wired; the memory store
// is only intended for single-threaded test paths.
func (s *Store) WithTx(_ context.Context, fn func(tx organization.Store) error) error {
	s.mu.Lock()
	snap := s.snapshotOrgState()
	s.mu.Unlock()

	if err := fn(s); err != nil {
		s.mu.Lock()
		s.restoreOrgState(snap)
		s.mu.Unlock()
		return err
	}
	return nil
}

// DeleteOrganizationCascade deletes the organization and all of its
// dependent rows (members, teams, invitations) atomically via the
// snapshot/restore WithTx wrapper. On any inner-store error (e.g. a
// fault injection) the entire cascade rolls back.
//
// Returns nil if the org doesn't exist (idempotent — matches the
// SQL-backend behaviour where DELETE on a missing PK is a no-op).
func (s *Store) DeleteOrganizationCascade(ctx context.Context, orgID id.OrgID) error {
	return s.WithTx(ctx, func(tx organization.Store) error {
		if members, err := tx.ListMembers(ctx, orgID); err == nil {
			for _, m := range members {
				if m == nil {
					continue
				}
				if err := tx.DeleteMember(ctx, m.ID); err != nil {
					return fmt.Errorf("delete member %s: %w", m.ID, err)
				}
			}
		}
		if teams, err := tx.ListTeams(ctx, orgID); err == nil {
			for _, t := range teams {
				if t == nil {
					continue
				}
				if err := tx.DeleteTeam(ctx, t.ID); err != nil {
					return fmt.Errorf("delete team %s: %w", t.ID, err)
				}
			}
		}
		if invs, err := tx.ListInvitations(ctx, orgID); err == nil {
			for _, inv := range invs {
				if inv == nil {
					continue
				}
				if err := tx.DeleteInvitation(ctx, inv.ID); err != nil {
					return fmt.Errorf("delete invitation %s: %w", inv.ID, err)
				}
			}
		}
		// Idempotent: ignore not-found on the org row delete.
		if err := tx.DeleteOrganization(ctx, orgID); err != nil && !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("delete org row: %w", err)
		}
		return nil
	})
}

func (s *Store) Migrate(_ context.Context, _ ...*migrate.Group) error { return nil }
func (s *Store) Ping(context.Context) error                           { return nil }
func (s *Store) Close() error                                         { return nil }

// ──────────────────────────────────────────────────
// User Store
// ──────────────────────────────────────────────────

func (s *Store) CreateUser(_ context.Context, u *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	u.UpdatedAt = u.CreatedAt
	s.users[u.ID.String()] = u
	return nil
}

func (s *Store) GetUser(_ context.Context, userID id.UserID) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[userID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return u, nil
}

func (s *Store) GetUserByEmail(_ context.Context, appID id.AppID, email string) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.AppID.String() == appID.String() && u.Email == email {
			return u, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetUserByPhone(_ context.Context, appID id.AppID, phone string) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.AppID.String() == appID.String() && u.Phone == phone {
			return u, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetUserByUsername(_ context.Context, appID id.AppID, username string) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.AppID.String() == appID.String() && u.Username == username {
			return u, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateUser(_ context.Context, u *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.ID.String()]; !ok {
		return store.ErrNotFound
	}
	u.UpdatedAt = time.Now()
	s.users[u.ID.String()] = u
	return nil
}

func (s *Store) DeleteUser(_ context.Context, userID id.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, userID.String())
	return nil
}

func (s *Store) ListUsers(_ context.Context, q *user.Query) (*user.List, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*user.User
	for _, u := range s.users {
		if u.AppID.String() != q.AppID.String() {
			continue
		}
		if !q.EnvID.IsNil() && u.EnvID.String() != q.EnvID.String() {
			continue
		}
		result = append(result, u)
	}
	return &user.List{Users: result, Total: len(result)}, nil
}

// ──────────────────────────────────────────────────
// Session Store
// ──────────────────────────────────────────────────

func (s *Store) CreateSession(_ context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now()
	}
	sess.UpdatedAt = sess.CreatedAt
	s.sessions[sess.ID.String()] = sess
	return nil
}

func (s *Store) GetSession(_ context.Context, sessionID id.SessionID) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return sess, nil
}

func (s *Store) GetSessionByToken(_ context.Context, token string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.sessions {
		if sess.Token == token {
			return sess, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetSessionByRefreshToken(_ context.Context, refreshToken string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.sessions {
		if sess.RefreshToken == refreshToken {
			return sess, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateSession(_ context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sess.ID.String()]; !ok {
		return store.ErrNotFound
	}
	sess.UpdatedAt = time.Now()
	s.sessions[sess.ID.String()] = sess
	return nil
}

func (s *Store) TouchSession(_ context.Context, sessionID id.SessionID, lastActivityAt, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[sessionID.String()]
	if !ok {
		return store.ErrNotFound
	}
	sess.LastActivityAt = lastActivityAt
	sess.ExpiresAt = expiresAt
	sess.UpdatedAt = time.Now()
	return nil
}

func (s *Store) DeleteSession(_ context.Context, sessionID id.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID.String())
	return nil
}

func (s *Store) DeleteUserSessions(_ context.Context, userID id.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, sess := range s.sessions {
		if sess.UserID.String() == userID.String() {
			delete(s.sessions, k)
		}
	}
	return nil
}

func (s *Store) ListUserSessions(_ context.Context, userID id.UserID) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*session.Session
	for _, sess := range s.sessions {
		if sess.UserID.String() == userID.String() {
			result = append(result, sess)
		}
	}
	return result, nil
}

func (s *Store) ListSessions(_ context.Context, limit int) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*session.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	// Sort by created_at descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Refresh-token replay detection
// ──────────────────────────────────────────────────

// IsRefreshTokenRevoked reports whether tokenHash is in the revoked set.
func (s *Store) IsRefreshTokenRevoked(_ context.Context, tokenHash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.revokedRefreshTokens[tokenHash]
	return ok, nil
}

// MarkRefreshTokenRevoked records tokenHash as revoked. Idempotent: a duplicate
// insert keeps the original record (the original family + first-seen timestamp).
func (s *Store) MarkRefreshTokenRevoked(_ context.Context, tokenHash string, familyID id.SessionFamilyID, reason string) error {
	if tokenHash == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.revokedRefreshTokens[tokenHash]; exists {
		return nil
	}
	s.revokedRefreshTokens[tokenHash] = &session.RevokedRefreshToken{
		TokenHash: tokenHash,
		FamilyID:  familyID,
		RevokedAt: time.Now(),
		Reason:    reason,
	}
	return nil
}

// GetRevokedRefreshTokenFamily returns the family of a previously-revoked
// token, or store.ErrNotFound if the hash is unknown.
func (s *Store) GetRevokedRefreshTokenFamily(_ context.Context, tokenHash string) (id.SessionFamilyID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.revokedRefreshTokens[tokenHash]
	if !ok {
		return id.Nil, store.ErrNotFound
	}
	return rec.FamilyID, nil
}

// RevokeRefreshTokenFamily revokes every active session sharing familyID. Each
// surviving refresh-token hash is also recorded as revoked with reason so that
// later replays of any sibling token are detected too.
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, familyID id.SessionFamilyID, reason string) error {
	if familyID.IsNil() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, sess := range s.sessions {
		if sess == nil {
			continue
		}
		if sess.FamilyID.String() != familyID.String() {
			continue
		}
		if sess.RefreshToken != "" {
			h := hashRefreshToken(sess.RefreshToken)
			if _, exists := s.revokedRefreshTokens[h]; !exists {
				s.revokedRefreshTokens[h] = &session.RevokedRefreshToken{
					TokenHash: h,
					FamilyID:  familyID,
					RevokedAt: now,
					Reason:    reason,
				}
			}
		}
		delete(s.sessions, k)
	}
	return nil
}

// hashRefreshToken returns the hex-encoded SHA-256 of a refresh token. Kept as
// a small helper so memory + service share the exact same canonicalisation.
func hashRefreshToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// ──────────────────────────────────────────────────
// Account Store (Verification + PasswordReset)
// ──────────────────────────────────────────────────

func (s *Store) CreateVerification(_ context.Context, v *account.Verification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verifications[v.Token] = v
	return nil
}

func (s *Store) GetVerification(_ context.Context, token string) (*account.Verification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.verifications[token]
	if !ok {
		return nil, store.ErrNotFound
	}
	return v, nil
}

func (s *Store) ConsumeVerification(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.verifications[token]
	if !ok {
		return store.ErrNotFound
	}
	v.Consumed = true
	return nil
}

func (s *Store) GetActiveEmailVerification(_ context.Context, userID id.UserID) (*account.Verification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var latest *account.Verification
	for _, v := range s.verifications {
		if v.UserID != userID || v.Type != account.VerificationEmail {
			continue
		}
		if v.Consumed || now.After(v.ExpiresAt) {
			continue
		}
		if latest == nil || v.CreatedAt.After(latest.CreatedAt) {
			latest = v
		}
	}
	if latest == nil {
		return nil, store.ErrNotFound
	}
	return latest, nil
}

func (s *Store) UpdateVerification(_ context.Context, v *account.Verification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.verifications[v.Token]; !ok {
		return store.ErrNotFound
	}
	s.verifications[v.Token] = v
	return nil
}

func (s *Store) CreatePasswordReset(_ context.Context, pr *account.PasswordReset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.passwordResets[pr.Token] = pr
	return nil
}

func (s *Store) GetPasswordReset(_ context.Context, token string) (*account.PasswordReset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pr, ok := s.passwordResets[token]
	if !ok {
		return nil, store.ErrNotFound
	}
	return pr, nil
}

func (s *Store) ConsumePasswordReset(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr, ok := s.passwordResets[token]
	if !ok {
		return store.ErrNotFound
	}
	pr.Consumed = true
	return nil
}

// ──────────────────────────────────────────────────
// App Store
// ──────────────────────────────────────────────────

func (s *Store) CreateApp(_ context.Context, a *app.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	a.UpdatedAt = a.CreatedAt
	s.apps[a.ID.String()] = a
	return nil
}

func (s *Store) GetApp(_ context.Context, appID id.AppID) (*app.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.apps[appID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return a, nil
}

func (s *Store) GetAppBySlug(_ context.Context, slug string) (*app.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.apps {
		if a.Slug == slug {
			return a, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetAppByPublishableKey(_ context.Context, key string) (*app.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.apps {
		if a.PublishableKey == key {
			return a, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetPlatformApp(_ context.Context) (*app.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.apps {
		if a.IsPlatform {
			return a, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateApp(_ context.Context, a *app.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[a.ID.String()]; !ok {
		return store.ErrNotFound
	}
	a.UpdatedAt = time.Now()
	s.apps[a.ID.String()] = a
	return nil
}

// DeleteApp removes the app and every child record scoped to it, mirroring
// the ON DELETE CASCADE behaviour of the PostgreSQL backend so the in-memory
// store stays consistent (no orphaned users/sessions/etc.). Plugin-owned
// tables (passkey, oauth connections, oauth2 clients, ...) live in their own
// stores and are not tracked here.
func (s *Store) DeleteApp(_ context.Context, appID id.AppID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aid := appID.String()

	// Organizations key their dependent rows (members, teams, invitations)
	// off org_id, so collect this app's org IDs before dropping the orgs.
	orgIDs := make(map[string]bool)
	for k, o := range s.orgs {
		if o.AppID.String() == aid {
			orgIDs[k] = true
			delete(s.orgs, k)
		}
	}
	for k, m := range s.members {
		if orgIDs[m.OrgID.String()] {
			delete(s.members, k)
		}
	}
	for k, t := range s.teams {
		if orgIDs[t.OrgID.String()] {
			delete(s.teams, k)
		}
	}
	for k, inv := range s.invitations {
		if orgIDs[inv.OrgID.String()] {
			delete(s.invitations, k)
		}
	}

	// App-scoped child records (every map keyed by an entity with an AppID).
	for k, u := range s.users {
		if u.AppID.String() == aid {
			delete(s.users, k)
		}
	}
	for k, sess := range s.sessions {
		if sess.AppID.String() == aid {
			delete(s.sessions, k)
		}
	}
	for k, v := range s.verifications {
		if v.AppID.String() == aid {
			delete(s.verifications, k)
		}
	}
	for k, pr := range s.passwordResets {
		if pr.AppID.String() == aid {
			delete(s.passwordResets, k)
		}
	}
	for k, d := range s.devices {
		if d.AppID.String() == aid {
			delete(s.devices, k)
		}
	}
	for k, w := range s.webhooks {
		if w.AppID.String() == aid {
			delete(s.webhooks, k)
		}
	}
	for k, n := range s.notifications {
		if n.AppID.String() == aid {
			delete(s.notifications, k)
		}
	}
	for k, ak := range s.apikeys {
		if ak.AppID.String() == aid {
			delete(s.apikeys, k)
		}
	}
	for k, e := range s.environments {
		if e.AppID.String() == aid {
			delete(s.environments, k)
		}
	}
	for k, fc := range s.formConfigs {
		if fc.AppID.String() == aid {
			delete(s.formConfigs, k)
		}
	}
	for k, bc := range s.brandingConfigs {
		if bc.AppID.String() == aid {
			delete(s.brandingConfigs, k)
		}
	}
	for k, c := range s.appSessionConfigs {
		if c.AppID.String() == aid {
			delete(s.appSessionConfigs, k)
		}
	}
	for k, c := range s.appClientConfigs {
		if c.AppID.String() == aid {
			delete(s.appClientConfigs, k)
		}
	}
	for k, sa := range s.serviceAccounts {
		if sa.AppID.String() == aid {
			delete(s.serviceAccounts, k)
		}
	}
	for k, st := range s.settingsMap {
		if st.AppID == aid {
			delete(s.settingsMap, k)
		}
	}
	for k, ue := range s.userEmails {
		if ue.AppID.String() == aid {
			delete(s.userEmails, k)
		}
	}

	delete(s.apps, aid)
	return nil
}

func (s *Store) ListApps(_ context.Context) ([]*app.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*app.App, 0, len(s.apps))
	for _, a := range s.apps {
		result = append(result, a)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Organization Store
// ──────────────────────────────────────────────────

func (s *Store) CreateOrganization(_ context.Context, o *organization.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	o.UpdatedAt = o.CreatedAt
	s.orgs[o.ID.String()] = o
	return nil
}

func (s *Store) GetOrganization(_ context.Context, orgID id.OrgID) (*organization.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orgs[orgID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return o, nil
}

func (s *Store) GetOrganizationBySlug(_ context.Context, appID id.AppID, slug string) (*organization.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.orgs {
		if o.AppID.String() == appID.String() && o.Slug == slug {
			return o, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateOrganization(_ context.Context, o *organization.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orgs[o.ID.String()]; !ok {
		return store.ErrNotFound
	}
	o.UpdatedAt = time.Now()
	s.orgs[o.ID.String()] = o
	return nil
}

func (s *Store) DeleteOrganization(_ context.Context, orgID id.OrgID) error {
	if err := s.takeFault("DeleteOrganization"); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.orgs, orgID.String())
	return nil
}

func (s *Store) ListOrganizations(_ context.Context, appID id.AppID) ([]*organization.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*organization.Organization
	for _, o := range s.orgs {
		if o.AppID.String() == appID.String() {
			result = append(result, o)
		}
	}
	return result, nil
}

func (s *Store) ListUserOrganizations(_ context.Context, userID id.UserID) ([]*organization.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orgIDs := make(map[string]bool)
	for _, m := range s.members {
		if m.UserID.String() == userID.String() {
			orgIDs[m.OrgID.String()] = true
		}
	}
	var result []*organization.Organization
	for _, o := range s.orgs {
		if orgIDs[o.ID.String()] {
			result = append(result, o)
		}
	}
	return result, nil
}

func (s *Store) CreateMember(_ context.Context, m *organization.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	m.UpdatedAt = m.CreatedAt
	s.members[m.ID.String()] = m
	return nil
}

func (s *Store) GetMember(_ context.Context, memberID id.MemberID) (*organization.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.members[memberID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return m, nil
}

func (s *Store) GetMemberByUserAndOrg(_ context.Context, userID id.UserID, orgID id.OrgID) (*organization.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.members {
		if m.UserID.String() == userID.String() && m.OrgID.String() == orgID.String() {
			return m, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateMember(_ context.Context, m *organization.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.members[m.ID.String()]; !ok {
		return store.ErrNotFound
	}
	m.UpdatedAt = time.Now()
	s.members[m.ID.String()] = m
	return nil
}

func (s *Store) DeleteMember(_ context.Context, memberID id.MemberID) error {
	if err := s.takeFault("DeleteMember"); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.members, memberID.String())
	return nil
}

func (s *Store) ListMembers(_ context.Context, orgID id.OrgID) ([]*organization.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*organization.Member
	for _, m := range s.members {
		if m.OrgID.String() == orgID.String() {
			result = append(result, m)
		}
	}
	return result, nil
}

func (s *Store) CreateInvitation(_ context.Context, inv *organization.Invitation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	s.invitations[inv.ID.String()] = inv
	return nil
}

func (s *Store) GetInvitation(_ context.Context, invID id.InvitationID) (*organization.Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inv, ok := s.invitations[invID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return inv, nil
}

func (s *Store) GetInvitationByToken(_ context.Context, token string) (*organization.Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, inv := range s.invitations {
		if inv.Token == token {
			return inv, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateInvitation(_ context.Context, inv *organization.Invitation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.invitations[inv.ID.String()]; !ok {
		return store.ErrNotFound
	}
	s.invitations[inv.ID.String()] = inv
	return nil
}

func (s *Store) DeleteInvitation(_ context.Context, invID id.InvitationID) error {
	if err := s.takeFault("DeleteInvitation"); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.invitations[invID.String()]; !ok {
		return store.ErrNotFound
	}
	delete(s.invitations, invID.String())
	return nil
}

func (s *Store) ListInvitations(_ context.Context, orgID id.OrgID) ([]*organization.Invitation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*organization.Invitation
	for _, inv := range s.invitations {
		if inv.OrgID.String() == orgID.String() {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (s *Store) CreateTeam(_ context.Context, t *organization.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	t.UpdatedAt = t.CreatedAt
	s.teams[t.ID.String()] = t
	return nil
}

func (s *Store) GetTeam(_ context.Context, teamID id.TeamID) (*organization.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.teams[teamID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return t, nil
}

func (s *Store) UpdateTeam(_ context.Context, t *organization.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[t.ID.String()]; !ok {
		return store.ErrNotFound
	}
	t.UpdatedAt = time.Now()
	s.teams[t.ID.String()] = t
	return nil
}

func (s *Store) DeleteTeam(_ context.Context, teamID id.TeamID) error {
	if err := s.takeFault("DeleteTeam"); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.teams, teamID.String())
	return nil
}

func (s *Store) ListTeams(_ context.Context, orgID id.OrgID) ([]*organization.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*organization.Team
	for _, t := range s.teams {
		if t.OrgID.String() == orgID.String() {
			result = append(result, t)
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Device Store
// ──────────────────────────────────────────────────

func (s *Store) CreateDevice(_ context.Context, d *device.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	d.UpdatedAt = d.CreatedAt
	s.devices[d.ID.String()] = d
	return nil
}

func (s *Store) GetDevice(_ context.Context, deviceID id.DeviceID) (*device.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[deviceID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return d, nil
}

func (s *Store) GetDeviceByFingerprint(_ context.Context, userID id.UserID, fingerprint string) (*device.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.devices {
		if d.UserID.String() == userID.String() && d.Fingerprint == fingerprint {
			return d, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateDevice(_ context.Context, d *device.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.devices[d.ID.String()]; !ok {
		return store.ErrNotFound
	}
	d.UpdatedAt = time.Now()
	s.devices[d.ID.String()] = d
	return nil
}

func (s *Store) DeleteDevice(_ context.Context, deviceID id.DeviceID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, deviceID.String())
	return nil
}

func (s *Store) ListUserDevices(_ context.Context, userID id.UserID) ([]*device.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*device.Device
	for _, d := range s.devices {
		if d.UserID.String() == userID.String() {
			result = append(result, d)
		}
	}
	return result, nil
}

func (s *Store) ListDevices(_ context.Context, limit int) ([]*device.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*device.Device, 0, len(s.devices))
	for _, d := range s.devices {
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeenAt.After(result[j].LastSeenAt)
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Webhook Store
// ──────────────────────────────────────────────────

func (s *Store) CreateWebhook(_ context.Context, w *webhook.Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now()
	}
	w.UpdatedAt = w.CreatedAt
	s.webhooks[w.ID.String()] = w
	return nil
}

func (s *Store) GetWebhook(_ context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.webhooks[webhookID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return w, nil
}

func (s *Store) UpdateWebhook(_ context.Context, w *webhook.Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.webhooks[w.ID.String()]; !ok {
		return store.ErrNotFound
	}
	w.UpdatedAt = time.Now()
	s.webhooks[w.ID.String()] = w
	return nil
}

func (s *Store) DeleteWebhook(_ context.Context, webhookID id.WebhookID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.webhooks, webhookID.String())
	return nil
}

func (s *Store) ListWebhooks(_ context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*webhook.Webhook
	for _, w := range s.webhooks {
		if w.AppID.String() == appID.String() {
			result = append(result, w)
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Notification Store
// ──────────────────────────────────────────────────

func (s *Store) CreateNotification(_ context.Context, n *notification.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	s.notifications[n.ID.String()] = n
	return nil
}

func (s *Store) GetNotification(_ context.Context, notifID id.NotificationID) (*notification.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notifications[notifID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return n, nil
}

func (s *Store) MarkSent(_ context.Context, notifID id.NotificationID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notifications[notifID.String()]
	if !ok {
		return store.ErrNotFound
	}
	now := time.Now()
	n.Sent = true
	n.SentAt = &now
	return nil
}

func (s *Store) ListUserNotifications(_ context.Context, userID id.UserID) ([]*notification.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*notification.Notification
	for _, n := range s.notifications {
		if n.UserID.String() == userID.String() {
			result = append(result, n)
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// API Key Store
// ──────────────────────────────────────────────────

func (s *Store) CreateAPIKey(_ context.Context, k *apikey.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	k.UpdatedAt = k.CreatedAt
	s.apikeys[k.ID.String()] = k
	return nil
}

func (s *Store) GetAPIKey(_ context.Context, keyID id.APIKeyID) (*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.apikeys[keyID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return k, nil
}

func (s *Store) GetAPIKeyByPrefix(_ context.Context, appID id.AppID, prefix string) (*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.apikeys {
		if k.AppID.String() == appID.String() && k.KeyPrefix == prefix {
			return k, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) FindByPrefix(_ context.Context, prefix string) (*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.apikeys {
		if k.KeyPrefix == prefix {
			return k, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetAPIKeyByPublicKey(_ context.Context, appID id.AppID, publicKey string) (*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.apikeys {
		if k.AppID.String() == appID.String() && k.PublicKey == publicKey {
			return k, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateAPIKey(_ context.Context, k *apikey.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apikeys[k.ID.String()]; !ok {
		return store.ErrNotFound
	}
	k.UpdatedAt = time.Now()
	s.apikeys[k.ID.String()] = k
	return nil
}

func (s *Store) DeleteAPIKey(_ context.Context, keyID id.APIKeyID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.apikeys, keyID.String())
	return nil
}

func (s *Store) ListAPIKeysByApp(_ context.Context, appID id.AppID) ([]*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*apikey.APIKey
	for _, k := range s.apikeys {
		if k.AppID.String() == appID.String() {
			result = append(result, k)
		}
	}
	return result, nil
}

func (s *Store) ListAPIKeysByUser(_ context.Context, appID id.AppID, userID id.UserID) ([]*apikey.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*apikey.APIKey
	for _, k := range s.apikeys {
		if k.AppID.String() == appID.String() && k.UserID.String() == userID.String() {
			result = append(result, k)
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Environment Store
// ──────────────────────────────────────────────────

func (s *Store) CreateEnvironment(_ context.Context, e *environment.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	e.UpdatedAt = e.CreatedAt
	s.environments[e.ID.String()] = e
	return nil
}

func (s *Store) GetEnvironment(_ context.Context, envID id.EnvironmentID) (*environment.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.environments[envID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return e, nil
}

func (s *Store) GetEnvironmentBySlug(_ context.Context, appID id.AppID, slug string) (*environment.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.environments {
		if e.AppID.String() == appID.String() && e.Slug == slug {
			return e, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) GetDefaultEnvironment(_ context.Context, appID id.AppID) (*environment.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.environments {
		if e.AppID.String() == appID.String() && e.IsDefault {
			return e, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateEnvironment(_ context.Context, e *environment.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.environments[e.ID.String()]; !ok {
		return store.ErrNotFound
	}
	e.UpdatedAt = time.Now()
	s.environments[e.ID.String()] = e
	return nil
}

func (s *Store) DeleteEnvironment(_ context.Context, envID id.EnvironmentID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.environments[envID.String()]
	if !ok {
		return store.ErrNotFound
	}
	if e.IsDefault {
		return fmt.Errorf("authsome/memory: cannot delete the default environment")
	}
	delete(s.environments, envID.String())
	return nil
}

func (s *Store) ListEnvironments(_ context.Context, appID id.AppID) ([]*environment.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*environment.Environment
	for _, e := range s.environments {
		if e.AppID.String() == appID.String() {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *Store) SetDefaultEnvironment(_ context.Context, appID id.AppID, envID id.EnvironmentID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Clear the current default for this app.
	for _, e := range s.environments {
		if e.AppID.String() == appID.String() && e.IsDefault {
			e.IsDefault = false
			e.UpdatedAt = time.Now()
		}
	}
	// Set the new default.
	e, ok := s.environments[envID.String()]
	if !ok {
		return store.ErrNotFound
	}
	e.IsDefault = true
	e.UpdatedAt = time.Now()
	return nil
}

// ──────────────────────────────────────────────────
// FormConfig Store
// ──────────────────────────────────────────────────

func (s *Store) CreateFormConfig(_ context.Context, fc *formconfig.FormConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fc.CreatedAt.IsZero() {
		fc.CreatedAt = time.Now()
	}
	fc.UpdatedAt = fc.CreatedAt
	s.formConfigs[fc.ID.String()] = fc
	return nil
}

func (s *Store) GetFormConfig(_ context.Context, appID id.AppID, formType string) (*formconfig.FormConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, fc := range s.formConfigs {
		if fc.AppID.String() == appID.String() && fc.FormType == formType {
			return fc, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) UpdateFormConfig(_ context.Context, fc *formconfig.FormConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.formConfigs[fc.ID.String()]; !ok {
		return store.ErrNotFound
	}
	fc.UpdatedAt = time.Now()
	s.formConfigs[fc.ID.String()] = fc
	return nil
}

func (s *Store) DeleteFormConfig(_ context.Context, appID id.AppID, formType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, fc := range s.formConfigs {
		if fc.AppID.String() == appID.String() && fc.FormType == formType {
			delete(s.formConfigs, k)
			return nil
		}
	}
	return store.ErrNotFound
}

func (s *Store) ListFormConfigs(_ context.Context, appID id.AppID) ([]*formconfig.FormConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*formconfig.FormConfig
	for _, fc := range s.formConfigs {
		if fc.AppID.String() == appID.String() {
			result = append(result, fc)
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Branding Store
// ──────────────────────────────────────────────────

func (s *Store) GetBranding(_ context.Context, orgID id.OrgID) (*formconfig.BrandingConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.brandingConfigs {
		if b.OrgID.String() == orgID.String() {
			return b, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) SaveBranding(_ context.Context, b *formconfig.BrandingConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	b.UpdatedAt = time.Now()
	s.brandingConfigs[b.ID.String()] = b
	return nil
}

func (s *Store) DeleteBranding(_ context.Context, orgID id.OrgID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, b := range s.brandingConfigs {
		if b.OrgID.String() == orgID.String() {
			delete(s.brandingConfigs, k)
			return nil
		}
	}
	return store.ErrNotFound
}

// ──────────────────────────────────────────────────
// App Session Config Store
// ──────────────────────────────────────────────────

func (s *Store) GetAppSessionConfig(_ context.Context, appID id.AppID) (*appsessionconfig.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.appSessionConfigs[appID.String()]
	if !ok {
		return nil, appsessionconfig.ErrNotFound
	}
	return cfg, nil
}

func (s *Store) SetAppSessionConfig(_ context.Context, cfg *appsessionconfig.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppSessionConfigID()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	s.appSessionConfigs[cfg.AppID.String()] = cfg
	return nil
}

func (s *Store) DeleteAppSessionConfig(_ context.Context, appID id.AppID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.appSessionConfigs[appID.String()]; !ok {
		return appsessionconfig.ErrNotFound
	}
	delete(s.appSessionConfigs, appID.String())
	return nil
}

// ──────────────────────────────────────────────────
// App Client Config Store
// ──────────────────────────────────────────────────

func (s *Store) GetAppClientConfig(_ context.Context, appID id.AppID) (*appclientconfig.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.appClientConfigs[appID.String()]
	if !ok {
		return nil, appclientconfig.ErrNotFound
	}
	return cfg, nil
}

func (s *Store) SetAppClientConfig(_ context.Context, cfg *appclientconfig.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppClientConfigID()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	s.appClientConfigs[cfg.AppID.String()] = cfg
	return nil
}

func (s *Store) DeleteAppClientConfig(_ context.Context, appID id.AppID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.appClientConfigs[appID.String()]; !ok {
		return appclientconfig.ErrNotFound
	}
	delete(s.appClientConfigs, appID.String())
	return nil
}

// ──────────────────────────────────────────────────
// Service Account Store
// ──────────────────────────────────────────────────

func (s *Store) CreateServiceAccount(_ context.Context, svc *serviceaccount.ServiceAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc.CreatedAt.IsZero() {
		svc.CreatedAt = time.Now()
	}
	svc.UpdatedAt = svc.CreatedAt
	// Enforce unique (app_id, name).
	for _, existing := range s.serviceAccounts {
		if existing.AppID.String() == svc.AppID.String() && existing.Name == svc.Name {
			return store.ErrConflict
		}
	}
	s.serviceAccounts[svc.ID.String()] = svc
	return nil
}

func (s *Store) GetServiceAccount(_ context.Context, svcID id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svc, ok := s.serviceAccounts[svcID.String()]
	if !ok {
		return nil, store.ErrNotFound
	}
	return svc, nil
}

func (s *Store) ListServiceAccounts(_ context.Context, q *serviceaccount.Query) (*serviceaccount.List, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*serviceaccount.ServiceAccount
	for _, svc := range s.serviceAccounts {
		if svc.AppID.String() != q.AppID.String() {
			continue
		}
		if q.Active != nil && svc.Active != *q.Active {
			continue
		}
		result = append(result, svc)
	}
	if q.Limit > 0 && len(result) > q.Limit {
		result = result[:q.Limit]
	}
	return &serviceaccount.List{ServiceAccounts: result, Total: len(result)}, nil
}

func (s *Store) UpdateServiceAccount(_ context.Context, svc *serviceaccount.ServiceAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceAccounts[svc.ID.String()]; !ok {
		return store.ErrNotFound
	}
	svc.UpdatedAt = time.Now()
	s.serviceAccounts[svc.ID.String()] = svc
	return nil
}

func (s *Store) DeleteServiceAccount(_ context.Context, svcID id.ServiceAccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceAccounts[svcID.String()]; !ok {
		return store.ErrNotFound
	}
	delete(s.serviceAccounts, svcID.String())
	return nil
}
