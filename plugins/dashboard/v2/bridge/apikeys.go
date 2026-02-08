package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// ============================================================================
// API Key Bridge Types
// ============================================================================

// APIKeyListInput represents list API keys request
type APIKeyListInput struct {
	AppID    string `json:"appId" validate:"required"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// APIKeyListOutput represents list API keys response
type APIKeyListOutput struct {
	APIKeys    []APIKeyItem `json:"apiKeys"`
	TotalItems int          `json:"totalItems"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalPages int          `json:"totalPages"`
}

// APIKeyItem represents an API key in the list
type APIKeyItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	KeyType     string   `json:"keyType"`
	KeyPrefix   string   `json:"keyPrefix"`
	Scopes      []string `json:"scopes"`
	Active      bool     `json:"active"`
	RateLimit   int      `json:"rateLimit"`
	LastUsedAt  string   `json:"lastUsedAt,omitempty"`
	ExpiresAt   string   `json:"expiresAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	IsExpired   bool     `json:"isExpired"`
}

// APIKeyCreateInput represents create API key request
type APIKeyCreateInput struct {
	AppID       string `json:"appId" validate:"required"`
	Name        string `json:"name" validate:"required"`
	KeyType     string `json:"keyType,omitempty"` // pk, sk, rk
	Scopes      string `json:"scopes,omitempty"`  // comma-separated
	RateLimit   int    `json:"rateLimit,omitempty"`
	ExpiresIn   int    `json:"expiresIn,omitempty"` // days
	Description string `json:"description,omitempty"`
}

// APIKeyCreateOutput represents create API key response
type APIKeyCreateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Key     string `json:"key"` // The actual API key - shown only once
}

// APIKeyRotateInput represents rotate API key request
type APIKeyRotateInput struct {
	AppID string `json:"appId" validate:"required"`
	KeyID string `json:"keyId" validate:"required"`
}

// APIKeyRotateOutput represents rotate API key response
type APIKeyRotateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Key     string `json:"key"` // The new API key - shown only once
}

// APIKeyRevokeInput represents revoke API key request
type APIKeyRevokeInput struct {
	AppID string `json:"appId" validate:"required"`
	KeyID string `json:"keyId" validate:"required"`
}

// APIKeyRevokeOutput represents revoke API key response
type APIKeyRevokeOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// APIKeyDetailInput represents get API key detail request
type APIKeyDetailInput struct {
	AppID string `json:"appId" validate:"required"`
	KeyID string `json:"keyId" validate:"required"`
}

// APIKeyDetailOutput represents API key detail response
type APIKeyDetailOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	KeyType     string            `json:"keyType"`
	KeyPrefix   string            `json:"keyPrefix"`
	Scopes      []string          `json:"scopes"`
	Active      bool              `json:"active"`
	RateLimit   int               `json:"rateLimit"`
	Permissions map[string]string `json:"permissions,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	UsageCount  int64             `json:"usageCount"`
	LastUsedAt  string            `json:"lastUsedAt,omitempty"`
	ExpiresAt   string            `json:"expiresAt,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
	IsExpired   bool              `json:"isExpired"`
}

// ============================================================================
// Registration
// ============================================================================

// registerAPIKeyFunctions registers API key bridge functions
func (bm *BridgeManager) registerAPIKeyFunctions() error {
	if bm.apikeySvc == nil {
		bm.log.Warn("apikey service not available, skipping API key bridge functions")
		return nil
	}

	if err := bm.bridge.Register("getAPIKeysList", bm.getAPIKeysList,
		bridge.WithDescription("Get list of API keys for an app"),
	); err != nil {
		return fmt.Errorf("failed to register getAPIKeysList: %w", err)
	}

	if err := bm.bridge.Register("getAPIKeyDetail", bm.getAPIKeyDetail,
		bridge.WithDescription("Get API key details"),
	); err != nil {
		return fmt.Errorf("failed to register getAPIKeyDetail: %w", err)
	}

	if err := bm.bridge.Register("createAPIKey", bm.createAPIKey,
		bridge.WithDescription("Create a new API key"),
	); err != nil {
		return fmt.Errorf("failed to register createAPIKey: %w", err)
	}

	if err := bm.bridge.Register("rotateAPIKey", bm.rotateAPIKey,
		bridge.WithDescription("Rotate an API key"),
	); err != nil {
		return fmt.Errorf("failed to register rotateAPIKey: %w", err)
	}

	if err := bm.bridge.Register("revokeAPIKey", bm.revokeAPIKey,
		bridge.WithDescription("Revoke an API key"),
	); err != nil {
		return fmt.Errorf("failed to register revokeAPIKey: %w", err)
	}

	bm.log.Info("API key bridge functions registered")
	return nil
}

// ============================================================================
// Implementation
// ============================================================================

// getAPIKeysList retrieves list of API keys for an app
func (bm *BridgeManager) getAPIKeysList(ctx bridge.Context, input APIKeyListInput) (*APIKeyListOutput, error) {
	if input.AppID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId is required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId format")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Set pagination defaults
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	// Create filter with pagination
	filter := &apikey.ListAPIKeysFilter{
		AppID: appID,
		PaginationParams: pagination.PaginationParams{
			Limit: pageSize,
			Page:  page,
		},
	}

	// List API keys
	result, err := bm.apikeySvc.ListAPIKeys(goCtx, filter)
	if err != nil {
		bm.log.Error("failed to list API keys", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to list API keys")
	}

	// Convert to output format
	keys := make([]APIKeyItem, 0, len(result.Data))
	for _, key := range result.Data {
		item := APIKeyItem{
			ID:        key.ID.String(),
			Name:      key.Name,
			KeyType:   string(key.KeyType),
			KeyPrefix: key.Prefix,
			Scopes:    key.Scopes,
			Active:    key.Active,
			RateLimit: key.RateLimit,
			CreatedAt: key.CreatedAt.Format(time.RFC3339),
			IsExpired: key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()),
		}

		if key.LastUsedAt != nil {
			item.LastUsedAt = key.LastUsedAt.Format(time.RFC3339)
		}
		if key.ExpiresAt != nil {
			item.ExpiresAt = key.ExpiresAt.Format(time.RFC3339)
		}

		keys = append(keys, item)
	}

	// Get pagination metadata
	totalItems := int(result.Pagination.Total)
	totalPages := result.Pagination.TotalPages

	return &APIKeyListOutput{
		APIKeys:    keys,
		TotalItems: totalItems,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// getAPIKeyDetail retrieves API key details
func (bm *BridgeManager) getAPIKeyDetail(ctx bridge.Context, input APIKeyDetailInput) (*APIKeyDetailOutput, error) {
	if input.AppID == "" || input.KeyID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and keyId are required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId format")
	}

	keyID, err := xid.FromString(input.KeyID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid keyId format")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Get environment ID from context
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Get API key
	key, err := bm.apikeySvc.GetAPIKey(goCtx, appID, keyID, envID, nil)
	if err != nil {
		bm.log.Error("failed to get API key", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "API key not found")
	}

	output := &APIKeyDetailOutput{
		ID:          key.ID.String(),
		Name:        key.Name,
		Description: key.Description,
		KeyType:     string(key.KeyType),
		KeyPrefix:   key.Prefix,
		Scopes:      key.Scopes,
		Active:      key.Active,
		RateLimit:   key.RateLimit,
		Permissions: key.Permissions,
		Metadata:    key.Metadata,
		UsageCount:  key.UsageCount,
		CreatedAt:   key.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   key.UpdatedAt.Format(time.RFC3339),
		IsExpired:   key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()),
	}

	if key.LastUsedAt != nil {
		output.LastUsedAt = key.LastUsedAt.Format(time.RFC3339)
	}
	if key.ExpiresAt != nil {
		output.ExpiresAt = key.ExpiresAt.Format(time.RFC3339)
	}

	return output, nil
}

// createAPIKey creates a new API key
func (bm *BridgeManager) createAPIKey(ctx bridge.Context, input APIKeyCreateInput) (*APIKeyCreateOutput, error) {
	if input.AppID == "" || input.Name == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and name are required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId format")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Parse key type
	var keyType apikey.KeyType
	switch input.KeyType {
	case "pk":
		keyType = apikey.KeyTypePublishable
	case "sk":
		keyType = apikey.KeyTypeSecret
	case "rk":
		keyType = apikey.KeyTypeRestricted
	default:
		keyType = apikey.KeyTypeRestricted
	}

	// Parse scopes
	var scopes []string
	if input.Scopes != "" {
		scopes = strings.Split(strings.ReplaceAll(input.Scopes, " ", ""), ",")
	}
	if len(scopes) == 0 {
		switch keyType {
		case apikey.KeyTypePublishable:
			scopes = []string{"app:identify", "sessions:create", "users:verify"}
		case apikey.KeyTypeSecret:
			scopes = []string{"admin:full"}
		default:
			scopes = []string{"read"}
		}
	}

	// Parse rate limit
	rateLimit := 1000 // default
	if input.RateLimit > 0 {
		rateLimit = input.RateLimit
	}

	// Parse expiry
	var expiresAt *time.Time
	if input.ExpiresIn > 0 {
		expiry := time.Now().AddDate(0, 0, input.ExpiresIn)
		expiresAt = &expiry
	}

	// Get user ID and environment ID from context
	userID, _ := contexts.GetUserID(goCtx)
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Create API key
	req := &apikey.CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envID,
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		KeyType:       keyType,
		Scopes:        scopes,
		RateLimit:     rateLimit,
		ExpiresAt:     expiresAt,
		Permissions:   make(map[string]string),
		Metadata:      make(map[string]string),
	}

	key, err := bm.apikeySvc.CreateAPIKey(goCtx, req)
	if err != nil {
		bm.log.Error("failed to create API key", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to create API key: "+err.Error())
	}

	return &APIKeyCreateOutput{
		Success: true,
		ID:      key.ID.String(),
		Name:    key.Name,
		Key:     key.Key, // The full key - shown only once
	}, nil
}

// rotateAPIKey rotates an API key
func (bm *BridgeManager) rotateAPIKey(ctx bridge.Context, input APIKeyRotateInput) (*APIKeyRotateOutput, error) {
	if input.AppID == "" || input.KeyID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and keyId are required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId format")
	}

	keyID, err := xid.FromString(input.KeyID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid keyId format")
	}

	goCtx := bm.buildContext(ctx, appID)

	// Get user ID and environment ID from context
	userID, _ := contexts.GetUserID(goCtx)
	envID, _ := contexts.GetEnvironmentID(goCtx)

	req := &apikey.RotateAPIKeyRequest{
		ID:            keyID,
		AppID:         appID,
		EnvironmentID: envID,
		UserID:        userID,
	}

	key, err := bm.apikeySvc.RotateAPIKey(goCtx, req)
	if err != nil {
		bm.log.Error("failed to rotate API key", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to rotate API key: "+err.Error())
	}

	return &APIKeyRotateOutput{
		Success: true,
		ID:      key.ID.String(),
		Key:     key.Key, // The new key - shown only once
	}, nil
}

// revokeAPIKey revokes an API key
func (bm *BridgeManager) revokeAPIKey(ctx bridge.Context, input APIKeyRevokeInput) (*APIKeyRevokeOutput, error) {
	if input.AppID == "" || input.KeyID == "" {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "appId and keyId are required")
	}

	appID, err := xid.FromString(input.AppID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid appId format")
	}

	keyID, err := xid.FromString(input.KeyID)
	if err != nil {
		return nil, bridge.NewError(bridge.ErrCodeBadRequest, "invalid keyId format")
	}

	goCtx := bm.buildContext(ctx, appID)

	err = bm.apikeySvc.DeleteAPIKey(goCtx, appID, keyID, xid.NilID(), nil)
	if err != nil {
		bm.log.Error("failed to revoke API key", forge.F("error", err.Error()))
		return nil, bridge.NewError(bridge.ErrCodeInternal, "failed to revoke API key: "+err.Error())
	}

	return &APIKeyRevokeOutput{
		Success: true,
		Message: "API key revoked successfully",
	}, nil
}
