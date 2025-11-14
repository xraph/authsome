package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/schema"
)

// Config holds the API key service configuration
type Config struct {
	DefaultRateLimit int           `json:"default_rate_limit"`
	MaxRateLimit     int           `json:"max_rate_limit"`
	DefaultExpiry    time.Duration `json:"default_expiry"`
	MaxKeysPerUser   int           `json:"max_keys_per_user"`
	MaxKeysPerOrg    int           `json:"max_keys_per_org"`
	KeyLength        int           `json:"key_length"`
}

// Service handles API key operations
// Updated for V2 architecture: App → Environment → Organization
type Service struct {
	repo     Repository
	auditSvc *audit.Service
	config   Config
}

// NewService creates a new API key service
func NewService(repo Repository, auditSvc *audit.Service, cfg Config) *Service {
	// Set defaults
	if cfg.DefaultRateLimit == 0 {
		cfg.DefaultRateLimit = 1000
	}
	if cfg.MaxRateLimit == 0 {
		cfg.MaxRateLimit = 10000
	}
	if cfg.DefaultExpiry == 0 {
		cfg.DefaultExpiry = 365 * 24 * time.Hour // 1 year
	}
	if cfg.MaxKeysPerUser == 0 {
		cfg.MaxKeysPerUser = 10
	}
	if cfg.MaxKeysPerOrg == 0 {
		cfg.MaxKeysPerOrg = 100
	}
	if cfg.KeyLength == 0 {
		cfg.KeyLength = 32
	}

	return &Service{
		repo:     repo,
		auditSvc: auditSvc,
		config:   cfg,
	}
}

// CreateAPIKey creates a new API key
func (s *Service) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, error) {
	// Validate request
	if err := s.validateCreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// Check limits
	if err := s.checkLimits(ctx, req.AppID, req.UserID, req.OrgID); err != nil {
		return nil, err
	}

	// Generate key
	keyBytes := make([]byte, s.config.KeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, APIKeyCreationFailed(fmt.Errorf("failed to generate key: %w", err))
	}
	key := base64.URLEncoding.EncodeToString(keyBytes)

	// Generate prefix for identification
	prefix := s.generatePrefix(req.AppID, req.OrgID)

	// Hash the key for storage
	keyHash := s.hashKey(key)

	// Set defaults
	rateLimit := req.RateLimit
	if rateLimit == 0 {
		rateLimit = s.config.DefaultRateLimit
	}
	if rateLimit > s.config.MaxRateLimit {
		rateLimit = s.config.MaxRateLimit
	}

	expiresAt := req.ExpiresAt
	if expiresAt == nil {
		expiry := time.Now().Add(s.config.DefaultExpiry)
		expiresAt = &expiry
	}

	// Create API key schema
	apiKey := &schema.APIKey{
		ID:             xid.New(),
		AppID:          req.AppID,
		EnvironmentID:  req.EnvironmentID,
		OrganizationID: req.OrgID,
		UserID:         req.UserID,
		Name:           req.Name,
		Description:    req.Description,
		Prefix:         prefix,
		KeyHash:        keyHash,
		Scopes:         req.Scopes,
		Permissions:    req.Permissions,
		RateLimit:      rateLimit,
		AllowedIPs:     req.AllowedIPs,
		Active:         true,
		ExpiresAt:      expiresAt,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, APIKeyCreationFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &req.UserID, "api_key.created", "api_key:"+apiKey.ID.String(), "", "", fmt.Sprintf(`{"name":"%s","scopes":["%s"]}`, req.Name, strings.Join(req.Scopes, `","`)))

	// Convert to DTO and return the full key only once
	result := FromSchemaAPIKey(apiKey)
	result.Key = prefix + "." + key
	return result, nil
}

// VerifyAPIKey verifies an API key and returns the associated key info
func (s *Service) VerifyAPIKey(ctx context.Context, req *VerifyAPIKeyRequest) (*VerifyAPIKeyResponse, error) {
	parts := strings.Split(req.Key, ".")
	if len(parts) != 2 {
		err := InvalidKeyFormat()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: err.Message,
		}, nil
	}

	prefix := parts[0]
	keyPart := parts[1]

	// Find by prefix
	apiKey, err := s.repo.FindAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		notFoundErr := APIKeyNotFound()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: notFoundErr.Message,
		}, nil
	}

	// Verify hash
	if !s.verifyKeyHash(keyPart, apiKey.KeyHash) {
		invalidErr := InvalidAPIKeyHash()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: invalidErr.Message,
		}, nil
	}

	// Check if active
	if !apiKey.Active {
		inactiveErr := APIKeyInactive()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: inactiveErr.Message,
		}, nil
	}

	// Check expiration
	if apiKey.IsExpired() {
		expiredErr := APIKeyExpired()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: expiredErr.Message,
		}, nil
	}

	// Check scope if required
	if req.RequiredScope != "" && !apiKey.HasScope(req.RequiredScope) {
		scopeErr := InsufficientScope(req.RequiredScope)
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: scopeErr.Message,
		}, nil
	}

	// Check permission if required
	if req.RequiredPermission != "" && !apiKey.HasPermission(req.RequiredPermission) {
		permErr := InsufficientPermission(req.RequiredPermission)
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: permErr.Message,
		}, nil
	}

	// Update usage
	if err := s.repo.UpdateAPIKeyUsage(ctx, apiKey.ID, req.IP, req.UserAgent); err != nil {
		// Log error but don't fail verification
		fmt.Printf("Failed to update API key usage: %v\n", err)
	}

	return &VerifyAPIKeyResponse{
		Valid:  true,
		APIKey: FromSchemaAPIKey(apiKey),
	}, nil
}

// GetAPIKey retrieves an API key by ID
func (s *Service) GetAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID) (*APIKey, error) {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership - app context
	if apiKey.AppID != appID {
		return nil, AccessDenied("wrong app")
	}

	// Check ownership - user context
	if apiKey.UserID != userID {
		return nil, AccessDenied("wrong user")
	}

	// Check ownership - org context (if org-scoped)
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return nil, AccessDenied("wrong organization")
		}
	}

	return FromSchemaAPIKey(apiKey), nil
}

// ListAPIKeys lists API keys with filtering and pagination
func (s *Service) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) (*ListAPIKeysResponse, error) {
	// Get paginated results from repository (returns schema types)
	pageResp, err := s.repo.ListAPIKeys(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema types to DTOs
	dtoKeys := FromSchemaAPIKeys(pageResp.Data)

	// Return paginated response with DTOs
	return &ListAPIKeysResponse{
		Data:       dtoKeys,
		Pagination: pageResp.Pagination,
	}, nil
}

// UpdateAPIKey updates an API key
func (s *Service) UpdateAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID, req *UpdateAPIKeyRequest) (*APIKey, error) {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership
	if apiKey.AppID != appID {
		return nil, AccessDenied("wrong app")
	}
	if apiKey.UserID != userID {
		return nil, AccessDenied("wrong user")
	}
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return nil, AccessDenied("wrong organization")
		}
	}

	// Update fields
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Description != nil {
		apiKey.Description = *req.Description
	}
	if req.Scopes != nil {
		apiKey.Scopes = req.Scopes
	}
	if req.Permissions != nil {
		apiKey.Permissions = req.Permissions
	}
	if req.RateLimit != nil {
		rateLimit := *req.RateLimit
		if rateLimit > s.config.MaxRateLimit {
			rateLimit = s.config.MaxRateLimit
		}
		apiKey.RateLimit = rateLimit
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if req.Active != nil {
		apiKey.Active = *req.Active
	}
	if req.Metadata != nil {
		apiKey.Metadata = req.Metadata
	}

	if err := s.repo.UpdateAPIKey(ctx, apiKey); err != nil {
		return nil, APIKeyUpdateFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &userID, "api_key.updated", "api_key:"+id.String(), "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return FromSchemaAPIKey(apiKey), nil
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID) error {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return APIKeyNotFound()
	}

	// Check ownership
	if apiKey.AppID != appID {
		return AccessDenied("wrong app")
	}
	if apiKey.UserID != userID {
		return AccessDenied("wrong user")
	}
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return AccessDenied("wrong organization")
		}
	}

	if err := s.repo.DeleteAPIKey(ctx, id); err != nil {
		return APIKeyDeletionFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &userID, "api_key.deleted", "api_key:"+id.String(), "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return nil
}

// RotateAPIKey rotates an API key (creates a new key with same settings)
func (s *Service) RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*APIKey, error) {
	// Get existing key
	existingKey, err := s.repo.FindAPIKeyByID(ctx, req.ID)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership
	if existingKey.AppID != req.AppID {
		return nil, AccessDenied("wrong app")
	}
	if existingKey.UserID != req.UserID {
		return nil, AccessDenied("wrong user")
	}
	if req.OrganizationID != nil && !req.OrganizationID.IsNil() {
		if existingKey.OrganizationID == nil || *existingKey.OrganizationID != *req.OrganizationID {
			return nil, AccessDenied("wrong organization")
		}
	}

	// Create new key with same settings
	createReq := &CreateAPIKeyRequest{
		AppID:         existingKey.AppID,
		EnvironmentID: existingKey.EnvironmentID,
		OrgID:         existingKey.OrganizationID,
		UserID:        existingKey.UserID,
		Name:          existingKey.Name,
		Description:   existingKey.Description,
		Scopes:        existingKey.Scopes,
		Permissions:   existingKey.Permissions,
		RateLimit:     existingKey.RateLimit,
		ExpiresAt:     req.ExpiresAt,
		Metadata:      existingKey.Metadata,
	}

	newKey, err := s.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, APIKeyRotationFailed(err)
	}

	// Deactivate old key
	if err := s.repo.DeactivateAPIKey(ctx, req.ID); err != nil {
		// Log error but don't fail rotation
		fmt.Printf("Failed to deactivate old API key: %v\n", err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &req.UserID, "api_key.rotated", "api_key:"+req.ID.String(), "", "", fmt.Sprintf(`{"name":"%s","old_key_id":"%s","new_key_id":"%s"}`, existingKey.Name, req.ID.String(), newKey.ID.String()))

	return newKey, nil
}

// CleanupExpired removes expired API keys
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	return s.repo.CleanupExpiredAPIKeys(ctx)
}

// Helper methods

func (s *Service) validateCreateRequest(ctx context.Context, req *CreateAPIKeyRequest) error {
	if req.AppID.IsNil() {
		return MissingAppContext()
	}
	if req.EnvironmentID.IsNil() {
		return MissingEnvContext()
	}
	if req.UserID.IsNil() {
		return AccessDenied("user ID required")
	}
	if req.Name == "" {
		return AccessDenied("name is required")
	}
	if len(req.Scopes) == 0 {
		return AccessDenied("at least one scope is required")
	}
	return nil
}

func (s *Service) checkLimits(ctx context.Context, appID, userID xid.ID, orgID *xid.ID) error {
	// Check user limit (per app)
	userIDPtr := &userID
	userCount, err := s.repo.CountAPIKeys(ctx, appID, nil, nil, userIDPtr)
	if err != nil {
		return err
	}
	if userCount >= s.config.MaxKeysPerUser {
		return MaxKeysReached(s.config.MaxKeysPerUser)
	}

	// Check org limit (if org-scoped)
	if orgID != nil && !orgID.IsNil() {
		orgCount, err := s.repo.CountAPIKeys(ctx, appID, nil, orgID, nil)
		if err != nil {
			return err
		}
		if orgCount >= s.config.MaxKeysPerOrg {
			return MaxKeysReached(s.config.MaxKeysPerOrg)
		}
	}

	return nil
}

func (s *Service) generatePrefix(appID xid.ID, orgID *xid.ID) string {
	// Generate a short random suffix
	bytes := make([]byte, 4)
	rand.Read(bytes)
	suffix := base64.URLEncoding.EncodeToString(bytes)[:6]

	// Create prefix based on scope
	if orgID != nil && !orgID.IsNil() {
		// Org-scoped: ak_org_<suffix>
		orgShort := orgID.String()
		if len(orgShort) > 8 {
			orgShort = orgShort[:8]
		}
		return fmt.Sprintf("ak_org_%s_%s", orgShort, suffix)
	} else {
		// App-scoped: ak_app_<suffix>
		appShort := appID.String()
		if len(appShort) > 8 {
			appShort = appShort[:8]
		}
		return fmt.Sprintf("ak_app_%s_%s", appShort, suffix)
	}
}

func (s *Service) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (s *Service) verifyKeyHash(key, hash string) bool {
	return s.hashKey(key) == hash
}
