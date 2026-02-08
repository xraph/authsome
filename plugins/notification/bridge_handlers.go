package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/notification/builder"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Bridge Function Implementations
// =============================================================================

// bridgeGetSettings handles the getSettings bridge call
func (e *DashboardExtension) bridgeGetSettings(ctx bridge.Context, input GetSettingsInput) (*GetSettingsResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Get current plugin config
	cfg := e.plugin.config

	// Build DTO from config
	settings := NotificationSettingsDTO{
		AppName: cfg.AppName,
		Auth: AuthAutoSendDTO{
			Welcome:           cfg.AutoSend.Auth.Welcome,
			VerificationEmail: cfg.AutoSend.Auth.VerificationEmail,
			MagicLink:         cfg.AutoSend.Auth.MagicLink,
			EmailOTP:          cfg.AutoSend.Auth.EmailOTP,
			MFACode:           cfg.AutoSend.Auth.MFACode,
			PasswordReset:     cfg.AutoSend.Auth.PasswordReset,
		},
		Organization: OrganizationAutoSendDTO{
			Invite:        cfg.AutoSend.Organization.Invite,
			MemberAdded:   cfg.AutoSend.Organization.MemberAdded,
			MemberRemoved: cfg.AutoSend.Organization.MemberRemoved,
			RoleChanged:   cfg.AutoSend.Organization.RoleChanged,
			Transfer:      cfg.AutoSend.Organization.Transfer,
			Deleted:       cfg.AutoSend.Organization.Deleted,
			MemberLeft:    cfg.AutoSend.Organization.MemberLeft,
		},
		Session: SessionAutoSendDTO{
			NewDevice:       cfg.AutoSend.Session.NewDevice,
			NewLocation:     cfg.AutoSend.Session.NewLocation,
			SuspiciousLogin: cfg.AutoSend.Session.SuspiciousLogin,
			DeviceRemoved:   cfg.AutoSend.Session.DeviceRemoved,
			AllRevoked:      cfg.AutoSend.Session.AllRevoked,
		},
		Account: AccountAutoSendDTO{
			EmailChangeRequest: cfg.AutoSend.Account.EmailChangeRequest,
			EmailChanged:       cfg.AutoSend.Account.EmailChanged,
			PasswordChanged:    cfg.AutoSend.Account.PasswordChanged,
			UsernameChanged:    cfg.AutoSend.Account.UsernameChanged,
			Deleted:            cfg.AutoSend.Account.Deleted,
			Suspended:          cfg.AutoSend.Account.Suspended,
			Reactivated:        cfg.AutoSend.Account.Reactivated,
		},
	}

	// Log access
	userID, _ := contexts.GetUserID(goCtx)
	e.plugin.logger.Debug("notification settings retrieved via bridge",
		forge.F("appId", appID.String()),
		forge.F("userId", userID.String()))

	return &GetSettingsResult{
		Settings: settings,
	}, nil
}

// bridgeUpdateSettings handles the updateSettings bridge call
func (e *DashboardExtension) bridgeUpdateSettings(ctx bridge.Context, input UpdateSettingsInput) (*UpdateSettingsResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Update app name if provided
	if input.AppName != "" {
		e.plugin.config.AppName = input.AppName
	}

	// Update auth settings if provided
	if input.Auth != nil {
		e.plugin.config.AutoSend.Auth.Welcome = input.Auth.Welcome
		e.plugin.config.AutoSend.Auth.VerificationEmail = input.Auth.VerificationEmail
		e.plugin.config.AutoSend.Auth.MagicLink = input.Auth.MagicLink
		e.plugin.config.AutoSend.Auth.EmailOTP = input.Auth.EmailOTP
		e.plugin.config.AutoSend.Auth.MFACode = input.Auth.MFACode
		e.plugin.config.AutoSend.Auth.PasswordReset = input.Auth.PasswordReset
	}

	// Update organization settings if provided
	if input.Organization != nil {
		e.plugin.config.AutoSend.Organization.Invite = input.Organization.Invite
		e.plugin.config.AutoSend.Organization.MemberAdded = input.Organization.MemberAdded
		e.plugin.config.AutoSend.Organization.MemberRemoved = input.Organization.MemberRemoved
		e.plugin.config.AutoSend.Organization.RoleChanged = input.Organization.RoleChanged
		e.plugin.config.AutoSend.Organization.Transfer = input.Organization.Transfer
		e.plugin.config.AutoSend.Organization.Deleted = input.Organization.Deleted
		e.plugin.config.AutoSend.Organization.MemberLeft = input.Organization.MemberLeft
	}

	// Update session settings if provided
	if input.Session != nil {
		e.plugin.config.AutoSend.Session.NewDevice = input.Session.NewDevice
		e.plugin.config.AutoSend.Session.NewLocation = input.Session.NewLocation
		e.plugin.config.AutoSend.Session.SuspiciousLogin = input.Session.SuspiciousLogin
		e.plugin.config.AutoSend.Session.DeviceRemoved = input.Session.DeviceRemoved
		e.plugin.config.AutoSend.Session.AllRevoked = input.Session.AllRevoked
	}

	// Update account settings if provided
	if input.Account != nil {
		e.plugin.config.AutoSend.Account.EmailChangeRequest = input.Account.EmailChangeRequest
		e.plugin.config.AutoSend.Account.EmailChanged = input.Account.EmailChanged
		e.plugin.config.AutoSend.Account.PasswordChanged = input.Account.PasswordChanged
		e.plugin.config.AutoSend.Account.UsernameChanged = input.Account.UsernameChanged
		e.plugin.config.AutoSend.Account.Deleted = input.Account.Deleted
		e.plugin.config.AutoSend.Account.Suspended = input.Account.Suspended
		e.plugin.config.AutoSend.Account.Reactivated = input.Account.Reactivated
	}

	// Build updated settings DTO
	settings := NotificationSettingsDTO{
		AppName: e.plugin.config.AppName,
		Auth: AuthAutoSendDTO{
			Welcome:           e.plugin.config.AutoSend.Auth.Welcome,
			VerificationEmail: e.plugin.config.AutoSend.Auth.VerificationEmail,
			MagicLink:         e.plugin.config.AutoSend.Auth.MagicLink,
			EmailOTP:          e.plugin.config.AutoSend.Auth.EmailOTP,
			MFACode:           e.plugin.config.AutoSend.Auth.MFACode,
			PasswordReset:     e.plugin.config.AutoSend.Auth.PasswordReset,
		},
		Organization: OrganizationAutoSendDTO{
			Invite:        e.plugin.config.AutoSend.Organization.Invite,
			MemberAdded:   e.plugin.config.AutoSend.Organization.MemberAdded,
			MemberRemoved: e.plugin.config.AutoSend.Organization.MemberRemoved,
			RoleChanged:   e.plugin.config.AutoSend.Organization.RoleChanged,
			Transfer:      e.plugin.config.AutoSend.Organization.Transfer,
			Deleted:       e.plugin.config.AutoSend.Organization.Deleted,
			MemberLeft:    e.plugin.config.AutoSend.Organization.MemberLeft,
		},
		Session: SessionAutoSendDTO{
			NewDevice:       e.plugin.config.AutoSend.Session.NewDevice,
			NewLocation:     e.plugin.config.AutoSend.Session.NewLocation,
			SuspiciousLogin: e.plugin.config.AutoSend.Session.SuspiciousLogin,
			DeviceRemoved:   e.plugin.config.AutoSend.Session.DeviceRemoved,
			AllRevoked:      e.plugin.config.AutoSend.Session.AllRevoked,
		},
		Account: AccountAutoSendDTO{
			EmailChangeRequest: e.plugin.config.AutoSend.Account.EmailChangeRequest,
			EmailChanged:       e.plugin.config.AutoSend.Account.EmailChanged,
			PasswordChanged:    e.plugin.config.AutoSend.Account.PasswordChanged,
			UsernameChanged:    e.plugin.config.AutoSend.Account.UsernameChanged,
			Deleted:            e.plugin.config.AutoSend.Account.Deleted,
			Suspended:          e.plugin.config.AutoSend.Account.Suspended,
			Reactivated:        e.plugin.config.AutoSend.Account.Reactivated,
		},
	}

	// Log update
	userID, _ := contexts.GetUserID(goCtx)
	e.plugin.logger.Info("notification settings updated via bridge",
		forge.F("appId", appID.String()),
		forge.F("userId", userID.String()))

	return &UpdateSettingsResult{
		Success:  true,
		Settings: settings,
		Message:  "Settings updated successfully",
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildContextFromBridge retrieves the Go context from the HTTP request.
// The context has already been enriched by the dashboard v2 BridgeContextMiddleware.
func (e *DashboardExtension) buildContextFromBridge(bridgeCtx bridge.Context) (context.Context, xid.ID, error) {
	// Get the already-enriched context from the HTTP request
	var goCtx context.Context
	req := bridgeCtx.Request()

	if req != nil {
		goCtx = req.Context()
	} else {
		goCtx = bridgeCtx.Context()
	}

	// Extract app ID from the existing context (set by middleware from URL)
	appID, hasAppID := contexts.GetAppID(goCtx)
	if !hasAppID || appID == xid.NilID() {
		e.plugin.logger.Error("[NotificationBridge] No app ID in context")
		return nil, xid.NilID(), errs.BadRequest("invalid app context")
	}

	// Verify that user is authenticated
	userID, hasUserID := contexts.GetUserID(goCtx)
	if !hasUserID || userID == xid.NilID() {
		e.plugin.logger.Error("[NotificationBridge] Unauthorized - no user ID in context")
		return nil, xid.NilID(), errs.Unauthorized()
	}

	return goCtx, appID, nil
}

// =============================================================================
// Template Bridge Handlers
// =============================================================================

// bridgeListTemplates handles the listTemplates bridge call
func (e *DashboardExtension) bridgeListTemplates(ctx bridge.Context, input ListTemplatesInput) (*ListTemplatesResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build filter
	filter := &notification.ListTemplatesFilter{
		AppID: appID,
		PaginationParams: pagination.PaginationParams{
			Page:  input.Page,
			Limit: input.Limit,
		},
	}

	if input.Type != nil {
		notifType := notification.NotificationType(*input.Type)
		filter.Type = &notifType
	}
	if input.Language != nil {
		filter.Language = input.Language
	}
	if input.Active != nil {
		filter.Active = input.Active
	}

	// Fetch templates
	response, err := e.plugin.service.ListTemplates(goCtx, filter)
	if err != nil {
		e.plugin.logger.Error("failed to list templates", forge.F("error", err))
		return nil, errs.InternalServerError("failed to list templates", err)
	}

	// Convert to DTOs
	templates := make([]TemplateDTO, len(response.Data))
	for i, tmpl := range response.Data {
		templates[i] = e.templateToDTO(tmpl)
	}

	pagination := PaginationDTO{
		CurrentPage: 1,
		TotalPages:  1,
		TotalCount:  0,
		PageSize:    filter.Limit,
		HasNext:     false,
		HasPrev:     false,
	}

	if response.Pagination != nil {
		pagination = PaginationDTO{
			CurrentPage: response.Pagination.CurrentPage,
			TotalPages:  response.Pagination.TotalPages,
			TotalCount:  response.Pagination.Total,
			PageSize:    response.Pagination.Limit,
			HasNext:     response.Pagination.HasNext,
			HasPrev:     response.Pagination.HasPrev,
		}
	}

	return &ListTemplatesResult{
		Templates:  templates,
		Pagination: pagination,
	}, nil
}

// bridgeGetTemplate handles the getTemplate bridge call
func (e *DashboardExtension) bridgeGetTemplate(ctx bridge.Context, input GetTemplateInput) (*GetTemplateResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	templateID, err := xid.FromString(input.TemplateID)
	if err != nil {
		return nil, errs.BadRequest("invalid templateId")
	}

	template, err := e.plugin.service.GetTemplate(goCtx, templateID)
	if err != nil {
		return nil, errs.NotFound("template not found")
	}

	return &GetTemplateResult{
		Template: e.templateToDTO(template),
	}, nil
}

// bridgeCreateTemplate handles the createTemplate bridge call
func (e *DashboardExtension) bridgeCreateTemplate(ctx bridge.Context, input CreateTemplateInput) (*CreateTemplateResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build create request
	createReq := &notification.CreateTemplateRequest{
		AppID:       appID,
		TemplateKey: input.TemplateKey,
		Name:        input.Name,
		Type:        notification.NotificationType(input.Type),
		Language:    input.Language,
		Subject:     input.Subject,
		Body:        input.Body,
		Variables:   input.Variables,
		Metadata:    input.Metadata,
	}

	template, err := e.plugin.service.CreateTemplate(goCtx, createReq)
	if err != nil {
		e.plugin.logger.Error("failed to create template", forge.F("error", err))
		return nil, errs.InternalServerError("failed to create template", err)
	}

	return &CreateTemplateResult{
		Success:  true,
		Template: e.templateToDTO(template),
		Message:  "Template created successfully",
	}, nil
}

// bridgeUpdateTemplate handles the updateTemplate bridge call
func (e *DashboardExtension) bridgeUpdateTemplate(ctx bridge.Context, input UpdateTemplateInput) (*UpdateTemplateResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	templateID, err := xid.FromString(input.TemplateID)
	if err != nil {
		return nil, errs.BadRequest("invalid templateId")
	}

	// Build update request
	updateReq := &notification.UpdateTemplateRequest{
		Name:      input.Name,
		Subject:   input.Subject,
		Body:      input.Body,
		Variables: input.Variables,
		Metadata:  input.Metadata,
		Active:    input.Active,
	}

	err = e.plugin.service.UpdateTemplate(goCtx, templateID, updateReq)
	if err != nil {
		e.plugin.logger.Error("failed to update template", forge.F("error", err))
		return nil, errs.InternalServerError("failed to update template", err)
	}

	// Fetch updated template
	template, err := e.plugin.service.GetTemplate(goCtx, templateID)
	if err != nil {
		return nil, errs.InternalServerError("failed to fetch updated template", err)
	}

	return &UpdateTemplateResult{
		Success:  true,
		Template: e.templateToDTO(template),
		Message:  "Template updated successfully",
	}, nil
}

// bridgeDeleteTemplate handles the deleteTemplate bridge call
func (e *DashboardExtension) bridgeDeleteTemplate(ctx bridge.Context, input DeleteTemplateInput) (*DeleteTemplateResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	templateID, err := xid.FromString(input.TemplateID)
	if err != nil {
		return nil, errs.BadRequest("invalid templateId")
	}

	err = e.plugin.service.DeleteTemplate(goCtx, templateID)
	if err != nil {
		e.plugin.logger.Error("failed to delete template", forge.F("error", err))
		return nil, errs.InternalServerError("failed to delete template", err)
	}

	return &DeleteTemplateResult{
		Success: true,
		Message: "Template deleted successfully",
	}, nil
}

// bridgePreviewTemplate handles the previewTemplate bridge call
func (e *DashboardExtension) bridgePreviewTemplate(ctx bridge.Context, input PreviewTemplateInput) (*PreviewTemplateResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	templateID, err := xid.FromString(input.TemplateID)
	if err != nil {
		return nil, errs.BadRequest("invalid templateId")
	}

	// Get template
	template, err := e.plugin.service.GetTemplate(goCtx, templateID)
	if err != nil {
		return nil, errs.NotFound("template not found")
	}

	// Create template engine for rendering
	engine := notification.NewSimpleTemplateEngine()

	// Render template with variables
	renderedBody, err := engine.Render(template.Body, input.Variables)
	if err != nil {
		return nil, errs.InternalServerError("failed to render template body", err)
	}

	renderedSubject := template.Subject
	if template.Subject != "" {
		renderedSubject, err = engine.Render(template.Subject, input.Variables)
		if err != nil {
			return nil, errs.InternalServerError("failed to render template subject", err)
		}
	}

	return &PreviewTemplateResult{
		Subject:    renderedSubject,
		Body:       renderedBody,
		RenderedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// bridgeTestSendTemplate handles the testSendTemplate bridge call
func (e *DashboardExtension) bridgeTestSendTemplate(ctx bridge.Context, input TestSendTemplateInput) (*TestSendTemplateResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	templateID, err := xid.FromString(input.TemplateID)
	if err != nil {
		return nil, errs.BadRequest("invalid templateId")
	}

	// Get template
	template, err := e.plugin.service.GetTemplate(goCtx, templateID)
	if err != nil {
		return nil, errs.NotFound("template not found")
	}

	// Build send request
	sendReq := &notification.SendRequest{
		AppID:     template.AppID,
		Type:      template.Type,
		Recipient: input.Recipient,
		Variables: input.Variables,
	}

	// Create template engine for rendering
	engine := notification.NewSimpleTemplateEngine()

	// Render template
	renderedBody, err := engine.Render(template.Body, input.Variables)
	if err != nil {
		return nil, errs.InternalServerError("failed to render template", err)
	}
	sendReq.Body = renderedBody

	if template.Subject != "" {
		renderedSubject, err := engine.Render(template.Subject, input.Variables)
		if err != nil {
			return nil, errs.InternalServerError("failed to render subject", err)
		}
		sendReq.Subject = renderedSubject
	}

	// Send notification
	_, err = e.plugin.service.Send(goCtx, sendReq)
	if err != nil {
		e.plugin.logger.Error("failed to send test notification", forge.F("error", err))
		return nil, errs.InternalServerError("failed to send test notification", err)
	}

	return &TestSendTemplateResult{
		Success: true,
		Message: "Test notification sent successfully",
	}, nil
}

// =============================================================================
// Overview/Statistics Bridge Handlers
// =============================================================================

// bridgeGetOverviewStats handles the getOverviewStats bridge call
func (e *DashboardExtension) bridgeGetOverviewStats(ctx bridge.Context, input GetOverviewStatsInput) (*GetOverviewStatsResult, error) {
	_, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build date range
	var startDate, endDate time.Time
	if input.StartDate != nil && input.EndDate != nil {
		startDate, err = time.Parse(time.RFC3339, *input.StartDate)
		if err != nil {
			return nil, errs.BadRequest("invalid startDate format")
		}
		endDate, err = time.Parse(time.RFC3339, *input.EndDate)
		if err != nil {
			return nil, errs.BadRequest("invalid endDate format")
		}
	} else {
		days := 30
		if input.Days != nil {
			days = *input.Days
		}
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -days)
	}

	// Get analytics summary (placeholder - implement actual logic in service)
	// For now, return mock data
	stats := OverviewStatsDTO{
		TotalSent:      1234,
		TotalDelivered: 1150,
		TotalOpened:    456,
		TotalClicked:   123,
		TotalBounced:   34,
		TotalFailed:    50,
		DeliveryRate:   93.2,
		OpenRate:       39.7,
		ClickRate:      27.0,
		BounceRate:     2.8,
	}

	e.plugin.logger.Debug("overview stats retrieved",
		forge.F("appId", appID.String()),
		forge.F("startDate", startDate),
		forge.F("endDate", endDate))

	return &GetOverviewStatsResult{
		Stats: stats,
	}, nil
}

// =============================================================================
// Provider Bridge Handlers
// =============================================================================

// bridgeGetProviders handles the getProviders bridge call
func (e *DashboardExtension) bridgeGetProviders(ctx bridge.Context, input GetProvidersInput) (*GetProvidersResult, error) {
	_, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Get current provider configuration from plugin config
	cfg := e.plugin.config

	providers := ProvidersConfigDTO{
		EmailProvider: EmailProviderDTO{
			Type:      cfg.Providers.Email.Provider,
			Enabled:   true, // Email is always enabled if configured
			FromName:  cfg.Providers.Email.FromName,
			FromEmail: cfg.Providers.Email.From,
			Config:    cfg.Providers.Email.Config,
		},
		SMSProvider: SMSProviderDTO{
			Type:    "",
			Enabled: false,
			Config:  map[string]interface{}{},
		},
	}

	// Check if SMS is configured
	if cfg.Providers.SMS != nil {
		providers.SMSProvider = SMSProviderDTO{
			Type:    cfg.Providers.SMS.Provider,
			Enabled: true,
			Config:  cfg.Providers.SMS.Config,
		}
	}

	e.plugin.logger.Debug("providers config retrieved via bridge",
		forge.F("appId", appID.String()))

	return &GetProvidersResult{
		Providers: providers,
	}, nil
}

// bridgeUpdateProviders handles the updateProviders bridge call
func (e *DashboardExtension) bridgeUpdateProviders(ctx bridge.Context, input UpdateProvidersInput) (*UpdateProvidersResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Update email provider if provided
	if input.EmailProvider != nil {
		e.plugin.config.Providers.Email.Provider = input.EmailProvider.Type
		e.plugin.config.Providers.Email.FromName = input.EmailProvider.FromName
		e.plugin.config.Providers.Email.From = input.EmailProvider.FromEmail
		if input.EmailProvider.Config != nil {
			e.plugin.config.Providers.Email.Config = input.EmailProvider.Config
		}
	}

	// Update SMS provider if provided
	if input.SMSProvider != nil {
		if e.plugin.config.Providers.SMS == nil {
			e.plugin.config.Providers.SMS = &SMSProviderConfig{}
		}
		e.plugin.config.Providers.SMS.Provider = input.SMSProvider.Type
		if input.SMSProvider.Config != nil {
			e.plugin.config.Providers.SMS.Config = input.SMSProvider.Config
		}
	}

	// Build updated providers DTO
	providers := ProvidersConfigDTO{
		EmailProvider: EmailProviderDTO{
			Type:      e.plugin.config.Providers.Email.Provider,
			Enabled:   true,
			FromName:  e.plugin.config.Providers.Email.FromName,
			FromEmail: e.plugin.config.Providers.Email.From,
			Config:    e.plugin.config.Providers.Email.Config,
		},
		SMSProvider: SMSProviderDTO{
			Type:    "",
			Enabled: false,
			Config:  map[string]interface{}{},
		},
	}

	// Include SMS if configured
	if e.plugin.config.Providers.SMS != nil {
		providers.SMSProvider = SMSProviderDTO{
			Type:    e.plugin.config.Providers.SMS.Provider,
			Enabled: true,
			Config:  e.plugin.config.Providers.SMS.Config,
		}
	}

	// Log update
	userID, _ := contexts.GetUserID(goCtx)
	e.plugin.logger.Info("providers config updated via bridge",
		forge.F("appId", appID.String()),
		forge.F("userId", userID.String()))

	return &UpdateProvidersResult{
		Success:   true,
		Providers: providers,
		Message:   "Providers updated successfully",
	}, nil
}

// bridgeTestProvider handles the testProvider bridge call
func (e *DashboardExtension) bridgeTestProvider(ctx bridge.Context, input TestProviderInput) (*TestProviderResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build test notification
	sendReq := &notification.SendRequest{
		AppID:     appID,
		Recipient: input.Recipient,
		Subject:   "Test Notification",
		Body:      "This is a test notification from the notification plugin.",
	}

	if input.ProviderType == "email" {
		sendReq.Type = notification.NotificationTypeEmail
	} else if input.ProviderType == "sms" {
		sendReq.Type = notification.NotificationTypeSMS
	} else {
		return nil, errs.BadRequest("invalid provider type")
	}

	// Send test notification
	_, err = e.plugin.service.Send(goCtx, sendReq)
	if err != nil {
		e.plugin.logger.Error("provider test failed", forge.F("error", err))
		return &TestProviderResult{
			Success: false,
			Message: fmt.Sprintf("Test failed: %v", err),
		}, nil
	}

	return &TestProviderResult{
		Success: true,
		Message: "Test notification sent successfully",
	}, nil
}

// =============================================================================
// Analytics Bridge Handlers
// =============================================================================

// bridgeGetAnalytics handles the getAnalytics bridge call
func (e *DashboardExtension) bridgeGetAnalytics(ctx bridge.Context, input GetAnalyticsInput) (*GetAnalyticsResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build date range
	var startDate, endDate time.Time
	if input.StartDate != nil && input.EndDate != nil {
		startDate, err = time.Parse(time.RFC3339, *input.StartDate)
		if err != nil {
			return nil, errs.BadRequest("invalid startDate format")
		}
		endDate, err = time.Parse(time.RFC3339, *input.EndDate)
		if err != nil {
			return nil, errs.BadRequest("invalid endDate format")
		}
	} else {
		days := 30
		if input.Days != nil {
			days = *input.Days
		}
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -days)
	}

	// Create analytics service instance
	repo := e.plugin.service.GetRepository()
	analyticsService := notification.NewAnalyticsService(repo)

	// Get real analytics data from database
	appReport, err := analyticsService.GetAppAnalytics(goCtx, appID, startDate, endDate)
	if err != nil {
		e.plugin.logger.Error("failed to get app analytics", forge.F("error", err), forge.F("appId", appID.String()))
		// Return empty analytics on error rather than failing
		appReport = &notification.AppAnalyticsReport{
			AppID:          appID,
			TotalSent:      0,
			TotalDelivered: 0,
			TotalOpened:    0,
			TotalClicked:   0,
			TotalBounced:   0,
			TotalFailed:    0,
			DeliveryRate:   0,
			OpenRate:       0,
			ClickRate:      0,
			BounceRate:     0,
			StartDate:      startDate,
			EndDate:        endDate,
		}
	}

	analytics := AnalyticsDTO{
		Overview: OverviewStatsDTO{
			TotalSent:      appReport.TotalSent,
			TotalDelivered: appReport.TotalDelivered,
			TotalOpened:    appReport.TotalOpened,
			TotalClicked:   appReport.TotalClicked,
			TotalBounced:   appReport.TotalBounced,
			TotalFailed:    appReport.TotalFailed,
			DeliveryRate:   appReport.DeliveryRate,
			OpenRate:       appReport.OpenRate,
			ClickRate:      appReport.ClickRate,
			BounceRate:     appReport.BounceRate,
		},
		ByTemplate:   []TemplateAnalyticsDTO{},   // TODO: Implement template-level analytics
		ByDay:        []DailyAnalyticsDTO{},      // TODO: Implement daily breakdown
		TopTemplates: []TemplatePerformanceDTO{}, // TODO: Implement top templates
	}

	e.plugin.logger.Debug("analytics retrieved",
		forge.F("appId", appID.String()),
		forge.F("startDate", startDate),
		forge.F("endDate", endDate))

	return &GetAnalyticsResult{
		Analytics: analytics,
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// templateToDTO converts a notification.Template to TemplateDTO
func (e *DashboardExtension) templateToDTO(tmpl *notification.Template) TemplateDTO {
	return TemplateDTO{
		ID:          tmpl.ID.String(),
		AppID:       tmpl.AppID.String(),
		TemplateKey: tmpl.TemplateKey,
		Name:        tmpl.Name,
		Type:        string(tmpl.Type),
		Language:    tmpl.Language,
		Subject:     tmpl.Subject,
		Body:        tmpl.Body,
		Variables:   tmpl.Variables,
		Metadata:    tmpl.Metadata,
		Active:      tmpl.Active,
		IsDefault:   tmpl.IsDefault,
		IsModified:  tmpl.IsModified,
		CreatedAt:   tmpl.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   tmpl.UpdatedAt.Format(time.RFC3339),
	}
}

// =============================================================================
// Notification History Bridge Handlers
// =============================================================================

// bridgeListNotificationsHistory handles the listNotificationsHistory bridge call
func (e *DashboardExtension) bridgeListNotificationsHistory(ctx bridge.Context, input ListNotificationsHistoryInput) (*ListNotificationsHistoryResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Build filter
	filter := &notification.ListNotificationsFilter{
		AppID: appID,
		PaginationParams: pagination.PaginationParams{
			Page:  input.Page,
			Limit: input.Limit,
		},
	}

	if input.Type != nil {
		notifType := notification.NotificationType(*input.Type)
		filter.Type = &notifType
	}
	if input.Status != nil {
		notifStatus := notification.NotificationStatus(*input.Status)
		filter.Status = &notifStatus
	}
	if input.Recipient != nil {
		filter.Recipient = input.Recipient
	}

	// Fetch notifications
	response, err := e.plugin.service.ListNotifications(goCtx, filter)
	if err != nil {
		e.plugin.logger.Error("failed to list notifications history", forge.F("error", err))
		return nil, errs.InternalServerError("failed to list notifications", err)
	}

	// Convert to DTOs
	notifications := make([]NotificationHistoryDTO, len(response.Data))
	for i, n := range response.Data {
		notifications[i] = notificationToDTO(n)
	}

	return &ListNotificationsHistoryResult{
		Notifications: notifications,
		Pagination: PaginationDTO{
			CurrentPage: response.Pagination.CurrentPage,
			PageSize:    response.Pagination.Limit,
			TotalCount:  response.Pagination.Total,
			TotalPages:  response.Pagination.TotalPages,
			HasNext:     response.Pagination.HasNext,
		},
	}, nil
}

// bridgeGetNotificationDetail handles the getNotificationDetail bridge call
func (e *DashboardExtension) bridgeGetNotificationDetail(ctx bridge.Context, input GetNotificationDetailInput) (*GetNotificationDetailResult, error) {
	goCtx, _, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Parse notification ID
	notifID, err := xid.FromString(input.NotificationID)
	if err != nil {
		return nil, errs.BadRequest("invalid notification ID")
	}

	// Fetch notification
	notif, err := e.plugin.service.GetNotification(goCtx, notifID)
	if err != nil {
		e.plugin.logger.Error("failed to get notification detail", forge.F("error", err))
		return nil, errs.InternalServerError("failed to get notification", err)
	}

	return &GetNotificationDetailResult{
		Notification: notificationToDTO(notif),
	}, nil
}

// Helper function to convert notification to DTO
func notificationToDTO(n *notification.Notification) NotificationHistoryDTO {
	dto := NotificationHistoryDTO{
		ID:         n.ID.String(),
		AppID:      n.AppID.String(),
		Type:       string(n.Type),
		Recipient:  n.Recipient,
		Subject:    n.Subject,
		Body:       n.Body,
		Status:     string(n.Status),
		Error:      n.Error,
		ProviderID: n.ProviderID,
		Metadata:   n.Metadata,
		CreatedAt:  n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  n.UpdatedAt.Format(time.RFC3339),
	}

	if n.TemplateID != nil {
		tid := n.TemplateID.String()
		dto.TemplateID = &tid
	}
	if n.SentAt != nil {
		sentAt := n.SentAt.Format(time.RFC3339)
		dto.SentAt = &sentAt
	}
	if n.DeliveredAt != nil {
		deliveredAt := n.DeliveredAt.Format(time.RFC3339)
		dto.DeliveredAt = &deliveredAt
	}

	return dto
}

// getBridgeFunctions returns the bridge functions for registration
func (e *DashboardExtension) getBridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		// Settings
		{
			Name:        "getSettings",
			Handler:     e.bridgeGetSettings,
			Description: "Get notification plugin settings",
		},
		{
			Name:        "updateSettings",
			Handler:     e.bridgeUpdateSettings,
			Description: "Update notification plugin settings",
		},
		// Templates
		{
			Name:        "listTemplates",
			Handler:     e.bridgeListTemplates,
			Description: "List all notification templates",
		},
		{
			Name:        "getTemplate",
			Handler:     e.bridgeGetTemplate,
			Description: "Get a single notification template",
		},
		{
			Name:        "createTemplate",
			Handler:     e.bridgeCreateTemplate,
			Description: "Create a new notification template",
		},
		{
			Name:        "updateTemplate",
			Handler:     e.bridgeUpdateTemplate,
			Description: "Update a notification template",
		},
		{
			Name:        "deleteTemplate",
			Handler:     e.bridgeDeleteTemplate,
			Description: "Delete a notification template",
		},
		{
			Name:        "previewTemplate",
			Handler:     e.bridgePreviewTemplate,
			Description: "Preview a template with variables",
		},
		{
			Name:        "testSendTemplate",
			Handler:     e.bridgeTestSendTemplate,
			Description: "Send a test notification",
		},
		// Overview/Statistics
		{
			Name:        "getOverviewStats",
			Handler:     e.bridgeGetOverviewStats,
			Description: "Get overview statistics",
		},
		// Providers
		{
			Name:        "getProviders",
			Handler:     e.bridgeGetProviders,
			Description: "Get providers configuration",
		},
		{
			Name:        "updateProviders",
			Handler:     e.bridgeUpdateProviders,
			Description: "Update providers configuration",
		},
		{
			Name:        "testProvider",
			Handler:     e.bridgeTestProvider,
			Description: "Test provider configuration",
		},
		// Analytics
		{
			Name:        "getAnalytics",
			Handler:     e.bridgeGetAnalytics,
			Description: "Get detailed analytics",
		},
		// Email Builder
		{
			Name:        "saveBuilderTemplate",
			Handler:     e.bridgeSaveBuilderTemplate,
			Description: "Save a template from the visual email builder",
		},
		// Notification History
		{
			Name:        "listNotificationsHistory",
			Description: "List notification history with filtering",
			Handler:     e.bridgeListNotificationsHistory,
		},
		{
			Name:        "getNotificationDetail",
			Description: "Get a single notification detail",
			Handler:     e.bridgeGetNotificationDetail,
		},
	}
}

// =============================================================================
// Email Builder Bridge Handlers
// =============================================================================

// bridgeSaveBuilderTemplate saves a template from the visual builder
func (e *DashboardExtension) bridgeSaveBuilderTemplate(ctx bridge.Context, input SaveBuilderTemplateInput) (*SaveBuilderTemplateResult, error) {
	goCtx, appID, err := e.buildContextFromBridge(ctx)
	if err != nil {
		return nil, err
	}

	// Validate input
	if input.Name == "" {
		return nil, errs.BadRequest("template name is required")
	}
	if input.TemplateKey == "" {
		return nil, errs.BadRequest("template key is required")
	}
	if input.BuilderJSON == "" {
		return nil, errs.BadRequest("builder document is required")
	}

	// Parse builder document JSON
	doc, err := builder.FromJSON(input.BuilderJSON)
	if err != nil {
		e.plugin.logger.Error("invalid builder document", forge.F("error", err))
		return nil, errs.BadRequest("invalid builder document format")
	}

	// Validate document structure
	if err := doc.Validate(); err != nil {
		e.plugin.logger.Error("builder document validation failed", forge.F("error", err))
		return nil, errs.BadRequest("invalid builder document structure")
	}

	// Render to HTML
	renderer := builder.NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		e.plugin.logger.Error("failed to render builder template to HTML", forge.F("error", err))
		return nil, errs.InternalServerError("failed to render template", err)
	}

	// Create or update template
	var savedTemplateID string

	if input.TemplateID != "" {
		// Update existing template
		templateID, err := xid.FromString(input.TemplateID)
		if err != nil {
			return nil, errs.BadRequest("invalid templateId")
		}

		updateReq := &notification.UpdateTemplateRequest{
			Name:    &input.Name,
			Subject: &input.Subject,
			Body:    &html,
			Metadata: map[string]interface{}{
				"builderType":   "visual",
				"builderBlocks": input.BuilderJSON,
			},
		}

		err = e.plugin.service.UpdateTemplate(goCtx, templateID, updateReq)
		if err != nil {
			e.plugin.logger.Error("failed to update builder template", forge.F("error", err))
			return nil, errs.InternalServerError("failed to update template", err)
		}

		savedTemplateID = templateID.String()

		// Log update
		userID, _ := contexts.GetUserID(goCtx)
		e.plugin.logger.Info("builder template updated",
			forge.F("appId", appID.String()),
			forge.F("templateId", savedTemplateID),
			forge.F("userId", userID.String()))

		return &SaveBuilderTemplateResult{
			Success:    true,
			TemplateID: savedTemplateID,
			Message:    "Template updated successfully",
		}, nil
	}

	// Create new template
	createReq := &notification.CreateTemplateRequest{
		AppID:       appID,
		TemplateKey: input.TemplateKey,
		Name:        input.Name,
		Type:        notification.NotificationTypeEmail,
		Subject:     input.Subject,
		Body:        html,
		Metadata: map[string]interface{}{
			"builderType":   "visual",
			"builderBlocks": input.BuilderJSON,
		},
	}

	template, err := e.plugin.service.CreateTemplate(goCtx, createReq)
	if err != nil {
		e.plugin.logger.Error("failed to create builder template", forge.F("error", err))
		return nil, errs.InternalServerError("failed to create template", err)
	}

	savedTemplateID = template.ID.String()

	// Log creation
	userID, _ := contexts.GetUserID(goCtx)
	e.plugin.logger.Info("builder template created",
		forge.F("appId", appID.String()),
		forge.F("templateId", savedTemplateID),
		forge.F("userId", userID.String()))

	return &SaveBuilderTemplateResult{
		Success:    true,
		TemplateID: savedTemplateID,
		Message:    "Template created successfully",
	}, nil
}
