package repository

import (
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/webhook"
	repository "github.com/xraph/authsome/repository/organization"
)

type Repository interface {
	// Core repositories
	User() *UserRepository
	Session() *SessionRepository
	SocialAccount() SocialAccountRepository

	// Authentication & Security
	APIKey() *APIKeyRepository
	Audit() *AuditRepository
	Device() *DeviceRepository
	JWTKey() *JWTKeyRepository
	Role() *RoleRepository
	Security() *SecurityRepository
	Policy() *PolicyRepository
	Permission() *PermissionRepository
	RolePermission() *RolePermissionRepository
	UserRole() *UserRoleRepository
	UserBan() *UserBanRepository

	Notification() notification.Repository

	// OAuth & SSO
	OAuthToken() *OAuthTokenRepository
	OAuthClient() *OAuthClientRepository
	AuthorizationCode() *AuthorizationCodeRepository
	SSOProvider() *SSOProviderRepository
	SocialProviderConfig() SocialProviderConfigRepository

	// Multi-factor Authentication
	TwoFA() *TwoFARepository
	MFA() *MFARepository
	EmailOTP() *EmailOTPRepository

	// Authentication Methods
	Phone() *PhoneRepository
	MagicLink() *MagicLinkRepository

	// Identity & Verification
	IdentityVerification() *IdentityVerificationRepository
	Verification() *verificationRepository

	// App & Environment
	App() *AppRepository
	Environment() environment.Repository

	// Impersonation
	Impersonation() *ImpersonationRepository

	// Forms & Webhooks
	Forms() forms.Repository
	Webhook() webhook.Repository

	// Organization repositories
	Organization() organization.OrganizationRepository
	OrganizationMember() organization.MemberRepository
	OrganizationTeam() organization.TeamRepository
	OrganizationInvitation() organization.InvitationRepository
}

type Repo struct {
	// Core repositories
	user          *UserRepository
	session       *SessionRepository
	socialAccount SocialAccountRepository

	// Authentication & Security
	apiKey         *APIKeyRepository
	audit          *AuditRepository
	device         *DeviceRepository
	jwtKey         *JWTKeyRepository
	role           *RoleRepository
	security       *SecurityRepository
	policy         *PolicyRepository
	permission     *PermissionRepository
	rolePermission *RolePermissionRepository
	userRole       *UserRoleRepository
	userBan        *UserBanRepository

	// Notification
	notification notification.Repository

	// OAuth & SSO
	oauthToken           *OAuthTokenRepository
	oauthClient          *OAuthClientRepository
	authorizationCode    *AuthorizationCodeRepository
	ssoProvider          *SSOProviderRepository
	socialProviderConfig SocialProviderConfigRepository

	// Multi-factor Authentication
	twoFA    *TwoFARepository
	mfa      *MFARepository
	emailOTP *EmailOTPRepository

	// Authentication Methods
	phone     *PhoneRepository
	magicLink *MagicLinkRepository

	// Identity & Verification
	identityVerification *IdentityVerificationRepository
	verification         *verificationRepository

	// App & Environment
	app         *AppRepository
	environment environment.Repository

	// Impersonation
	impersonation *ImpersonationRepository

	// Forms & Webhooks
	forms   forms.Repository
	webhook webhook.Repository

	// Organization repositories
	organization           organization.OrganizationRepository
	organizationMember     organization.MemberRepository
	organizationTeam       organization.TeamRepository
	organizationInvitation organization.InvitationRepository
}

func NewRepo(db *bun.DB) Repository {
	return &Repo{
		// Core repositories
		user:          NewUserRepository(db),
		session:       NewSessionRepository(db),
		socialAccount: NewSocialAccountRepository(db),

		// Authentication & Security
		apiKey:         NewAPIKeyRepository(db),
		audit:          NewAuditRepository(db),
		device:         NewDeviceRepository(db),
		jwtKey:         NewJWTKeyRepository(db),
		role:           NewRoleRepository(db),
		security:       NewSecurityRepository(db),
		policy:         NewPolicyRepository(db),
		permission:     NewPermissionRepository(db),
		rolePermission: NewRolePermissionRepository(db),
		userRole:       NewUserRoleRepository(db),
		userBan:        NewUserBanRepository(db),

		// Notification
		notification: NewNotificationRepository(db),

		// OAuth & SSO
		oauthToken:           NewOAuthTokenRepository(db),
		oauthClient:          NewOAuthClientRepository(db),
		authorizationCode:    NewAuthorizationCodeRepository(db),
		ssoProvider:          NewSSOProviderRepository(db),
		socialProviderConfig: NewSocialProviderConfigRepository(db),

		// Multi-factor Authentication
		twoFA:    NewTwoFARepository(db),
		mfa:      NewMFARepository(db),
		emailOTP: NewEmailOTPRepository(db),

		// Authentication Methods
		phone:     NewPhoneRepository(db),
		magicLink: NewMagicLinkRepository(db),

		// Identity & Verification
		identityVerification: NewIdentityVerificationRepository(db),
		verification:         NewVerificationRepository(db),

		// App & Environment
		app:         NewAppRepository(db),
		environment: NewEnvironmentRepository(db),

		// Impersonation
		impersonation: NewImpersonationRepository(db),

		// Forms & Webhooks
		forms:   NewFormsRepository(db),
		webhook: NewWebhookRepository(db),

		// Organization repositories
		organization:           repository.NewOrganizationRepository(db),
		organizationMember:     repository.NewOrganizationMemberRepository(db),
		organizationTeam:       repository.NewOrganizationTeamRepository(db),
		organizationInvitation: repository.NewOrganizationInvitationRepository(db),
	}
}

// Core repositories
func (r *Repo) User() *UserRepository {
	return r.user
}

func (r *Repo) Session() *SessionRepository {
	return r.session
}

func (r *Repo) SocialAccount() SocialAccountRepository {
	return r.socialAccount
}

// Authentication & Security
func (r *Repo) APIKey() *APIKeyRepository {
	return r.apiKey
}

func (r *Repo) Audit() *AuditRepository {
	return r.audit
}

func (r *Repo) Device() *DeviceRepository {
	return r.device
}

func (r *Repo) JWTKey() *JWTKeyRepository {
	return r.jwtKey
}

func (r *Repo) Role() *RoleRepository {
	return r.role
}

func (r *Repo) Security() *SecurityRepository {
	return r.security
}

func (r *Repo) Policy() *PolicyRepository {
	return r.policy
}

func (r *Repo) Permission() *PermissionRepository {
	return r.permission
}

func (r *Repo) RolePermission() *RolePermissionRepository {
	return r.rolePermission
}

func (r *Repo) UserRole() *UserRoleRepository {
	return r.userRole
}

func (r *Repo) UserBan() *UserBanRepository {
	return r.userBan
}

func (r *Repo) Notification() notification.Repository {
	return r.notification
}

// OAuth & SSO
func (r *Repo) OAuthToken() *OAuthTokenRepository {
	return r.oauthToken
}

func (r *Repo) OAuthClient() *OAuthClientRepository {
	return r.oauthClient
}

func (r *Repo) AuthorizationCode() *AuthorizationCodeRepository {
	return r.authorizationCode
}

func (r *Repo) SSOProvider() *SSOProviderRepository {
	return r.ssoProvider
}

func (r *Repo) SocialProviderConfig() SocialProviderConfigRepository {
	return r.socialProviderConfig
}

// Multi-factor Authentication
func (r *Repo) TwoFA() *TwoFARepository {
	return r.twoFA
}

func (r *Repo) MFA() *MFARepository {
	return r.mfa
}

func (r *Repo) EmailOTP() *EmailOTPRepository {
	return r.emailOTP
}

// Authentication Methods
func (r *Repo) Phone() *PhoneRepository {
	return r.phone
}

func (r *Repo) MagicLink() *MagicLinkRepository {
	return r.magicLink
}

// Identity & Verification
func (r *Repo) IdentityVerification() *IdentityVerificationRepository {
	return r.identityVerification
}

func (r *Repo) Verification() *verificationRepository {
	return r.verification
}

// App & Environment
func (r *Repo) App() *AppRepository {
	return r.app
}

func (r *Repo) Environment() environment.Repository {
	return r.environment
}

// Impersonation
func (r *Repo) Impersonation() *ImpersonationRepository {
	return r.impersonation
}

// Forms & Webhooks
func (r *Repo) Forms() forms.Repository {
	return r.forms
}

func (r *Repo) Webhook() webhook.Repository {
	return r.webhook
}

// Organization repositories
func (r *Repo) Organization() organization.OrganizationRepository {
	return r.organization
}

func (r *Repo) OrganizationMember() organization.MemberRepository {
	return r.organizationMember
}

func (r *Repo) OrganizationTeam() organization.TeamRepository {
	return r.organizationTeam
}

func (r *Repo) OrganizationInvitation() organization.InvitationRepository {
	return r.organizationInvitation
}
