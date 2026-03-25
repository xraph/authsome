// Package sqlite implements the AuthSome store interface using grove ORM (SQLite).
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	"github.com/xraph/grove/migrate"

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
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// Store implements store.Store using grove ORM (SQLite).
type Store struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// Compile-time interface checks.
var _ store.Store = (*Store)(nil)

// New creates a new SQLite-backed store.
func New(db *grove.DB) *Store {
	return &Store{
		db:  db,
		sdb: sqlitedriver.Unwrap(db),
	}
}

// DB returns the underlying grove.DB for advanced use cases.
func (s *Store) DB() *grove.DB { return s.db }

// ──────────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────────

// Migrate runs all registered migrations via the grove orchestrator.
func (s *Store) Migrate(ctx context.Context, extraGroups ...*migrate.Group) error {
	executor, err := migrate.NewExecutorFor(s.sdb)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: create migration executor: %w", err)
	}

	groups := make([]*migrate.Group, 0, 1+len(extraGroups))
	groups = append(groups, Migrations)
	groups = append(groups, extraGroups...)

	orch := migrate.NewOrchestrator(executor, groups...)
	if _, err := orch.Migrate(ctx); err != nil {
		return fmt.Errorf("authsome/sqlite: migration failed: %w", err)
	}

	return nil
}

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ──────────────────────────────────────────────────
// App store
// ──────────────────────────────────────────────────

func (s *Store) CreateApp(ctx context.Context, a *app.App) error {
	m := fromApp(a)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetApp(ctx context.Context, appID id.AppID) (*app.App, error) {
	m := new(AppModel)
	err := s.sdb.NewSelect(m).Where("id = ?", appID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toApp(m)
}

func (s *Store) GetAppBySlug(ctx context.Context, slug string) (*app.App, error) {
	m := new(AppModel)
	err := s.sdb.NewSelect(m).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toApp(m)
}

func (s *Store) GetAppByPublishableKey(ctx context.Context, key string) (*app.App, error) {
	m := new(AppModel)
	err := s.sdb.NewSelect(m).Where("publishable_key = ?", key).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toApp(m)
}

func (s *Store) GetPlatformApp(ctx context.Context) (*app.App, error) {
	m := new(AppModel)
	err := s.sdb.NewSelect(m).Where("is_platform = ?", true).Limit(1).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toApp(m)
}

func (s *Store) UpdateApp(ctx context.Context, a *app.App) error {
	m := fromApp(a)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteApp(ctx context.Context, appID id.AppID) error {
	_, err := s.sdb.NewDelete((*AppModel)(nil)).Where("id = ?", appID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListApps(ctx context.Context) ([]*app.App, error) {
	var models []AppModel
	err := s.sdb.NewSelect(&models).OrderExpr("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*app.App, 0, len(models))
	for i := range models {
		a, err := toApp(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// User store
// ──────────────────────────────────────────────────

func (s *Store) CreateUser(ctx context.Context, u *user.User) error {
	m := fromUser(u)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	m := new(UserModel)
	err := s.sdb.NewSelect(m).Where("id = ?", userID.String()).Where("deleted_at IS NULL").Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByEmail(ctx context.Context, appID id.AppID, email string) (*user.User, error) {
	m := new(UserModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", email).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByPhone(ctx context.Context, appID id.AppID, phone string) (*user.User, error) {
	m := new(UserModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("phone = ?", phone).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByUsername(ctx context.Context, appID id.AppID, username string) (*user.User, error) {
	m := new(UserModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("username = ?", username).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toUser(m)
}

func (s *Store) UpdateUser(ctx context.Context, u *user.User) error {
	m := fromUser(u)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteUser(ctx context.Context, userID id.UserID) error {
	now := time.Now()
	_, err := s.sdb.NewUpdate((*UserModel)(nil)).
		Set("deleted_at = ?", now).
		Where("id = ?", userID.String()).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListUsers(ctx context.Context, q *user.Query) (*user.List, error) {
	var models []UserModel
	query := s.sdb.NewSelect(&models).
		Where("app_id = ?", q.AppID.String()).
		Where("deleted_at IS NULL")

	if !q.EnvID.IsNil() {
		query = query.Where("env_id = ?", q.EnvID.String())
	}

	if q.Email != "" {
		query = query.Where("email LIKE ?", "%"+q.Email+"%")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	if q.Cursor != "" {
		query = query.Where("id < ?", q.Cursor)
	}

	query = query.OrderExpr("id DESC").Limit(limit + 1)

	err := query.Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}

	list := &user.List{
		Users: make([]*user.User, 0, len(models)),
		Total: len(models),
	}

	// Adjust total if we fetched the extra sentinel row (limit+1).
	if len(models) > limit {
		list.Total = limit
	}

	for i := range models {
		if i >= limit {
			list.NextCursor = models[i].ID
			break
		}
		u, err := toUser(&models[i])
		if err != nil {
			return nil, err
		}
		list.Users = append(list.Users, u)
	}
	return list, nil
}

// ──────────────────────────────────────────────────
// Session store
// ──────────────────────────────────────────────────

func (s *Store) CreateSession(ctx context.Context, sess *session.Session) error {
	m := fromSession(sess)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetSession(ctx context.Context, sessionID id.SessionID) (*session.Session, error) {
	m := new(SessionModel)
	err := s.sdb.NewSelect(m).Where("id = ?", sessionID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toSession(m)
}

func (s *Store) GetSessionByToken(ctx context.Context, token string) (*session.Session, error) {
	m := new(SessionModel)
	err := s.sdb.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toSession(m)
}

func (s *Store) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*session.Session, error) {
	m := new(SessionModel)
	err := s.sdb.NewSelect(m).Where("refresh_token = ?", refreshToken).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toSession(m)
}

func (s *Store) UpdateSession(ctx context.Context, sess *session.Session) error {
	m := fromSession(sess)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) TouchSession(ctx context.Context, sessionID id.SessionID, lastActivityAt, expiresAt time.Time) error {
	_, err := s.sdb.NewUpdate((*SessionModel)(nil)).
		Set("last_activity_at = ?", lastActivityAt).
		Set("expires_at = ?", expiresAt).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", sessionID.String()).
		Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteSession(ctx context.Context, sessionID id.SessionID) error {
	_, err := s.sdb.NewDelete((*SessionModel)(nil)).Where("id = ?", sessionID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID id.UserID) error {
	_, err := s.sdb.NewDelete((*SessionModel)(nil)).Where("user_id = ?", userID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListUserSessions(ctx context.Context, userID id.UserID) ([]*session.Session, error) {
	var models []SessionModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*session.Session, 0, len(models))
	for i := range models {
		sess, err := toSession(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, sess)
	}
	return result, nil
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]*session.Session, error) {
	var models []SessionModel
	q := s.sdb.NewSelect(&models).
		OrderExpr("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*session.Session, 0, len(models))
	for i := range models {
		sess, err := toSession(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, sess)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Account store (verification + password reset)
// ──────────────────────────────────────────────────

func (s *Store) CreateVerification(ctx context.Context, v *account.Verification) error {
	m := fromVerification(v)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetVerification(ctx context.Context, token string) (*account.Verification, error) {
	m := new(VerificationModel)
	err := s.sdb.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toVerification(m)
}

func (s *Store) ConsumeVerification(ctx context.Context, token string) error {
	_, err := s.sdb.NewUpdate((*VerificationModel)(nil)).
		Set("consumed = TRUE").
		Where("token = ?", token).
		Where("consumed = FALSE").
		Exec(ctx)
	return sqliteError(err)
}

func (s *Store) CreatePasswordReset(ctx context.Context, pr *account.PasswordReset) error {
	m := fromPasswordReset(pr)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetPasswordReset(ctx context.Context, token string) (*account.PasswordReset, error) {
	m := new(PasswordResetModel)
	err := s.sdb.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toPasswordReset(m)
}

func (s *Store) ConsumePasswordReset(ctx context.Context, token string) error {
	_, err := s.sdb.NewUpdate((*PasswordResetModel)(nil)).
		Set("consumed = TRUE").
		Where("token = ?", token).
		Where("consumed = FALSE").
		Exec(ctx)
	return sqliteError(err)
}

// ──────────────────────────────────────────────────
// Organization store
// ──────────────────────────────────────────────────

func (s *Store) CreateOrganization(ctx context.Context, o *organization.Organization) error {
	m := fromOrganization(o)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetOrganization(ctx context.Context, orgID id.OrgID) (*organization.Organization, error) {
	m := new(OrganizationModel)
	err := s.sdb.NewSelect(m).Where("id = ?", orgID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toOrganization(m)
}

func (s *Store) GetOrganizationBySlug(ctx context.Context, appID id.AppID, slug string) (*organization.Organization, error) {
	m := new(OrganizationModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toOrganization(m)
}

func (s *Store) UpdateOrganization(ctx context.Context, o *organization.Organization) error {
	m := fromOrganization(o)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteOrganization(ctx context.Context, orgID id.OrgID) error {
	_, err := s.sdb.NewDelete((*OrganizationModel)(nil)).Where("id = ?", orgID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListOrganizations(ctx context.Context, appID id.AppID) ([]*organization.Organization, error) {
	var models []OrganizationModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toOrganizations(models)
}

func (s *Store) ListUserOrganizations(ctx context.Context, userID id.UserID) ([]*organization.Organization, error) {
	var models []OrganizationModel
	err := s.sdb.NewSelect(&models).
		Join("JOIN", "authsome_members AS mem", "mem.org_id = o.id").
		Where("mem.user_id = ?", userID.String()).
		OrderExpr("o.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toOrganizations(models)
}

func toOrganizations(models []OrganizationModel) ([]*organization.Organization, error) {
	result := make([]*organization.Organization, 0, len(models))
	for i := range models {
		o, err := toOrganization(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Member store
// ──────────────────────────────────────────────────

func (s *Store) CreateMember(ctx context.Context, m *organization.Member) error {
	model := fromMember(m)
	_, err := s.sdb.NewInsert(model).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetMember(ctx context.Context, memberID id.MemberID) (*organization.Member, error) {
	m := new(MemberModel)
	err := s.sdb.NewSelect(m).Where("id = ?", memberID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toMember(m)
}

func (s *Store) GetMemberByUserAndOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) (*organization.Member, error) {
	m := new(MemberModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("org_id = ?", orgID.String()).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toMember(m)
}

func (s *Store) UpdateMember(ctx context.Context, mem *organization.Member) error {
	m := fromMember(mem)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteMember(ctx context.Context, memberID id.MemberID) error {
	_, err := s.sdb.NewDelete((*MemberModel)(nil)).Where("id = ?", memberID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListMembers(ctx context.Context, orgID id.OrgID) ([]*organization.Member, error) {
	var models []MemberModel
	err := s.sdb.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*organization.Member, 0, len(models))
	for i := range models {
		mem, err := toMember(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, mem)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Invitation store
// ──────────────────────────────────────────────────

func (s *Store) CreateInvitation(ctx context.Context, inv *organization.Invitation) error {
	m := fromInvitation(inv)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetInvitation(ctx context.Context, invID id.InvitationID) (*organization.Invitation, error) {
	m := new(InvitationModel)
	err := s.sdb.NewSelect(m).Where("id = ?", invID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toInvitation(m)
}

func (s *Store) GetInvitationByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	m := new(InvitationModel)
	err := s.sdb.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toInvitation(m)
}

func (s *Store) UpdateInvitation(ctx context.Context, inv *organization.Invitation) error {
	m := fromInvitation(inv)
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListInvitations(ctx context.Context, orgID id.OrgID) ([]*organization.Invitation, error) {
	var models []InvitationModel
	err := s.sdb.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*organization.Invitation, 0, len(models))
	for i := range models {
		inv, err := toInvitation(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, inv)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Team store
// ──────────────────────────────────────────────────

func (s *Store) CreateTeam(ctx context.Context, t *organization.Team) error {
	m := fromTeam(t)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetTeam(ctx context.Context, teamID id.TeamID) (*organization.Team, error) {
	m := new(TeamModel)
	err := s.sdb.NewSelect(m).Where("id = ?", teamID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toTeam(m)
}

func (s *Store) UpdateTeam(ctx context.Context, t *organization.Team) error {
	m := fromTeam(t)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteTeam(ctx context.Context, teamID id.TeamID) error {
	_, err := s.sdb.NewDelete((*TeamModel)(nil)).Where("id = ?", teamID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListTeams(ctx context.Context, orgID id.OrgID) ([]*organization.Team, error) {
	var models []TeamModel
	err := s.sdb.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*organization.Team, 0, len(models))
	for i := range models {
		t, err := toTeam(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Device store
// ──────────────────────────────────────────────────

func (s *Store) CreateDevice(ctx context.Context, d *device.Device) error {
	m := fromDevice(d)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetDevice(ctx context.Context, deviceID id.DeviceID) (*device.Device, error) {
	m := new(DeviceModel)
	err := s.sdb.NewSelect(m).Where("id = ?", deviceID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toDevice(m)
}

func (s *Store) GetDeviceByFingerprint(ctx context.Context, userID id.UserID, fingerprint string) (*device.Device, error) {
	m := new(DeviceModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("fingerprint = ?", fingerprint).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toDevice(m)
}

func (s *Store) UpdateDevice(ctx context.Context, d *device.Device) error {
	m := fromDevice(d)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteDevice(ctx context.Context, deviceID id.DeviceID) error {
	_, err := s.sdb.NewDelete((*DeviceModel)(nil)).Where("id = ?", deviceID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListUserDevices(ctx context.Context, userID id.UserID) ([]*device.Device, error) {
	var models []DeviceModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("last_seen_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*device.Device, 0, len(models))
	for i := range models {
		d, err := toDevice(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

func (s *Store) ListDevices(ctx context.Context, limit int) ([]*device.Device, error) {
	var models []DeviceModel
	q := s.sdb.NewSelect(&models).
		OrderExpr("last_seen_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*device.Device, 0, len(models))
	for i := range models {
		d, err := toDevice(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Webhook store
// ──────────────────────────────────────────────────

func (s *Store) CreateWebhook(ctx context.Context, w *webhook.Webhook) error {
	m := fromWebhook(w)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetWebhook(ctx context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	m := new(WebhookModel)
	err := s.sdb.NewSelect(m).Where("id = ?", webhookID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toWebhook(m)
}

func (s *Store) UpdateWebhook(ctx context.Context, w *webhook.Webhook) error {
	m := fromWebhook(w)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error {
	_, err := s.sdb.NewDelete((*WebhookModel)(nil)).Where("id = ?", webhookID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListWebhooks(ctx context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	var models []WebhookModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*webhook.Webhook, 0, len(models))
	for i := range models {
		w, err := toWebhook(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Notification store
// ──────────────────────────────────────────────────

func (s *Store) CreateNotification(ctx context.Context, n *notification.Notification) error {
	m := fromNotification(n)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetNotification(ctx context.Context, notifID id.NotificationID) (*notification.Notification, error) {
	m := new(NotificationModel)
	err := s.sdb.NewSelect(m).Where("id = ?", notifID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toNotification(m)
}

func (s *Store) MarkSent(ctx context.Context, notifID id.NotificationID) error {
	now := time.Now()
	_, err := s.sdb.NewUpdate((*NotificationModel)(nil)).
		Set("sent = TRUE").
		Set("sent_at = ?", now).
		Where("id = ?", notifID.String()).
		Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListUserNotifications(ctx context.Context, userID id.UserID) ([]*notification.Notification, error) {
	var models []NotificationModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*notification.Notification, 0, len(models))
	for i := range models {
		n, err := toNotification(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// API Key store
// ──────────────────────────────────────────────────

func (s *Store) CreateAPIKey(ctx context.Context, k *apikey.APIKey) error {
	m := fromAPIKey(k)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetAPIKey(ctx context.Context, keyID id.APIKeyID) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.sdb.NewSelect(m).Where("id = ?", keyID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toAPIKey(m)
}

func (s *Store) GetAPIKeyByPrefix(ctx context.Context, appID id.AppID, prefix string) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("key_prefix = ?", prefix).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toAPIKey(m)
}

func (s *Store) GetAPIKeyByPublicKey(ctx context.Context, appID id.AppID, publicKey string) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("public_key = ?", publicKey).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toAPIKey(m)
}

func (s *Store) UpdateAPIKey(ctx context.Context, k *apikey.APIKey) error {
	m := fromAPIKey(k)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID id.APIKeyID) error {
	_, err := s.sdb.NewDelete((*APIKeyModel)(nil)).Where("id = ?", keyID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListAPIKeysByApp(ctx context.Context, appID id.AppID) ([]*apikey.APIKey, error) {
	var models []APIKeyModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*apikey.APIKey, 0, len(models))
	for i := range models {
		k, err := toAPIKey(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, nil
}

func (s *Store) ListAPIKeysByUser(ctx context.Context, appID id.AppID, userID id.UserID) ([]*apikey.APIKey, error) {
	var models []APIKeyModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*apikey.APIKey, 0, len(models))
	for i := range models {
		k, err := toAPIKey(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Environment store
// ──────────────────────────────────────────────────

func (s *Store) CreateEnvironment(ctx context.Context, e *environment.Environment) error {
	m := fromEnvironment(e)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetEnvironment(ctx context.Context, envID id.EnvironmentID) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.sdb.NewSelect(m).Where("id = ?", envID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toEnvironment(m)
}

func (s *Store) GetEnvironmentBySlug(ctx context.Context, appID id.AppID, slug string) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toEnvironment(m)
}

func (s *Store) GetDefaultEnvironment(ctx context.Context, appID id.AppID) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("is_default = TRUE").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toEnvironment(m)
}

func (s *Store) UpdateEnvironment(ctx context.Context, e *environment.Environment) error {
	m := fromEnvironment(e)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteEnvironment(ctx context.Context, envID id.EnvironmentID) error {
	m := new(EnvironmentModel)
	err := s.sdb.NewSelect(m).Where("id = ?", envID.String()).Scan(ctx)
	if err != nil {
		return sqliteError(err)
	}
	if m.IsDefault {
		return fmt.Errorf("authsome/sqlite: cannot delete the default environment")
	}
	_, err = s.sdb.NewDelete((*EnvironmentModel)(nil)).Where("id = ?", envID.String()).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListEnvironments(ctx context.Context, appID id.AppID) ([]*environment.Environment, error) {
	var models []EnvironmentModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*environment.Environment, 0, len(models))
	for i := range models {
		e, err := toEnvironment(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *Store) SetDefaultEnvironment(ctx context.Context, appID id.AppID, envID id.EnvironmentID) error {
	_, err := s.sdb.NewUpdate((*EnvironmentModel)(nil)).
		Set("is_default = FALSE").
		Where("app_id = ?", appID.String()).
		Where("is_default = TRUE").
		Exec(ctx)
	if err != nil {
		return sqliteError(err)
	}
	_, err = s.sdb.NewUpdate((*EnvironmentModel)(nil)).
		Set("is_default = TRUE").
		Where("id = ?", envID.String()).
		Exec(ctx)
	return sqliteError(err)
}

// ──────────────────────────────────────────────────
// FormConfig store
// ──────────────────────────────────────────────────

func (s *Store) CreateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := fromFormConfig(fc)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetFormConfig(ctx context.Context, appID id.AppID, formType string) (*formconfig.FormConfig, error) {
	m := new(FormConfigModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("form_type = ?", formType).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toFormConfig(m)
}

func (s *Store) UpdateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := fromFormConfig(fc)
	m.UpdatedAt = time.Now()
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteFormConfig(ctx context.Context, appID id.AppID, formType string) error {
	_, err := s.sdb.NewDelete((*FormConfigModel)(nil)).
		Where("app_id = ?", appID.String()).
		Where("form_type = ?", formType).
		Exec(ctx)
	return sqliteError(err)
}

func (s *Store) ListFormConfigs(ctx context.Context, appID id.AppID) ([]*formconfig.FormConfig, error) {
	var models []FormConfigModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	result := make([]*formconfig.FormConfig, 0, len(models))
	for i := range models {
		fc, err := toFormConfig(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, fc)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Branding store
// ──────────────────────────────────────────────────

func (s *Store) GetBranding(ctx context.Context, orgID id.OrgID) (*formconfig.BrandingConfig, error) {
	m := new(BrandingConfigModel)
	err := s.sdb.NewSelect(m).Where("org_id = ?", orgID.String()).Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return toBrandingConfig(m)
}

func (s *Store) SaveBranding(ctx context.Context, b *formconfig.BrandingConfig) error {
	m := fromBrandingConfig(b)
	m.UpdatedAt = time.Now()
	res, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return sqliteError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // driver always returns valid count
	if n > 0 {
		return nil
	}
	_, err = s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteBranding(ctx context.Context, orgID id.OrgID) error {
	_, err := s.sdb.NewDelete((*BrandingConfigModel)(nil)).Where("org_id = ?", orgID.String()).Exec(ctx)
	return sqliteError(err)
}

// ──────────────────────────────────────────────────
// AppSessionConfig store
// ──────────────────────────────────────────────────

func (s *Store) GetAppSessionConfig(ctx context.Context, appID id.AppID) (*appsessionconfig.Config, error) {
	m := new(AppSessionConfigModel)
	err := s.sdb.NewSelect(m).Where("app_id = ?", appID.String()).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appsessionconfig.ErrNotFound
		}
		return nil, err
	}
	return toAppSessionConfig(m)
}

func (s *Store) SetAppSessionConfig(ctx context.Context, cfg *appsessionconfig.Config) error {
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppSessionConfigID()
	}
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	m := fromAppSessionConfig(cfg)
	res, err := s.sdb.NewUpdate(m).Where("app_id = ?", cfg.AppID.String()).Exec(ctx)
	if err != nil {
		return sqliteError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // driver always returns valid count
	if n > 0 {
		return nil
	}
	_, err = s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) DeleteAppSessionConfig(ctx context.Context, appID id.AppID) error {
	_, err := s.sdb.NewDelete((*AppSessionConfigModel)(nil)).Where("app_id = ?", appID.String()).Exec(ctx)
	return sqliteError(err)
}

// ──────────────────────────────────────────────────
// App Client Config Store
// ──────────────────────────────────────────────────

func (s *Store) GetAppClientConfig(ctx context.Context, appID id.AppID) (*appclientconfig.Config, error) {
	m := new(AppClientConfigModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Scan(ctx)
	if err != nil {
		return nil, appclientconfig.ErrNotFound
	}
	return fromAppClientConfigModel(m)
}

func (s *Store) SetAppClientConfig(ctx context.Context, cfg *appclientconfig.Config) error {
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppClientConfigID()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = time.Now()
	}
	cfg.UpdatedAt = time.Now()

	m := toAppClientConfigModel(cfg)

	res, err := s.sdb.NewUpdate(m).
		Where("app_id = ?", m.AppID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: set app client config (update): %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}

	_, err = s.sdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: set app client config (insert): %w", err)
	}
	return nil
}

func (s *Store) DeleteAppClientConfig(ctx context.Context, appID id.AppID) error {
	res, err := s.sdb.NewDelete((*AppClientConfigModel)(nil)).
		Where("app_id = ?", appID.String()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: delete app client config: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return appclientconfig.ErrNotFound
	}
	return nil
}

// ──────────────────────────────────────────────────
// Settings Store
// ──────────────────────────────────────────────────

func (s *Store) GetSetting(ctx context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	m := new(SettingModel)
	err := s.sdb.NewSelect(m).
		Where("key = ?", key).
		Where("scope = ?", string(scope)).
		Where("scope_id = ?", scopeID).
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	return fromSettingModel(m)
}

func (s *Store) SetSetting(ctx context.Context, st *settings.Setting) error {
	m := toSettingModel(st)

	res, err := s.sdb.NewUpdate(m).
		Where("key = ?", m.Key).
		Where("scope = ?", m.Scope).
		Where("scope_id = ?", m.ScopeID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: set setting (update): %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}

	_, err = s.sdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: set setting (insert): %w", err)
	}
	return nil
}

func (s *Store) DeleteSetting(ctx context.Context, key string, scope settings.Scope, scopeID string) error {
	_, err := s.sdb.NewDelete((*SettingModel)(nil)).
		Where("key = ?", key).
		Where("scope = ?", string(scope)).
		Where("scope_id = ?", scopeID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: delete setting: %w", err)
	}
	return nil
}

func (s *Store) ListSettings(ctx context.Context, opts settings.ListOpts) ([]*settings.Setting, error) {
	var models []SettingModel

	q := s.sdb.NewSelect(&models)
	if opts.Namespace != "" {
		q = q.Where("namespace = ?", opts.Namespace)
	}
	if opts.Scope != "" {
		q = q.Where("scope = ?", string(opts.Scope))
	}
	if opts.ScopeID != "" {
		q = q.Where("scope_id = ?", opts.ScopeID)
	}
	if opts.AppID != "" {
		q = q.Where("app_id = ?", opts.AppID)
	}
	if opts.OrgID != "" {
		q = q.Where("org_id = ?", opts.OrgID)
	}
	q = q.OrderExpr("created_at DESC")
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("authsome/sqlite: list settings: %w", err)
	}

	result := make([]*settings.Setting, 0, len(models))
	for i := range models {
		st, err := fromSettingModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, nil
}

func (s *Store) ResolveSettings(ctx context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	type scopeQuery struct {
		scope   settings.Scope
		scopeID string
	}
	queries := []scopeQuery{
		{settings.ScopeGlobal, ""},
	}
	if opts.AppID != "" {
		queries = append(queries, scopeQuery{settings.ScopeApp, opts.AppID})
	}
	if opts.OrgID != "" {
		queries = append(queries, scopeQuery{settings.ScopeOrg, opts.OrgID})
	}
	if opts.UserID != "" {
		queries = append(queries, scopeQuery{settings.ScopeUser, opts.UserID})
	}

	var result []*settings.Setting
	for _, sq := range queries {
		st, err := s.GetSetting(ctx, key, sq.scope, sq.scopeID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return nil, err
		}
		result = append(result, st)
	}

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if settings.ScopePriority(result[i].Scope) > settings.ScopePriority(result[j].Scope) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (s *Store) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	result := make(map[string][]*settings.Setting, len(keys))
	for _, key := range keys {
		resolved, err := s.ResolveSettings(ctx, key, opts)
		if err != nil {
			return nil, err
		}
		if len(resolved) > 0 {
			result[key] = resolved
		}
	}
	return result, nil
}

func (s *Store) DeleteSettingsByNamespace(ctx context.Context, namespace string) error {
	_, err := s.sdb.NewDelete((*SettingModel)(nil)).
		Where("namespace = ?", namespace).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: delete settings by namespace: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// sqliteError maps sql.ErrNoRows to a standard sentinel and passes through other errors.
func sqliteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	return err
}
