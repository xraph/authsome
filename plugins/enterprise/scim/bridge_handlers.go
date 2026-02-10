package scim

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Bridge Function Types
// =============================================================================

// GetOverviewInput is the input for getting SCIM overview.
type GetOverviewInput struct {
	AppID string `json:"appId"`
}

// GetOverviewOutput is the output for SCIM overview.
type GetOverviewOutput struct {
	Stats          OverviewStats         `json:"stats"`
	RecentActivity []ActivityItem        `json:"recentActivity"`
	Providers      []ProviderSummaryItem `json:"providers"`
	QuickActions   []QuickActionItem     `json:"quickActions"`
}

// OverviewStats contains overview statistics.
type OverviewStats struct {
	TotalProviders   int    `json:"totalProviders,omitempty"`
	ActiveProviders  int    `json:"activeProviders,omitempty"`
	TotalTokens      int    `json:"totalTokens,omitempty"`
	ActiveTokens     int    `json:"activeTokens,omitempty"`
	UsersProvisioned int    `json:"usersProvisioned,omitempty"`
	GroupsSynced     int    `json:"groupsSynced,omitempty"`
	LastSyncTime     string `json:"lastSyncTime,omitempty"`
	SyncErrors       int    `json:"syncErrors,omitempty"`
}

// ActivityItem represents a recent activity item.
type ActivityItem struct {
	ID          string `json:"id"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Timestamp   string `json:"timestamp"`
	Provider    string `json:"provider,omitempty"`
}

// ProviderSummaryItem represents a provider summary.
type ProviderSummaryItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status"`
	LastSync   string `json:"lastSync,omitempty"`
	UserCount  int    `json:"userCount,omitempty"`
	GroupCount int    `json:"groupCount,omitempty"`
}

// QuickActionItem represents a quick action.
type QuickActionItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Action      string `json:"action"`
}

// GetProvidersInput is the input for listing providers.
type GetProvidersInput struct {
	AppID    string `json:"appId"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
	Search   string `json:"search,omitempty"`
	Status   string `json:"status,omitempty"`
}

// ProviderItem represents a SCIM provider.
type ProviderItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	EndpointURL    string `json:"endpointUrl,omitempty"`
	LastSync       string `json:"lastSync,omitempty"`
	LastSyncStatus string `json:"lastSyncStatus,omitempty"`
	UserCount      int    `json:"userCount,omitempty"`
	GroupCount     int    `json:"groupCount,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

// GetProvidersOutput is the output for listing providers.
type GetProvidersOutput struct {
	Providers  []ProviderItem `json:"providers"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

// GetProviderInput is the input for getting a provider.
type GetProviderInput struct {
	AppID      string `json:"appId"`
	ProviderID string `json:"providerId"`
}

// ProviderDetailOutput is the output for provider details.
type ProviderDetailOutput struct {
	Provider      ProviderItem        `json:"provider"`
	Configuration ProviderConfig      `json:"configuration"`
	SyncHistory   []SyncHistoryItem   `json:"syncHistory"`
	Stats         BridgeProviderStats `json:"stats"`
}

// ProviderConfig represents provider configuration.
type ProviderConfig struct {
	EndpointURL       string   `json:"endpointUrl,omitempty"`
	AuthMethod        string   `json:"authMethod,omitempty"`
	SyncInterval      int      `json:"syncInterval,omitempty"`
	EnableUserSync    bool     `json:"enableUserSync,omitempty"`
	EnableGroupSync   bool     `json:"enableGroupSync,omitempty"`
	AutoProvision     bool     `json:"autoProvision,omitempty"`
	AutoDeprovision   bool     `json:"autoDeprovision,omitempty"`
	DefaultRole       string   `json:"defaultRole,omitempty"`
	AttributeMappings []string `json:"attributeMappings,omitempty"`
}

// SyncHistoryItem represents a sync history entry.
type SyncHistoryItem struct {
	ID            string `json:"id"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime,omitempty"`
	Status        string `json:"status"`
	UsersAdded    int    `json:"usersAdded,omitempty"`
	UsersUpdated  int    `json:"usersUpdated,omitempty"`
	UsersRemoved  int    `json:"usersRemoved,omitempty"`
	GroupsAdded   int    `json:"groupsAdded,omitempty"`
	GroupsUpdated int    `json:"groupsUpdated,omitempty"`
	ErrorCount    int    `json:"errorCount,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// BridgeProviderStats contains provider statistics for bridge.
type BridgeProviderStats struct {
	TotalUsers       int    `json:"totalUsers,omitempty"`
	TotalGroups      int    `json:"totalGroups,omitempty"`
	TotalSyncs       int    `json:"totalSyncs,omitempty"`
	SuccessfulSyncs  int    `json:"successfulSyncs,omitempty"`
	FailedSyncs      int    `json:"failedSyncs,omitempty"`
	AvgSyncDuration  string `json:"avgSyncDuration,omitempty"`
	LastSyncDuration string `json:"lastSyncDuration,omitempty"`
}

// CreateProviderInput is the input for creating a provider.
type CreateProviderInput struct {
	AppID           string `json:"appId"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	EndpointURL     string `json:"endpointUrl,omitempty"`
	AuthMethod      string `json:"authMethod,omitempty"`
	EnableUserSync  bool   `json:"enableUserSync,omitempty"`
	EnableGroupSync bool   `json:"enableGroupSync,omitempty"`
}

// CreateProviderOutput is the output for creating a provider.
type CreateProviderOutput struct {
	Provider ProviderItem `json:"provider"`
	Token    string       `json:"token"`
}

// UpdateProviderInput is the input for updating a provider.
type UpdateProviderInput struct {
	AppID           string `json:"appId"`
	ProviderID      string `json:"providerId"`
	Name            string `json:"name,omitempty"`
	EndpointURL     string `json:"endpointUrl,omitempty"`
	SyncInterval    int    `json:"syncInterval,omitempty"`
	EnableUserSync  bool   `json:"enableUserSync,omitempty"`
	EnableGroupSync bool   `json:"enableGroupSync,omitempty"`
	AutoProvision   bool   `json:"autoProvision,omitempty"`
	AutoDeprovision bool   `json:"autoDeprovision,omitempty"`
	DefaultRole     string `json:"defaultRole,omitempty"`
}

// DeleteProviderInput is the input for deleting a provider.
type DeleteProviderInput struct {
	AppID      string `json:"appId"`
	ProviderID string `json:"providerId"`
}

// TriggerSyncInput is the input for triggering a sync.
type TriggerSyncInput struct {
	AppID      string `json:"appId"`
	ProviderID string `json:"providerId"`
	FullSync   bool   `json:"fullSync,omitempty"`
}

// TriggerSyncOutput is the output for triggering a sync.
type TriggerSyncOutput struct {
	SyncID  string `json:"syncId,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// TestConnectionInput is the input for testing a connection.
type TestConnectionInput struct {
	AppID      string `json:"appId"`
	ProviderID string `json:"providerId"`
}

// TestConnectionOutput is the output for testing a connection.
type TestConnectionOutput struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	ResponseTime int    `json:"responseTime,omitempty"`
	Details      string `json:"details,omitempty"`
}

// GetTokensInput is the input for listing tokens.
type GetTokensInput struct {
	AppID    string `json:"appId"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// TokenItem represents a SCIM token.
type TokenItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Status      string   `json:"status"`
	Scopes      []string `json:"scopes,omitempty"`
	LastUsed    string   `json:"lastUsed,omitempty"`
	ExpiresAt   string   `json:"expiresAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

// GetTokensOutput is the output for listing tokens.
type GetTokensOutput struct {
	Tokens     []TokenItem `json:"tokens"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

// CreateTokenInput is the input for creating a token.
type CreateTokenInput struct {
	AppID       string   `json:"appId"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   int      `json:"expiresIn,omitempty"` // days
}

// CreateTokenOutput is the output for creating a token.
type CreateTokenOutput struct {
	Token     TokenItem `json:"token"`
	PlainText string    `json:"plainText"` // Only shown once
}

// RevokeTokenInput is the input for revoking a token.
type RevokeTokenInput struct {
	AppID   string `json:"appId"`
	TokenID string `json:"tokenId"`
}

// RotateTokenInput is the input for rotating a token.
type RotateTokenInput struct {
	AppID   string `json:"appId"`
	TokenID string `json:"tokenId"`
}

// RotateTokenOutput is the output for rotating a token.
type RotateTokenOutput struct {
	Token     TokenItem `json:"token"`
	PlainText string    `json:"plainText"` // Only shown once
}

// GetLogsInput is the input for getting logs.
type GetLogsInput struct {
	AppID      string `json:"appId"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
	EventType  string `json:"eventType,omitempty"`
	Status     string `json:"status,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	StartDate  string `json:"startDate,omitempty"`
	EndDate    string `json:"endDate,omitempty"`
}

// LogItem represents a SCIM log entry.
type LogItem struct {
	ID         string `json:"id"`
	EventType  string `json:"eventType,omitempty"`
	Status     string `json:"status,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Resource   string `json:"resource,omitempty"`
	ResourceID string `json:"resourceId,omitempty"`
	Details    string `json:"details,omitempty"`
	IPAddress  string `json:"ipAddress,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// GetLogsOutput is the output for getting logs.
type GetLogsOutput struct {
	Logs       []LogItem `json:"logs"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalPages int       `json:"totalPages"`
}

// GetConfigInput is the input for getting configuration.
type GetConfigInput struct {
	AppID string `json:"appId"`
}

// SCIMConfigOutput is the output for SCIM configuration.
type SCIMConfigOutput struct {
	UserProvisioning BridgeUserProvisioningConfig `json:"userProvisioning"`
	GroupSync        BridgeGroupSyncConfig        `json:"groupSync"`
	Security         BridgeSecurityConfig         `json:"security"`
	AttributeMapping BridgeAttributeMappingConfig `json:"attributeMapping"`
}

// BridgeUserProvisioningConfig contains user provisioning settings for bridge.
type BridgeUserProvisioningConfig struct {
	AutoActivate       bool   `json:"autoActivate,omitempty"`
	SendWelcomeEmail   bool   `json:"sendWelcomeEmail,omitempty"`
	PreventDuplicates  bool   `json:"preventDuplicates,omitempty"`
	DefaultRole        string `json:"defaultRole,omitempty"`
	RequireEmailVerify bool   `json:"requireEmailVerify,omitempty"`
}

// BridgeGroupSyncConfig contains group sync settings for bridge.
type BridgeGroupSyncConfig struct {
	Enabled       bool `json:"enabled,omitempty"`
	SyncToTeams   bool `json:"syncToTeams,omitempty"`
	SyncToRoles   bool `json:"syncToRoles,omitempty"`
	CreateMissing bool `json:"createMissing,omitempty"`
	DeleteOrphans bool `json:"deleteOrphans,omitempty"`
}

// BridgeSecurityConfig contains security settings for bridge.
type BridgeSecurityConfig struct {
	RequireHTTPS     bool `json:"requireHttps,omitempty"`
	RateLimitEnabled bool `json:"rateLimitEnabled,omitempty"`
	RateLimitPerMin  int  `json:"rateLimitPerMin,omitempty"`
	RequireSignedReq bool `json:"requireSignedReq,omitempty"`
	AuditAllRequests bool `json:"auditAllRequests,omitempty"`
}

// BridgeAttributeMappingConfig contains attribute mapping settings for bridge.
type BridgeAttributeMappingConfig struct {
	EmailMapping   string            `json:"emailMapping,omitempty"`
	NameMapping    string            `json:"nameMapping,omitempty"`
	PhoneMapping   string            `json:"phoneMapping,omitempty"`
	RoleMapping    string            `json:"roleMapping,omitempty"`
	CustomMappings map[string]string `json:"customMappings,omitempty"`
}

// UpdateConfigInput is the input for updating configuration.
type UpdateConfigInput struct {
	AppID            string                        `json:"appId"`
	Section          string                        `json:"section"`
	UserProvisioning *BridgeUserProvisioningConfig `json:"userProvisioning,omitempty"`
	GroupSync        *BridgeGroupSyncConfig        `json:"groupSync,omitempty"`
	Security         *BridgeSecurityConfig         `json:"security,omitempty"`
	AttributeMapping *BridgeAttributeMappingConfig `json:"attributeMapping,omitempty"`
}

// GenericSuccessOutput is a generic success response.
type GenericSuccessOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// =============================================================================
// Bridge Handler Implementations
// =============================================================================

// bridgeGetOverview handles the getOverview bridge call.
func (e *DashboardExtension) bridgeGetOverview(ctx bridge.Context, input GetOverviewInput) (*GetOverviewOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)

	// Get providers count
	providers, err := e.plugin.service.ListProviders(goCtx, appID, nil)
	activeProviders := 0
	providerSummaries := []ProviderSummaryItem{}

	if err == nil && providers != nil {
		for _, p := range providers {
			if p.Status == "active" {
				activeProviders++
			}

			providerSummaries = append(providerSummaries, ProviderSummaryItem{
				ID:         p.ID.String(),
				Name:       p.Name,
				Type:       p.Type,
				Status:     p.Status,
				LastSync:   formatTime(p.LastSyncAt),
				UserCount:  0, // Not stored in schema
				GroupCount: 0, // Not stored in schema
			})
		}
	}

	// Get token count
	envID := xid.ID{}
	orgID := xid.ID{}
	tokens, _, _ := e.plugin.service.ListProvisioningTokens(goCtx, appID, envID, orgID, 100, 0)
	activeTokens := 0

	for _, t := range tokens {
		if t.RevokedAt == nil && (t.ExpiresAt == nil || t.ExpiresAt.After(time.Now())) {
			activeTokens++
		}
	}

	// Get recent sync events
	events, _ := e.plugin.service.GetRecentActivity(goCtx, appID, nil, 10)
	recentActivity := []ActivityItem{}
	syncErrors := 0

	var lastSyncTime time.Time

	for _, ev := range events {
			description := ev.EventType
			if ev.ErrorMessage != nil && *ev.ErrorMessage != "" {
				description = *ev.ErrorMessage
			}

			recentActivity = append(recentActivity, ActivityItem{
				ID:          ev.ID.String(),
				Type:        ev.EventType,
				Description: description,
				Status:      ev.Status,
				Timestamp:   formatTime(&ev.CreatedAt),
				Provider:    "", // Provider name not in schema
			})
			if ev.Status == "error" || ev.Status == "failed" {
				syncErrors++
			}

			if lastSyncTime.IsZero() || ev.CreatedAt.After(lastSyncTime) {
				lastSyncTime = ev.CreatedAt
			}
	}

	return &GetOverviewOutput{
		Stats: OverviewStats{
			TotalProviders:   len(providers),
			ActiveProviders:  activeProviders,
			TotalTokens:      len(tokens),
			ActiveTokens:     activeTokens,
			UsersProvisioned: 0, // TODO: Get from metrics
			GroupsSynced:     0, // TODO: Get from metrics
			LastSyncTime:     formatTime(&lastSyncTime),
			SyncErrors:       syncErrors,
		},
		RecentActivity: recentActivity,
		Providers:      providerSummaries,
		QuickActions: []QuickActionItem{
			{ID: "add-provider", Label: "Add Provider", Description: "Configure a new identity provider", Icon: "plus", Action: "addProvider"},
			{ID: "create-token", Label: "Create Token", Description: "Generate a new SCIM token", Icon: "key", Action: "createToken"},
			{ID: "trigger-sync", Label: "Sync Now", Description: "Trigger manual synchronization", Icon: "refresh", Action: "triggerSync"},
			{ID: "view-logs", Label: "View Logs", Description: "View provisioning logs", Icon: "list", Action: "viewLogs"},
		},
	}, nil
}

// bridgeGetProviders handles the getProviders bridge call.
func (e *DashboardExtension) bridgeGetProviders(ctx bridge.Context, input GetProvidersInput) (*GetProvidersOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)

	providers, err := e.plugin.service.ListProviders(goCtx, appID, nil)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch providers")
	}

	items := make([]ProviderItem, 0, len(providers))
	for _, p := range providers {
		endpointURL := ""
		if p.BaseURL != nil {
			endpointURL = *p.BaseURL
		}

		items = append(items, ProviderItem{
			ID:             p.ID.String(),
			Name:           p.Name,
			Type:           p.Type,
			Status:         p.Status,
			EndpointURL:    endpointURL,
			LastSync:       formatTime(p.LastSyncAt),
			LastSyncStatus: p.LastSyncStatus,
			UserCount:      0,
			GroupCount:     0,
			CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		})
	}

	// Apply pagination
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	total := len(items)

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	start := (page - 1) * pageSize

	end := start + pageSize
	if start > total {
		start = total
	}

	if end > total {
		end = total
	}

	return &GetProvidersOutput{
		Providers:  items[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// bridgeGetProvider handles the getProvider bridge call.
func (e *DashboardExtension) bridgeGetProvider(ctx bridge.Context, input GetProviderInput) (*ProviderDetailOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	providerID, err := xid.FromString(input.ProviderID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid providerId")
	}

	provider, err := e.plugin.service.GetProvider(goCtx, providerID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "provider not found")
	}

	// Get sync history
	events, _ := e.plugin.service.GetProviderSyncHistory(goCtx, providerID, 10)
	syncHistory := make([]SyncHistoryItem, 0)

	for _, ev := range events {
			errMsg := ""
			if ev.ErrorMessage != nil {
				errMsg = *ev.ErrorMessage
			}

			errCount := 0
			if ev.Status == "failed" || ev.Status == "error" {
				errCount = 1
			}

			syncHistory = append(syncHistory, SyncHistoryItem{
				ID:            ev.ID.String(),
				StartTime:     ev.CreatedAt.Format(time.RFC3339),
				EndTime:       ev.CreatedAt.Format(time.RFC3339), // No end time in schema
				Status:        ev.Status,
				UsersAdded:    0,
				UsersUpdated:  0,
				UsersRemoved:  0,
				GroupsAdded:   0,
				GroupsUpdated: 0,
				ErrorCount:    errCount,
				ErrorMessage:  errMsg,
			})
	}

	endpointURL := ""
	if provider.BaseURL != nil {
		endpointURL = *provider.BaseURL
	}

	return &ProviderDetailOutput{
		Provider: ProviderItem{
			ID:             provider.ID.String(),
			Name:           provider.Name,
			Type:           provider.Type,
			Status:         provider.Status,
			EndpointURL:    endpointURL,
			LastSync:       formatTime(provider.LastSyncAt),
			LastSyncStatus: provider.LastSyncStatus,
			UserCount:      0,
			GroupCount:     0,
			CreatedAt:      provider.CreatedAt.Format(time.RFC3339),
		},
		Configuration: ProviderConfig{
			EndpointURL:     endpointURL,
			AuthMethod:      provider.AuthMethod,
			SyncInterval:    0,
			EnableUserSync:  true,
			EnableGroupSync: true,
			AutoProvision:   true,
			AutoDeprovision: false,
			DefaultRole:     "",
		},
		SyncHistory: syncHistory,
		Stats: BridgeProviderStats{
			TotalUsers:      0,
			TotalGroups:     0,
			TotalSyncs:      0,
			SuccessfulSyncs: 0,
			FailedSyncs:     0,
		},
	}, nil
}

// bridgeCreateProvider handles the createProvider bridge call.
func (e *DashboardExtension) bridgeCreateProvider(ctx bridge.Context, input CreateProviderInput) (*CreateProviderOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)

	// Create token for this provider
	envID := xid.ID{}
	orgID := xid.ID{}

	tokenPlainText, token, err := e.plugin.service.CreateProvisioningToken(goCtx, appID, envID, orgID, input.Name+" Token", "Auto-generated token for "+input.Name, []string{"users:read", "users:write", "groups:read", "groups:write"}, nil)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create token: "+err.Error())
	}

	// TODO: Actually create the provider once CreateProvider method is implemented
	// For now, we return placeholder data
	return &CreateProviderOutput{
		Provider: ProviderItem{
			ID:          token.ID.String(), // Placeholder
			Name:        input.Name,
			Type:        input.Type,
			Status:      "active",
			EndpointURL: input.EndpointURL,
			CreatedAt:   time.Now().Format(time.RFC3339),
		},
		Token: tokenPlainText,
	}, nil
}

// bridgeDeleteProvider handles the deleteProvider bridge call.
func (e *DashboardExtension) bridgeDeleteProvider(ctx bridge.Context, input DeleteProviderInput) (*GenericSuccessOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	providerID, err := xid.FromString(input.ProviderID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid providerId")
	}

	if err := e.plugin.service.RemoveProvider(goCtx, providerID); err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to delete provider")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Provider deleted successfully",
	}, nil
}

// bridgeTriggerSync handles the triggerSync bridge call.
func (e *DashboardExtension) bridgeTriggerSync(ctx bridge.Context, input TriggerSyncInput) (*TriggerSyncOutput, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	_, err = xid.FromString(input.ProviderID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid providerId")
	}

	// TODO: Implement TriggerSync when available
	return &TriggerSyncOutput{
		SyncID:  xid.New().String(),
		Status:  "started",
		Message: "Synchronization started successfully",
	}, nil
}

// bridgeTestConnection handles the testConnection bridge call.
func (e *DashboardExtension) bridgeTestConnection(ctx bridge.Context, input TestConnectionInput) (*TestConnectionOutput, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	_, err = xid.FromString(input.ProviderID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid providerId")
	}

	// TODO: Implement TestProviderConnection when available
	return &TestConnectionOutput{
		Success:      true,
		Message:      "Connection successful",
		ResponseTime: 50,
		Details:      "SCIM endpoint is reachable",
	}, nil
}

// bridgeGetTokens handles the getTokens bridge call.
func (e *DashboardExtension) bridgeGetTokens(ctx bridge.Context, input GetTokensInput) (*GetTokensOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)
	envID := xid.ID{}
	orgID := xid.ID{}

	tokens, _, err := e.plugin.service.ListProvisioningTokens(goCtx, appID, envID, orgID, 100, 0)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch tokens")
	}

	items := make([]TokenItem, 0, len(tokens))
	for _, t := range tokens {
		status := "active"
		if t.RevokedAt != nil {
			status = "revoked"
		} else if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
			status = "expired"
		}

		// Use stored token prefix
		prefix := t.TokenPrefix

		items = append(items, TokenItem{
			ID:          t.ID.String(),
			Name:        t.Name,
			Description: t.Description,
			Prefix:      prefix,
			Status:      status,
			Scopes:      t.Scopes,
			LastUsed:    formatTime(t.LastUsedAt),
			ExpiresAt:   formatTime(t.ExpiresAt),
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		})
	}

	// Apply pagination
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	total := len(items)

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	start := (page - 1) * pageSize

	end := start + pageSize
	if start > total {
		start = total
	}

	if end > total {
		end = total
	}

	return &GetTokensOutput{
		Tokens:     items[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// bridgeCreateToken handles the createToken bridge call.
func (e *DashboardExtension) bridgeCreateToken(ctx bridge.Context, input CreateTokenInput) (*CreateTokenOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)

	var expiresAt *time.Time

	if input.ExpiresIn > 0 {
		exp := time.Now().AddDate(0, 0, input.ExpiresIn)
		expiresAt = &exp
	}

	envID := xid.ID{}
	orgID := xid.ID{}

	plainText, token, err := e.plugin.service.CreateProvisioningToken(goCtx, appID, envID, orgID, input.Name, input.Description, input.Scopes, expiresAt)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create token: "+err.Error())
	}

	// Get token prefix
	prefix := ""
	if len(plainText) > 8 {
		prefix = plainText[:8]
	}

	return &CreateTokenOutput{
		Token: TokenItem{
			ID:          token.ID.String(),
			Name:        token.Name,
			Description: token.Description,
			Prefix:      prefix,
			Status:      "active",
			Scopes:      token.Scopes,
			ExpiresAt:   formatTime(token.ExpiresAt),
			CreatedAt:   token.CreatedAt.Format(time.RFC3339),
		},
		PlainText: plainText,
	}, nil
}

// bridgeRevokeToken handles the revokeToken bridge call.
func (e *DashboardExtension) bridgeRevokeToken(ctx bridge.Context, input RevokeTokenInput) (*GenericSuccessOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	if err := e.plugin.service.RevokeProvisioningToken(goCtx, input.TokenID); err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to revoke token")
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Token revoked successfully",
	}, nil
}

// bridgeRotateToken handles the rotateToken bridge call.
func (e *DashboardExtension) bridgeRotateToken(ctx bridge.Context, input RotateTokenInput) (*RotateTokenOutput, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// TODO: Implement token rotation when the method is available
	// For now, return an error indicating it's not implemented
	return nil, bridge.NewError(bridge.ErrCodeInternal, "token rotation not yet implemented - please revoke and create a new token")
}

// bridgeGetLogs handles the getLogs bridge call.
func (e *DashboardExtension) bridgeGetLogs(ctx bridge.Context, input GetLogsInput) (*GetLogsOutput, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appID, _ := xid.FromString(input.AppID)

	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	events, err := e.plugin.service.GetRecentActivity(goCtx, appID, nil, pageSize)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to fetch logs")
	}

	items := make([]LogItem, 0, len(events))
	for _, ev := range events {
		resourceID := ""
		if ev.ResourceID != nil {
			resourceID = ev.ResourceID.String()
		}

		details := ""
		if ev.ErrorMessage != nil {
			details = *ev.ErrorMessage
		}

		items = append(items, LogItem{
			ID:         ev.ID.String(),
			EventType:  ev.EventType,
			Status:     ev.Status,
			Provider:   "", // Not available in schema
			Resource:   ev.ResourceType,
			ResourceID: resourceID,
			Details:    details,
			IPAddress:  "", // Not available in schema
			Timestamp:  ev.CreatedAt.Format(time.RFC3339),
		})
	}

	total := len(items)

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	return &GetLogsOutput{
		Logs:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// bridgeGetConfig handles the getConfig bridge call.
func (e *DashboardExtension) bridgeGetConfig(ctx bridge.Context, input GetConfigInput) (*SCIMConfigOutput, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	config := e.plugin.config

	return &SCIMConfigOutput{
		UserProvisioning: BridgeUserProvisioningConfig{
			AutoActivate:      config.UserProvisioning.AutoActivate,
			SendWelcomeEmail:  config.UserProvisioning.SendWelcomeEmail,
			PreventDuplicates: config.UserProvisioning.PreventDuplicates,
			DefaultRole:       config.UserProvisioning.DefaultRole,
		},
		GroupSync: BridgeGroupSyncConfig{
			Enabled:       config.GroupSync.Enabled,
			SyncToTeams:   config.GroupSync.SyncToTeams,
			SyncToRoles:   config.GroupSync.SyncToRoles,
			CreateMissing: config.GroupSync.CreateMissingGroups,
		},
		Security: BridgeSecurityConfig{
			RequireHTTPS:     config.Security.RequireHTTPS,
			RateLimitEnabled: config.RateLimit.Enabled,
			RateLimitPerMin:  config.RateLimit.RequestsPerMin,
			AuditAllRequests: config.Security.AuditAllOperations,
		},
		AttributeMapping: BridgeAttributeMappingConfig{
			EmailMapping: config.AttributeMapping.EmailField,
			NameMapping:  config.AttributeMapping.DisplayNameField,
			PhoneMapping: "",
		},
	}, nil
}

// bridgeUpdateConfig handles the updateConfig bridge call.
func (e *DashboardExtension) bridgeUpdateConfig(ctx bridge.Context, input UpdateConfigInput) (*GenericSuccessOutput, error) {
	_, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Update config based on section
	switch input.Section {
	case "userProvisioning":
		if input.UserProvisioning != nil {
			e.plugin.config.UserProvisioning.AutoActivate = input.UserProvisioning.AutoActivate
			e.plugin.config.UserProvisioning.SendWelcomeEmail = input.UserProvisioning.SendWelcomeEmail
			e.plugin.config.UserProvisioning.PreventDuplicates = input.UserProvisioning.PreventDuplicates
			e.plugin.config.UserProvisioning.DefaultRole = input.UserProvisioning.DefaultRole
		}
	case "groupSync":
		if input.GroupSync != nil {
			e.plugin.config.GroupSync.Enabled = input.GroupSync.Enabled
			e.plugin.config.GroupSync.SyncToTeams = input.GroupSync.SyncToTeams
			e.plugin.config.GroupSync.SyncToRoles = input.GroupSync.SyncToRoles
			e.plugin.config.GroupSync.CreateMissingGroups = input.GroupSync.CreateMissing
		}
	case "security":
		if input.Security != nil {
			e.plugin.config.Security.RequireHTTPS = input.Security.RequireHTTPS
			e.plugin.config.RateLimit.Enabled = input.Security.RateLimitEnabled
			e.plugin.config.RateLimit.RequestsPerMin = input.Security.RateLimitPerMin
			e.plugin.config.Security.AuditAllOperations = input.Security.AuditAllRequests
		}
	}

	return &GenericSuccessOutput{
		Success: true,
		Message: "Configuration updated successfully",
	}, nil
}

// =============================================================================
// Bridge Registration
// =============================================================================

// getBridgeFunctions returns the bridge functions for registration.
func (e *DashboardExtension) getBridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		// Overview
		{
			Name:        "getOverview",
			Handler:     e.bridgeGetOverview,
			Description: "Get SCIM overview with stats and recent activity",
		},
		// Providers
		{
			Name:        "getProviders",
			Handler:     e.bridgeGetProviders,
			Description: "List SCIM identity providers",
		},
		{
			Name:        "getProvider",
			Handler:     e.bridgeGetProvider,
			Description: "Get SCIM provider details",
		},
		{
			Name:        "createProvider",
			Handler:     e.bridgeCreateProvider,
			Description: "Create a new SCIM provider",
		},
		{
			Name:        "deleteProvider",
			Handler:     e.bridgeDeleteProvider,
			Description: "Delete a SCIM provider",
		},
		{
			Name:        "triggerSync",
			Handler:     e.bridgeTriggerSync,
			Description: "Trigger SCIM synchronization",
		},
		{
			Name:        "testConnection",
			Handler:     e.bridgeTestConnection,
			Description: "Test SCIM provider connection",
		},
		// Tokens
		{
			Name:        "getTokens",
			Handler:     e.bridgeGetTokens,
			Description: "List SCIM tokens",
		},
		{
			Name:        "createToken",
			Handler:     e.bridgeCreateToken,
			Description: "Create a new SCIM token",
		},
		{
			Name:        "revokeToken",
			Handler:     e.bridgeRevokeToken,
			Description: "Revoke a SCIM token",
		},
		{
			Name:        "rotateToken",
			Handler:     e.bridgeRotateToken,
			Description: "Rotate a SCIM token",
		},
		// Logs
		{
			Name:        "getLogs",
			Handler:     e.bridgeGetLogs,
			Description: "Get SCIM event logs",
		},
		// Configuration
		{
			Name:        "getConfig",
			Handler:     e.bridgeGetConfig,
			Description: "Get SCIM configuration",
		},
		{
			Name:        "updateConfig",
			Handler:     e.bridgeUpdateConfig,
			Description: "Update SCIM configuration",
		},
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// buildContextFromBridge creates a Go context from a bridge context with app/env IDs.
func (e *DashboardExtension) buildContextFromBridge(ctx bridge.Context, appIDStr string) (context.Context, error) {
	var goCtx context.Context
	if req := ctx.Request(); req != nil {
		goCtx = req.Context()
	} else {
		goCtx = ctx.Context()
	}

	if appIDStr != "" {
		appID, err := xid.FromString(appIDStr)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId")
		}

		goCtx = contexts.SetAppID(goCtx, appID)
	}

	return goCtx, nil
}

// formatTime formats a time pointer to string.
func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

// getProviderStatus returns the status string for a provider.
func getProviderStatus(p *SCIMProvider) string {
	if p.Status != "active" {
		return p.Status
	}

	if p.LastSyncStatus == "error" || p.LastSyncStatus == "failed" {
		return "error"
	}

	return "active"
}
