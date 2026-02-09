package bridge

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard/services"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// BridgeManager manages all bridge functions for the dashboard.
type BridgeManager struct {
	bridge   *bridge.Bridge
	services *services.Services
	log      forge.Logger
	basePath string

	// Service references for easy access
	userSvc    user.ServiceInterface
	sessionSvc session.ServiceInterface
	appSvc     app.Service
	orgSvc     *organization.Service
	rbacSvc    rbac.ServiceInterface
	apikeySvc  *apikey.Service
	auditSvc   *audit.Service
	envSvc     environment.EnvironmentService

	// Enabled plugins map for checking plugin status
	enabledPlugins map[string]bool

	// Extension registry for dashboard widgets
	extensionRegistry ExtensionRegistry
}

// ExtensionRegistry interface for accessing dashboard extensions.
type ExtensionRegistry interface {
	GetDashboardWidgets() []ui.DashboardWidget
}

// NewBridgeManager creates a new bridge manager with an existing bridge instance.
func NewBridgeManager(
	existingBridge *bridge.Bridge,
	services *services.Services,
	log forge.Logger,
	basePath string,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	appSvc app.Service,
	orgSvc *organization.Service,
	rbacSvc rbac.ServiceInterface,
	apikeySvc *apikey.Service,
	auditSvc *audit.Service,
	envSvc environment.EnvironmentService,
	enabledPlugins map[string]bool,
	extensionRegistry ExtensionRegistry,
) *BridgeManager {
	return &BridgeManager{
		bridge:            existingBridge,
		services:          services,
		log:               log,
		basePath:          basePath,
		userSvc:           userSvc,
		sessionSvc:        sessionSvc,
		appSvc:            appSvc,
		orgSvc:            orgSvc,
		rbacSvc:           rbacSvc,
		apikeySvc:         apikeySvc,
		auditSvc:          auditSvc,
		envSvc:            envSvc,
		enabledPlugins:    enabledPlugins,
		extensionRegistry: extensionRegistry,
	}
}

// RegisterFunctions registers all bridge functions.
func (bm *BridgeManager) RegisterFunctions() error {
	// Auth functions
	if err := bm.registerAuthFunctions(); err != nil {
		return err
	}

	// Stats functions
	if err := bm.registerStatsFunctions(); err != nil {
		return err
	}

	// User management functions
	if err := bm.registerUserFunctions(); err != nil {
		return err
	}

	// Session management functions
	if err := bm.registerSessionFunctions(); err != nil {
		return err
	}

	// Organization management functions
	if err := bm.registerOrganizationFunctions(); err != nil {
		return err
	}

	// App management functions
	if err := bm.registerAppFunctions(); err != nil {
		return err
	}

	// Settings functions
	if err := bm.registerSettingsFunctions(); err != nil {
		return err
	}

	// Advanced features functions
	if err := bm.registerAdvancedFunctions(); err != nil {
		return err
	}

	// API key functions
	if err := bm.registerAPIKeyFunctions(); err != nil {
		return err
	}

	bm.log.Info("all bridge functions registered successfully")

	return nil
}

// GetBridge returns the bridge instance.
func (bm *BridgeManager) GetBridge() *bridge.Bridge {
	return bm.bridge
}

// buildContext retrieves the Go context from the HTTP request.
// The context has already been enriched by BridgeContextMiddleware with user ID, app ID,
// and environment ID from the session and cookies.
//
// This method supports optional appID override for cases where the bridge function
// needs to specify a different app context than the session's default.
//
// Optional request-level parameters (orgId from form values) are still extracted here
// as they're function-specific and not global context.
func (bm *BridgeManager) buildContext(bridgeCtx bridge.Context, additionalAppID ...xid.ID) context.Context {
	// Get the already-enriched context from the HTTP request
	// The BridgeContextMiddleware has already set:
	// - User ID (from session)
	// - App ID (from session)
	// - Environment ID (from cookie)
	var goCtx context.Context
	if req := bridgeCtx.Request(); req != nil {
		// IMPORTANT: Get context from HTTP request, not bridgeCtx.Context()
		// because the middleware enriches the request's context
		goCtx = req.Context()
	} else {
		// Fallback to bridge context if no request available
		goCtx = bridgeCtx.Context()
	}

	// Allow override with additional appID parameter (highest priority)
	// This is useful when a bridge function needs to operate on a different app
	// than the one in the user's session
	if len(additionalAppID) > 0 && !additionalAppID[0].IsNil() {
		goCtx = contexts.SetAppID(goCtx, additionalAppID[0])
		bm.log.Debug("overriding appID from parameter", forge.F("appId", additionalAppID[0].String()))
	}

	// Extract function-specific parameters that aren't part of global context
	// These are per-request parameters, not session-level context
	if req := bridgeCtx.Request(); req != nil {
		// Organization ID is function-specific (not global context)
		if orgIDStr := req.FormValue("orgId"); orgIDStr != "" {
			if orgID, err := xid.FromString(orgIDStr); err == nil {
				goCtx = contexts.SetOrganizationID(goCtx, orgID)
				bm.log.Debug("set orgID from request", forge.F("orgId", orgID.String()))
			}
		}
	}

	return goCtx
}
