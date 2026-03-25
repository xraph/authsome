package mongo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/xraph/grove"

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
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// ──────────────────────────────────────────────────
// App model
// ──────────────────────────────────────────────────

type appModel struct {
	grove.BaseModel `grove:"table:authsome_apps"`

	ID             string          `grove:"id,pk"             bson:"_id"`
	Name           string          `grove:"name"              bson:"name"`
	Slug           string          `grove:"slug"              bson:"slug"`
	Logo           string          `grove:"logo"              bson:"logo"`
	PublishableKey string          `grove:"publishable_key"   bson:"publishable_key,omitempty"`
	IsPlatform     bool            `grove:"is_platform"       bson:"is_platform"`
	Metadata       json.RawMessage `grove:"metadata"          bson:"metadata,omitempty"`
	CreatedAt      time.Time       `grove:"created_at"        bson:"created_at"`
	UpdatedAt      time.Time       `grove:"updated_at"        bson:"updated_at"`
}

func toAppModel(a *app.App) *appModel {
	md, _ := json.Marshal(a.Metadata) //nolint:errcheck // best-effort encode
	return &appModel{
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

func fromAppModel(m *appModel) (*app.App, error) {
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

// ──────────────────────────────────────────────────
// User model
// ──────────────────────────────────────────────────

type userModel struct {
	grove.BaseModel `grove:"table:authsome_users"`

	ID              string          `grove:"id,pk"             bson:"_id"`
	AppID           string          `grove:"app_id"            bson:"app_id"`
	EnvID           string          `grove:"env_id"            bson:"env_id"`
	Email           string          `grove:"email"             bson:"email"`
	EmailVerified   bool            `grove:"email_verified"    bson:"email_verified"`
	FirstName       string          `grove:"first_name"        bson:"first_name"`
	LastName        string          `grove:"last_name"         bson:"last_name"`
	Image           string          `grove:"image"             bson:"image"`
	Username        string          `grove:"username"          bson:"username"`
	DisplayUsername string          `grove:"display_username"  bson:"display_username"`
	Phone           string          `grove:"phone"             bson:"phone"`
	PhoneVerified   bool            `grove:"phone_verified"    bson:"phone_verified"`
	PasswordHash    string          `grove:"password_hash"     bson:"password_hash"`
	Banned          bool            `grove:"banned"            bson:"banned"`
	BanReason       string          `grove:"ban_reason"        bson:"ban_reason"`
	BanExpires      *time.Time      `grove:"ban_expires"       bson:"ban_expires,omitempty"`
	Metadata        json.RawMessage `grove:"metadata"          bson:"metadata,omitempty"`
	CreatedAt       time.Time       `grove:"created_at"        bson:"created_at"`
	UpdatedAt       time.Time       `grove:"updated_at"        bson:"updated_at"`
	DeletedAt       *time.Time      `grove:"deleted_at"        bson:"deleted_at,omitempty"`
}

func toUserModel(u *user.User) *userModel {
	md, _ := json.Marshal(u.Metadata) //nolint:errcheck // best-effort encode
	return &userModel{
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
		BanExpires:      u.BanExpires,
		Metadata:        md,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
		DeletedAt:       u.DeletedAt,
	}
}

func fromUserModel(m *userModel) (*user.User, error) {
	userID, err := id.ParseUserID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse //nolint:errcheck // best-effort parse
	md := make(user.Metadata)
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &md) //nolint:errcheck // best-effort decode
	}
	return &user.User{
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
		BanExpires:      m.BanExpires,
		Metadata:        md,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		DeletedAt:       m.DeletedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Session model
// ──────────────────────────────────────────────────

type sessionModel struct {
	grove.BaseModel `grove:"table:authsome_sessions"`

	ID                    string    `grove:"id,pk"                     bson:"_id"`
	AppID                 string    `grove:"app_id"                    bson:"app_id"`
	EnvID                 string    `grove:"env_id"                    bson:"env_id"`
	UserID                string    `grove:"user_id"                   bson:"user_id"`
	OrgID                 string    `grove:"org_id"                    bson:"org_id,omitempty"`
	Token                 string    `grove:"token"                     bson:"token"`
	RefreshToken          string    `grove:"refresh_token"             bson:"refresh_token"`
	IPAddress             string    `grove:"ip_address"                bson:"ip_address"`
	UserAgent             string    `grove:"user_agent"                bson:"user_agent"`
	DeviceID              string    `grove:"device_id"                 bson:"device_id,omitempty"`
	ImpersonatedBy        string    `grove:"impersonated_by"           bson:"impersonated_by,omitempty"`
	LastActivityAt        time.Time `grove:"last_activity_at"          bson:"last_activity_at,omitempty"`
	ExpiresAt             time.Time `grove:"expires_at"                bson:"expires_at"`
	RefreshTokenExpiresAt time.Time `grove:"refresh_token_expires_at"  bson:"refresh_token_expires_at"`
	CreatedAt             time.Time `grove:"created_at"                bson:"created_at"`
	UpdatedAt             time.Time `grove:"updated_at"                bson:"updated_at"`
}

func toSessionModel(s *session.Session) *sessionModel {
	m := &sessionModel{
		ID:                    s.ID.String(),
		AppID:                 s.AppID.String(),
		EnvID:                 s.EnvID.String(),
		UserID:                s.UserID.String(),
		Token:                 s.Token,
		RefreshToken:          s.RefreshToken,
		IPAddress:             s.IPAddress,
		UserAgent:             s.UserAgent,
		LastActivityAt:        s.LastActivityAt,
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

func fromSessionModel(m *sessionModel) (*session.Session, error) {
	sessID, err := id.ParseSessionID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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
		LastActivityAt:        m.LastActivityAt,
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

// ──────────────────────────────────────────────────
// Verification model
// ──────────────────────────────────────────────────

type verificationModel struct {
	grove.BaseModel `grove:"table:authsome_verifications"`

	ID        string    `grove:"id,pk"       bson:"_id"`
	AppID     string    `grove:"app_id"      bson:"app_id"`
	EnvID     string    `grove:"env_id"      bson:"env_id"`
	UserID    string    `grove:"user_id"     bson:"user_id"`
	Token     string    `grove:"token"       bson:"token"`
	Type      string    `grove:"type"        bson:"type"`
	ExpiresAt time.Time `grove:"expires_at"  bson:"expires_at"`
	Consumed  bool      `grove:"consumed"    bson:"consumed"`
	CreatedAt time.Time `grove:"created_at"  bson:"created_at"`
}

func toVerificationModel(v *account.Verification) *verificationModel {
	return &verificationModel{
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

func fromVerificationModel(m *verificationModel) (*account.Verification, error) {
	vID, err := id.ParseVerificationID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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

// ──────────────────────────────────────────────────
// Password reset model
// ──────────────────────────────────────────────────

type passwordResetModel struct {
	grove.BaseModel `grove:"table:authsome_password_resets"`

	ID        string    `grove:"id,pk"       bson:"_id"`
	AppID     string    `grove:"app_id"      bson:"app_id"`
	EnvID     string    `grove:"env_id"      bson:"env_id"`
	UserID    string    `grove:"user_id"     bson:"user_id"`
	Token     string    `grove:"token"       bson:"token"`
	ExpiresAt time.Time `grove:"expires_at"  bson:"expires_at"`
	Consumed  bool      `grove:"consumed"    bson:"consumed"`
	CreatedAt time.Time `grove:"created_at"  bson:"created_at"`
}

func toPasswordResetModel(pr *account.PasswordReset) *passwordResetModel {
	return &passwordResetModel{
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

func fromPasswordResetModel(m *passwordResetModel) (*account.PasswordReset, error) {
	prID, err := id.ParsePasswordResetID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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

// ──────────────────────────────────────────────────
// Organization model
// ──────────────────────────────────────────────────

type organizationModel struct {
	grove.BaseModel `grove:"table:authsome_organizations"`

	ID        string          `grove:"id,pk"        bson:"_id"`
	AppID     string          `grove:"app_id"       bson:"app_id"`
	EnvID     string          `grove:"env_id"       bson:"env_id"`
	Name      string          `grove:"name"         bson:"name"`
	Slug      string          `grove:"slug"         bson:"slug"`
	Logo      string          `grove:"logo"         bson:"logo"`
	Metadata  json.RawMessage `grove:"metadata"     bson:"metadata,omitempty"`
	CreatedBy string          `grove:"created_by"   bson:"created_by"`
	CreatedAt time.Time       `grove:"created_at"   bson:"created_at"`
	UpdatedAt time.Time       `grove:"updated_at"   bson:"updated_at"`
}

func toOrganizationModel(o *organization.Organization) *organizationModel {
	md, _ := json.Marshal(o.Metadata) //nolint:errcheck // best-effort encode
	return &organizationModel{
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

func fromOrganizationModel(m *organizationModel) (*organization.Organization, error) {
	orgID, err := id.ParseOrgID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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

// ──────────────────────────────────────────────────
// Member model
// ──────────────────────────────────────────────────

type memberModel struct {
	grove.BaseModel `grove:"table:authsome_members"`

	ID        string    `grove:"id,pk"       bson:"_id"`
	OrgID     string    `grove:"org_id"      bson:"org_id"`
	UserID    string    `grove:"user_id"     bson:"user_id"`
	Role      string    `grove:"role"        bson:"role"`
	CreatedAt time.Time `grove:"created_at"  bson:"created_at"`
	UpdatedAt time.Time `grove:"updated_at"  bson:"updated_at"`
}

func toMemberModel(mem *organization.Member) *memberModel {
	return &memberModel{
		ID:        mem.ID.String(),
		OrgID:     mem.OrgID.String(),
		UserID:    mem.UserID.String(),
		Role:      string(mem.Role),
		CreatedAt: mem.CreatedAt,
		UpdatedAt: mem.UpdatedAt,
	}
}

func fromMemberModel(m *memberModel) (*organization.Member, error) {
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

// ──────────────────────────────────────────────────
// Invitation model
// ──────────────────────────────────────────────────

type invitationModel struct {
	grove.BaseModel `grove:"table:authsome_invitations"`

	ID        string    `grove:"id,pk"        bson:"_id"`
	OrgID     string    `grove:"org_id"       bson:"org_id"`
	Email     string    `grove:"email"        bson:"email"`
	Role      string    `grove:"role"         bson:"role"`
	InviterID string    `grove:"inviter_id"   bson:"inviter_id"`
	Status    string    `grove:"status"       bson:"status"`
	Token     string    `grove:"token"        bson:"token"`
	ExpiresAt time.Time `grove:"expires_at"   bson:"expires_at"`
	CreatedAt time.Time `grove:"created_at"   bson:"created_at"`
}

func toInvitationModel(inv *organization.Invitation) *invitationModel {
	return &invitationModel{
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

func fromInvitationModel(m *invitationModel) (*organization.Invitation, error) {
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

// ──────────────────────────────────────────────────
// Team model
// ──────────────────────────────────────────────────

type teamModel struct {
	grove.BaseModel `grove:"table:authsome_teams"`

	ID        string    `grove:"id,pk"       bson:"_id"`
	OrgID     string    `grove:"org_id"      bson:"org_id"`
	Name      string    `grove:"name"        bson:"name"`
	Slug      string    `grove:"slug"        bson:"slug"`
	CreatedAt time.Time `grove:"created_at"  bson:"created_at"`
	UpdatedAt time.Time `grove:"updated_at"  bson:"updated_at"`
}

func toTeamModel(t *organization.Team) *teamModel {
	return &teamModel{
		ID:        t.ID.String(),
		OrgID:     t.OrgID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func fromTeamModel(m *teamModel) (*organization.Team, error) {
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

// ──────────────────────────────────────────────────
// Device model
// ──────────────────────────────────────────────────

type deviceModel struct {
	grove.BaseModel `grove:"table:authsome_devices"`

	ID          string    `grove:"id,pk"          bson:"_id"`
	UserID      string    `grove:"user_id"        bson:"user_id"`
	AppID       string    `grove:"app_id"         bson:"app_id"`
	EnvID       string    `grove:"env_id"         bson:"env_id"`
	Name        string    `grove:"name"           bson:"name"`
	Type        string    `grove:"type"           bson:"type"`
	Browser     string    `grove:"browser"        bson:"browser"`
	OS          string    `grove:"os"             bson:"os"`
	IPAddress   string    `grove:"ip_address"     bson:"ip_address"`
	Fingerprint string    `grove:"fingerprint"    bson:"fingerprint"`
	Trusted     bool      `grove:"trusted"        bson:"trusted"`
	LastSeenAt  time.Time `grove:"last_seen_at"   bson:"last_seen_at"`
	CreatedAt   time.Time `grove:"created_at"     bson:"created_at"`
	UpdatedAt   time.Time `grove:"updated_at"     bson:"updated_at"`
}

func toDeviceModel(d *device.Device) *deviceModel {
	return &deviceModel{
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

func fromDeviceModel(m *deviceModel) (*device.Device, error) {
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
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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

// ──────────────────────────────────────────────────
// Webhook model
// ──────────────────────────────────────────────────

type webhookModel struct {
	grove.BaseModel `grove:"table:authsome_webhooks"`

	ID        string    `grove:"id,pk"       bson:"_id"`
	AppID     string    `grove:"app_id"      bson:"app_id"`
	EnvID     string    `grove:"env_id"      bson:"env_id"`
	URL       string    `grove:"url"         bson:"url"`
	Events    []string  `grove:"events"      bson:"events,omitempty"`
	Secret    string    `grove:"secret"      bson:"secret"`
	Active    bool      `grove:"active"      bson:"active"`
	CreatedAt time.Time `grove:"created_at" bson:"created_at"`
	UpdatedAt time.Time `grove:"updated_at" bson:"updated_at"`
}

func toWebhookModel(w *webhook.Webhook) *webhookModel {
	return &webhookModel{
		ID:        w.ID.String(),
		AppID:     w.AppID.String(),
		EnvID:     w.EnvID.String(),
		URL:       w.URL,
		Events:    w.Events,
		Secret:    w.Secret,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func fromWebhookModel(m *webhookModel) (*webhook.Webhook, error) {
	whID, err := id.ParseWebhookID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
	return &webhook.Webhook{
		ID:        whID,
		AppID:     appID,
		EnvID:     envID,
		URL:       m.URL,
		Events:    m.Events,
		Secret:    m.Secret,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Notification model
// ──────────────────────────────────────────────────

type notificationModel struct {
	grove.BaseModel `grove:"table:authsome_notifications"`

	ID        string     `grove:"id,pk"       bson:"_id"`
	AppID     string     `grove:"app_id"      bson:"app_id"`
	EnvID     string     `grove:"env_id"      bson:"env_id"`
	UserID    string     `grove:"user_id"     bson:"user_id"`
	Type      string     `grove:"type"        bson:"type"`
	Channel   string     `grove:"channel"     bson:"channel"`
	Subject   string     `grove:"subject"     bson:"subject"`
	Body      string     `grove:"body"        bson:"body"`
	Sent      bool       `grove:"sent"        bson:"sent"`
	SentAt    *time.Time `grove:"sent_at"     bson:"sent_at,omitempty"`
	CreatedAt time.Time  `grove:"created_at"  bson:"created_at"`
}

func toNotificationModel(n *notification.Notification) *notificationModel {
	return &notificationModel{
		ID:        n.ID.String(),
		AppID:     n.AppID.String(),
		EnvID:     n.EnvID.String(),
		UserID:    n.UserID.String(),
		Type:      n.Type,
		Channel:   string(n.Channel),
		Subject:   n.Subject,
		Body:      n.Body,
		Sent:      n.Sent,
		SentAt:    n.SentAt,
		CreatedAt: n.CreatedAt,
	}
}

func fromNotificationModel(m *notificationModel) (*notification.Notification, error) {
	nID, err := id.ParseNotificationID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return &notification.Notification{
		ID:        nID,
		AppID:     appID,
		EnvID:     envID,
		UserID:    userID,
		Type:      m.Type,
		Channel:   notification.Channel(m.Channel),
		Subject:   m.Subject,
		Body:      m.Body,
		Sent:      m.Sent,
		SentAt:    m.SentAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// API Key model
// ──────────────────────────────────────────────────

type apiKeyModel struct {
	grove.BaseModel `grove:"table:authsome_api_keys"`

	ID              string     `grove:"id,pk"          bson:"_id"`
	AppID           string     `grove:"app_id"         bson:"app_id"`
	EnvID           string     `grove:"env_id"         bson:"env_id"`
	UserID          string     `grove:"user_id"        bson:"user_id"`
	Name            string     `grove:"name"           bson:"name"`
	KeyHash         string     `grove:"key_hash"            bson:"key_hash"`
	KeyPrefix       string     `grove:"key_prefix"          bson:"key_prefix"`
	PublicKey       string     `grove:"public_key"          bson:"public_key"`
	PublicKeyPrefix string     `grove:"public_key_prefix"   bson:"public_key_prefix"`
	Scopes          string     `grove:"scopes"              bson:"scopes"`
	ExpiresAt       *time.Time `grove:"expires_at"     bson:"expires_at,omitempty"`
	LastUsedAt      *time.Time `grove:"last_used_at"   bson:"last_used_at,omitempty"`
	Revoked         bool       `grove:"revoked"        bson:"revoked"`
	CreatedAt       time.Time  `grove:"created_at"     bson:"created_at"`
	UpdatedAt       time.Time  `grove:"updated_at"     bson:"updated_at"`
}

func toAPIKeyModel(k *apikey.APIKey) *apiKeyModel {
	m := &apiKeyModel{
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
		ExpiresAt:       k.ExpiresAt,
		LastUsedAt:      k.LastUsedAt,
		CreatedAt:       k.CreatedAt,
		UpdatedAt:       k.UpdatedAt,
	}
	if len(k.Scopes) > 0 {
		m.Scopes = strings.Join(k.Scopes, ",")
	}
	return m
}

func fromAPIKeyModel(m *apiKeyModel) (*apikey.APIKey, error) {
	keyID, err := id.ParseAPIKeyID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, _ := id.ParseEnvironmentID(m.EnvID) //nolint:errcheck // best-effort parse
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
		ExpiresAt:       m.ExpiresAt,
		LastUsedAt:      m.LastUsedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.Scopes != "" {
		k.Scopes = strings.Split(m.Scopes, ",")
	}
	return k, nil
}

// ──────────────────────────────────────────────────
// Environment model
// ──────────────────────────────────────────────────

type environmentModel struct {
	grove.BaseModel `grove:"table:authsome_environments"`

	ID          string          `grove:"id,pk"         bson:"_id"`
	AppID       string          `grove:"app_id"        bson:"app_id"`
	Name        string          `grove:"name"          bson:"name"`
	Slug        string          `grove:"slug"          bson:"slug"`
	Type        string          `grove:"type"          bson:"type"`
	IsDefault   bool            `grove:"is_default"    bson:"is_default"`
	Color       string          `grove:"color"         bson:"color,omitempty"`
	Description string          `grove:"description"   bson:"description,omitempty"`
	Settings    json.RawMessage `grove:"settings"      bson:"settings,omitempty"`
	ClonedFrom  string          `grove:"cloned_from"   bson:"cloned_from,omitempty"`
	Metadata    json.RawMessage `grove:"metadata"      bson:"metadata,omitempty"`
	CreatedAt   time.Time       `grove:"created_at"    bson:"created_at"`
	UpdatedAt   time.Time       `grove:"updated_at"    bson:"updated_at"`
}

func toEnvironmentModel(e *environment.Environment) *environmentModel {
	envSettings, _ := json.Marshal(e.Settings) //nolint:errcheck // best-effort encode
	md, _ := json.Marshal(e.Metadata)          //nolint:errcheck // best-effort encode
	m := &environmentModel{
		ID:          e.ID.String(),
		AppID:       e.AppID.String(),
		Name:        e.Name,
		Slug:        e.Slug,
		Type:        string(e.Type),
		IsDefault:   e.IsDefault,
		Color:       e.Color,
		Description: e.Description,
		Settings:    envSettings,
		Metadata:    md,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if !e.ClonedFrom.IsNil() {
		m.ClonedFrom = e.ClonedFrom.String()
	}
	return m
}

func fromEnvironmentModel(m *environmentModel) (*environment.Environment, error) {
	envID, err := id.ParseEnvironmentID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	var envSettings *environment.Settings
	if len(m.Settings) > 0 {
		envSettings = new(environment.Settings)
		_ = json.Unmarshal(m.Settings, envSettings) //nolint:errcheck // best-effort decode
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
		Settings:    envSettings,
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

// ──────────────────────────────────────────────────
// FormConfig model
// ──────────────────────────────────────────────────

type formConfigModel struct {
	grove.BaseModel `grove:"table:authsome_form_configs"`

	ID        string          `grove:"id,pk"        bson:"_id"`
	AppID     string          `grove:"app_id"       bson:"app_id"`
	FormType  string          `grove:"form_type"    bson:"form_type"`
	Fields    json.RawMessage `grove:"fields"       bson:"fields,omitempty"`
	Active    bool            `grove:"active"       bson:"active"`
	Version   int             `grove:"version"      bson:"version"`
	CreatedAt time.Time       `grove:"created_at"   bson:"created_at"`
	UpdatedAt time.Time       `grove:"updated_at"   bson:"updated_at"`
}

func toFormConfigModel(fc *formconfig.FormConfig) *formConfigModel {
	fields, _ := json.Marshal(fc.Fields) //nolint:errcheck // best-effort encode
	return &formConfigModel{
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

func fromFormConfigModel(m *formConfigModel) (*formconfig.FormConfig, error) {
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

// ──────────────────────────────────────────────────
// BrandingConfig model
// ──────────────────────────────────────────────────

type brandingConfigModel struct {
	grove.BaseModel `grove:"table:authsome_branding_configs"`

	ID              string    `grove:"id,pk"              bson:"_id"`
	OrgID           string    `grove:"org_id"             bson:"org_id"`
	AppID           string    `grove:"app_id"             bson:"app_id"`
	LogoURL         string    `grove:"logo_url"           bson:"logo_url,omitempty"`
	PrimaryColor    string    `grove:"primary_color"      bson:"primary_color,omitempty"`
	BackgroundColor string    `grove:"background_color"   bson:"background_color,omitempty"`
	AccentColor     string    `grove:"accent_color"       bson:"accent_color,omitempty"`
	FontFamily      string    `grove:"font_family"        bson:"font_family,omitempty"`
	CustomCSS       string    `grove:"custom_css"         bson:"custom_css,omitempty"`
	CompanyName     string    `grove:"company_name"       bson:"company_name,omitempty"`
	Tagline         string    `grove:"tagline"            bson:"tagline,omitempty"`
	CreatedAt       time.Time `grove:"created_at"         bson:"created_at"`
	UpdatedAt       time.Time `grove:"updated_at"         bson:"updated_at"`
}

func toBrandingConfigModel(b *formconfig.BrandingConfig) *brandingConfigModel {
	return &brandingConfigModel{
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

func fromBrandingConfigModel(m *brandingConfigModel) (*formconfig.BrandingConfig, error) {
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

// ──────────────────────────────────────────────────
// AppSessionConfig model
// ──────────────────────────────────────────────────

type appSessionConfigModel struct {
	grove.BaseModel `grove:"table:authsome_app_session_configs"`

	ID                     string    `grove:"id,pk"                      bson:"_id"`
	AppID                  string    `grove:"app_id"                     bson:"app_id"`
	TokenTTLSeconds        *int      `grove:"token_ttl_seconds"          bson:"token_ttl_seconds,omitempty"`
	RefreshTokenTTLSeconds *int      `grove:"refresh_token_ttl_seconds"  bson:"refresh_token_ttl_seconds,omitempty"`
	MaxActiveSessions      *int      `grove:"max_active_sessions"        bson:"max_active_sessions,omitempty"`
	RotateRefreshToken     *bool     `grove:"rotate_refresh_token"       bson:"rotate_refresh_token,omitempty"`
	BindToIP               *bool     `grove:"bind_to_ip"                 bson:"bind_to_ip,omitempty"`
	BindToDevice           *bool     `grove:"bind_to_device"             bson:"bind_to_device,omitempty"`
	TokenFormat            string    `grove:"token_format"               bson:"token_format,omitempty"`
	CreatedAt              time.Time `grove:"created_at"                 bson:"created_at"`
	UpdatedAt              time.Time `grove:"updated_at"                 bson:"updated_at"`
}

func toAppSessionConfigModel(c *appsessionconfig.Config) *appSessionConfigModel {
	return &appSessionConfigModel{
		ID:                     c.ID.String(),
		AppID:                  c.AppID.String(),
		TokenTTLSeconds:        c.TokenTTLSeconds,
		RefreshTokenTTLSeconds: c.RefreshTokenTTLSeconds,
		MaxActiveSessions:      c.MaxActiveSessions,
		RotateRefreshToken:     c.RotateRefreshToken,
		BindToIP:               c.BindToIP,
		BindToDevice:           c.BindToDevice,
		TokenFormat:            c.TokenFormat,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
	}
}

func fromAppSessionConfigModel(m *appSessionConfigModel) (*appsessionconfig.Config, error) {
	cfgID, err := id.ParseAppSessionConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	return &appsessionconfig.Config{
		ID:                     cfgID,
		AppID:                  appID,
		TokenTTLSeconds:        m.TokenTTLSeconds,
		RefreshTokenTTLSeconds: m.RefreshTokenTTLSeconds,
		MaxActiveSessions:      m.MaxActiveSessions,
		RotateRefreshToken:     m.RotateRefreshToken,
		BindToIP:               m.BindToIP,
		BindToDevice:           m.BindToDevice,
		TokenFormat:            m.TokenFormat,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// AppClientConfig model
// ──────────────────────────────────────────────────

type appClientConfigModel struct {
	grove.BaseModel `grove:"table:authsome_app_client_configs"`

	ID               string          `grove:"id,pk"               bson:"_id"`
	AppID            string          `grove:"app_id"              bson:"app_id"`
	PasswordEnabled  *bool           `grove:"password_enabled"    bson:"password_enabled,omitempty"`
	PasskeyEnabled   *bool           `grove:"passkey_enabled"     bson:"passkey_enabled,omitempty"`
	MagicLinkEnabled *bool           `grove:"magic_link_enabled"  bson:"magic_link_enabled,omitempty"`
	MFAEnabled       *bool           `grove:"mfa_enabled"         bson:"mfa_enabled,omitempty"`
	SSOEnabled       *bool           `grove:"sso_enabled"         bson:"sso_enabled,omitempty"`
	SocialEnabled    *bool           `grove:"social_enabled"      bson:"social_enabled,omitempty"`
	WaitlistEnabled          *bool           `grove:"waitlist_enabled"              bson:"waitlist_enabled,omitempty"`
	RequireEmailVerification *bool           `grove:"require_email_verification"    bson:"require_email_verification,omitempty"`
	SocialProviders          json.RawMessage `grove:"social_providers"              bson:"social_providers,omitempty"`
	MFAMethods       json.RawMessage `grove:"mfa_methods"         bson:"mfa_methods,omitempty"`
	AppName          *string         `grove:"app_name"            bson:"app_name,omitempty"`
	LogoURL          *string         `grove:"logo_url"            bson:"logo_url,omitempty"`
	CreatedAt        time.Time       `grove:"created_at"          bson:"created_at"`
	UpdatedAt        time.Time       `grove:"updated_at"          bson:"updated_at"`
}

func toAppClientConfigModel(c *appclientconfig.Config) *appClientConfigModel {
	sp, _ := json.Marshal(c.SocialProviders) //nolint:errcheck // best-effort encode
	mm, _ := json.Marshal(c.MFAMethods)      //nolint:errcheck // best-effort encode
	return &appClientConfigModel{
		ID:               c.ID.String(),
		AppID:            c.AppID.String(),
		PasswordEnabled:  c.PasswordEnabled,
		PasskeyEnabled:   c.PasskeyEnabled,
		MagicLinkEnabled: c.MagicLinkEnabled,
		MFAEnabled:       c.MFAEnabled,
		SSOEnabled:       c.SSOEnabled,
		SocialEnabled:    c.SocialEnabled,
		WaitlistEnabled:          c.WaitlistEnabled,
		RequireEmailVerification: c.RequireEmailVerification,
		SocialProviders:          sp,
		MFAMethods:               mm,
		AppName:                  c.AppName,
		LogoURL:                  c.LogoURL,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}
}

func fromAppClientConfigModel(m *appClientConfigModel) (*appclientconfig.Config, error) {
	cfgID, err := id.ParseAppClientConfigID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	var sp []string
	if len(m.SocialProviders) > 0 {
		_ = json.Unmarshal(m.SocialProviders, &sp) //nolint:errcheck // best-effort decode
	}
	var mm []string
	if len(m.MFAMethods) > 0 {
		_ = json.Unmarshal(m.MFAMethods, &mm) //nolint:errcheck // best-effort decode
	}
	return &appclientconfig.Config{
		ID:               cfgID,
		AppID:            appID,
		PasswordEnabled:  m.PasswordEnabled,
		PasskeyEnabled:   m.PasskeyEnabled,
		MagicLinkEnabled: m.MagicLinkEnabled,
		MFAEnabled:       m.MFAEnabled,
		SSOEnabled:       m.SSOEnabled,
		SocialEnabled:    m.SocialEnabled,
		WaitlistEnabled:          m.WaitlistEnabled,
		RequireEmailVerification: m.RequireEmailVerification,
		SocialProviders:          sp,
		MFAMethods:               mm,
		AppName:                  m.AppName,
		LogoURL:                  m.LogoURL,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Setting model
// ──────────────────────────────────────────────────

type settingModel struct {
	grove.BaseModel `grove:"table:authsome_settings"`

	ID        string          `grove:"id,pk"        bson:"_id"`
	Key       string          `grove:"key"          bson:"key"`
	Value     json.RawMessage `grove:"value"        bson:"value"`
	Scope     string          `grove:"scope"        bson:"scope"`
	ScopeID   string          `grove:"scope_id"     bson:"scope_id"`
	AppID     string          `grove:"app_id"       bson:"app_id,omitempty"`
	OrgID     string          `grove:"org_id"       bson:"org_id,omitempty"`
	Enforced  bool            `grove:"enforced"     bson:"enforced"`
	Namespace string          `grove:"namespace"    bson:"namespace,omitempty"`
	Version   int64           `grove:"version"      bson:"version"`
	UpdatedBy string          `grove:"updated_by"   bson:"updated_by,omitempty"`
	CreatedAt time.Time       `grove:"created_at"   bson:"created_at"`
	UpdatedAt time.Time       `grove:"updated_at"   bson:"updated_at"`
}

func toSettingModel(s *settings.Setting) *settingModel {
	return &settingModel{
		ID:        s.ID.String(),
		Key:       s.Key,
		Value:     s.Value,
		Scope:     string(s.Scope),
		ScopeID:   s.ScopeID,
		AppID:     s.AppID,
		OrgID:     s.OrgID,
		Enforced:  s.Enforced,
		Namespace: s.Namespace,
		Version:   s.Version,
		UpdatedBy: s.UpdatedBy,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func fromSettingModel(m *settingModel) (*settings.Setting, error) {
	sID, err := id.ParseSettingID(m.ID)
	if err != nil {
		return nil, err
	}
	return &settings.Setting{
		ID:        sID,
		Key:       m.Key,
		Value:     m.Value,
		Scope:     settings.Scope(m.Scope),
		ScopeID:   m.ScopeID,
		AppID:     m.AppID,
		OrgID:     m.OrgID,
		Enforced:  m.Enforced,
		Namespace: m.Namespace,
		Version:   m.Version,
		UpdatedBy: m.UpdatedBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}
