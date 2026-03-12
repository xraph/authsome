// Package postgres implements the AuthSome store interface using grove ORM (PostgreSQL).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"
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

// Store implements store.Store using grove ORM (PostgreSQL).
type Store struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// Compile-time interface checks.
var _ store.Store = (*Store)(nil)

// New creates a new PostgreSQL-backed store.
func New(db *grove.DB) *Store {
	return &Store{
		db: db,
		pg: pgdriver.Unwrap(db),
	}
}

// DB returns the underlying grove.DB for advanced use cases.
func (s *Store) DB() *grove.DB { return s.db }

// ──────────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────────

// Migrate runs all registered migrations via the grove orchestrator.
func (s *Store) Migrate(ctx context.Context, extraGroups ...*migrate.Group) error {
	executor, err := migrate.NewExecutorFor(s.pg)
	if err != nil {
		return fmt.Errorf("authsome/postgres: create migration executor: %w", err)
	}

	groups := make([]*migrate.Group, 0, 1+len(extraGroups))
	groups = append(groups, Migrations)
	groups = append(groups, extraGroups...)

	orch := migrate.NewOrchestrator(executor, groups...)
	if _, err := orch.Migrate(ctx); err != nil {
		return fmt.Errorf("authsome/postgres: migration failed: %w", err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetApp(ctx context.Context, appID id.AppID) (*app.App, error) {
	m := new(AppModel)
	err := s.pg.NewSelect(m).Where("id = ?", appID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toApp(m)
}

func (s *Store) GetAppBySlug(ctx context.Context, slug string) (*app.App, error) {
	m := new(AppModel)
	err := s.pg.NewSelect(m).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toApp(m)
}

func (s *Store) GetAppByPublishableKey(ctx context.Context, key string) (*app.App, error) {
	m := new(AppModel)
	err := s.pg.NewSelect(m).Where("publishable_key = ?", key).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toApp(m)
}

func (s *Store) UpdateApp(ctx context.Context, a *app.App) error {
	m := fromApp(a)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteApp(ctx context.Context, appID id.AppID) error {
	_, err := s.pg.NewDelete((*AppModel)(nil)).Where("id = ?", appID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListApps(ctx context.Context) ([]*app.App, error) {
	var models []AppModel
	err := s.pg.NewSelect(&models).OrderExpr("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	m := new(UserModel)
	err := s.pg.NewSelect(m).Where("id = ?", userID.String()).Where("deleted_at IS NULL").Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByEmail(ctx context.Context, appID id.AppID, email string) (*user.User, error) {
	m := new(UserModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", email).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByPhone(ctx context.Context, appID id.AppID, phone string) (*user.User, error) {
	m := new(UserModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("phone = ?", phone).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toUser(m)
}

func (s *Store) GetUserByUsername(ctx context.Context, appID id.AppID, username string) (*user.User, error) {
	m := new(UserModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("username = ?", username).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toUser(m)
}

func (s *Store) UpdateUser(ctx context.Context, u *user.User) error {
	m := fromUser(u)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteUser(ctx context.Context, userID id.UserID) error {
	now := time.Now()
	_, err := s.pg.NewUpdate((*UserModel)(nil)).
		Set("deleted_at = ?", now).
		Where("id = ?", userID.String()).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return pgError(err)
}

func (s *Store) ListUsers(ctx context.Context, q *user.Query) (*user.List, error) {
	var models []UserModel
	query := s.pg.NewSelect(&models).
		Where("app_id = ?", q.AppID.String()).
		Where("deleted_at IS NULL")

	if !q.EnvID.IsNil() {
		query = query.Where("env_id = ?", q.EnvID.String())
	}

	if q.Email != "" {
		query = query.Where("email ILIKE ?", "%"+q.Email+"%")
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
		return nil, pgError(err)
	}

	list := &user.List{
		Users: make([]*user.User, 0, len(models)),
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetSession(ctx context.Context, sessionID id.SessionID) (*session.Session, error) {
	m := new(SessionModel)
	err := s.pg.NewSelect(m).Where("id = ?", sessionID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toSession(m)
}

func (s *Store) GetSessionByToken(ctx context.Context, token string) (*session.Session, error) {
	m := new(SessionModel)
	err := s.pg.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toSession(m)
}

func (s *Store) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*session.Session, error) {
	m := new(SessionModel)
	err := s.pg.NewSelect(m).Where("refresh_token = ?", refreshToken).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toSession(m)
}

func (s *Store) UpdateSession(ctx context.Context, sess *session.Session) error {
	m := fromSession(sess)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteSession(ctx context.Context, sessionID id.SessionID) error {
	_, err := s.pg.NewDelete((*SessionModel)(nil)).Where("id = ?", sessionID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID id.UserID) error {
	_, err := s.pg.NewDelete((*SessionModel)(nil)).Where("user_id = ?", userID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListUserSessions(ctx context.Context, userID id.UserID) ([]*session.Session, error) {
	var models []SessionModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	q := s.pg.NewSelect(&models).
		OrderExpr("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetVerification(ctx context.Context, token string) (*account.Verification, error) {
	m := new(VerificationModel)
	err := s.pg.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toVerification(m)
}

func (s *Store) ConsumeVerification(ctx context.Context, token string) error {
	_, err := s.pg.NewUpdate((*VerificationModel)(nil)).
		Set("consumed = TRUE").
		Where("token = ?", token).
		Where("consumed = FALSE").
		Exec(ctx)
	return pgError(err)
}

func (s *Store) CreatePasswordReset(ctx context.Context, pr *account.PasswordReset) error {
	m := fromPasswordReset(pr)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetPasswordReset(ctx context.Context, token string) (*account.PasswordReset, error) {
	m := new(PasswordResetModel)
	err := s.pg.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toPasswordReset(m)
}

func (s *Store) ConsumePasswordReset(ctx context.Context, token string) error {
	_, err := s.pg.NewUpdate((*PasswordResetModel)(nil)).
		Set("consumed = TRUE").
		Where("token = ?", token).
		Where("consumed = FALSE").
		Exec(ctx)
	return pgError(err)
}

// ──────────────────────────────────────────────────
// Organization store
// ──────────────────────────────────────────────────

func (s *Store) CreateOrganization(ctx context.Context, o *organization.Organization) error {
	m := fromOrganization(o)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetOrganization(ctx context.Context, orgID id.OrgID) (*organization.Organization, error) {
	m := new(OrganizationModel)
	err := s.pg.NewSelect(m).Where("id = ?", orgID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toOrganization(m)
}

func (s *Store) GetOrganizationBySlug(ctx context.Context, appID id.AppID, slug string) (*organization.Organization, error) {
	m := new(OrganizationModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toOrganization(m)
}

func (s *Store) UpdateOrganization(ctx context.Context, o *organization.Organization) error {
	m := fromOrganization(o)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteOrganization(ctx context.Context, orgID id.OrgID) error {
	_, err := s.pg.NewDelete((*OrganizationModel)(nil)).Where("id = ?", orgID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListOrganizations(ctx context.Context, appID id.AppID) ([]*organization.Organization, error) {
	var models []OrganizationModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toOrganizations(models)
}

func (s *Store) ListUserOrganizations(ctx context.Context, userID id.UserID) ([]*organization.Organization, error) {
	var models []OrganizationModel
	err := s.pg.NewSelect(&models).
		Join("JOIN", "authsome_members AS mem", "mem.org_id = o.id").
		Where("mem.user_id = ?", userID.String()).
		OrderExpr("o.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(model).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetMember(ctx context.Context, memberID id.MemberID) (*organization.Member, error) {
	m := new(MemberModel)
	err := s.pg.NewSelect(m).Where("id = ?", memberID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toMember(m)
}

func (s *Store) GetMemberByUserAndOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) (*organization.Member, error) {
	m := new(MemberModel)
	err := s.pg.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("org_id = ?", orgID.String()).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toMember(m)
}

func (s *Store) UpdateMember(ctx context.Context, mem *organization.Member) error {
	m := fromMember(mem)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteMember(ctx context.Context, memberID id.MemberID) error {
	_, err := s.pg.NewDelete((*MemberModel)(nil)).Where("id = ?", memberID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListMembers(ctx context.Context, orgID id.OrgID) ([]*organization.Member, error) {
	var models []MemberModel
	err := s.pg.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetInvitation(ctx context.Context, invID id.InvitationID) (*organization.Invitation, error) {
	m := new(InvitationModel)
	err := s.pg.NewSelect(m).Where("id = ?", invID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toInvitation(m)
}

func (s *Store) GetInvitationByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	m := new(InvitationModel)
	err := s.pg.NewSelect(m).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toInvitation(m)
}

func (s *Store) UpdateInvitation(ctx context.Context, inv *organization.Invitation) error {
	m := fromInvitation(inv)
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) ListInvitations(ctx context.Context, orgID id.OrgID) ([]*organization.Invitation, error) {
	var models []InvitationModel
	err := s.pg.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetTeam(ctx context.Context, teamID id.TeamID) (*organization.Team, error) {
	m := new(TeamModel)
	err := s.pg.NewSelect(m).Where("id = ?", teamID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toTeam(m)
}

func (s *Store) UpdateTeam(ctx context.Context, t *organization.Team) error {
	m := fromTeam(t)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteTeam(ctx context.Context, teamID id.TeamID) error {
	_, err := s.pg.NewDelete((*TeamModel)(nil)).Where("id = ?", teamID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListTeams(ctx context.Context, orgID id.OrgID) ([]*organization.Team, error) {
	var models []TeamModel
	err := s.pg.NewSelect(&models).
		Where("org_id = ?", orgID.String()).
		OrderExpr("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetDevice(ctx context.Context, deviceID id.DeviceID) (*device.Device, error) {
	m := new(DeviceModel)
	err := s.pg.NewSelect(m).Where("id = ?", deviceID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toDevice(m)
}

func (s *Store) GetDeviceByFingerprint(ctx context.Context, userID id.UserID, fingerprint string) (*device.Device, error) {
	m := new(DeviceModel)
	err := s.pg.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("fingerprint = ?", fingerprint).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toDevice(m)
}

func (s *Store) UpdateDevice(ctx context.Context, d *device.Device) error {
	m := fromDevice(d)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteDevice(ctx context.Context, deviceID id.DeviceID) error {
	_, err := s.pg.NewDelete((*DeviceModel)(nil)).Where("id = ?", deviceID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListUserDevices(ctx context.Context, userID id.UserID) ([]*device.Device, error) {
	var models []DeviceModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("last_seen_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	q := s.pg.NewSelect(&models).
		OrderExpr("last_seen_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetWebhook(ctx context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	m := new(WebhookModel)
	err := s.pg.NewSelect(m).Where("id = ?", webhookID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toWebhook(m)
}

func (s *Store) UpdateWebhook(ctx context.Context, w *webhook.Webhook) error {
	m := fromWebhook(w)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error {
	_, err := s.pg.NewDelete((*WebhookModel)(nil)).Where("id = ?", webhookID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListWebhooks(ctx context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	var models []WebhookModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetNotification(ctx context.Context, notifID id.NotificationID) (*notification.Notification, error) {
	m := new(NotificationModel)
	err := s.pg.NewSelect(m).Where("id = ?", notifID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toNotification(m)
}

func (s *Store) MarkSent(ctx context.Context, notifID id.NotificationID) error {
	now := time.Now()
	_, err := s.pg.NewUpdate((*NotificationModel)(nil)).
		Set("sent = TRUE").
		Set("sent_at = ?", now).
		Where("id = ?", notifID.String()).
		Exec(ctx)
	return pgError(err)
}

func (s *Store) ListUserNotifications(ctx context.Context, userID id.UserID) ([]*notification.Notification, error) {
	var models []NotificationModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetAPIKey(ctx context.Context, keyID id.APIKeyID) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.pg.NewSelect(m).Where("id = ?", keyID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toAPIKey(m)
}

func (s *Store) GetAPIKeyByPrefix(ctx context.Context, appID id.AppID, prefix string) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("key_prefix = ?", prefix).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toAPIKey(m)
}

func (s *Store) GetAPIKeyByPublicKey(ctx context.Context, appID id.AppID, publicKey string) (*apikey.APIKey, error) {
	m := new(APIKeyModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("public_key = ?", publicKey).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toAPIKey(m)
}

func (s *Store) UpdateAPIKey(ctx context.Context, k *apikey.APIKey) error {
	m := fromAPIKey(k)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID id.APIKeyID) error {
	_, err := s.pg.NewDelete((*APIKeyModel)(nil)).Where("id = ?", keyID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListAPIKeysByApp(ctx context.Context, appID id.AppID) ([]*apikey.APIKey, error) {
	var models []APIKeyModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetEnvironment(ctx context.Context, envID id.EnvironmentID) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.pg.NewSelect(m).Where("id = ?", envID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toEnvironment(m)
}

func (s *Store) GetEnvironmentBySlug(ctx context.Context, appID id.AppID, slug string) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toEnvironment(m)
}

func (s *Store) GetDefaultEnvironment(ctx context.Context, appID id.AppID) (*environment.Environment, error) {
	m := new(EnvironmentModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("is_default = TRUE").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toEnvironment(m)
}

func (s *Store) UpdateEnvironment(ctx context.Context, e *environment.Environment) error {
	m := fromEnvironment(e)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteEnvironment(ctx context.Context, envID id.EnvironmentID) error {
	m := new(EnvironmentModel)
	err := s.pg.NewSelect(m).Where("id = ?", envID.String()).Scan(ctx)
	if err != nil {
		return pgError(err)
	}
	if m.IsDefault {
		return fmt.Errorf("authsome/postgres: cannot delete the default environment")
	}
	_, err = s.pg.NewDelete((*EnvironmentModel)(nil)).Where("id = ?", envID.String()).Exec(ctx)
	return pgError(err)
}

func (s *Store) ListEnvironments(ctx context.Context, appID id.AppID) ([]*environment.Environment, error) {
	var models []EnvironmentModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	_, err := s.pg.NewUpdate((*EnvironmentModel)(nil)).
		Set("is_default = FALSE").
		Where("app_id = ?", appID.String()).
		Where("is_default = TRUE").
		Exec(ctx)
	if err != nil {
		return pgError(err)
	}
	_, err = s.pg.NewUpdate((*EnvironmentModel)(nil)).
		Set("is_default = TRUE").
		Where("id = ?", envID.String()).
		Exec(ctx)
	return pgError(err)
}

// ──────────────────────────────────────────────────
// FormConfig store
// ──────────────────────────────────────────────────

func (s *Store) CreateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := fromFormConfig(fc)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) GetFormConfig(ctx context.Context, appID id.AppID, formType string) (*formconfig.FormConfig, error) {
	m := new(FormConfigModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("form_type = ?", formType).
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toFormConfig(m)
}

func (s *Store) UpdateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := fromFormConfig(fc)
	m.UpdatedAt = time.Now()
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteFormConfig(ctx context.Context, appID id.AppID, formType string) error {
	_, err := s.pg.NewDelete((*FormConfigModel)(nil)).
		Where("app_id = ?", appID.String()).
		Where("form_type = ?", formType).
		Exec(ctx)
	return pgError(err)
}

func (s *Store) ListFormConfigs(ctx context.Context, appID id.AppID) ([]*formconfig.FormConfig, error) {
	var models []FormConfigModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, pgError(err)
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
	err := s.pg.NewSelect(m).Where("org_id = ?", orgID.String()).Scan(ctx)
	if err != nil {
		return nil, pgError(err)
	}
	return toBrandingConfig(m)
}

func (s *Store) SaveBranding(ctx context.Context, b *formconfig.BrandingConfig) error {
	// Try update first; if no rows affected, insert.
	m := fromBrandingConfig(b)
	m.UpdatedAt = time.Now()
	res, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return pgError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // driver always returns valid count
	if n > 0 {
		return nil
	}
	_, err = s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteBranding(ctx context.Context, orgID id.OrgID) error {
	_, err := s.pg.NewDelete((*BrandingConfigModel)(nil)).Where("org_id = ?", orgID.String()).Exec(ctx)
	return pgError(err)
}

// ──────────────────────────────────────────────────
// AppSessionConfig store
// ──────────────────────────────────────────────────

func (s *Store) GetAppSessionConfig(ctx context.Context, appID id.AppID) (*appsessionconfig.Config, error) {
	m := new(AppSessionConfigModel)
	err := s.pg.NewSelect(m).Where("app_id = ?", appID.String()).Scan(ctx)
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
	cfg.UpdatedAt = now
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	m := fromAppSessionConfig(cfg)
	// Try update first; if no rows affected, insert.
	res, err := s.pg.NewUpdate(m).Where("app_id = ?", cfg.AppID.String()).Exec(ctx)
	if err != nil {
		return pgError(err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck // driver always returns valid count
	if n > 0 {
		return nil
	}
	_, err = s.pg.NewInsert(m).Exec(ctx)
	return pgError(err)
}

func (s *Store) DeleteAppSessionConfig(ctx context.Context, appID id.AppID) error {
	_, err := s.pg.NewDelete((*AppSessionConfigModel)(nil)).Where("app_id = ?", appID.String()).Exec(ctx)
	return pgError(err)
}

// ──────────────────────────────────────────────────
// App Client Config Store
// ──────────────────────────────────────────────────

func (s *Store) GetAppClientConfig(_ context.Context, _ id.AppID) (*appclientconfig.Config, error) {
	return nil, appclientconfig.ErrNotFound
}

func (s *Store) SetAppClientConfig(_ context.Context, _ *appclientconfig.Config) error {
	return fmt.Errorf("postgres: app client config not implemented")
}

func (s *Store) DeleteAppClientConfig(_ context.Context, _ id.AppID) error {
	return appclientconfig.ErrNotFound
}

// ──────────────────────────────────────────────────
// Settings Store
// ──────────────────────────────────────────────────

func (s *Store) GetSetting(_ context.Context, _ string, _ settings.Scope, _ string) (*settings.Setting, error) {
	return nil, store.ErrNotFound
}

func (s *Store) SetSetting(_ context.Context, _ *settings.Setting) error {
	return fmt.Errorf("postgres: settings not implemented")
}

func (s *Store) DeleteSetting(_ context.Context, _ string, _ settings.Scope, _ string) error {
	return nil
}

func (s *Store) ListSettings(_ context.Context, _ settings.ListOpts) ([]*settings.Setting, error) {
	return nil, nil
}

func (s *Store) ResolveSettings(_ context.Context, _ string, _ settings.ResolveOpts) ([]*settings.Setting, error) {
	return nil, nil
}

func (s *Store) BatchResolve(_ context.Context, _ []string, _ settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	return nil, nil
}

func (s *Store) DeleteSettingsByNamespace(_ context.Context, _ string) error {
	return nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// pgError maps sql.ErrNoRows to a standard sentinel and passes through other errors.
func pgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	return err
}
