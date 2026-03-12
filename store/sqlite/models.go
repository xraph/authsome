package sqlite

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// ──────────────────────────────────────────────────
// App model
// ──────────────────────────────────────────────────

type AppModel struct {
	grove.BaseModel `grove:"table:authsome_apps,alias:a"`

	ID             string          `grove:"id,pk"`
	Name           string          `grove:"name,notnull"`
	Slug           string          `grove:"slug,notnull"`
	Logo           string          `grove:"logo"`
	PublishableKey string          `grove:"publishable_key"`
	IsPlatform     bool            `grove:"is_platform"`
	Metadata       json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedAt      time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt      time.Time       `grove:"updated_at,notnull,default:now()"`
}

func toApp(m *AppModel) (*app.App, error) {
	appID, err := id.ParseAppID(m.ID)
	if err != nil {
		return nil, err
	}
	md := make(app.Metadata)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}
	return &app.App{
		ID:             appID,
		Name:           m.Name,
		Slug:           m.Slug,
		Logo:           m.Logo,
		PublishableKey: m.PublishableKey,
		IsPlatform:     m.IsPlatform,
		Metadata:       md,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}, nil
}

func fromApp(a *app.App) *AppModel {
	md, _ := json.Marshal(a.Metadata) //nolint:errcheck // best-effort encode
	return &AppModel{
		ID:             a.ID.String(),
		Name:           a.Name,
		Slug:           a.Slug,
		Logo:           a.Logo,
		PublishableKey: a.PublishableKey,
		IsPlatform:     a.IsPlatform,
		Metadata:       md,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// User model
// ──────────────────────────────────────────────────

type UserModel struct {
	grove.BaseModel `grove:"table:authsome_users,alias:u"`

	ID              string          `grove:"id,pk"`
	AppID           string          `grove:"app_id,notnull"`
	EnvID           string          `grove:"env_id,notnull"`
	Email           string          `grove:"email,notnull"`
	EmailVerified   bool            `grove:"email_verified"`
	FirstName       string          `grove:"first_name"`
	LastName        string          `grove:"last_name"`
	Image           string          `grove:"image"`
	Username        string          `grove:"username"`
	DisplayUsername string          `grove:"display_username"`
	Phone           string          `grove:"phone"`
	PhoneVerified   bool            `grove:"phone_verified"`
	PasswordHash    string          `grove:"password_hash"`
	Banned          bool            `grove:"banned"`
	BanReason       string          `grove:"ban_reason"`
	BanExpires      sql.NullTime    `grove:"ban_expires"`
	Metadata        json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedAt       time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt       time.Time       `grove:"updated_at,notnull,default:now()"`
	DeletedAt       sql.NullTime    `grove:"deleted_at"`
}

func toUser(m *UserModel) (*user.User, error) {
	userID, err := id.ParseUserID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	md := make(user.Metadata)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}
	u := &user.User{
		ID:              userID,
		AppID:           appID,
		EnvID:           envID,
		Email:           m.Email,
		EmailVerified:   m.EmailVerified,
		FirstName:       m.FirstName,
		LastName:        m.LastName,
		Image:           m.Image,
		Username:        m.Username,
		DisplayUsername: m.DisplayUsername,
		Phone:           m.Phone,
		PhoneVerified:   m.PhoneVerified,
		PasswordHash:    m.PasswordHash,
		Banned:          m.Banned,
		BanReason:       m.BanReason,
		Metadata:        md,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.BanExpires.Valid {
		u.BanExpires = &m.BanExpires.Time
	}
	if m.DeletedAt.Valid {
		u.DeletedAt = &m.DeletedAt.Time
	}
	return u, nil
}

func fromUser(u *user.User) *UserModel {
	md, _ := json.Marshal(u.Metadata) //nolint:errcheck // best-effort encode
	m := &UserModel{
		ID:              u.ID.String(),
		AppID:           u.AppID.String(),
		EnvID:           u.EnvID.String(),
		Email:           u.Email,
		EmailVerified:   u.EmailVerified,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Image:           u.Image,
		Username:        u.Username,
		DisplayUsername: u.DisplayUsername,
		Phone:           u.Phone,
		PhoneVerified:   u.PhoneVerified,
		PasswordHash:    u.PasswordHash,
		Banned:          u.Banned,
		BanReason:       u.BanReason,
		Metadata:        md,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
	if u.BanExpires != nil {
		m.BanExpires = sql.NullTime{Time: *u.BanExpires, Valid: true}
	}
	if u.DeletedAt != nil {
		m.DeletedAt = sql.NullTime{Time: *u.DeletedAt, Valid: true}
	}
	return m
}

// ──────────────────────────────────────────────────
// Session model
// ──────────────────────────────────────────────────

type SessionModel struct {
	grove.BaseModel `grove:"table:authsome_sessions,alias:s"`

	ID                    string    `grove:"id,pk"`
	AppID                 string    `grove:"app_id,notnull"`
	EnvID                 string    `grove:"env_id,notnull"`
	UserID                string    `grove:"user_id,notnull"`
	OrgID                 string    `grove:"org_id"`
	Token                 string    `grove:"token,notnull"`
	RefreshToken          string    `grove:"refresh_token,notnull"`
	IPAddress             string    `grove:"ip_address"`
	UserAgent             string    `grove:"user_agent"`
	DeviceID              string    `grove:"device_id"`
	ImpersonatedBy        string    `grove:"impersonated_by"`
	ExpiresAt             time.Time `grove:"expires_at,notnull"`
	RefreshTokenExpiresAt time.Time `grove:"refresh_token_expires_at,notnull"`
	CreatedAt             time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt             time.Time `grove:"updated_at,notnull,default:now()"`
}

func toSession(m *SessionModel) (*session.Session, error) {
	sessID, err := id.ParseSessionID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	s := &session.Session{
		ID:                    sessID,
		AppID:                 appID,
		EnvID:                 envID,
		UserID:                userID,
		Token:                 m.Token,
		RefreshToken:          m.RefreshToken,
		IPAddress:             m.IPAddress,
		UserAgent:             m.UserAgent,
		ExpiresAt:             m.ExpiresAt,
		RefreshTokenExpiresAt: m.RefreshTokenExpiresAt,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
	if m.OrgID != "" {
		orgID, err := id.ParseOrgID(m.OrgID)
		if err != nil {
			return nil, err
		}
		s.OrgID = orgID
	}
	if m.DeviceID != "" {
		devID, err := id.ParseDeviceID(m.DeviceID)
		if err != nil {
			return nil, err
		}
		s.DeviceID = devID
	}
	if m.ImpersonatedBy != "" {
		impID, err := id.ParseUserID(m.ImpersonatedBy)
		if err != nil {
			return nil, err
		}
		s.ImpersonatedBy = impID
	}
	return s, nil
}

func fromSession(s *session.Session) *SessionModel {
	m := &SessionModel{
		ID:                    s.ID.String(),
		AppID:                 s.AppID.String(),
		EnvID:                 s.EnvID.String(),
		UserID:                s.UserID.String(),
		Token:                 s.Token,
		RefreshToken:          s.RefreshToken,
		IPAddress:             s.IPAddress,
		UserAgent:             s.UserAgent,
		ExpiresAt:             s.ExpiresAt,
		RefreshTokenExpiresAt: s.RefreshTokenExpiresAt,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
	if s.OrgID.Prefix() != "" {
		m.OrgID = s.OrgID.String()
	}
	if s.DeviceID.Prefix() != "" {
		m.DeviceID = s.DeviceID.String()
	}
	if s.ImpersonatedBy.Prefix() != "" {
		m.ImpersonatedBy = s.ImpersonatedBy.String()
	}
	return m
}

// ──────────────────────────────────────────────────
// Verification model
// ──────────────────────────────────────────────────

type VerificationModel struct {
	grove.BaseModel `grove:"table:authsome_verifications,alias:v"`

	ID        string    `grove:"id,pk"`
	AppID     string    `grove:"app_id,notnull"`
	EnvID     string    `grove:"env_id,notnull"`
	UserID    string    `grove:"user_id,notnull"`
	Token     string    `grove:"token,notnull"`
	Type      string    `grove:"type,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	Consumed  bool      `grove:"consumed"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
}

func toVerification(m *VerificationModel) (*account.Verification, error) {
	vID, err := id.ParseVerificationID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &account.Verification{
		ID:        vID,
		AppID:     appID,
		EnvID:     envID,
		UserID:    userID,
		Token:     m.Token,
		Type:      account.VerificationType(m.Type),
		ExpiresAt: m.ExpiresAt,
		Consumed:  m.Consumed,
		CreatedAt: m.CreatedAt,
	}, nil
}

func fromVerification(v *account.Verification) *VerificationModel {
	return &VerificationModel{
		ID:        v.ID.String(),
		AppID:     v.AppID.String(),
		EnvID:     v.EnvID.String(),
		UserID:    v.UserID.String(),
		Token:     v.Token,
		Type:      string(v.Type),
		ExpiresAt: v.ExpiresAt,
		Consumed:  v.Consumed,
		CreatedAt: v.CreatedAt,
	}
}

// ──────────────────────────────────────────────────
// Password reset model
// ──────────────────────────────────────────────────

type PasswordResetModel struct {
	grove.BaseModel `grove:"table:authsome_password_resets,alias:pr"`

	ID        string    `grove:"id,pk"`
	AppID     string    `grove:"app_id,notnull"`
	EnvID     string    `grove:"env_id,notnull"`
	UserID    string    `grove:"user_id,notnull"`
	Token     string    `grove:"token,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	Consumed  bool      `grove:"consumed"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
}

func toPasswordReset(m *PasswordResetModel) (*account.PasswordReset, error) {
	prID, err := id.ParsePasswordResetID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &account.PasswordReset{
		ID:        prID,
		AppID:     appID,
		EnvID:     envID,
		UserID:    userID,
		Token:     m.Token,
		ExpiresAt: m.ExpiresAt,
		Consumed:  m.Consumed,
		CreatedAt: m.CreatedAt,
	}, nil
}

func fromPasswordReset(pr *account.PasswordReset) *PasswordResetModel {
	return &PasswordResetModel{
		ID:        pr.ID.String(),
		AppID:     pr.AppID.String(),
		EnvID:     pr.EnvID.String(),
		UserID:    pr.UserID.String(),
		Token:     pr.Token,
		ExpiresAt: pr.ExpiresAt,
		Consumed:  pr.Consumed,
		CreatedAt: pr.CreatedAt,
	}
}

// ──────────────────────────────────────────────────
// Organization model
// ──────────────────────────────────────────────────

type OrganizationModel struct {
	grove.BaseModel `grove:"table:authsome_organizations,alias:o"`

	ID        string          `grove:"id,pk"`
	AppID     string          `grove:"app_id,notnull"`
	EnvID     string          `grove:"env_id,notnull"`
	Name      string          `grove:"name,notnull"`
	Slug      string          `grove:"slug,notnull"`
	Logo      string          `grove:"logo"`
	Metadata  json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedBy string          `grove:"created_by,notnull"`
	CreatedAt time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time       `grove:"updated_at,notnull,default:now()"`
}

func toOrganization(m *OrganizationModel) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	createdBy, err := id.ParseUserID(m.CreatedBy)
	if err != nil {
		return nil, err
	}
	md := make(organization.Metadata)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}
	return &organization.Organization{
		ID:        orgID,
		AppID:     appID,
		EnvID:     envID,
		Name:      m.Name,
		Slug:      m.Slug,
		Logo:      m.Logo,
		Metadata:  md,
		CreatedBy: createdBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromOrganization(o *organization.Organization) *OrganizationModel {
	md, _ := json.Marshal(o.Metadata) //nolint:errcheck // best-effort encode
	return &OrganizationModel{
		ID:        o.ID.String(),
		AppID:     o.AppID.String(),
		EnvID:     o.EnvID.String(),
		Name:      o.Name,
		Slug:      o.Slug,
		Logo:      o.Logo,
		Metadata:  md,
		CreatedBy: o.CreatedBy.String(),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Member model
// ──────────────────────────────────────────────────

type MemberModel struct {
	grove.BaseModel `grove:"table:authsome_members,alias:m"`

	ID        string    `grove:"id,pk"`
	OrgID     string    `grove:"org_id,notnull"`
	UserID    string    `grove:"user_id,notnull"`
	Role      string    `grove:"role,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `grove:"updated_at,notnull,default:now()"`
}

func toMember(m *MemberModel) (*organization.Member, error) {
	memID, err := id.ParseMemberID(m.ID)
	if err != nil {
		return nil, err
	}
	orgID, err := id.ParseOrgID(m.OrgID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &organization.Member{
		ID:        memID,
		OrgID:     orgID,
		UserID:    userID,
		Role:      organization.MemberRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromMember(mem *organization.Member) *MemberModel {
	return &MemberModel{
		ID:        mem.ID.String(),
		OrgID:     mem.OrgID.String(),
		UserID:    mem.UserID.String(),
		Role:      string(mem.Role),
		CreatedAt: mem.CreatedAt,
		UpdatedAt: mem.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Invitation model
// ──────────────────────────────────────────────────

type InvitationModel struct {
	grove.BaseModel `grove:"table:authsome_invitations,alias:inv"`

	ID        string    `grove:"id,pk"`
	OrgID     string    `grove:"org_id,notnull"`
	Email     string    `grove:"email,notnull"`
	Role      string    `grove:"role,notnull"`
	InviterID string    `grove:"inviter_id,notnull"`
	Status    string    `grove:"status,notnull"`
	Token     string    `grove:"token,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
}

func toInvitation(m *InvitationModel) (*organization.Invitation, error) {
	invID, err := id.ParseInvitationID(m.ID)
	if err != nil {
		return nil, err
	}
	orgID, err := id.ParseOrgID(m.OrgID)
	if err != nil {
		return nil, err
	}
	inviterID, err := id.ParseUserID(m.InviterID)
	if err != nil {
		return nil, err
	}
	return &organization.Invitation{
		ID:        invID,
		OrgID:     orgID,
		Email:     m.Email,
		Role:      organization.MemberRole(m.Role),
		InviterID: inviterID,
		Status:    organization.InvitationStatus(m.Status),
		Token:     m.Token,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

func fromInvitation(inv *organization.Invitation) *InvitationModel {
	return &InvitationModel{
		ID:        inv.ID.String(),
		OrgID:     inv.OrgID.String(),
		Email:     inv.Email,
		Role:      string(inv.Role),
		InviterID: inv.InviterID.String(),
		Status:    string(inv.Status),
		Token:     inv.Token,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}
}

// ──────────────────────────────────────────────────
// Team model
// ──────────────────────────────────────────────────

type TeamModel struct {
	grove.BaseModel `grove:"table:authsome_teams,alias:t"`

	ID        string    `grove:"id,pk"`
	OrgID     string    `grove:"org_id,notnull"`
	Name      string    `grove:"name,notnull"`
	Slug      string    `grove:"slug,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `grove:"updated_at,notnull,default:now()"`
}

func toTeam(m *TeamModel) (*organization.Team, error) {
	teamID, err := id.ParseTeamID(m.ID)
	if err != nil {
		return nil, err
	}
	orgID, err := id.ParseOrgID(m.OrgID)
	if err != nil {
		return nil, err
	}
	return &organization.Team{
		ID:        teamID,
		OrgID:     orgID,
		Name:      m.Name,
		Slug:      m.Slug,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromTeam(t *organization.Team) *TeamModel {
	return &TeamModel{
		ID:        t.ID.String(),
		OrgID:     t.OrgID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Device model
// ──────────────────────────────────────────────────

type DeviceModel struct {
	grove.BaseModel `grove:"table:authsome_devices,alias:d"`

	ID          string    `grove:"id,pk"`
	UserID      string    `grove:"user_id,notnull"`
	AppID       string    `grove:"app_id,notnull"`
	EnvID       string    `grove:"env_id,notnull"`
	Name        string    `grove:"name"`
	Type        string    `grove:"type"`
	Browser     string    `grove:"browser"`
	OS          string    `grove:"os"`
	IPAddress   string    `grove:"ip_address"`
	Fingerprint string    `grove:"fingerprint"`
	Trusted     bool      `grove:"trusted"`
	LastSeenAt  time.Time `grove:"last_seen_at,notnull,default:now()"`
	CreatedAt   time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `grove:"updated_at,notnull,default:now()"`
}

func toDevice(m *DeviceModel) (*device.Device, error) {
	devID, err := id.ParseDeviceID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	return &device.Device{
		ID:          devID,
		UserID:      userID,
		AppID:       appID,
		EnvID:       envID,
		Name:        m.Name,
		Type:        m.Type,
		Browser:     m.Browser,
		OS:          m.OS,
		IPAddress:   m.IPAddress,
		Fingerprint: m.Fingerprint,
		Trusted:     m.Trusted,
		LastSeenAt:  m.LastSeenAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}, nil
}

func fromDevice(d *device.Device) *DeviceModel {
	return &DeviceModel{
		ID:          d.ID.String(),
		UserID:      d.UserID.String(),
		AppID:       d.AppID.String(),
		EnvID:       d.EnvID.String(),
		Name:        d.Name,
		Type:        d.Type,
		Browser:     d.Browser,
		OS:          d.OS,
		IPAddress:   d.IPAddress,
		Fingerprint: d.Fingerprint,
		Trusted:     d.Trusted,
		LastSeenAt:  d.LastSeenAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Webhook model
// ──────────────────────────────────────────────────

type WebhookModel struct {
	grove.BaseModel `grove:"table:authsome_webhooks,alias:wh"`

	ID        string          `grove:"id,pk"`
	AppID     string          `grove:"app_id,notnull"`
	EnvID     string          `grove:"env_id,notnull"`
	URL       string          `grove:"url,notnull"`
	Events    json.RawMessage `grove:"events,type:jsonb"`
	Secret    string          `grove:"secret"`
	Active    bool            `grove:"active"`
	CreatedAt time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time       `grove:"updated_at,notnull,default:now()"`
}

func toWebhook(m *WebhookModel) (*webhook.Webhook, error) {
	whID, err := id.ParseWebhookID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	var events []string
	if len(m.Events) > 0 {
		_ = json.Unmarshal(m.Events, &events) //nolint:errcheck // best-effort decode
	}
	return &webhook.Webhook{
		ID:        whID,
		AppID:     appID,
		EnvID:     envID,
		URL:       m.URL,
		Events:    events,
		Secret:    m.Secret,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromWebhook(w *webhook.Webhook) *WebhookModel {
	events, _ := json.Marshal(w.Events) //nolint:errcheck // best-effort encode
	return &WebhookModel{
		ID:        w.ID.String(),
		AppID:     w.AppID.String(),
		EnvID:     w.EnvID.String(),
		URL:       w.URL,
		Events:    events,
		Secret:    w.Secret,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Notification model
// ──────────────────────────────────────────────────

type NotificationModel struct {
	grove.BaseModel `grove:"table:authsome_notifications,alias:n"`

	ID        string       `grove:"id,pk"`
	AppID     string       `grove:"app_id,notnull"`
	EnvID     string       `grove:"env_id,notnull"`
	UserID    string       `grove:"user_id,notnull"`
	Type      string       `grove:"type,notnull"`
	Channel   string       `grove:"channel,notnull"`
	Subject   string       `grove:"subject"`
	Body      string       `grove:"body"`
	Sent      bool         `grove:"sent"`
	SentAt    sql.NullTime `grove:"sent_at"`
	CreatedAt time.Time    `grove:"created_at,notnull,default:now()"`
}

func toNotification(m *NotificationModel) (*notification.Notification, error) {
	nID, err := id.ParseNotificationID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	n := &notification.Notification{
		ID:        nID,
		AppID:     appID,
		EnvID:     envID,
		UserID:    userID,
		Type:      m.Type,
		Channel:   notification.Channel(m.Channel),
		Subject:   m.Subject,
		Body:      m.Body,
		Sent:      m.Sent,
		CreatedAt: m.CreatedAt,
	}
	if m.SentAt.Valid {
		n.SentAt = &m.SentAt.Time
	}
	return n, nil
}

func fromNotification(n *notification.Notification) *NotificationModel {
	m := &NotificationModel{
		ID:        n.ID.String(),
		AppID:     n.AppID.String(),
		EnvID:     n.EnvID.String(),
		UserID:    n.UserID.String(),
		Type:      n.Type,
		Channel:   string(n.Channel),
		Subject:   n.Subject,
		Body:      n.Body,
		Sent:      n.Sent,
		CreatedAt: n.CreatedAt,
	}
	if n.SentAt != nil {
		m.SentAt = sql.NullTime{Time: *n.SentAt, Valid: true}
	}
	return m
}

// ──────────────────────────────────────────────────
// API Key model
// ──────────────────────────────────────────────────

type APIKeyModel struct {
	grove.BaseModel `grove:"table:authsome_api_keys,alias:ak"`

	ID              string       `grove:"id,pk"`
	AppID           string       `grove:"app_id,notnull"`
	EnvID           string       `grove:"env_id,notnull"`
	UserID          string       `grove:"user_id,notnull"`
	Name            string       `grove:"name,notnull"`
	KeyHash         string       `grove:"key_hash,notnull"`
	KeyPrefix       string       `grove:"key_prefix,notnull"`
	PublicKey       string       `grove:"public_key,notnull"`
	PublicKeyPrefix string       `grove:"public_key_prefix,notnull"`
	Scopes          string       `grove:"scopes"` // comma-separated
	ExpiresAt       sql.NullTime `grove:"expires_at"`
	LastUsedAt      sql.NullTime `grove:"last_used_at"`
	Revoked         bool         `grove:"revoked"`
	CreatedAt       time.Time    `grove:"created_at,notnull,default:now()"`
	UpdatedAt       time.Time    `grove:"updated_at,notnull,default:now()"`
}

func toAPIKey(m *APIKeyModel) (*apikey.APIKey, error) {
	keyID, err := id.ParseAPIKeyID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	k := &apikey.APIKey{
		ID:              keyID,
		AppID:           appID,
		EnvID:           envID,
		UserID:          userID,
		Name:            m.Name,
		KeyHash:         m.KeyHash,
		KeyPrefix:       m.KeyPrefix,
		PublicKey:       m.PublicKey,
		PublicKeyPrefix: m.PublicKeyPrefix,
		Revoked:         m.Revoked,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.Scopes != "" {
		k.Scopes = strings.Split(m.Scopes, ",")
	}
	if m.ExpiresAt.Valid {
		k.ExpiresAt = &m.ExpiresAt.Time
	}
	if m.LastUsedAt.Valid {
		k.LastUsedAt = &m.LastUsedAt.Time
	}
	return k, nil
}

func fromAPIKey(k *apikey.APIKey) *APIKeyModel {
	m := &APIKeyModel{
		ID:              k.ID.String(),
		AppID:           k.AppID.String(),
		EnvID:           k.EnvID.String(),
		UserID:          k.UserID.String(),
		Name:            k.Name,
		KeyHash:         k.KeyHash,
		KeyPrefix:       k.KeyPrefix,
		PublicKey:       k.PublicKey,
		PublicKeyPrefix: k.PublicKeyPrefix,
		Revoked:         k.Revoked,
		CreatedAt:       k.CreatedAt,
		UpdatedAt:       k.UpdatedAt,
	}
	if len(k.Scopes) > 0 {
		m.Scopes = strings.Join(k.Scopes, ",")
	}
	if k.ExpiresAt != nil {
		m.ExpiresAt = sql.NullTime{Time: *k.ExpiresAt, Valid: true}
	}
	if k.LastUsedAt != nil {
		m.LastUsedAt = sql.NullTime{Time: *k.LastUsedAt, Valid: true}
	}
	return m
}

// ──────────────────────────────────────────────────
// Environment model
// ──────────────────────────────────────────────────

type EnvironmentModel struct {
	grove.BaseModel `grove:"table:authsome_environments,alias:env"`

	ID          string          `grove:"id,pk"`
	AppID       string          `grove:"app_id,notnull"`
	Name        string          `grove:"name,notnull"`
	Slug        string          `grove:"slug,notnull"`
	Type        string          `grove:"type,notnull"`
	IsDefault   bool            `grove:"is_default"`
	Color       string          `grove:"color"`
	Description string          `grove:"description"`
	Settings    json.RawMessage `grove:"settings,type:jsonb"`
	ClonedFrom  string          `grove:"cloned_from"`
	Metadata    json.RawMessage `grove:"metadata,type:jsonb"`
	CreatedAt   time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time       `grove:"updated_at,notnull,default:now()"`
}

func toEnvironment(m *EnvironmentModel) (*environment.Environment, error) {
	envID, err := id.ParseEnvironmentID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	var settings *environment.Settings
	if len(m.Settings) > 0 {
		settings = new(environment.Settings)
		_ = json.Unmarshal(m.Settings, settings) //nolint:errcheck // best-effort decode
	}
	md := make(environment.Metadata)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}
	e := &environment.Environment{
		ID:          envID,
		AppID:       appID,
		Name:        m.Name,
		Slug:        m.Slug,
		Type:        environment.Type(m.Type),
		IsDefault:   m.IsDefault,
		Color:       m.Color,
		Description: m.Description,
		Settings:    settings,
		Metadata:    md,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.ClonedFrom != "" {
		clonedFrom, err := id.ParseEnvironmentID(m.ClonedFrom)
		if err != nil {
			return nil, err
		}
		e.ClonedFrom = clonedFrom
	}
	return e, nil
}

func fromEnvironment(e *environment.Environment) *EnvironmentModel {
	settings, _ := json.Marshal(e.Settings) //nolint:errcheck // best-effort encode
	md, _ := json.Marshal(e.Metadata)       //nolint:errcheck // best-effort encode
	m := &EnvironmentModel{
		ID:          e.ID.String(),
		AppID:       e.AppID.String(),
		Name:        e.Name,
		Slug:        e.Slug,
		Type:        string(e.Type),
		IsDefault:   e.IsDefault,
		Color:       e.Color,
		Description: e.Description,
		Settings:    settings,
		Metadata:    md,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if !e.ClonedFrom.IsNil() {
		m.ClonedFrom = e.ClonedFrom.String()
	}
	return m
}

// ──────────────────────────────────────────────────
// FormConfig model
// ──────────────────────────────────────────────────

type FormConfigModel struct {
	grove.BaseModel `grove:"table:authsome_form_configs,alias:fc"`

	ID        string          `grove:"id,pk"`
	AppID     string          `grove:"app_id,notnull"`
	FormType  string          `grove:"form_type,notnull"`
	Fields    json.RawMessage `grove:"fields"`
	Active    bool            `grove:"active"`
	Version   int             `grove:"version,notnull"`
	CreatedAt time.Time       `grove:"created_at,notnull"`
	UpdatedAt time.Time       `grove:"updated_at,notnull"`
}

func toFormConfig(m *FormConfigModel) (*formconfig.FormConfig, error) {
	fcID, err := id.ParseFormConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	var fields []formconfig.FormField
	if len(m.Fields) > 0 {
		_ = json.Unmarshal(m.Fields, &fields) //nolint:errcheck // best-effort decode
	}
	return &formconfig.FormConfig{
		ID:        fcID,
		AppID:     appID,
		FormType:  m.FormType,
		Fields:    fields,
		Active:    m.Active,
		Version:   m.Version,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func fromFormConfig(fc *formconfig.FormConfig) *FormConfigModel {
	fields, _ := json.Marshal(fc.Fields) //nolint:errcheck // best-effort encode
	return &FormConfigModel{
		ID:        fc.ID.String(),
		AppID:     fc.AppID.String(),
		FormType:  fc.FormType,
		Fields:    fields,
		Active:    fc.Active,
		Version:   fc.Version,
		CreatedAt: fc.CreatedAt,
		UpdatedAt: fc.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// BrandingConfig model
// ──────────────────────────────────────────────────

type BrandingConfigModel struct {
	grove.BaseModel `grove:"table:authsome_branding_configs,alias:bc"`

	ID              string    `grove:"id,pk"`
	OrgID           string    `grove:"org_id,notnull"`
	AppID           string    `grove:"app_id,notnull"`
	LogoURL         string    `grove:"logo_url"`
	PrimaryColor    string    `grove:"primary_color"`
	BackgroundColor string    `grove:"background_color"`
	AccentColor     string    `grove:"accent_color"`
	FontFamily      string    `grove:"font_family"`
	CustomCSS       string    `grove:"custom_css"`
	CompanyName     string    `grove:"company_name"`
	Tagline         string    `grove:"tagline"`
	CreatedAt       time.Time `grove:"created_at,notnull"`
	UpdatedAt       time.Time `grove:"updated_at,notnull"`
}

func toBrandingConfig(m *BrandingConfigModel) (*formconfig.BrandingConfig, error) {
	bcID, err := id.ParseBrandingConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	orgID, err := id.ParseOrgID(m.OrgID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	return &formconfig.BrandingConfig{
		ID:              bcID,
		OrgID:           orgID,
		AppID:           appID,
		LogoURL:         m.LogoURL,
		PrimaryColor:    m.PrimaryColor,
		BackgroundColor: m.BackgroundColor,
		AccentColor:     m.AccentColor,
		FontFamily:      m.FontFamily,
		CustomCSS:       m.CustomCSS,
		CompanyName:     m.CompanyName,
		Tagline:         m.Tagline,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

func fromBrandingConfig(b *formconfig.BrandingConfig) *BrandingConfigModel {
	return &BrandingConfigModel{
		ID:              b.ID.String(),
		OrgID:           b.OrgID.String(),
		AppID:           b.AppID.String(),
		LogoURL:         b.LogoURL,
		PrimaryColor:    b.PrimaryColor,
		BackgroundColor: b.BackgroundColor,
		AccentColor:     b.AccentColor,
		FontFamily:      b.FontFamily,
		CustomCSS:       b.CustomCSS,
		CompanyName:     b.CompanyName,
		Tagline:         b.Tagline,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// AppSessionConfig model
// ──────────────────────────────────────────────────

type AppSessionConfigModel struct {
	grove.BaseModel `grove:"table:authsome_app_session_configs,alias:asc"`

	ID                     string        `grove:"id,pk"`
	AppID                  string        `grove:"app_id,notnull"`
	TokenTTLSeconds        sql.NullInt64 `grove:"token_ttl_seconds"`
	RefreshTokenTTLSeconds sql.NullInt64 `grove:"refresh_token_ttl_seconds"`
	MaxActiveSessions      sql.NullInt64 `grove:"max_active_sessions"`
	RotateRefreshToken     sql.NullBool  `grove:"rotate_refresh_token"`
	BindToIP               sql.NullBool  `grove:"bind_to_ip"`
	BindToDevice           sql.NullBool  `grove:"bind_to_device"`
	TokenFormat            string        `grove:"token_format"`
	CreatedAt              time.Time     `grove:"created_at,notnull,default:now()"`
	UpdatedAt              time.Time     `grove:"updated_at,notnull,default:now()"`
}

func toAppSessionConfig(m *AppSessionConfigModel) (*appsessionconfig.Config, error) {
	cfgID, err := id.ParseAppSessionConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	cfg := &appsessionconfig.Config{
		ID:          cfgID,
		AppID:       appID,
		TokenFormat: m.TokenFormat,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.TokenTTLSeconds.Valid {
		v := int(m.TokenTTLSeconds.Int64)
		cfg.TokenTTLSeconds = &v
	}
	if m.RefreshTokenTTLSeconds.Valid {
		v := int(m.RefreshTokenTTLSeconds.Int64)
		cfg.RefreshTokenTTLSeconds = &v
	}
	if m.MaxActiveSessions.Valid {
		v := int(m.MaxActiveSessions.Int64)
		cfg.MaxActiveSessions = &v
	}
	if m.RotateRefreshToken.Valid {
		cfg.RotateRefreshToken = &m.RotateRefreshToken.Bool
	}
	if m.BindToIP.Valid {
		cfg.BindToIP = &m.BindToIP.Bool
	}
	if m.BindToDevice.Valid {
		cfg.BindToDevice = &m.BindToDevice.Bool
	}
	return cfg, nil
}

func fromAppSessionConfig(cfg *appsessionconfig.Config) *AppSessionConfigModel {
	m := &AppSessionConfigModel{
		ID:          cfg.ID.String(),
		AppID:       cfg.AppID.String(),
		TokenFormat: cfg.TokenFormat,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt,
	}
	if cfg.TokenTTLSeconds != nil {
		m.TokenTTLSeconds = sql.NullInt64{Int64: int64(*cfg.TokenTTLSeconds), Valid: true}
	}
	if cfg.RefreshTokenTTLSeconds != nil {
		m.RefreshTokenTTLSeconds = sql.NullInt64{Int64: int64(*cfg.RefreshTokenTTLSeconds), Valid: true}
	}
	if cfg.MaxActiveSessions != nil {
		m.MaxActiveSessions = sql.NullInt64{Int64: int64(*cfg.MaxActiveSessions), Valid: true}
	}
	if cfg.RotateRefreshToken != nil {
		m.RotateRefreshToken = sql.NullBool{Bool: *cfg.RotateRefreshToken, Valid: true}
	}
	if cfg.BindToIP != nil {
		m.BindToIP = sql.NullBool{Bool: *cfg.BindToIP, Valid: true}
	}
	if cfg.BindToDevice != nil {
		m.BindToDevice = sql.NullBool{Bool: *cfg.BindToDevice, Valid: true}
	}
	return m
}
