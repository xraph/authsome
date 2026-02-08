package bridge

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// OIDCServiceInterface defines the OIDC service methods needed by bridge functions
// Using interface{} for config to avoid import cycle issues
type OIDCServiceInterface interface {
	GetConfig() interface{}
	GetCurrentKeyID() (string, error)
	GetLastKeyRotation() time.Time
	RotateKeys() error
	GetDeviceFlowService() interface{}
}

// BridgeManager manages all bridge functions for the OIDC provider plugin
type BridgeManager struct {
	clientRepo     *repository.OAuthClientRepository
	tokenRepo      *repository.OAuthTokenRepository
	consentRepo    *repository.OAuthConsentRepository
	deviceCodeRepo *repository.DeviceCodeRepository
	service        OIDCServiceInterface
	logger         forge.Logger
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager(
	clientRepo *repository.OAuthClientRepository,
	tokenRepo *repository.OAuthTokenRepository,
	consentRepo *repository.OAuthConsentRepository,
	deviceCodeRepo *repository.DeviceCodeRepository,
	service OIDCServiceInterface,
	logger forge.Logger,
) *BridgeManager {
	return &BridgeManager{
		clientRepo:     clientRepo,
		tokenRepo:      tokenRepo,
		consentRepo:    consentRepo,
		deviceCodeRepo: deviceCodeRepo,
		service:        service,
		logger:         logger,
	}
}

// buildContext creates a Go context from bridge context with authentication
// This leverages the enriched context from dashboard v2 middleware
func (bm *BridgeManager) buildContext(ctx bridge.Context) (context.Context, xid.ID, xid.ID, error) {
	// Get the enriched context from the HTTP request
	goCtx := context.Background()
	if req := ctx.Request(); req != nil {
		goCtx = req.Context()
	} else {
		goCtx = ctx.Context()
	}

	// Verify authentication (UserID should be set by middleware)
	userID, hasUserID := contexts.GetUserID(goCtx)
	if !hasUserID || userID == xid.NilID() {
		return nil, xid.NilID(), xid.NilID(), errs.Unauthorized()
	}

	// Get app ID (should be set by middleware)
	appID, hasAppID := contexts.GetAppID(goCtx)
	if !hasAppID || appID == xid.NilID() {
		return nil, xid.NilID(), xid.NilID(), errs.BadRequest("missing appId")
	}

	// Get environment ID (optional, may not always be set)
	envID, _ := contexts.GetEnvironmentID(goCtx)
	if envID == xid.NilID() {
		// If no environment is set, you might want to get a default one
		// For now, we'll just use the nil ID
	}

	return goCtx, userID, appID, nil
}

// buildContextWithAppID creates a context with a specific app ID override
func (bm *BridgeManager) buildContextWithAppID(ctx bridge.Context, appIDStr string) (context.Context, xid.ID, xid.ID, error) {
	goCtx, userID, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, xid.NilID(), xid.NilID(), err
	}

	// Parse and override app ID
	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, xid.NilID(), xid.NilID(), errs.BadRequest("invalid appId")
	}

	// Set the app ID in context
	goCtx = contexts.SetAppID(goCtx, appID)

	return goCtx, userID, appID, nil
}
