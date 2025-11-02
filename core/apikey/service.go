package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/repository"
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
type Service struct {
	repo     *repository.APIKeyRepository
	auditSvc *audit.Service
	config   Config
}

// NewService creates a new API key service
func NewService(repo *repository.APIKeyRepository, auditSvc *audit.Service, cfg Config) *Service {
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
	if err := s.checkLimits(ctx, req.UserID, req.OrgID); err != nil {
		return nil, err
	}

	// Generate key
	keyBytes := make([]byte, s.config.KeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	key := base64.URLEncoding.EncodeToString(keyBytes)

	// Generate prefix for identification
	prefix := s.generatePrefix(req.OrgID)

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

	// Create API key
	apiKey := &schema.APIKey{
		ID:          xid.New(),
		OrgID:       req.OrgID,
		UserID:      req.UserID,
		Name:        req.Name,
		Description: req.Description,
		Prefix:      prefix,
		KeyHash:     keyHash,
		Scopes:      req.Scopes,
		Permissions: req.Permissions,
		RateLimit:   rateLimit,
		AllowedIPs:  req.AllowedIPs,
		Active:      true,
		ExpiresAt:   expiresAt,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Audit log
	userID, _ := xid.FromString(req.UserID)
	_ = s.auditSvc.Log(ctx, &userID, "api_key.created", "api_key:"+apiKey.ID.String(), "", "", fmt.Sprintf(`{"name":"%s","scopes":["%s"]}`, req.Name, strings.Join(req.Scopes, `","`)))

	// Convert to domain model and return the full key only once
	result := s.schemaToAPIKey(apiKey)
	result.Key = prefix + "." + key
	return result, nil
}

// VerifyAPIKey verifies an API key and returns the associated key info
func (s *Service) VerifyAPIKey(ctx context.Context, req *VerifyAPIKeyRequest) (*VerifyAPIKeyResponse, error) {
	parts := strings.Split(req.Key, ".")
	if len(parts) != 2 {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "invalid key format",
		}, nil
	}

	prefix := parts[0]
	keyPart := parts[1]

	// Find by prefix
	apiKey, err := s.repo.FindByPrefix(ctx, prefix)
	if err != nil {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "key not found",
		}, nil
	}

	// Verify hash
	if !s.verifyKeyHash(keyPart, apiKey.KeyHash) {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "invalid key",
		}, nil
	}

	// Check if active
	if !apiKey.Active {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "key deactivated",
		}, nil
	}

	// Check expiration
	if apiKey.IsExpired() {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "key expired",
		}, nil
	}

	// Check scope if required
	if req.RequiredScope != "" && !apiKey.HasScope(req.RequiredScope) {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "insufficient scope",
		}, nil
	}

	// Check permission if required
	if req.RequiredPermission != "" && !apiKey.HasPermission(req.RequiredPermission) {
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: "insufficient permission",
		}, nil
	}

	// Update usage
	if err := s.repo.UpdateUsage(ctx, apiKey.ID.String(), req.IP, req.UserAgent); err != nil {
		// Log error but don't fail verification
		fmt.Printf("Failed to update API key usage: %v\n", err)
	}

	return &VerifyAPIKeyResponse{
		Valid:  true,
		APIKey: s.schemaToAPIKey(apiKey),
	}, nil
}

// GetAPIKey retrieves an API key by ID
func (s *Service) GetAPIKey(ctx context.Context, id, userID, orgID string) (*APIKey, error) {
	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API key not found: %w", err)
	}

	// Check ownership
	if apiKey.UserID != userID || apiKey.OrgID != orgID {
		return nil, errors.New("access denied")
	}

	return s.schemaToAPIKey(apiKey), nil
}

// ListAPIKeys lists API keys for a user or organization
func (s *Service) ListAPIKeys(ctx context.Context, req *ListAPIKeysRequest) (*ListAPIKeysResponse, error) {
	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 50
	}

	var apiKeys []*schema.APIKey
	var total int
	var err error

	if req.UserID != "" {
		apiKeys, err = s.repo.FindByUserID(ctx, req.UserID, limit, req.Offset)
		if err != nil {
			return nil, fmt.Errorf("failed to list API keys: %w", err)
		}
		total, err = s.repo.CountByUserID(ctx, req.UserID)
	} else if req.OrgID != "" {
		apiKeys, err = s.repo.FindByOrgID(ctx, req.OrgID, limit, req.Offset)
		if err != nil {
			return nil, fmt.Errorf("failed to list API keys: %w", err)
		}
		total, err = s.repo.CountByOrgID(ctx, req.OrgID)
	} else {
		return nil, errors.New("either user_id or org_id must be provided")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Convert to domain models
	result := make([]*APIKey, len(apiKeys))
	for i, key := range apiKeys {
		result[i] = s.schemaToAPIKey(key)
	}

	return &ListAPIKeysResponse{
		APIKeys: result,
		Total:   total,
		Limit:   limit,
		Offset:  req.Offset,
	}, nil
}

// UpdateAPIKey updates an API key
func (s *Service) UpdateAPIKey(ctx context.Context, id, userID, orgID string, req *UpdateAPIKeyRequest) (*APIKey, error) {
	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("API key not found: %w", err)
	}

	// Check ownership
	if apiKey.UserID != userID || apiKey.OrgID != orgID {
		return nil, errors.New("access denied")
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

	if err := s.repo.Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	// Audit log
	uid, _ := xid.FromString(userID)
	_ = s.auditSvc.Log(ctx, &uid, "api_key.updated", "api_key:"+id, "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return s.schemaToAPIKey(apiKey), nil
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(ctx context.Context, id, userID, orgID string) error {
	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	// Check ownership
	if apiKey.UserID != userID || apiKey.OrgID != orgID {
		return errors.New("access denied")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	// Audit log
	uid, _ := xid.FromString(userID)
	_ = s.auditSvc.Log(ctx, &uid, "api_key.deleted", "api_key:"+id, "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return nil
}

// RotateAPIKey rotates an API key (creates a new key with same settings)
func (s *Service) RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*APIKey, error) {
	// Get existing key
	existingKey, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("API key not found: %w", err)
	}

	// Check ownership
	if existingKey.UserID != req.UserID || existingKey.OrgID != req.OrgID {
		return nil, errors.New("access denied")
	}

	// Create new key with same settings
	createReq := &CreateAPIKeyRequest{
		OrgID:       existingKey.OrgID,
		UserID:      existingKey.UserID,
		Name:        existingKey.Name,
		Description: existingKey.Description,
		Scopes:      existingKey.Scopes,
		Permissions: existingKey.Permissions,
		RateLimit:   existingKey.RateLimit,
		ExpiresAt:   req.ExpiresAt,
		Metadata:    existingKey.Metadata,
	}

	newKey, err := s.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create new API key: %w", err)
	}

	// Deactivate old key
	if err := s.repo.Deactivate(ctx, req.ID); err != nil {
		// Log error but don't fail rotation
		fmt.Printf("Failed to deactivate old API key: %v\n", err)
	}

	// Audit log
	userID, _ := xid.FromString(req.UserID)
	_ = s.auditSvc.Log(ctx, &userID, "api_key.rotated", "api_key:"+req.ID, "", "", fmt.Sprintf(`{"name":"%s","old_key_id":"%s","new_key_id":"%s"}`, existingKey.Name, req.ID, newKey.ID.String()))

	return newKey, nil
}

// CleanupExpired removes expired API keys
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	return s.repo.CleanupExpired(ctx)
}

// Helper methods

func (s *Service) validateCreateRequest(ctx context.Context, req *CreateAPIKeyRequest) error {
	if req.OrgID == "" {
		return errors.New("org_id is required")
	}
	if req.UserID == "" {
		return errors.New("user_id is required")
	}
	if req.Name == "" {
		return errors.New("name is required")
	}
	if len(req.Scopes) == 0 {
		return errors.New("at least one scope is required")
	}
	return nil
}

func (s *Service) checkLimits(ctx context.Context, userID, orgID string) error {
	// Check user limit
	userCount, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check user limit: %w", err)
	}
	if userCount >= s.config.MaxKeysPerUser {
		return fmt.Errorf("user has reached maximum API key limit (%d)", s.config.MaxKeysPerUser)
	}

	// Check org limit
	orgCount, err := s.repo.CountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to check org limit: %w", err)
	}
	if orgCount >= s.config.MaxKeysPerOrg {
		return fmt.Errorf("organization has reached maximum API key limit (%d)", s.config.MaxKeysPerOrg)
	}

	return nil
}

func (s *Service) generatePrefix(orgID string) string {
	// Generate a short random suffix
	bytes := make([]byte, 4)
	rand.Read(bytes)
	suffix := base64.URLEncoding.EncodeToString(bytes)[:6]

	// Create prefix: ak_<org_short>_<random>
	orgShort := orgID
	if len(orgShort) > 8 {
		orgShort = orgShort[:8]
	}

	return fmt.Sprintf("ak_%s_%s", orgShort, suffix)
}

func (s *Service) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (s *Service) verifyKeyHash(key, hash string) bool {
	return s.hashKey(key) == hash
}

func (s *Service) schemaToAPIKey(schema *schema.APIKey) *APIKey {
	return &APIKey{
		ID:          schema.ID,
		OrgID:       schema.OrgID,
		UserID:      schema.UserID,
		Name:        schema.Name,
		Description: schema.Description,
		Prefix:      schema.Prefix,
		Scopes:      schema.Scopes,
		Permissions: schema.Permissions,
		RateLimit:   schema.RateLimit,
		AllowedIPs:  schema.AllowedIPs,
		Active:      schema.Active,
		ExpiresAt:   schema.ExpiresAt,
		UsageCount:  schema.UsageCount,
		LastUsedAt:  schema.LastUsedAt,
		LastUsedIP:  schema.LastUsedIP,
		LastUsedUA:  schema.LastUsedUA,
		CreatedAt:   schema.CreatedAt,
		UpdatedAt:   schema.UpdatedAt,
		Metadata:    schema.Metadata,
	}
}
