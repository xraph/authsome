package migrations

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// Register m2m models before any table operations
		// These models are used as join tables for many-to-many relationships
		// and must be explicitly registered with Bun before creating tables with m2m relations
		db.RegisterModel((*schema.TeamMember)(nil))
		db.RegisterModel((*schema.OrganizationTeamMember)(nil))
		db.RegisterModel((*schema.RolePermission)(nil))
		db.RegisterModel((*schema.APIKeyRole)(nil))

		// Drop existing tables to allow a clean reset (ignoring backward compatibility)
		drop := []string{
			"identity_verification_documents",
			"identity_verification_sessions",
			"identity_verifications",
			"authorization_codes",
			"oauth_tokens",
			"oauth_clients",
			"social_accounts",
			"impersonation_audit",
			"impersonation_sessions",
			"mfa_risk_assessments",
			"mfa_attempts",
			"mfa_trusted_devices",
			"mfa_sessions",
			"mfa_challenges",
			"mfa_factors",
			"phone_verifications",
			"email_otps",
			"passkeys",
			"team_members",
			"teams",
			"organization_team_members",
			"organization_teams",
			"organization_members",
			"organization_invitations",
			"usage_events",
			"form_schemas",
			"webhook_deliveries",
			"webhook_events",
			"webhooks",
			"notification_templates",
			"notifications",
			"apikey_roles",
			"api_keys",
			"policies",
			"user_roles",
			"role_permissions",
			"permissions",
			"roles",
			"user_bans",
			"audit_events",
			"devices",
			"verifications",
			"accounts",
			"sessions",
			"members",
			"users",
			"organizations",
			"environment_promotions",
			"environments",
			"jwt_keys",
			"sso_providers",
			"apps",
		}
		for _, t := range drop {
			_, _ = db.NewDropTable().Table(t).IfExists().Cascade().Exec(ctx)
		}

		// Create core tables
		if _, err := db.NewCreateTable().Model((*schema.App)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Environment)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.EnvironmentPromotion)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Identity core - Users must be created before Organization tables that reference them
		if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// User organizations (created after Users since OrganizationMember references users)
		if _, err := db.NewCreateTable().Model((*schema.Organization)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.OrganizationMember)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.OrganizationTeam)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.OrganizationTeamMember)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.OrganizationInvitation)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Additional identity tables (depend on User)
		if _, err := db.NewCreateTable().Model((*schema.Member)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Account)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Verification)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Device)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.AuditEvent)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.UserBan)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// RBAC
		if _, err := db.NewCreateTable().Model((*schema.Role)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Permission)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.RolePermission)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.UserRole)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Policy)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Keys & Providers
		if _, err := db.NewCreateTable().Model((*schema.JWTKey)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.APIKey)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.APIKeyRole)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.SSOProvider)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Notifications & Webhooks
		if _, err := db.NewCreateTable().Model((*schema.NotificationTemplate)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Notification)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Webhook)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Event)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.Delivery)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Forms & Usage
		if _, err := db.NewCreateTable().Model((*schema.FormSchema)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.UsageEvent)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Teams (app-level)
		if _, err := db.NewCreateTable().Model((*schema.Team)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.TeamMember)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Plugins: passkey, magic link, phone, email OTP
		if _, err := db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.EmailOTP)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// MFA suite
		if _, err := db.NewCreateTable().Model((*schema.MFAFactor)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFAChallenge)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFASession)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFATrustedDevice)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFAPolicy)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFAAttempt)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.MFARiskAssessment)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// OAuth/OIDC
		if _, err := db.NewCreateTable().Model((*schema.OAuthClient)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.AuthorizationCode)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.OAuthToken)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.SocialAccount)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Impersonation
		if _, err := db.NewCreateTable().Model((*schema.ImpersonationSession)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model((*schema.ImpersonationAuditEvent)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Indexes
		if _, err := db.NewCreateIndex().Model((*schema.User)(nil)).Index("idx_users_email").Column("email").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.Session)(nil)).Index("idx_sessions_token").Column("token").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.Member)(nil)).Index("idx_members_app_user").Column("app_id", "user_id").Unique().IfNotExists().Exec(ctx); err != nil {
			return err
		}
		// Unique constraint for organization members to prevent duplicate memberships
		if _, err := db.NewCreateIndex().Model((*schema.OrganizationMember)(nil)).Index("idx_organization_members_org_user").Column("organization_id", "user_id").Unique().IfNotExists().Exec(ctx); err != nil {
			return err
		}
		// Event (webhook_events) now uses app_id instead of organization_id after app-scoped refactoring
		if _, err := db.NewCreateIndex().Model((*schema.Event)(nil)).Index("idx_webhook_events_app").Column("app_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.Delivery)(nil)).Index("idx_webhook_deliveries_webhook_id").Column("webhook_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.Delivery)(nil)).Index("idx_webhook_deliveries_event_id").Column("event_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.Delivery)(nil)).Index("idx_webhook_deliveries_status").Column("status", "next_retry_at").IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Composite indexes for usage events
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_endpoint_method").Column("endpoint", "method").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_plugin_feature").Column("plugin", "feature").IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Unique constraints where Bun builder support is limited
		if _, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS idx_env_app_slug ON environments(app_id, slug)"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS idx_org_app_env_slug ON organizations(app_id, environment_id, slug)"); err != nil {
			return err
		}

		// Usage events indexes (Bun doesn't support `index` in struct tags)
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_user_id").Column("user_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_organization_id").Column("organization_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_session_id").Column("session_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_api_key_id").Column("api_key_id").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_method").Column("method").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_endpoint").Column("endpoint").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_status_code").Column("status_code").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_auth_method").Column("auth_method").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_country").Column("country").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_plugin").Column("plugin").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*schema.UsageEvent)(nil)).Index("idx_usage_feature").Column("feature").IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// JWT Keys indexes (app-scoped architecture)
		if _, err := db.NewCreateIndex().
			Model((*schema.JWTKey)(nil)).
			Index("idx_jwt_keys_app_id").
			Column("app_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.JWTKey)(nil)).
			Index("idx_jwt_keys_platform").
			Column("is_platform_key").
			Where("is_platform_key = true AND deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.JWTKey)(nil)).
			Index("idx_jwt_keys_key_id_app").
			Column("key_id", "app_id").
			Where("deleted_at IS NULL").
			Unique().
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.JWTKey)(nil)).
			Index("idx_jwt_keys_active").
			Column("active", "app_id").
			Where("active = true AND deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.JWTKey)(nil)).
			Index("idx_jwt_keys_expires_at").
			Column("expires_at").
			Where("expires_at IS NOT NULL AND deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		// API Keys indexes (environment-scoped)
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_app_id").
			Column("app_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_environment_id").
			Column("environment_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_app_env").
			Column("app_id", "environment_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_active").
			Column("active").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_user_id").
			Column("user_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_key_type").
			Column("key_type").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_app_key_type").
			Column("app_id", "key_type").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKey)(nil)).
			Index("idx_apikeys_impersonate_user").
			Column("impersonate_user_id").
			Where("impersonate_user_id IS NOT NULL AND deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		// API Key Roles indexes (RBAC join table)
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKeyRole)(nil)).
			Index("idx_apikey_roles_apikey").
			Column("api_key_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKeyRole)(nil)).
			Index("idx_apikey_roles_role").
			Column("role_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.APIKeyRole)(nil)).
			Index("idx_apikey_roles_org").
			Column("organization_id").
			Where("deleted_at IS NULL AND organization_id IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		// Unique constraint to prevent duplicate role assignments
		if _, err := db.ExecContext(ctx, `
			CREATE UNIQUE INDEX IF NOT EXISTS idx_apikey_roles_unique 
			ON apikey_roles(api_key_id, role_id, COALESCE(organization_id, '')) 
			WHERE deleted_at IS NULL
		`); err != nil {
			return err
		}

		// Session indexes (app-scoped with environment and organization)
		if _, err := db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_app_id").
			Column("app_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_app_user").
			Column("app_id", "user_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_expires_at").
			Column("expires_at").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_environment_id").
			Column("environment_id").
			Where("deleted_at IS NULL AND environment_id IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Session)(nil)).
			Index("idx_sessions_organization_id").
			Column("organization_id").
			Where("deleted_at IS NULL AND organization_id IS NOT NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		// Webhook indexes (app-scoped with environment)
		if _, err := db.NewCreateIndex().
			Model((*schema.Webhook)(nil)).
			Index("idx_webhooks_app_id").
			Column("app_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Webhook)(nil)).
			Index("idx_webhooks_app_env").
			Column("app_id", "environment_id").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Webhook)(nil)).
			Index("idx_webhooks_enabled").
			Column("enabled").
			Where("deleted_at IS NULL").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		// Webhook Events indexes (app-scoped with environment)
		if _, err := db.NewCreateIndex().
			Model((*schema.Event)(nil)).
			Index("idx_webhook_events_app_env").
			Column("app_id", "environment_id").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Event)(nil)).
			Index("idx_webhook_events_type").
			Column("type").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().
			Model((*schema.Event)(nil)).
			Index("idx_webhook_events_occurred_at").
			Column("occurred_at").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		// Optional seed: create platform app and default dev environment if not present
		var count int
		if err := db.NewSelect().Model((*schema.App)(nil)).ColumnExpr("COUNT(*)").Scan(ctx, &count); err == nil && count == 0 {
			platformID := xid.New()
			systemActor := xid.New()
			_, err := db.NewInsert().Model(&schema.App{ID: platformID, Name: "Platform", Slug: "platform", IsPlatform: true, AuditableModel: schema.AuditableModel{CreatedBy: systemActor, UpdatedBy: systemActor}}).Exec(ctx)
			if err != nil {
				return err
			}
			_, err = db.NewInsert().Model(&schema.Environment{
				ID:             xid.New(),
				AppID:          platformID,
				Name:           "Development",
				Slug:           "dev",
				Type:           schema.EnvironmentTypeDevelopment,
				Status:         schema.EnvironmentStatusActive,
				IsDefault:      true,
				Config:         map[string]interface{}{},
				AuditableModel: schema.AuditableModel{CreatedBy: systemActor, UpdatedBy: systemActor},
			}).Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// Drop all tables in reverse dependency order
		tables := []string{
			"identity_verification_documents",
			"identity_verification_sessions",
			"identity_verifications",
			"authorization_codes",
			"oauth_tokens",
			"oauth_clients",
			"social_accounts",
			"impersonation_audit",
			"impersonation_sessions",
			"mfa_risk_assessments",
			"mfa_attempts",
			"mfa_trusted_devices",
			"mfa_sessions",
			"mfa_challenges",
			"mfa_factors",
			"phone_verifications",
			"email_otps",
			"passkeys",
			"team_members",
			"teams",
			"organization_team_members",
			"organization_teams",
			"organization_members",
			"organization_invitations",
			"usage_events",
			"form_schemas",
			"webhook_deliveries",
			"webhook_events",
			"webhooks",
			"notification_templates",
			"notifications",
			"apikey_roles",
			"api_keys",
			"policies",
			"user_roles",
			"permissions",
			"roles",
			"user_bans",
			"audit_events",
			"devices",
			"verifications",
			"accounts",
			"sessions",
			"members",
			"users",
			"organizations",
			"environment_promotions",
			"environments",
			"jwt_keys",
			"sso_providers",
			"apps",
		}
		for _, table := range tables {
			if _, err := db.NewDropTable().Table(table).IfExists().Cascade().Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}
