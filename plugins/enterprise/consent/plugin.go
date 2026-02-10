package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

const (
	PluginID      = "consent"
	PluginName    = "Consent & Privacy Management"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for consent and privacy management.
type Plugin struct {
	service       *Service
	config        *Config
	handler       *Handler
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

// NewPlugin creates a new consent plugin instance.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique plugin identifier.
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name.
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Enterprise consent and privacy management with GDPR/CCPA compliance, cookie consent, data portability (Article 20), and right to be forgotten (Article 17)"
}

// Init initializes the plugin with dependencies from AuthSome.
func (p *Plugin) Init(auth any) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return errs.InternalServerErrorWithMessage("invalid auth instance type")
	}

	db := authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// config configuration from Forge config manager
	var config Config
	if err := configManager.Bind("auth.consent", &config); err != nil {
		// config defaults if binding fails
		config = *DefaultConfig()
	}

	config.Validate() // Ensure defaults are set
	p.config = &config

	if !p.config.Enabled {
		return nil
	}

	// Get user service (interface type)
	userSvcInterface := serviceRegistry.UserService()

	var userSvc *user.Service
	if userSvcInterface != nil {
		userSvc, _ = userSvcInterface.(*user.Service)
	}

	// Initialize repository
	repo := NewBunRepository(db)

	// Initialize service
	p.service = NewService(
		repo,
		p.config,
		userSvc,
	)

	// Initialize handler
	p.handler = NewHandler(p.service)

	// Start background tasks
	p.startBackgroundTasks()

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	// Consent management routes
	consentGroup := router.Group(p.config.Dashboard.Path)
	{
		// Consent records (user endpoints)
		if err := consentGroup.POST("/records", p.handler.CreateConsent,
			forge.WithName("consent.records.create"),
			forge.WithSummary("Create consent record"),
			forge.WithDescription("Record user consent for data processing activities"),
			forge.WithRequestSchema(CreateConsentRequest{}),
			forge.WithResponseSchema(200, "Consent recorded", ConsentRecordResponse{}),
			forge.WithTags("Consent", "GDPR"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/records/:id", p.handler.GetConsent,
			forge.WithName("consent.records.get"),
			forge.WithSummary("Get consent record"),
			forge.WithDescription("Retrieve a specific consent record"),
			forge.WithResponseSchema(200, "Consent record retrieved", ConsentRecordResponse{}),
			forge.WithTags("Consent", "GDPR"),
		
		); err != nil {
			return err
		}
		if err := consentGroup.PUT("/records/:id", p.handler.UpdateConsent,
			forge.WithName("consent.records.update"),
			forge.WithSummary("Update consent record"),
			forge.WithDescription("Update an existing consent record"),
			forge.WithRequestSchema(UpdateConsentRequest{}),
			forge.WithResponseSchema(200, "Consent updated", ConsentRecordResponse{}),
			forge.WithTags("Consent", "GDPR"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.POST("/revoke/:id", p.handler.RevokeConsent,
			forge.WithName("consent.records.revoke"),
			forge.WithSummary("Revoke consent"),
			forge.WithDescription("Revoke previously given consent"),
			forge.WithResponseSchema(200, "Consent revoked", ConsentStatusResponse{}),
			forge.WithTags("Consent", "GDPR"),
		
		); err != nil {
			return err
		}

		// Consent policies (read endpoints public, write endpoints admin only)
		if err := consentGroup.POST("/policies", p.handler.CreateConsentPolicy,
			forge.WithName("consent.policies.create"),
			forge.WithSummary("Create consent policy"),
			forge.WithDescription("Create a new consent policy (admin only)"),
			forge.WithRequestSchema(CreatePolicyRequest{}),
			forge.WithResponseSchema(200, "Policy created", ConsentPolicyResponse{}),
			forge.WithTags("Consent", "Policies"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/policies/:id", p.handler.GetConsentPolicy,
			forge.WithName("consent.policies.get"),
			forge.WithSummary("Get consent policy"),
			forge.WithDescription("Retrieve a specific consent policy"),
			forge.WithResponseSchema(200, "Policy retrieved", ConsentPolicyResponse{}),
			forge.WithTags("Consent", "Policies"),
		
		); err != nil {
			return err
		}

		// Cookie consent (public endpoints for anonymous users)
		if err := consentGroup.POST("/cookies", p.handler.RecordCookieConsent,
			forge.WithName("consent.cookies.record"),
			forge.WithSummary("Record cookie consent"),
			forge.WithDescription("Record user's cookie consent preferences"),
			forge.WithRequestSchema(CookieConsentRequest{}),
			forge.WithResponseSchema(200, "Cookie consent recorded", ConsentCookieResponse{}),
			forge.WithTags("Consent", "Cookies"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/cookies", p.handler.GetCookieConsent,
			forge.WithName("consent.cookies.get"),
			forge.WithSummary("Get cookie consent"),
			forge.WithDescription("Retrieve user's cookie consent preferences"),
			forge.WithResponseSchema(200, "Cookie consent retrieved", ConsentCookieResponse{}),
			forge.WithTags("Consent", "Cookies"),
		
		); err != nil {
			return err
		}

		// Data export (GDPR Article 20 - Right to Data Portability)
		if err := consentGroup.POST("/export", p.handler.RequestDataExport,
			forge.WithName("consent.export.request"),
			forge.WithSummary("Request data export"),
			forge.WithDescription("Request export of all user data (GDPR Article 20)"),
			forge.WithRequestSchema(DataExportRequestInput{}),
			forge.WithResponseSchema(200, "Export request created", ConsentExportResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/export/:id", p.handler.GetDataExport,
			forge.WithName("consent.export.get"),
			forge.WithSummary("Get data export status"),
			forge.WithDescription("Get status of a data export request"),
			forge.WithResponseSchema(200, "Export status retrieved", ConsentExportResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights"),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/export/:id/download", p.handler.DownloadDataExport,
			forge.WithName("consent.export.download"),
			forge.WithSummary("Download data export"),
			forge.WithDescription("Download exported user data"),
			forge.WithResponseSchema(200, "Export file", ConsentExportFileResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights"),
		
		); err != nil {
			return err
		}

		// Data deletion (GDPR Article 17 - Right to be Forgotten)
		if err := consentGroup.POST("/deletion", p.handler.RequestDataDeletion,
			forge.WithName("consent.deletion.request"),
			forge.WithSummary("Request data deletion"),
			forge.WithDescription("Request deletion of all user data (GDPR Article 17 - Right to be Forgotten)"),
			forge.WithRequestSchema(DataDeletionRequestInput{}),
			forge.WithResponseSchema(200, "Deletion request created", ConsentDeletionResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}
		if err := consentGroup.GET("/deletion/:id", p.handler.GetDataDeletion,
			forge.WithName("consent.deletion.get"),
			forge.WithSummary("Get data deletion status"),
			forge.WithDescription("Get status of a data deletion request"),
			forge.WithResponseSchema(200, "Deletion status retrieved", ConsentDeletionResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights"),
		
		); err != nil {
			return err
		}
		if err := consentGroup.POST("/deletion/:id/approve", p.handler.ApproveDeletionRequest,
			forge.WithName("consent.deletion.approve"),
			forge.WithSummary("Approve deletion request"),
			forge.WithDescription("Approve a data deletion request (admin only)"),
			forge.WithResponseSchema(200, "Deletion approved", ConsentStatusResponse{}),
			forge.WithTags("Consent", "GDPR", "Data Rights", "Admin"),
		
		); err != nil {
			return err
		}

		// Privacy settings (admin only)
		if err := consentGroup.GET("/settings", p.handler.GetPrivacySettings,
			forge.WithName("consent.settings.get"),
			forge.WithSummary("Get privacy settings"),
			forge.WithDescription("Get privacy and consent settings"),
			forge.WithResponseSchema(200, "Settings retrieved", ConsentSettingsResponse{}),
			forge.WithTags("Consent", "Settings"),
		
		); err != nil {
			return err
		}
		if err := consentGroup.PUT("/settings", p.handler.UpdatePrivacySettings,
			forge.WithName("consent.settings.update"),
			forge.WithSummary("Update privacy settings"),
			forge.WithDescription("Update privacy and consent settings (admin only)"),
			forge.WithRequestSchema(PrivacySettingsRequest{}),
			forge.WithResponseSchema(200, "Settings updated", ConsentSettingsResponse{}),
			forge.WithTags("Consent", "Settings", "Admin"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}

		// Audit logs
		if err := consentGroup.GET("/audit", p.handler.GetConsentAuditLogs,
			forge.WithName("consent.audit.list"),
			forge.WithSummary("List consent audit logs"),
			forge.WithDescription("List all consent and data processing audit logs"),
			forge.WithResponseSchema(200, "Audit logs retrieved", ConsentAuditLogsResponse{}),
			forge.WithTags("Consent", "Audit"),
		
		); err != nil {
			return err
		}

		// Reports (admin only)
		if err := consentGroup.POST("/reports", p.handler.GenerateConsentReport,
			forge.WithName("consent.reports.generate"),
			forge.WithSummary("Generate consent report"),
			forge.WithDescription("Generate a consent and GDPR compliance report (admin only)"),
			forge.WithResponseSchema(200, "Report generated", ConsentReportResponse{}),
			forge.WithTags("Consent", "Reports", "Admin"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}

		// Data Processing Agreements - TODO: Implement handlers
		// consentGroup.POST("/dpa", p.handler.CreateDPA)
		// consentGroup.GET("/dpa", p.handler.ListDPAs)
		// consentGroup.GET("/dpa/:id", p.handler.GetDPA)
	}

	return nil
}

// RegisterHooks registers plugin hooks with the hook registry.
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if !p.config.Enabled || p.service == nil {
		return nil
	}

	// Register user lifecycle hooks
	hookRegistry.RegisterAfterUserCreate(p.onUserCreated)
	// TODO: Hook signature mismatch - update when core hook registry is finalized
	// hookRegistry.RegisterBeforeUserDelete(p.onBeforeUserDelete)

	// Register organization hooks (multi-tenancy support)
	hookRegistry.RegisterAfterOrganizationCreate(p.onOrganizationCreated)

	return nil
}

// Hook Handlers

func (p *Plugin) onUserCreated(ctx context.Context, u *user.User) error {
	// When a new user is created, initialize default privacy settings
	// and check if there are required consents to be granted

	// In SaaS mode, get organization from context
	orgID := "default" // Standalone mode
	if orgIDCtx, ok := ctx.Value("organization_id").(string); ok {
		orgID = orgIDCtx
	}

	// Get privacy settings for organization
	settings, err := p.service.GetPrivacySettings(ctx, orgID)
	if err != nil {
		// Settings don't exist yet, will be created on first access
		return nil
	}

	// Check if explicit consent is required
	if settings.ConsentRequired && settings.RequireExplicitConsent {
		// Log that user needs to provide consent
		// (Would be enforced at login/first access)
	}

	return nil
}

func (p *Plugin) onBeforeUserDelete(ctx context.Context, u *user.User) error {
	// Before deleting a user, check if there's a data deletion request
	// This ensures compliance with GDPR Article 17
	orgID := "default"
	if orgIDCtx, ok := ctx.Value("organization_id").(string); ok {
		orgID = orgIDCtx
	}

	// Check for pending deletion request
	userIDStr := u.ID.String()

	deletionReq, err := p.service.repo.GetPendingDeletionRequest(ctx, userIDStr, orgID)
	if err != nil || deletionReq == nil {
		// No pending deletion request, create one for audit trail
		_, _ = p.service.RequestDataDeletion(ctx, userIDStr, orgID, &DataDeletionRequestInput{
			Reason:         "User account deletion",
			DeleteSections: []string{"all"},
		})
	}

	// Archive user consent data before deletion
	if p.config.DataDeletion.ArchiveBeforeDeletion {
		// Archive would happen in ProcessDeletionRequest
		// Just ensure it's marked for archiving
	}

	return nil
}

func (p *Plugin) onOrganizationCreated(ctx context.Context, org any) error {
	// When a new organization is created in SaaS mode,
	// initialize default privacy settings for that organization

	// orgID organization ID from interface
	var orgID string

	switch o := org.(type) {
	case *organization.Organization:
		orgID = o.ID.String()
	case map[string]any:
		if id, ok := o["id"].(string); ok {
			orgID = id
		} else if id, ok := o["id"].(xid.ID); ok {
			orgID = id.String()
		}
	}

	if orgID == "" {
		return errs.InternalServerErrorWithMessage("failed to extract organization ID")
	}

	// Create default privacy settings for the new organization
	_, err := p.service.createDefaultPrivacySettings(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to create default privacy settings: %w", err)
	}

	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Consent plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations.
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by the migration system
	// The schema is defined in schema.go and will be registered in migrations/bun/
	return nil
}

// startBackgroundTasks starts background tasks for the plugin.
func (p *Plugin) startBackgroundTasks() {
	if !p.config.Enabled {
		return
	}

	// Start consent expiry checker
	if p.config.Expiry.Enabled && p.config.Expiry.AutoExpireCheck {
		p.cleanupTicker = time.NewTicker(p.config.Expiry.ExpireCheckInterval)
		p.cleanupDone = make(chan bool)

		go func() {
			for {
				select {
				case <-p.cleanupTicker.C:
					p.runExpiryCheck()
					p.cleanupExpiredExports()
				case <-p.cleanupDone:
					return
				}
			}
		}()
	}
}

// runExpiryCheck checks and expires consents that have passed their expiry date.
func (p *Plugin) runExpiryCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := p.service.ExpireConsents(ctx)
	if err != nil {
		// Log error (would use proper logger in production)
		return
	}

	if count > 0 {
	}
}

// cleanupExpiredExports removes expired data export files.
func (p *Plugin) cleanupExpiredExports() {
	if !p.config.DataExport.AutoCleanup {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := p.service.repo.DeleteExpiredExports(ctx, time.Now())
	if err != nil {
		return
	}

	if count > 0 {
	}
}

// Shutdown gracefully shuts down the plugin.
func (p *Plugin) Shutdown(ctx context.Context) error {
	// Stop background tasks
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	return nil
}

// Health checks plugin health.
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return errs.InternalServerErrorWithMessage("service not initialized")
	}

	// Check if database is accessible
	settings, err := p.service.GetPrivacySettings(ctx, "default")
	if err != nil {
		// It's okay if settings don't exist yet
		if !errors.Is(err, ErrPrivacySettingsNotFound) {
			return fmt.Errorf("database health check failed: %w", err)
		}
	}

	_ = settings

	return nil
}

// Service returns the consent service for programmatic access (optional public method).
func (p *Plugin) Service() *Service {
	return p.service
}

// ====== Helper Methods for Integration ======

// RequireConsent middleware that checks if user has granted required consent.
func (p *Plugin) RequireConsent(consentType, purpose string) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			val := c.Get("user_id")
			userID, _ := val.(string)
			val = c.Get("organization_id")
			orgID, _ := val.(string)

			if userID == "" || orgID == "" {
				return c.JSON(401, map[string]any{
					"error": "unauthorized",
				})
			}

			// Check if consent exists and is granted
			consent, err := p.service.repo.GetConsentByUserAndType(c.Request().Context(), userID, orgID, consentType, purpose)
			if err != nil || consent == nil || !consent.Granted {
				return c.JSON(403, map[string]any{
					"error":       "consent required",
					"consentType": consentType,
					"purpose":     purpose,
				})
			}

			// Check if consent has expired
			if consent.ExpiresAt != nil && consent.ExpiresAt.Before(time.Now()) {
				return c.JSON(403, map[string]any{
					"error":       "consent expired",
					"consentType": consentType,
					"purpose":     purpose,
				})
			}

			return next(c)
		}
	}
}

// GetUserConsentStatus returns consent status for a user (for use by other plugins).
func (p *Plugin) GetUserConsentStatus(ctx context.Context, userID, orgID, consentType, purpose string) (bool, error) {
	consent, err := p.service.repo.GetConsentByUserAndType(ctx, userID, orgID, consentType, purpose)
	if err != nil || consent == nil {
		return false, err
	}

	// Check expiry
	if consent.ExpiresAt != nil && consent.ExpiresAt.Before(time.Now()) {
		return false, ErrConsentExpired
	}

	return consent.Granted, nil
}

// ConsentStatusResponse for consent routes.
type ConsentStatusResponse struct {
	Status string `example:"success" json:"status"`
}

type ConsentRecordResponse struct {
	ID string `example:"consent_123" json:"id"`
}

type ConsentPolicyResponse struct {
	ID string `example:"policy_123" json:"id"`
}

type ConsentCookieResponse struct {
	Preferences any `json:"preferences"`
}

type ConsentExportResponse struct {
	ID     string `example:"export_123" json:"id"`
	Status string `example:"processing" json:"status"`
}

type ConsentExportFileResponse struct {
	ContentType string `example:"application/zip" json:"content_type"`
	Data        []byte `json:"data"`
}

type ConsentDeletionResponse struct {
	ID     string `example:"deletion_123" json:"id"`
	Status string `example:"pending"      json:"status"`
}

type ConsentSettingsResponse struct {
	Settings any `json:"settings"`
}

type ConsentAuditLogsResponse struct {
	AuditLogs []any `json:"audit_logs"`
}

type ConsentReportResponse struct {
	ID string `example:"report_123" json:"id"`
}
